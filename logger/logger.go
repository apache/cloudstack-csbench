package logger

import (
	"log"
	"os"
)

var logger *log.Logger

func init() {
	logFile, err := os.OpenFile("csmetrics.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to create log file: %v", err)
	}

	logger = log.New(logFile, "", log.LstdFlags|log.Lshortfile)
}

func Log(message string) {
	logger.Printf("%s", message)
}
