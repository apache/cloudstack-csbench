package config

import (
    "bufio"
    "fmt"
    "os"
    "strings"
    "net/url"
)

type Profile struct {
    Name             string
    ApiKey           string
    SecretKey        string
    Expires          int
    SignatureVersion int
    Timeout          int
}

var URL = ""
var Iterations = 1
var Page = 0
var PageSize = 0
var Host = ""

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
                case "timeout":
                    var timeout int
                    _, err := fmt.Sscanf(value, "%d", &timeout)
                    if err == nil {
                        profiles[i].Timeout = timeout
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
        return nil, fmt.Errorf("No roles are defined in the configuration file: %w", err)
    }

    if URL == "" {
        fmt.Println("URL not found in the configuration, please verify")
        os.Exit(1)
    }

	parsedURL, err := url.Parse(URL)
	if err != nil {
        fmt.Println("Error parsing URL : %s with error : %s\n", URL, err)
		return nil, fmt.Errorf("Error parsing URL : %s with error : %s\n", URL, err)
	}
	Host = parsedURL.Hostname()


    return profiles, nil
}

