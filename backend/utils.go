package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
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

func (t *TransmissionRPC) sendRequest(method string, args interface{}) (*RPCResponse, error) {
	request := RPCRequest{
		Method:    method,
		Arguments: args,
		Tag:       1,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	maxRetries := 3
	for retry := 0; retry < maxRetries; retry++ {
		resp, err := t.doRequest(jsonData)
		if err != nil {
			if retry == maxRetries-1 {
				return nil, fmt.Errorf("final retry failed: %v", err)
			}
			log.Printf("Retry %d failed: %v", retry+1, err)
			time.Sleep(time.Second * time.Duration(retry+1))
			continue
		}
		return resp, nil
	}

	return nil, fmt.Errorf("all retries failed")
}

func (t *TransmissionRPC) doRequest(jsonData []byte) (*RPCResponse, error) {
	req, err := http.NewRequest("POST", t.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if t.Session != "" {
		req.Header.Set("X-Transmission-Session-Id", t.Session)
	}
	if t.Username != "" {
		req.SetBasicAuth(t.Username, t.Password)
	}

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 409 {
		t.Session = resp.Header.Get("X-Transmission-Session-Id")
		return t.doRequest(jsonData)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result RPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &result, nil
}
