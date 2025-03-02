package logging

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
)

func GetDefaultLogPath() string {
	var dir string
	// Determine the default log directory based on the OS.
	switch runtime.GOOS {
	case "windows":
		// Use %APPDATA% or a network standard folder on Windows
		appData := os.Getenv("APPDATA")
		if appData == "" {
			// fallback to the user's home directory
			homeDir, err := os.UserHomeDir()
			if err != nil {
				// If we can't determine a home directory, fallback to the current working directory.
				homeDir = "."
			}
			dir = homeDir
		} else {
			dir = appData
		}
	default:
		// For Unix-like systems, you might use /var/log if you have permissions,
		// otherwise use the user's home directory or the current working directory.
		// Here we'll try using a log subdirectory in the user's home directory.
		homeDir, err := os.UserHomeDir()
		if err != nil {
			dir = "."
		} else {
			dir = filepath.Join(homeDir, "logs")
		}
	}
	// Make sure the directory exists.
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("failed to create log directory %s: %v", dir, err)
	}

	// Return a full file path for your log file.
	return filepath.Join(dir, "ipmaster.log")
}
