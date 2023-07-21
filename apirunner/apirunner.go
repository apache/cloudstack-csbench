package apirunner

import (
    "encoding/json"
    "io/ioutil"
    "net/http"
    "net/url"
    "time"
    "strconv"
    "fmt"
    "bufio"
    "os"
    "crypto/hmac"
    "crypto/sha1"
    "encoding/base64"
    "strings"
    "math"
    "encoding/csv"

    logger "csmetrictool/logger"
)

var processedAPImap = make(map[string]bool)
var APIscount = 0
var SuccessAPIs = 0
var FailedAPIs = 0
var TotalTime = 0.0

func generateParams(apiKey string, secretKey string, signatureVersion int, expires int, command string, page int, pagesize int, keyword string) url.Values {
    logger.Log("Starting to generate parameters")
    params := url.Values{}
    params.Set("apiKey", apiKey)
    params.Set("response", "json")
    params.Set("signatureVersion", strconv.Itoa(signatureVersion))
    params.Set("listall", "true")
    params.Set("expires", time.Now().UTC().Add(time.Duration(expires) * time.Second).Format("2006-01-02T15:04:05Z"))

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

    logger.Log(fmt.Sprintf("Starting to run APIs from listCommands.txt file. Each command in the file will be run for multiple iterations and with page parameters mentioned in the configuration file."))

	commandsFile := "listCommands.txt"

	// Read commands from file
	commands, commandsKeywordMap, err := readCommandsFromFile(commandsFile)
	if err != nil {
		message := fmt.Sprintf("Error reading commands from file: %s\n", err.Error())
		fmt.Printf(message)
		logger.Log(message)
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
                message := fmt.Sprintf("Calling API [%s] with page %d and pagesize %d -> ", command, page, pagesize)
                logger.Log(message)
                fmt.Printf(message)
            } else {
                message := fmt.Sprintf("Calling API [%s] -> ", command)
                logger.Log(message)
                fmt.Printf(message)
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
        logger.Log(fmt.Sprintf("Calling API %s for %d number of iterations with parameters %s", command, iterations, params))
        for i := 1; i <= iterations; i++ {
            // fmt.Printf("%d,", i)
            logger.Log(fmt.Sprintf("Started with iteration %d for the command %s", i, command))
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
        message := fmt.Sprintf("count [%.f] : Time in seconds [Min - %.2f] [Max - %.2f] [Avg - %.2f]\n", count, minTime, maxTime, avgTime)
        fmt.Printf("%s", message)
        logger.Log(fmt.Sprintf("Time taken for the API %s\n %s", command, message))
        saveData(apiURL, count, minTime, maxTime, avgTime, page, pagesize, keyword, profileName, command, dbProfile, reportAppend)
    } else {
        elapsedTime, apicount, _ := executeAPI(apiURL, params)
        fmt.Printf("\n  Elapsed time [%.2f seconds] for the count [%.0f]\n", elapsedTime, apicount)
        logger.Log(fmt.Sprintf("\n  Elapsed time [%.2f seconds] for the count [%.0f]\n", elapsedTime, apicount))
        saveData(apiURL, count, elapsedTime, elapsedTime, elapsedTime, page, pagesize, keyword, profileName, command, dbProfile, reportAppend)
    }
}

func saveData(apiURL string, count float64, minTime float64, maxTime float64, avgTime float64, page int, pageSize int, keyword string, user string, filename string, dbProfile int, reportAppend bool) {

	parsedURL, err := url.Parse(apiURL)
	if err != nil {
        logger.Log(fmt.Sprintf("Error parsing URL : %s with error : %s\n", apiURL, err))
		return
	}
	host := parsedURL.Hostname()

	err = os.MkdirAll(fmt.Sprintf("report/accumulated/%s", host), 0755)
	if err != nil {
        logger.Log(fmt.Sprintf("Error creating host directory : report/accumulated/%s\n", host, err))
		return
	}

	err = os.MkdirAll(fmt.Sprintf("report/individual/%s", host), 0755)
	if err != nil {
        logger.Log(fmt.Sprintf("Error creating host directory : report/individual/%s\n", host, err))
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
        logger.Log(fmt.Sprintf("Error opening the file CSV : report/individual/%s/%s.csv with error %s\n", host, filename, apiURL, err))
		return
	}
	defer individualFile.Close()

	accumulatedFile, err := os.OpenFile(fmt.Sprintf("report/accumulated/%s/%s.csv", host, filename), os.O_WRONLY | os.O_CREATE | os.O_APPEND, 0644)
	if err != nil {
        logger.Log(fmt.Sprintf("Error opening the file CSV : report/accumulated/%s/%s.csv with error %s\n", host, filename, apiURL, err))
		return
	}
	defer accumulatedFile.Close()

    filePointers := []*os.File{individualFile, accumulatedFile}
    for _, file := range filePointers {
    	writer := csv.NewWriter(file)
    	defer writer.Flush()

        reader := bufio.NewReader(file)
        firstLine, err := reader.ReadString('\n')
        fmt.Printf("HARI %s", firstLine)
        if !strings.Contains(firstLine, "Count") {
            header := []string{"Count", "MinTime", "MaxTime", "AvgTime", "Page", "PageSize", "keyword", "User", "DBprofile"}
            err = writer.Write(header)
            if err != nil {
                logger.Log(fmt.Sprintf("Error writing CSV header for the API: %s with error %s\n", apiURL, err))
                return
            }
        }


    	record := []string{}
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
                fmt.Sprintf("-"),
                fmt.Sprintf("-"),
                keyword,
        		user,
        		strconv.Itoa(dbProfile),
        	}
    	}
    	err = writer.Write(record)
    	if err != nil {
            logger.Log(fmt.Sprintf("Error writing to CSV for the API: %s with error %s\n", apiURL, err))
    		return
    	}
    }

	message := fmt.Sprintf("Data saved to report/%s/%s.csv successfully.\n", host, filename)
	logger.Log(message)
}

func executeAPI() (apiURL string, params url.Values) (float64, float64, bool) {
	concurrentOptions := 5 // Change this value to the desired number of concurrent options.

	var wg sync.WaitGroup
	requestsChan := make(chan int, concurrentOptions)

	// Start the goroutines to make concurrent API requests.
	for i := 0; i < concurrentOptions; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range requestsChan {
                elapsedTime, apicount, _ := executeAPI(apiURL, params)
            }
		}()
	}

	// Send the API requests to the channel.
	for i := 0; i < 10; i++ { // Replace '10' with the total number of API requests you want to make.
		requestsChan <- i
	}

	close(requestsChan)
	wg.Wait()

}

func executeAPIconcurrent(apiURL string, params url.Values) (float64, float64, bool) {
    // Send the API request and calculate the time
    apiURL = fmt.Sprintf("%s?%s", apiURL, params.Encode())
    logger.Log(fmt.Sprintf("Running the API %s", apiURL))
    start := time.Now()
    resp, err := http.Get(apiURL)
    APIscount++
    if err != nil {
        logger.Log(fmt.Sprintf("Error sending API request: %s with error %s\n", apiURL, err))
        FailedAPIs++
        return 0, 0, false
    }
    defer resp.Body.Close()
    elapsed := time.Since(start)
    TotalTime += elapsed.Seconds()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        logger.Log(fmt.Sprintf("Error reading API response: %s with error %s\n", apiURL, err))
        FailedAPIs++
        return 0, 0, false
    }

    var data map[string]interface{}
    err = json.Unmarshal([]byte(body), &data)
    if err != nil {
        logger.Log(fmt.Sprintf("Error parsing JSON for the API response: %s with error %s\n", apiURL, err))
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
            message := fmt.Sprintf(" [Error] while calling the API ErrorCode[%.0f] ErrorText[%s]", errorCode, errorText)
            fmt.Printf(message)
            logger.Log(message)
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