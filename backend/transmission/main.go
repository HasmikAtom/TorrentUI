package transmission

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func (t *TransmissionRPC) SendRequest(method string, args any) (*RPCResponse, error) {
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
	for retry := range maxRetries {
		resp, err := t.doRequest(jsonData)
		if err != nil {
			if retry == maxRetries-1 {
				return nil, fmt.Errorf("final retry failed: %v", err)
			}
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

	// Reuse the client from the struct instead of creating new one
	resp, err := t.Client.Do(req)
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
