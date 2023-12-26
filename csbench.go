// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package main

import (
	"csbench/domain"
	"csbench/network"
	"csbench/vm"
	"csbench/volume"
	"flag"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	"csbench/apirunner"
	"csbench/config"

	log "github.com/sirupsen/logrus"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/montanaflynn/stats"
	"github.com/sourcegraph/conc/pool"
)

var (
	profiles = make(map[int]*config.Profile)
)

type Result struct {
	Success  bool
	Duration float64
}

func init() {
	logFile, err := os.OpenFile("csmetrics.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to create log file: %v", err)
	}

	mw := io.MultiWriter(os.Stdout, logFile)

	log.SetOutput(mw)
}

func readConfigurations(configFile string) map[int]*config.Profile {
	profiles, err := config.ReadProfiles(configFile)
	if err != nil {
		log.Fatal("Error reading profiles:", err)
	}

	return profiles
}

func logConfigurationDetails(profiles map[int]*config.Profile) {
	apiURL := config.URL
	iterations := config.Iterations
	page := config.Page
	pagesize := config.PageSize
	host := config.Host

	userProfileNames := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		userProfileNames = append(userProfileNames, profile.Name)
	}

	fmt.Printf("\n\n\033[1;34mBenchmarking the CloudStack environment [%s] with the following configuration\033[0m\n\n", apiURL)
	fmt.Printf("Management server : %s\n", host)
	fmt.Printf("Roles : %s\n", strings.Join(userProfileNames, ","))
	fmt.Printf("Iterations : %d\n", iterations)
	fmt.Printf("Page : %d\n", page)
	fmt.Printf("PageSize : %d\n\n", pagesize)

	log.Infof("Found %d profiles in the configuration: ", len(profiles))
	log.Infof("Management server : %s", host)
}

func logReport() {
	fmt.Printf("\n\n\nLog file : csmetrics.log\n")
	fmt.Printf("Reports directory per API : report/%s/\n", config.Host)
	fmt.Printf("Number of APIs : %d\n", apirunner.APIscount)
	fmt.Printf("Successful APIs : %d\n", apirunner.SuccessAPIs)
	fmt.Printf("Failed APIs : %d\n", apirunner.FailedAPIs)
	fmt.Printf("Time in seconds per API: %.2f (avg)\n", apirunner.TotalTime/float64(apirunner.APIscount))
	fmt.Printf("\n\n\033[1;34m--------------------------------------------------------------------------------\033[0m\n" +
		"                            Done with benchmarking\n" +
		"\033[1;34m--------------------------------------------------------------------------------\033[0m\n\n")
}

func getSamples(results []*Result) (stats.Float64Data, stats.Float64Data, stats.Float64Data) {
	var allExecutionsSample stats.Float64Data
	var successfulExecutionSample stats.Float64Data
	var failedExecutionSample stats.Float64Data

	for _, result := range results {
		duration := math.Round(result.Duration*1000) / 1000
		allExecutionsSample = append(allExecutionsSample, duration)
		if result.Success {
			successfulExecutionSample = append(successfulExecutionSample, duration)
		} else {
			failedExecutionSample = append(failedExecutionSample, duration)
		}
	}

	return allExecutionsSample, successfulExecutionSample, failedExecutionSample
}

func getRowFromSample(key string, sample stats.Float64Data) table.Row {
	min, _ := sample.Min()
	min = math.Round(min*1000) / 1000
	max, _ := sample.Max()
	max = math.Round(max*1000) / 1000
	mean, _ := sample.Mean()
	mean = math.Round(mean*1000) / 1000
	median, _ := sample.Median()
	median = math.Round(median*1000) / 1000
	percentile90, _ := sample.Percentile(90)
	percentile90 = math.Round(percentile90*1000) / 1000
	percentile95, _ := sample.Percentile(95)
	percentile95 = math.Round(percentile95*1000) / 1000
	percentile99, _ := sample.Percentile(99)
	percentile99 = math.Round(percentile99*1000) / 1000

	return table.Row{key, len(sample), min, max, mean, median, percentile90, percentile95, percentile99}
}

/*
This function will generate a report with the following details:
 1. Total Number of executions
 2. Number of successful executions
 3. Number of failed exections
 4. Different statistics like min, max, avg, median, 90th percentile, 95th percentile, 99th percentile for above 3

Output format:
 1. CSV
 2. TSV
 3. Table
*/
func generateReport(results map[string][]*Result, format string, outputFile string) {
	fmt.Println("Generating report")

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Type", "Count", "Min", "Max", "Avg", "Median", "90th percentile", "95th percentile", "99th percentile"})

	for key, result := range results {
		allExecutionsSample, successfulExecutionSample, failedExecutionSample := getSamples(result)
		t.AppendRow(getRowFromSample(fmt.Sprintf("%s - All", key), allExecutionsSample))

		if failedExecutionSample.Len() != 0 {
			t.AppendRow(getRowFromSample(fmt.Sprintf("%s - Successful", key), successfulExecutionSample))
			t.AppendRow(getRowFromSample(fmt.Sprintf("%s - Failed", key), failedExecutionSample))
		}
	}

	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			log.Error("Error creating file: ", err)
		}
		defer f.Close()
		t.SetOutputMirror(f)
	}
	switch format {
	case "csv":
		t.RenderCSV()
	case "tsv":
		t.RenderTSV()
	case "table":
		t.Render()
	}
}

func main() {
	dbprofile := flag.Int("dbprofile", 0, "DB profile number")
	create := flag.Bool("create", false, "Create resources")
	benchmark := flag.Bool("benchmark", false, "Benchmark list APIs")
	domainFlag := flag.Bool("domain", false, "Works with -create & -teardown\n\t"+
		"-create - Create subdomains and accounts\n\t"+
		"-teardown - Delete all subdomains and accounts")
	limitsFlag := flag.Bool("limits", false, "Update limits to -1 for subdomains and accounts")
	networkFlag := flag.Bool("network", false, "Works with -create & -teardown\n\t"+
		"-create - Create shared network in all subdomains\n\t"+
		"-teardown - Delete all networks in the subdomains")
	vmFlag := flag.Bool("vm", false, "Works with -create & -teardown\n\t"+
		"-create - Deploy VMs in all networks in the subdomains\n\t"+
		"-teardown - Delete all VMs in the subdomains")
	volumeFlag := flag.Bool("volume", false, "Works with -create & -teardown\n\t"+
		"-create - Create and attach Volumes to VMs\n\t"+
		"-teardown - Delete all volumes in the subdomains")
	vmAction := flag.String("vmaction", "", "Action to perform on VMs. Options:\n\t"+
		"start - start all VMs\n\t"+
		"stop - stop all VMs\n\t"+
		"reboot - reboot all running VMs\n\t"+
		"toggle - stop running VMs and start stopped VMs\n\t"+
		"random - Randomly toggle VMs")
	tearDown := flag.Bool("teardown", false, "Tear down resources")
	workers := flag.Int("workers", 10, "Number of workers to use while creating resources")
	format := flag.String("format", "table", "Format of the report (csv, tsv, table). Valid only for create")
	outputFile := flag.String("output", "", "Path to output file. Valid only for create")
	configFile := flag.String("config", "config/config", "Path to config file")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if !(*create || *benchmark || *tearDown || *vmAction != "") {
		log.Fatal("Please provide one of the following options: -create, -benchmark, -vmaction, -teardown")
	}

	if *create && *tearDown && *vmAction == "" {
		log.Fatal("Please provide only one of the following options: -create, -teardown, -vmaction")
	}

	if *vmAction != "" && !(*vmAction == "start" || *vmAction == "stop" || *vmAction == "reboot" || *vmAction == "toggle" || *vmAction == "random") {
		log.Fatal("Invalid VM action. Please provide one of the following: start, stop, reboot, toggle, random")
	}

	if *create && !(*domainFlag || *limitsFlag || *networkFlag || *vmFlag || *volumeFlag) {
		log.Fatal("Please provide one of the following options with create: -domain, -limits, -network, -vm, -volume")
	}

	if *tearDown && !(*domainFlag || *networkFlag || *vmFlag || *volumeFlag) {
		log.Fatal("Please provide one of the following options with teardown: -domain, -limits, -network, -vm, -volume")
	}

	switch *format {
	case "csv", "tsv", "table":
		// valid format, continue
	default:
		log.Fatal("Invalid format. Please provide one of the following: csv, tsv, table")
	}

	if *dbprofile < 0 {
		log.Fatal("Invalid DB profile number. Please provide a positive integer.")
	}

	profiles = readConfigurations(*configFile)
	apiURL := config.URL
	iterations := config.Iterations
	page := config.Page
	pagesize := config.PageSize

	if *create {
		results := createResources(domainFlag, limitsFlag, networkFlag, vmFlag, volumeFlag, workers)
		generateReport(results, *format, *outputFile)
	}

	if *vmAction != "" {
		results := executeVMAction(vmAction, workers)
		generateReport(results, *format, *outputFile)
	}

	if *tearDown {
		results := tearDownEnv(domainFlag, networkFlag, vmFlag, volumeFlag, workers)
		generateReport(results, *format, *outputFile)
	}

	if *benchmark {
		log.Infof("\nStarted benchmarking the CloudStack environment [%s]", apiURL)

		logConfigurationDetails(profiles)

		for i, profile := range profiles {
			userProfileName := profile.Name
			log.Infof("Using profile %d.%s for benchmarking", i, userProfileName)
			fmt.Printf("\n\033[1;34m============================================================\033[0m\n")
			fmt.Printf("                    Profile: [%s]\n", userProfileName)
			fmt.Printf("\033[1;34m============================================================\033[0m\n")
			apirunner.RunAPIs(userProfileName, apiURL, profile.ApiKey, profile.SecretKey, profile.Expires, profile.SignatureVersion, iterations, page, pagesize, *dbprofile)
		}
		logReport()

		log.Infof("Done with benchmarking the CloudStack environment [%s]", apiURL)
	}

}

func executeVMAction(vmAction *string, workers *int) map[string][]*Result {

	parentDomainId := config.ParentDomainId
	var cs *cloudstack.CloudStackClient
	workerPool := pool.NewWithResults[map[string]*Result]().WithMaxGoroutines(*workers)
	for _, profile := range profiles {
		if profile.Name == "admin" {
			cs = cloudstack.NewAsyncClient(config.URL, profile.ApiKey, profile.SecretKey, false)
		}
	}

	if cs == nil {
		log.Fatal("Failed to find admin profile")
	}

	log.Infof("Fetching all VMs in subdomains for domain %s", parentDomainId)
	domains := domain.ListSubDomains(cs, config.ParentDomainId)
	var allVMs []*cloudstack.VirtualMachine
	for _, dmn := range domains {
		vms, err := vm.ListVMs(cs, dmn.Id)
		if err != nil {
			log.Warn("Error listing VMs: ", err)
			continue
		}
		allVMs = append(allVMs, vms...)
	}

	progressMarker := int(math.Ceil(float64(len(allVMs)) / 10.0))
	start := time.Now()

	for i, virtualMachine := range allVMs {
		virtualMachine := virtualMachine

		if (i+1)%progressMarker == 0 {
			log.Infof("Executed %d VMs", i+1)
		}

		if *vmAction == "random" && rand.Intn(100) < 50 {
			continue
		}

		workerPool.Go(func() map[string]*Result {
			taskStart := time.Now()
			result := false
			action := "skipped"
			switch virtualMachine.State {
			case "Running":
				if *vmAction == "stop" || *vmAction == "toggle" || *vmAction == "random" {
					err := vm.StopVM(cs, virtualMachine.Id)
					result = err == nil
					action = "stop"
				} else if *vmAction == "reboot" {
					err := vm.RebootVM(cs, virtualMachine.Id)
					result = err == nil
					action = "reboot"
				}
			case "Stopped":
				if *vmAction == "start" || *vmAction == "toggle" || *vmAction == "random" {
					err := vm.StartVM(cs, virtualMachine.Id)
					result = err == nil
					action = "start"
				} else if *vmAction == "reboot" {
					result = false
					action = "stop"
				}
			}
			return map[string]*Result{
				action: {
					Success:  result,
					Duration: time.Since(taskStart).Seconds(),
				},
			}
		})
	}
	res := workerPool.Wait()
	log.Infof("Executed %s on %d VMs in %.2f seconds", *vmAction, len(allVMs), time.Since(start).Seconds())
	var results = make(map[string][]*Result)
	for _, result := range res {
		for key, value := range result {
			key = "vmaction-" + key
			if results[key] == nil {
				results[key] = make([]*Result, 0)
			}
			results[key] = append(results[key], value)
		}
	}
	return results
}

func createResources(domainFlag, limitsFlag, networkFlag, vmFlag, volumeFlag *bool, workers *int) map[string][]*Result {
	apiURL := config.URL

	for _, profile := range profiles {
		if profile.Name == "admin" {

			numNetworksPerDomain := config.NumNetworks
			numVmsPerNetwork := config.NumVms
			numVolumesPerVM := config.NumVolumes

			cs := cloudstack.NewAsyncClient(apiURL, profile.ApiKey, profile.SecretKey, false)

			var results = make(map[string][]*Result)

			if *domainFlag {
				workerPool := pool.NewWithResults[*Result]().WithMaxGoroutines(*workers)
				results["domain"] = createDomains(workerPool, cs, config.ParentDomainId, config.NumDomains)
			}

			if *limitsFlag {
				workerPool := pool.NewWithResults[*Result]().WithMaxGoroutines(*workers)
				results["limits"] = updateLimits(workerPool, cs, config.ParentDomainId)
			}

			if *networkFlag {
				workerPool := pool.NewWithResults[*Result]().WithMaxGoroutines(*workers)
				results["network"] = createNetwork(workerPool, cs, config.ParentDomainId, numNetworksPerDomain)
			}

			if *vmFlag {
				workerPool := pool.NewWithResults[*Result]().WithMaxGoroutines(*workers)
				results["vm"] = createVms(workerPool, cs, config.ParentDomainId, numVmsPerNetwork)
			}

			if *volumeFlag {
				workerPool := pool.NewWithResults[*Result]().WithMaxGoroutines(*workers)
				results["volume"] = createVolumes(workerPool, cs, config.ParentDomainId, numVolumesPerVM)
			}

			return results
		}
	}
	return nil
}

func createDomains(workerPool *pool.ResultPool[*Result], cs *cloudstack.CloudStackClient, parentDomainId string, count int) []*Result {
	progressMarker := int(math.Ceil(float64(count) / 10.0))
	start := time.Now()
	log.Infof("Creating %d domains", count)
	for i := 0; i < count; i++ {
		if (i+1)%progressMarker == 0 {
			log.Infof("Created %d domains", i+1)
		}
		workerPool.Go(func() *Result {
			taskStart := time.Now()
			dmn, err := domain.CreateDomain(cs, parentDomainId)
			if err != nil {
				return &Result{
					Success:  false,
					Duration: time.Since(taskStart).Seconds(),
				}
			}
			_, err = domain.CreateAccount(cs, dmn.Id)
			if err != nil {
				return &Result{
					Success:  false,
					Duration: time.Since(taskStart).Seconds(),
				}
			}

			return &Result{
				Success:  true,
				Duration: time.Since(taskStart).Seconds(),
			}
		})
	}
	res := workerPool.Wait()
	log.Infof("Created %d domains in %.2f seconds", count, time.Since(start).Seconds())
	return res
}

func updateLimits(workerPool *pool.ResultPool[*Result], cs *cloudstack.CloudStackClient, parentDomainId string) []*Result {
	log.Infof("Fetching subdomains for domain %s", parentDomainId)
	domains := domain.ListSubDomains(cs, parentDomainId)
	accounts := make([]*cloudstack.Account, 0)
	for _, dmn := range domains {
		accounts = append(accounts, domain.ListAccounts(cs, dmn.Id)...)
	}

	progressMarker := int(math.Ceil(float64(len(accounts)) / 10.0))
	start := time.Now()
	log.Infof("Updating limits for %d accounts", len(accounts))
	for i, account := range accounts {
		if (i+1)%progressMarker == 0 {
			log.Infof("Updated limits for %d accounts", i+1)
		}
		account := account
		workerPool.Go(func() *Result {
			taskStart := time.Now()
			resp := domain.UpdateLimits(cs, account)
			return &Result{
				Success:  resp,
				Duration: time.Since(taskStart).Seconds(),
			}
		})
	}
	res := workerPool.Wait()
	log.Infof("Updated limits for %d accounts in %.2f seconds", len(accounts), time.Since(start).Seconds())
	return res
}

func createNetwork(workerPool *pool.ResultPool[*Result], cs *cloudstack.CloudStackClient, parentDomainId string, numNetworkPerDomain int) []*Result {
	log.Infof("Fetching subdomains for domain %s", parentDomainId)
	domains := domain.ListSubDomains(cs, parentDomainId)

	progressMarker := int(math.Ceil(float64(len(domains)*numNetworkPerDomain) / 10.0))
	start := time.Now()
	log.Infof("Creating %d networks", len(domains))
	for i, dmn := range domains {
		for j := 1; j <= numNetworkPerDomain; j++ {
			counter := i*j + j
			dmn := dmn

			if counter%progressMarker == 0 {
				log.Infof("Created %d networks", counter)
			}

			workerPool.Go(func() *Result {
				taskStart := time.Now()
				_, err := network.CreateNetwork(cs, dmn.Id, counter-1)
				if err != nil {
					return &Result{
						Success:  false,
						Duration: time.Since(taskStart).Seconds(),
					}
				}
				return &Result{
					Success:  true,
					Duration: time.Since(taskStart).Seconds(),
				}
			})
		}
	}
	res := workerPool.Wait()
	log.Infof("Created %d networks in %.2f seconds", len(domains), time.Since(start).Seconds())
	return res
}

func createVms(workerPool *pool.ResultPool[*Result], cs *cloudstack.CloudStackClient, parentDomainId string, numVmPerNetwork int) []*Result {
	log.Infof("Fetching subdomains & accounts for domain %s", parentDomainId)
	domains := domain.ListSubDomains(cs, parentDomainId)
	var accounts []*cloudstack.Account
	for i := 0; i < len(domains); i++ {
		account := domain.ListAccounts(cs, domains[i].Id)
		accounts = append(accounts, account...)
	}

	domainIdAccountMapping := make(map[string]*cloudstack.Account)
	for _, account := range accounts {
		domainIdAccountMapping[account.Domainid] = account
	}

	log.Infof("Fetching networks for subdomains in domain %s", parentDomainId)
	var allNetworks []*cloudstack.Network
	for _, domain := range domains {
		network, _ := network.ListNetworks(cs, domain.Id)
		allNetworks = append(allNetworks, network...)
	}

	progressMarker := int(math.Ceil(float64(len(allNetworks)*numVmPerNetwork) / 10.0))
	start := time.Now()
	log.Infof("Creating %d VMs", len(allNetworks)*numVmPerNetwork)
	for i, network := range allNetworks {
		network := network
		for j := 1; j <= numVmPerNetwork; j++ {

			if (i*j+j)%progressMarker == 0 {
				log.Infof("Created %d VMs", i*j+j)
			}
			workerPool.Go(func() *Result {
				taskStart := time.Now()
				_, err := vm.DeployVm(cs, network.Domainid, network.Id, domainIdAccountMapping[network.Domainid].Name)
				if err != nil {
					return &Result{
						Success:  false,
						Duration: time.Since(taskStart).Seconds(),
					}
				}
				return &Result{
					Success:  true,
					Duration: time.Since(taskStart).Seconds(),
				}
			})
		}
	}
	res := workerPool.Wait()
	log.Infof("Created %d VMs in %.2f seconds", len(allNetworks)*numVmPerNetwork, time.Since(start).Seconds())
	return res
}

func createVolumes(workerPool *pool.ResultPool[*Result], cs *cloudstack.CloudStackClient, parentDomainId string, numVolumesPerVM int) []*Result {
	log.Infof("Fetching all VMs in subdomains for domain %s", parentDomainId)
	domains := domain.ListSubDomains(cs, parentDomainId)
	var allVMs []*cloudstack.VirtualMachine
	for _, dmn := range domains {
		vms, err := vm.ListVMs(cs, dmn.Id)
		if err != nil {
			log.Warn("Error listing VMs: ", err)
			continue
		}
		allVMs = append(allVMs, vms...)
	}

	progressMarker := int(math.Ceil(float64(len(allVMs)*numVolumesPerVM) / 10.0))
	start := time.Now()

	log.Infof("Creating %d volumes", len(allVMs)*numVolumesPerVM)
	unsuitableVmCount := 0

	for i, vm := range allVMs {
		vm := vm
		if vm.State != "Running" && vm.State != "Stopped" {
			unsuitableVmCount++
			continue
		}
		for j := 1; j <= numVolumesPerVM; j++ {
			if (i*j+j)%progressMarker == 0 {
				log.Infof("Created %d volumes", i*j+j)
			}

			workerPool.Go(func() *Result {
				taskStart := time.Now()
				vol, err := volume.CreateVolume(cs, vm.Domainid, vm.Account)
				if err != nil {
					return &Result{
						Success:  false,
						Duration: time.Since(taskStart).Seconds(),
					}
				}
				_, err = volume.AttachVolume(cs, vol.Id, vm.Id)
				if err != nil {
					return &Result{
						Success:  false,
						Duration: time.Since(taskStart).Seconds(),
					}
				}
				return &Result{
					Success:  true,
					Duration: time.Since(taskStart).Seconds(),
				}
			})
		}
	}
	if unsuitableVmCount > 0 {
		log.Warnf("Found %d VMs in unsuitable state", unsuitableVmCount)
	}
	res := workerPool.Wait()
	log.Infof("Created %d volumes in %.2f seconds", (len(allVMs)-unsuitableVmCount)*numVolumesPerVM, time.Since(start).Seconds())
	return res
}

func tearDownEnv(domainFlag, networkFlag, vmFlag, volumeFlag *bool, workers *int) map[string][]*Result {
	apiURL := config.URL

	for _, profile := range profiles {
		userProfileName := profile.Name
		if userProfileName == "admin" {
			cs := cloudstack.NewAsyncClient(apiURL, profile.ApiKey, profile.SecretKey, false)

			var results = make(map[string][]*Result)

			if *vmFlag {
				workerPool := pool.NewWithResults[*Result]().WithMaxGoroutines(*workers)
				results["vm-destroy"] = destroyVms(workerPool, cs, config.ParentDomainId)
			}

			if *volumeFlag {
				workerPool := pool.NewWithResults[*Result]().WithMaxGoroutines(*workers)
				results["volume-delete"] = deleteVolumes(workerPool, cs, config.ParentDomainId)
			}

			if *networkFlag {
				workerPool := pool.NewWithResults[*Result]().WithMaxGoroutines(*workers)
				results["network-delete"] = deleteNetworks(workerPool, cs, config.ParentDomainId)
			}

			if *domainFlag {
				workerPool := pool.NewWithResults[*Result]().WithMaxGoroutines(*workers)
				results["domain-delete"] = deleteDomains(workerPool, cs, config.ParentDomainId)
			}

			return results
		}
	}
	return nil
}

func destroyVms(workerPool *pool.ResultPool[*Result], cs *cloudstack.CloudStackClient, parentDomainId string) []*Result {
	log.Infof("Fetching subdomains & accounts for domain %s", parentDomainId)
	domains := domain.ListSubDomains(cs, parentDomainId)
	var allVMs []*cloudstack.VirtualMachine
	for _, dmn := range domains {
		vms, err := vm.ListVMs(cs, dmn.Id)
		if err != nil {
			log.Warn("Error listing VMs: ", err)
			continue
		}
		allVMs = append(allVMs, vms...)
	}

	progressMarker := int(math.Ceil(float64(len(allVMs)) / 10.0))
	start := time.Now()

	log.Infof("Destroying %d VMs", len(allVMs))

	for i, virtualMachine := range allVMs {
		virtualMachine := virtualMachine
		if i%progressMarker == 0 {
			log.Infof("Destroyed %d VMs", i+1)
		}

		workerPool.Go(func() *Result {
			taskStart := time.Now()
			err := vm.DestroyVm(cs, virtualMachine.Id)
			if err != nil {
				return &Result{
					Success:  false,
					Duration: time.Since(taskStart).Seconds(),
				}
			}
			return &Result{
				Success:  true,
				Duration: time.Since(taskStart).Seconds(),
			}
		})
	}

	res := workerPool.Wait()
	log.Infof("Destroyed %d VMs in %.2f seconds", len(allVMs), time.Since(start).Seconds())
	return res
}

func deleteNetworks(workerPool *pool.ResultPool[*Result], cs *cloudstack.CloudStackClient, parentDomainId string) []*Result {
	log.Infof("Fetching subdomains & accounts for domain %s", parentDomainId)
	domains := domain.ListSubDomains(cs, parentDomainId)

	log.Infof("Fetching networks for subdomains in domain %s", parentDomainId)
	var allNetworks []*cloudstack.Network
	for _, domain := range domains {
		network, _ := network.ListNetworks(cs, domain.Id)
		allNetworks = append(allNetworks, network...)
	}

	progressMarker := int(math.Ceil(float64(len(allNetworks)) / 10.0))
	start := time.Now()
	log.Infof("Deleting %d networks", len(allNetworks))
	for i, net := range allNetworks {
		net := net
		if (i+1)%progressMarker == 0 {
			log.Infof("Deleted %d networks", i+1)
		}
		workerPool.Go(func() *Result {
			taskStart := time.Now()
			resp, err := network.DeleteNetwork(cs, net.Id)
			if err != nil || !resp {
				return &Result{
					Success:  false,
					Duration: time.Since(taskStart).Seconds(),
				}
			}
			return &Result{
				Success:  true,
				Duration: time.Since(taskStart).Seconds(),
			}
		})
	}
	res := workerPool.Wait()
	log.Infof("Deleted %d networks in %.2f seconds", len(allNetworks), time.Since(start).Seconds())
	return res
}

func deleteVolumes(workerPool *pool.ResultPool[*Result], cs *cloudstack.CloudStackClient, parentDomainId string) []*Result {
	log.Infof("Fetching subdomains & accounts for domain %s", parentDomainId)
	domains := domain.ListSubDomains(cs, parentDomainId)

	log.Infof("Fetching volumes for subdomains in domain %s", parentDomainId)
	var allVolumes []*cloudstack.Volume
	for _, domain := range domains {
		volumes, _ := volume.ListVolumes(cs, domain.Id)
		allVolumes = append(allVolumes, volumes...)
	}

	progressMarker := int(math.Ceil(float64(len(allVolumes)) / 10.0))
	start := time.Now()
	log.Infof("Deleting %d volumes", len(allVolumes))
	for i, vol := range allVolumes {
		vol := vol
		if (i+1)%progressMarker == 0 {
			log.Infof("Deleted %d volumes", i+1)
		}
		workerPool.Go(func() *Result {
			taskStart := time.Now()
			_, err := volume.DestroyVolume(cs, vol.Id)
			if err != nil {
				return &Result{
					Success:  false,
					Duration: time.Since(taskStart).Seconds(),
				}
			}
			return &Result{
				Success:  true,
				Duration: time.Since(taskStart).Seconds(),
			}
		})
	}
	res := workerPool.Wait()
	log.Infof("Deleted %d volumes in %.2f seconds", len(allVolumes), time.Since(start).Seconds())
	return res
}

func deleteDomains(workerPool *pool.ResultPool[*Result], cs *cloudstack.CloudStackClient, parentDomainId string) []*Result {
	log.Infof("Fetching subdomains for domain %s", parentDomainId)
	domains := domain.ListSubDomains(cs, parentDomainId)

	progressMarker := int(math.Ceil(float64(len(domains)) / 10.0))
	start := time.Now()
	log.Infof("Deleting %d domains", len(domains))
	for i, dmn := range domains {
		dmn := dmn
		if (i+1)%progressMarker == 0 {
			log.Infof("Deleted %d domains", i+1)
		}
		workerPool.Go(func() *Result {
			taskStart := time.Now()
			resp, err := domain.DeleteDomain(cs, dmn.Id)
			if !resp || err != nil {
				return &Result{
					Success:  false,
					Duration: time.Since(taskStart).Seconds(),
				}
			}
			return &Result{
				Success:  true,
				Duration: time.Since(taskStart).Seconds(),
			}
		})
	}
	res := workerPool.Wait()
	log.Infof("Deleted %d domains in %.2f seconds", len(domains), time.Since(start).Seconds())
	return res
}
