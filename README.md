# CloudStack Bench - csbench

`csbench` is a tool designed to evaluate the performance and efficiency of
Apache CloudStack.

The tool is designed to be run from a single host, and can be used to benchmark
a single CloudStack Zone.

As of now, there are two modes of operation:
1. Setting up environment with multiple domains, accounts, users, networks, VMs, etc.
2. Benchmarking a list of APIs against an existing CloudStack environment

# Building
1. Install go 1.20 or above. Follow instructions [here](https://go.dev/doc/install) to install golang.
2. Clone the repository
3. Build the binary using the below command. This will generate a binary named `csbench` in the current directory.
```bash
go build
```

# Usage

Setup a config file. Check the sample config file [here](./config/config).

```
# URL of the CloudStack API endpoint
url = http://localhost:8080/client/api/
# Number of times to run the test. Used only for -benchmark
iterations = 1
# Page of results to start on. Used only for -benchmark
page = 0
# Max number of items to return per API call
pagesize = 5
# Zone to use for VMs. Used only for -create
zoneid = <zone id>
# Template to use for VMs. Used only for -create
templateid = <template id>
# Service offering to use for VMs. Used only for -create
serviceofferingid = <service offering id>
# Disk offering to use for volumes. Used only for -create
diskofferingid = <disk offering id>
# Shared network offering ID. Used only for -create
networkofferingid = <network offering id>
# Domain ID of the parent domain to create the subdomains under. Used only for -create
parentdomainid = <domain id>
# Number of domains to create. Creates a shared network for each domain. Used only for -create
numdomains = 2
# Number of networks to create per domain. Used only for -create
numnetworks = 1
# Subnet to use for the networks. Used only for -network with -create
subnet = 10.0.0.0
# Subnet mask to use for the networks. Used only for -network with -create
submask = 22
# Randomly allocated a VLAN from the range specified below. Used only for -network with -create
vlanrange = 80-1000
# Number of VMs to create per domain. Used only for -create
numvms = 2
# Whether to start the VMs after creation. Used only for -create
startvm = true
# Number of volumes to create & attach per VM. Used only for -create
numvolumes = 2

# Credentials to use to run -benchmark & -create. Name should be "admin" for -create
# Multiple profiles can be added and they will be used for -benchmark
[admin]
apikey = <api key>
secretkey = <secret key>
# Duration after which the signature included in the request is expired
expires = 600
# Signature version to allow the client to force a specific signature version
signatureversion = 3
```


```
$ csbench -h
Options:
  -benchmark
        Benchmark list APIs
  -config string
        Path to config file (default "config/config")
  -create
        Create resources. Specify at least one of the following options:
                -domain - Create subdomains and accounts
                -limits - Update limits to -1 for subdomains and accounts
                -network - Create shared network in all subdomains
                -vm - Deploy VMs in all networks in the subdomains
                -volume - Create and attach Volumes to VMs
  -dbprofile int
        DB profile number
  -domain
        Works with -create & -teardown
                -create - Create subdomains and accounts
                -teardown - Delete all subdomains and accounts
  -format string
        Format of the report (csv, tsv, table). Valid only for create (default "table")
  -limits
        Update limits to -1 for subdomains and accounts
  -network
        Works with -create & -teardown
                -create - Create shared network in all subdomains
                -teardown - Delete all networks in the subdomains
  -output string
        Path to output file. Valid only for create
  -teardown
        Tear down resources. Specify at least one of the following options:
                -domain - Delete all subdomains and accounts
                -network - Delete all networks in the subdomains
                -vm - Delete all VMs in the subdomains
                -volume - Delete all volumes in the subdomains
  -vm
        Works with -create & -teardown
                -create - Deploy VMs in all networks in the subdomains
                -teardown - Delete all VMs in the subdomains
  -vmaction string
        Action to perform on VMs. Options:
                start - start all VMs
                stop - stop all VMs
                reboot - reboot all running VMs
                toggle - stop running VMs and start stopped VMs
                random - Randomly toggle VMs
  -volume
        Works with -create & -teardown
                -create - Create and attach Volumes to VMs
                -teardown - Delete all volumes in the subdomains
  -workers int
        Number of workers to use while creating resources (default 10)
```

## Setting up an environment for benchmarking
This mode of operation is designed to set up a CloudStack environment with multiple domains, accounts, users, networks and VMs as per the configuration file.

To execute this mode, run the following command followed by the type of resources to be created:
```bash
csbench -create -domain -limits -network -vm -volume
```

This will create the resources under the domain specified in the config file. If there are existing domains, network and VMs present under the domain, they will be used as well for creating the resources.

If you wish to create just a single resource or a set of resources, you can specify the resource type as follows:
```bash
csbench -create -domain
csbench -create -limits -network
csbench -create -vm -volume
```

## Tearing down an environment
This mode of operation is designed to tear down subdomains and resources present in the subdomains.

To execute this mode, run the following command followed by the type of resources to be deleted:
```bash
csbench -teardown -domain -network -vm -volume
```

This will delete the resources under the domain specified in the config file.

If you wish to delete just a single resource or a set of resources, you can specify the resource type as follows:
```bash
csbench -teardown -domain
csbench -teardown -network
csbench -teardown -vm -volume
```

## Benchmarking actions on VMs
This mode of operation is designed to benchmark the actions on VMs. The actions that can be benchmarked are `start`, `stop`, `reboot`.

To execute this mode, run the following command followed by the type of action to be benchmarked:
```bash
csbench -vmaction <action>
```
Where action can be:
  - `start` - start all VMs
  - `stop` - stop all VMs
  - `reboot` - reboot all running VMs
  - `toggle` - stop running VMs and start stopped VMs
  - `random` - Randomly toggle VMs

## Output format
By default the results of setting up (`-create`)/tearing down (`-teardown`) the environment and actions on VM (`-vmaction`) are printed out to stdout, if you want to save the results to a file, you can pass the `-output` flag followed by the path to the file. And use `-format` flag to specify the format of the report (`csv`, `tsv`, `table`).

## Parallel execution
By default, the tool executes the APIs in parallel. The number of workers can be specified using the `-workers` flag. For example, to use 20 workers, you can run the following command:
```bash
csbench -create -domain -limits -network -vm -volume -workers 20
csbench -teardown -domain -network -vm -volume -workers 20
csbench -vmaction=<action> -workers 20
```

> *Note:* `-workers` flag is not applicable for `-benchmark` mode.

## Benchmarking list APIs
By internally executing a series of APIs, this tool meticulously measures the response times for various users, page sizes, and keyword combinations. 
With its comprehensive benchmarking capabilities, csbench provides invaluable insights into the system's overall performance, allowing cloud administrators 
and developers to fine-tune their configurations for optimal efficiency and seamless user experiences.

Currently, it looks like

```bash
/csbench$ ./csbench -benchmark
```

Note: this tool will go through several changes and is under development.
