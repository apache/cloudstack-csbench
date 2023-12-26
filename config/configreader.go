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

package config

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

type Profile struct {
	Name             string
	ApiKey           string
	SecretKey        string
	Expires          int `default:"600"`
	SignatureVersion int `default:"3"`
}

var URL = "http://localhost:8080/client/api/"
var Iterations = 1
var Page = 0
var PageSize = 0
var Host = ""
var ZoneId = ""
var NetworkOfferingId = ""
var ServiceOfferingId = ""
var DiskOfferingId = ""
var TemplateId = ""
var ParentDomainId = ""
var NumDomains = 0
var NumNetworks = 0
var Subnet = "10.0.0.0"
var Submask = 22
var VlanStart = 80
var VlanEnd = 1000
var StartVM = true
var NumVms = 1
var NumVolumes = 1

func ReadProfiles(filePath string) (map[int]*Profile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	profiles := make(map[int]*Profile)
	var currentProfile string

	i := 0
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentProfile = line[1 : len(line)-1]
			i++
			profiles[i] = &Profile{}
			profiles[i].Name = currentProfile
		} else {
			// Parse key-value pairs within the profile
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				switch strings.ToLower(key) {
				case "apikey":
					profiles[i].ApiKey = value
				case "secretkey":
					profiles[i].SecretKey = value
				case "url":
					URL = value
				case "iterations":
					var iterations int
					_, err := fmt.Sscanf(value, "%d", &iterations)
					if err == nil {
						Iterations = iterations
					}
				case "page":
					var page int
					_, err := fmt.Sscanf(value, "%d", &page)
					if err == nil {
						Page = page
					}
				case "pagesize":
					var pagesize int
					_, err := fmt.Sscanf(value, "%d", &pagesize)
					if err == nil {
						PageSize = pagesize
					}
				case "expires":
					var expires int
					_, err := fmt.Sscanf(value, "%d", &expires)
					if err == nil {
						profiles[i].Expires = expires
					}
				case "signatureversion":
					var signatureVersion int
					_, err := fmt.Sscanf(value, "%d", &signatureVersion)
					if err == nil {
						profiles[i].SignatureVersion = signatureVersion
					}
				case "zoneid":
					ZoneId = value
				case "networkofferingid":
					NetworkOfferingId = value
				case "serviceofferingid":
					ServiceOfferingId = value
				case "diskofferingid":
					DiskOfferingId = value
				case "templateid":
					TemplateId = value
				case "parentdomainid":
					ParentDomainId = value
				case "numdomains":
					var numDomains int
					_, err := fmt.Sscanf(value, "%d", &numDomains)
					if err == nil {
						NumDomains = numDomains
					}
				case "numnetworks":
					var numNetworks int
					_, err := fmt.Sscanf(value, "%d", &numNetworks)
					if err == nil {
						NumNetworks = numNetworks
					}
				case "subnet":
					var subnet string
					_, err := fmt.Sscanf(value, "%s", &subnet)
					if err == nil {
						Subnet = subnet
					}
				case "submask":
					var submask int
					_, err := fmt.Sscanf(value, "%d", &submask)
					if err == nil {
						Submask = submask
					}
				case "vlanrange":
					var vlanStart, vlanEnd int
					_, err := fmt.Sscanf(value, "%d-%d", &vlanStart, &vlanEnd)
					if err == nil {
						VlanStart = vlanStart
						VlanEnd = vlanEnd
					}
				case "numvms":
					var numVms int
					_, err := fmt.Sscanf(value, "%d", &numVms)
					if err == nil {
						NumVms = numVms
					}
				case "startvm":
					var startvm bool
					_, err := fmt.Sscanf(value, "%t", &startvm)
					if err == nil {
						StartVM = startvm
					}
				case "numvolumes":
					var numVolumes int
					_, err := fmt.Sscanf(value, "%d", &numVolumes)
					if err == nil {
						NumVolumes = numVolumes
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning file: %w", err)
	}

	if len(profiles) == 0 {
		fmt.Println("No roles are defined in the configuration file")
		return nil, fmt.Errorf("no roles are defined in the configuration file: %w", err)
	}

	if URL == "" {
		log.Fatalln("URL not found in the configuration, please verify")
	}

	parsedURL, err := url.Parse(URL)
	if err != nil {
		log.Errorf("Error parsing URL : %s with error : %s", URL, err)
		return nil, fmt.Errorf("error parsing url : %s with error : %s", URL, err)
	}
	Host = parsedURL.Hostname()

	validateConfig(profiles)

	return profiles, nil
}

func validateConfig(profiles map[int]*Profile) bool {

	result := true
	for i, profile := range profiles {
		if profile.ApiKey == "" || profile.SecretKey == "" {
			message := "Please check ApiKey, SecretKey of the profile. They should not be empty"
			fmt.Printf("Skipping profile [%s] : %s\n", profile.Name, message)
			delete(profiles, i)
			result = false
		}
	}

	return result
}
