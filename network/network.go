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
	"math"
	"math/rand"
	"net/netip"
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
		if len(result) < resp.Count {
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
	gateway, netmask, startIP, endIP, err := generateNetworkDetails(config.Subnet, count, config.Submask)
	if err != nil {
		log.Printf("Failed to generate network details due to %v", err)
		return nil, err
	}
	p.SetDomainid(domainId)
	p.SetAcltype("Domain")
	p.SetGateway(gateway)
	p.SetNetmask(netmask)
	p.SetDisplaytext(netName)
	p.SetStartip(startIP)
	p.SetEndip(endIP)
	p.SetVlan(strconv.Itoa(getRandomVlan()))
	p.SetBypassvlanoverlapcheck(true)

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

func getRandomVlan() int {
	return config.VlanStart + rand.Intn(config.VlanEnd-config.VlanStart)
}

// A function which generates gateway, netmask, start ip and end ip for a network with the specified
// subnet mask on the basis of an integer
// Generate CIDRs from 10.10.0.0
// IPs should be incremental. For 22 submask, the IPs should be generated as follows:
// for 0 -> 	10.10.0.1, 255.255.252.0, 10.10.0.2 - 10.10.3.255
// for 1 ->     10.10.4.1, 255.255.252.0, 10.10.4.2 - 10.10.7.255

func generateNetworkDetails(address string, count int, submask int) (string, string, string, string, error) {
	ip, err := netip.ParseAddr(address)
	if err != nil {
		return "", "", "", "", err
	}
	// Convert ip to uint32
	ipBin, _ := ip.MarshalBinary()
	ipUint32 := uint32(ipBin[0])<<24 | uint32(ipBin[1])<<16 | uint32(ipBin[2])<<8 | uint32(ipBin[3])
	// Convert submask to uint32
	incr := uint32(math.Pow(2, (32 - float64(submask))))
	gatewayInt32 := ipUint32 + (uint32(count) * incr) + 1
	startIPInt32 := gatewayInt32 + 1
	endIPInt32 := gatewayInt32 + incr - 2

	return getIPFromUint32(gatewayInt32),
		getIPFromUint32(uint32(math.Pow(2, 32) - math.Pow(2, 32-float64(submask)))),
		getIPFromUint32(startIPInt32),
		getIPFromUint32(endIPInt32),
		nil
}

func getIPFromUint32(ip uint32) string {
	IP := netip.Addr{}
	err := IP.UnmarshalBinary([]byte{byte(ip >> 24), byte(ip >> 16), byte(ip >> 8), byte(ip)})
	if err != nil {
		log.Printf("Failed to convert uint32 to ip due to %v", err)
	}
	return IP.String()
}
