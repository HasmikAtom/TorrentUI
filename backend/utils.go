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
		RutrackerUsername:    os.Getenv(fmt.Sprintf("%s_RUTRACKER_USERNAME", envPrefix)),
		RutrackerPassword:    os.Getenv(fmt.Sprintf("%s_RUTRACKER_PASSWORD", envPrefix)),
	}
}

// ValidMediaTypes defines allowed media type values to prevent path traversal
var ValidMediaTypes = map[string]bool{
	"Movies": true,
	"Series": true,
	"Music":  true,
}

// ValidateMediaType checks if the media type is allowed
func ValidateMediaType(mediaType string) bool {
	return ValidMediaTypes[mediaType]
}

// GetDownloadDir returns a safe download directory path
func GetDownloadDir(mediaType string) (string, error) {
	if !ValidateMediaType(mediaType) {
		return "", fmt.Errorf("invalid content type: %s", mediaType)
	}
	return "/mediastorage/" + mediaType, nil
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

// Safe type assertion helpers to prevent panics

// GetFloat64 safely extracts a float64 from a map
func GetFloat64(m map[string]interface{}, key string) (float64, bool) {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f, true
		}
	}
	return 0, false
}

// GetInt safely extracts an int from a map (handles float64 from JSON)
func GetInt(m map[string]interface{}, key string) (int, bool) {
	if f, ok := GetFloat64(m, key); ok {
		return int(f), true
	}
	return 0, false
}

// GetInt64 safely extracts an int64 from a map (handles float64 from JSON)
func GetInt64(m map[string]interface{}, key string) (int64, bool) {
	if f, ok := GetFloat64(m, key); ok {
		return int64(f), true
	}
	return 0, false
}

// GetString safely extracts a string from a map
func GetString(m map[string]interface{}, key string) (string, bool) {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s, true
		}
	}
	return "", false
}

// ParseTorrentStatus safely parses a torrent map into TorrentStatus
func ParseTorrentStatus(torrent map[string]interface{}) (TorrentStatus, bool) {
	id, idOk := GetInt(torrent, "id")
	name, nameOk := GetString(torrent, "name")
	percentDone, percentOk := GetFloat64(torrent, "percentDone")
	rateDownload, rateOk := GetInt64(torrent, "rateDownload")
	statusCode, statusOk := GetInt(torrent, "status")

	if !idOk || !nameOk || !percentOk || !rateOk || !statusOk {
		return TorrentStatus{}, false
	}

	status := TorrentStatus{
		ID:           id,
		Name:         name,
		PercentDone:  percentDone * 100,
		RateDownload: rateDownload,
		Status:       getStatusString(statusCode),
	}

	if errVal, ok := GetInt(torrent, "error"); ok {
		status.Error = errVal
	}
	if errStr, ok := GetString(torrent, "errorString"); ok {
		status.ErrorString = errStr
	}

	return status, true
}
