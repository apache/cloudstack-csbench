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

package network

import (
	"csbench/config"
	"csbench/utils"
	"log"
	"strconv"

	"github.com/apache/cloudstack-go/v2/cloudstack"
)

func ListNetworks(cs *cloudstack.CloudStackClient, domainId string) ([]*cloudstack.Network, error) {
	result := make([]*cloudstack.Network, 0)
	page := 1
	p := cs.Network.NewListNetworksParams()
	p.SetDomainid(domainId)
	p.SetListall(true)
	p.SetZoneid(config.ZoneId)
	p.SetPagesize(config.PageSize)
	for {
		p.SetPage(page)
		resp, err := cs.Network.ListNetworks(p)
		if err != nil {
			log.Printf("Failed to list networks due to %v", err)
			return result, err
		}
		result = append(result, resp.Networks...)
		if len(resp.Networks) < resp.Count {
			page++
		} else {
			break
		}
	}
	return result, nil
}

func CreateNetwork(cs *cloudstack.CloudStackClient, domainId string, count int) (*cloudstack.CreateNetworkResponse, error) {
	netName := "Network-" + utils.RandomString(10)
	p := cs.Network.NewCreateNetworkParams(netName, config.NetworkOfferingId, config.ZoneId)
	p.SetDomainid(domainId)
	p.SetAcltype("Domain")
	p.SetGateway("10.10.0.1")
	p.SetNetmask("255.255.252.0")
	p.SetDisplaytext(netName)
	p.SetStartip("10.10.0.2")
	p.SetEndip("10.10.3.255")
	p.SetVlan(strconv.Itoa(80 + count))

	resp, err := cs.Network.CreateNetwork(p)
	if err != nil {
		log.Printf("Failed to create network due to: %v", err)
		return nil, err
	}
	return resp, nil
}

func DeleteNetwork(cs *cloudstack.CloudStackClient, networkId string) (bool, error) {
	deleteParams := cs.Network.NewDeleteNetworkParams(networkId)
	delResp, err := cs.Network.DeleteNetwork(deleteParams)
	if err != nil {
		log.Printf("Failed to delete network with id %s due to %v", networkId, err)
		return false, err
	}
	return delResp.Success, nil
}
