package main

import (
    "os"
    "fmt"
    "flag"
    "strings"

    config "csmetrictool/config"
    apirunner "csmetrictool/apirunner"
    logger "csmetrictool/logger"
)

var (
    profiles = make(map[int]*config.Profile)
)

func readConfigurations() map[int]*config.Profile {
    profiles, err := config.ReadProfiles("config/config")
    if err != nil {
        fmt.Println("Error reading profiles:", err)
        os.Exit(1)
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

    logger.Log(fmt.Sprintf("Found %d profiles in the configuration: ", len(profiles)))

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

func main() {
	dbprofile := flag.Int("dbprofile", 0, "DB profile number")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: go run csmetrictool.go --dbprofile <DB profile number>\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *dbprofile < 0 {
		fmt.Println("Invalid DB profile number. Please provide a positive integer.")
		return
	}

    profiles = readConfigurations()
    apiURL := config.URL
    iterations := config.Iterations
    page := config.Page
    pagesize := config.PageSize

	logger.Log(fmt.Sprintf("\nStarted benchmarking the CloudStack environment [%s]", apiURL))

    logConfigurationDetails(profiles)

    for i := 1; i <= len(profiles); i++ {
        profile := profiles[i]
        userProfileName := profile.Name
        logger.Log(fmt.Sprintf("Using profile %d.%s for benchmarking", i, userProfileName))
        fmt.Printf("\n\033[1;34m============================================================\033[0m\n")
        fmt.Printf("                    Profile: [%s]\n", userProfileName)
        fmt.Printf("\033[1;34m============================================================\033[0m\n")
        apirunner.RunAPIs(userProfileName, apiURL, profile.ApiKey, profile.SecretKey, profile.Expires, profile.SignatureVersion, iterations, page, pagesize, *dbprofile)
    }

    logReport()

	logger.Log(fmt.Sprintf("Done with benchmarking the CloudStack environment [%s]", apiURL))
}

