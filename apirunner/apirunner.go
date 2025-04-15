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

package apirunner

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

var processedAPImap = make(map[string]bool)
var APIscount = 0
var SuccessAPIs = 0
var FailedAPIs = 0
var TotalTime = 0.0

func generateParams(apiKey string, secretKey string, signatureVersion int, expires int, command string, page int, pagesize int, keyword string) url.Values {
	log.Info("Starting to generate parameters")
	params := url.Values{}
	params.Set("apiKey", apiKey)
	params.Set("response", "json")
	params.Set("signatureVersion", strconv.Itoa(signatureVersion))
	params.Set("listall", "true")
	params.Set("expires", time.Now().UTC().Add(time.Duration(expires)*time.Second).Format("2006-01-02T15:04:05Z"))

	params.Set("command", command)
	if command == "listTemplates" {
		params.Set("templatefilter", "all")
	}

	if page != 0 {
		params.Set("page", strconv.Itoa(page))
		params.Set("pagesize", strconv.Itoa(pagesize))
	}

	if keyword != "" {
		params.Set("keyword", keyword)
	}

	// Generate and add the signature
	signature := generateSignature(params.Encode(), secretKey)
	params.Set("signature", signature)

	return params
}

func RunAPIs(profileName string, apiURL string, apiKey string, secretKey string, expires int, signatureVersion int, iterations int, page int, pagesize int, dbProfile int) {

	log.Infof("Starting to run APIs from listCommands.txt file. Each command in the file will be run for multiple iterations and with page parameters mentioned in the configuration file.")

	commandsFile := "listCommands.txt"

	// Read commands from file
	commands, commandsKeywordMap, err := readCommandsFromFile(commandsFile)
	if err != nil {
		log.Infof("Error reading commands from file: %s\n", err.Error())
		return
	}
	reportAppend := false
	for _, command := range commands {
		keyword := commandsKeywordMap[command]
		if processedAPImap[command] {
			reportAppend = true
		}
		if page != 0 {
			if iterations != 1 {
				log.Infof("Calling API [%s] with page %d and pagesize %d -> ", command, page, pagesize)
			} else {
				log.Infof("Calling API [%s] -> ", command)
			}

			params := generateParams(apiKey, secretKey, signatureVersion, expires, command, page, pagesize, "")
			executeAPIandCalculate(profileName, apiURL, command, params, iterations, page, pagesize, "", dbProfile, reportAppend)
			reportAppend = true
		}

		if len(keyword) != 0 || keyword != "" {
			fmt.Printf("Calling API [%s] with keyword -> ", command)
			params := generateParams(apiKey, secretKey, signatureVersion, expires, command, 0, 0, keyword)
			executeAPIandCalculate(profileName, apiURL, command, params, iterations, 0, 0, keyword, dbProfile, reportAppend)
		}

		fmt.Printf("Calling API [%s] -> ", command)
		params := generateParams(apiKey, secretKey, signatureVersion, expires, command, 0, 0, "")
		executeAPIandCalculate(profileName, apiURL, command, params, iterations, 0, 0, "", dbProfile, reportAppend)

		fmt.Printf("------------------------------------------------------------\n")
		processedAPImap[command] = true
	}
}

func executeAPIandCalculate(profileName string, apiURL string, command string, params url.Values, iterations int, page int, pagesize int, keyword string, dbProfile int, reportAppend bool) {
	var minTime = math.MaxFloat64
	var maxTime = 0.0
	var avgTime float64
	var totalTime float64
	var count float64
	if iterations != 1 {
		log.Infof("Calling API %s for %d number of iterations with parameters %s", command, iterations, params)
		for i := 1; i <= iterations; i++ {
			log.Infof("Started with iteration %d for the command %s", i, command)
			elapsedTime, apicount, result := executeAPI(apiURL, params)
			count = apicount
			if elapsedTime < minTime {
				minTime = elapsedTime
			}
			if elapsedTime > maxTime {
				maxTime = elapsedTime
			}
			totalTime += elapsedTime
			if !result {
				break
			}
		}
		avgTime = totalTime / float64(iterations)
		log.Infof("count [%.f] : Time in seconds [Min - %.2f] [Max - %.2f] [Avg - %.2f]\n", count, minTime, maxTime, avgTime)
		saveData(apiURL, count, minTime, maxTime, avgTime, page, pagesize, keyword, profileName, command, dbProfile, reportAppend)
	} else {
		elapsedTime, apicount, _ := executeAPI(apiURL, params)
		log.Infof("Elapsed time [%.2f seconds] for the count [%.0f]", elapsedTime, apicount)
		saveData(apiURL, count, elapsedTime, elapsedTime, elapsedTime, page, pagesize, keyword, profileName, command, dbProfile, reportAppend)
	}
}

func saveData(apiURL string, count float64, minTime float64, maxTime float64, avgTime float64, page int, pageSize int, keyword string, user string, filename string, dbProfile int, reportAppend bool) {

	parsedURL, err := url.Parse(apiURL)
	if err != nil {
		log.Infof("Error parsing URL : %s with error : %s\n", apiURL, err)
		return
	}
	host := parsedURL.Hostname()

	err = os.MkdirAll(fmt.Sprintf("report/accumulated/%s", host), 0755)
	if err != nil {
		log.Infof("Error creating host directory : report/accumulated/%s\n", host, err)
		return
	}

	err = os.MkdirAll(fmt.Sprintf("report/individual/%s", host), 0755)
	if err != nil {
		log.Infof("Error creating host directory : report/individual/%s\n", host, err)
		return
	}

	fileMode := os.O_WRONLY | os.O_CREATE
	if reportAppend {
		fileMode |= os.O_APPEND
	} else {
		fileMode |= os.O_TRUNC
	}

	individualFile, err := os.OpenFile(fmt.Sprintf("report/individual/%s/%s.csv", host, filename), fileMode, 0644)
	if err != nil {
		log.Fatalf("Error opening the file CSV : report/individual/%s/%s.csv with error %s\n", host, filename, apiURL, err)
	}
	defer individualFile.Close()

	accumulatedFile, err := os.OpenFile(fmt.Sprintf("report/accumulated/%s/%s.csv", host, filename), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Error opening the file CSV : report/accumulated/%s/%s.csv with error %s\n", host, filename, apiURL, err)
	}
	defer accumulatedFile.Close()

	filePointers := []*os.File{individualFile, accumulatedFile}
	for _, file := range filePointers {
		writer := csv.NewWriter(file)
		defer writer.Flush()

		filereader, err := os.Open(file.Name())
		if err != nil {
			log.Fatalf("Error opening the file CSV : %s with error %s\n", file.Name(), err)
		}
		defer filereader.Close()
		reader := bufio.NewReader(filereader)
		firstLine, _ := reader.ReadString('\n')
		containsCount := strings.Contains(strings.ToLower(firstLine), "count")

		if !containsCount {
			header := []string{"Count", "MinTime", "MaxTime", "AvgTime", "Page", "PageSize", "keyword", "User", "DBprofile"}
			err = writer.Write(header)
			if err != nil {
				log.Infof("Error writing CSV header for the API: %s with error %s\n", apiURL, err)
				return
			}
		}

		var record []string
		if page != 0 {
			record = []string{
				fmt.Sprintf("%.f", count),
				fmt.Sprintf("%.2f", minTime),
				fmt.Sprintf("%.2f", maxTime),
				fmt.Sprintf("%.2f", avgTime),
				strconv.Itoa(page),
				strconv.Itoa(pageSize),
				keyword,
				user,
				strconv.Itoa(dbProfile),
			}
		} else {
			record = []string{
				fmt.Sprintf("%.f", count),
				fmt.Sprintf("%.2f", minTime),
				fmt.Sprintf("%.2f", maxTime),
				fmt.Sprintf("%.2f", avgTime),
				"-",
				"-",
				keyword,
				user,
				strconv.Itoa(dbProfile),
			}
		}
		err = writer.Write(record)
		if err != nil {
			log.Infof("Error writing to CSV for the API: %s with error %s\n", apiURL, err)
			return
		}
	}

	message := fmt.Sprintf("Data saved to report/%s/%s.csv successfully.\n", host, filename)
	log.Info(message)
}

func executeAPI(apiURL string, params url.Values) (float64, float64, bool) {
	// Send the API request and calculate the time
	data := strings.NewReader(params.Encode())
	log.Infof("Running the API %s", apiURL)
	start := time.Now()
	resp, err := http.Post(
		apiURL,
		"application/x-www-form-urlencoded",
		data,
	)
	APIscount++
	if err != nil {
		log.Infof("Error sending API request: %s with error %s\n", apiURL, err)
		FailedAPIs++
		return 0, 0, false
	}
	defer resp.Body.Close()
	elapsed := time.Since(start)
	TotalTime += elapsed.Seconds()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Infof("Error reading API response: %s with error %s\n", apiURL, err)
		FailedAPIs++
		return 0, 0, false
	}

	var data map[string]interface{}
	err = json.Unmarshal([]byte(body), &data)
	if err != nil {
		log.Infof("Error parsing JSON for the API response: %s with error %s\n", apiURL, err)
		FailedAPIs++
		return 0, 0, false
	}
	var key string
	for k := range data {
		key = k
		break
	}

	count, ok := data[key].(map[string]interface{})["count"].(float64)
	if !ok {
		errorCode, ok := data[key].(map[string]interface{})["errorcode"].(float64)
		if ok {
			errorText := data[key].(map[string]interface{})["errortext"].(string)
			log.Infof(" [Error] while calling the API ErrorCode[%.0f] ErrorText[%s]", errorCode, errorText)
			FailedAPIs++
			return elapsed.Seconds(), count, false
		}
	}

	SuccessAPIs++
	return elapsed.Seconds(), count, true
}

func generateSignature(unsignedRequest string, secretKey string) string {
	unsignedRequest = strings.ToLower(unsignedRequest)
	hasher := hmac.New(sha1.New, []byte(secretKey))
	hasher.Write([]byte(unsignedRequest))
	encryptedBytes := hasher.Sum(nil)

	computedSignature := base64.StdEncoding.EncodeToString(encryptedBytes)

	return computedSignature
}

func readCommandsFromFile(filename string) ([]string, map[string]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	var commands []string
	var commandsKeywordMap = make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		keyword := ""
		if strings.Contains(line, "keyword=") {
			keywordStartIndex := strings.Index(line, "keyword=")
			keyword = strings.TrimSpace(line[keywordStartIndex+8:])
			line = strings.TrimSpace(line[:keywordStartIndex])
		}
		if line != "" {
			commands = append(commands, line)
			commandsKeywordMap[line] = keyword
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}

	return commands, commandsKeywordMap, nil
}
