package transmission

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func (t *TransmissionRPC) SendRequest(method string, args interface{}) (*RPCResponse, error) {
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
	log.Println("doing request")
	req, err := http.NewRequest("POST", t.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("error creating request ==>", err)
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	log.Println("setting content type")

	req.Header.Set("Content-Type", "application/json")
	if t.Session != "" {
		req.Header.Set("X-Transmission-Session-Id", t.Session)
	}
	if t.Username != "" {
		req.SetBasicAuth(t.Username, t.Password)
	}

	client := &http.Client{
		Timeout: time.Second * 30,
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("error sending request to transmission ===> ", err)
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 409 {
		t.Session = resp.Header.Get("X-Transmission-Session-Id")
		log.Println("received 409 status code, doing request again ===> ", resp.Header.Get("X-Transmission-Session-Id"))
		return t.doRequest(jsonData)
	}

	if resp.StatusCode != 200 {
		log.Println("Unexpected status code", resp.StatusCode)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result RPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Println("DECODING RESPONSE", err)
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &result, nil
}
