# csbench

This is Apache CloudStack Benchmark Tool, also known as "csbench"! Csbench is a tool designed to evaluate the performance and efficiency of Apache CloudStack. 
By internally executing a series of APIs, this tool meticulously measures the response times for various users, page sizes, and keyword combinations. 
With its comprehensive benchmarking capabilities, csbench provides invaluable insights into the system's overall performance, allowing cloud administrators 
and developers to fine-tune their configurations for optimal efficiency and seamless user experiences.

Currently, it looks like

/csbench$ go run csbench.go 
![image](https://github.com/shapeblue/csbench/assets/3348673/db37e176-474e-4b7d-8323-6a9a919414be)

The following are configurations options 

config/config file looks like the below having, CloudStack URL, user profiles for benchmarking and others,

![image](https://github.com/shapeblue/csbench/assets/3348673/bbdfcbd6-c57d-432f-bd63-799ad63d0b2f)

listCommants.txt file contains the list APIs that will be called for benchmarking

![image](https://github.com/shapeblue/csbench/assets/3348673/51402593-f330-4382-8e6e-4cec79a1bc1a)

Reports will be saved as CSV files for each API under report/individual/<management server host>/ report/accumulated/<management server host>
/individual folder contains the reports for each run
/accumulated folder contains the reports accumulated for all the runs

For example listDomains API Report looks like

![image](https://github.com/shapeblue/csbench/assets/3348673/4182b7ac-217a-489f-b7e6-fcb909633de8)
