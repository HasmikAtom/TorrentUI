package main

import (
	"fmt"
	"os"
)

func SetConfigs() *Config {
	envPrefix := "DEV"
	if _, err := os.Stat("/.dockerenv"); err == nil || os.Getenv("DOCKER_CONTAINER") == "true" {
		// its running in docker
		envPrefix = "PROD"
	}

	return &Config{
		AppPort:              os.Getenv(fmt.Sprintf("%s_APP_PORT", envPrefix)),
		TransmissionHost:     os.Getenv(fmt.Sprintf("%s_TRANSMISSION_HOST", envPrefix)),
		TransmissionPort:     os.Getenv(fmt.Sprintf("%s_TRANSMISSION_PORT", envPrefix)),
		TransmissionUsername: os.Getenv(fmt.Sprintf("%s_TRANSMISSION_USERNAME", envPrefix)),
		TransmissionPassword: os.Getenv(fmt.Sprintf("%s_TRANSMISSION_PASSWORD", envPrefix)),
	}
}

func getStatusString(status int) string {
	switch status {
	case 0:
		return "Stopped"
	case 1:
		return "Check waiting"
	case 2:
		return "Checking"
	case 3:
		return "Download waiting"
	case 4:
		return "Downloading"
	case 5:
		return "Seed waiting"
	case 6:
		return "Seeding"
	default:
		return "Unknown"
	}
}
