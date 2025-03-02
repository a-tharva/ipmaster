package main

import (
	"log"
	"os"

	"github.com/a-tharva/ipmaster/logging"
	"github.com/a-tharva/ipmaster/ui"
)

func main() {

	// ipAddresses := []string{"8.8.8.8", "1.1.1.1", "8.8.4.4"}
	// ping.StartPing(ipAddresses, "")

	logFile, err := os.OpenFile(logging.GetDefaultLogPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Failed to open log file:", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.Println("Starting IPmaster...")

	if err := ui.Start(); err != nil {
		log.Fatalf("Failed to start UI: %v", err)
	}
	log.Println("IPmaster terminated.")
	// fmt.Println("fin")
}
