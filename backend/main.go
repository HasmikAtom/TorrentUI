package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func handleDownload(c *gin.Context) {
	magnetLink := c.PostForm("magnetLink")
	torrentFile, _ := c.FormFile("torrentFile")
	mediaType := c.PostForm("contentType")

	if magnetLink == "" && torrentFile == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either magnet link or torrent file is required"})
		return
	}

	var filename string
	if torrentFile != nil {
		filename = fmt.Sprintf("/tmp/%s", torrentFile.Filename)
		if err := c.SaveUploadedFile(torrentFile, filename); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save torrent file"})
			return
		}
	} else {
		filename = magnetLink
	}

	args := map[string]interface{}{
		"filename":     filename,
		"download-dir": "/mediastorage/" + mediaType,
	}

	result, err := client.sendRequest("torrent-add", args)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result.Result != "success" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add torrent"})
		return
	}

	if torrentAdded, ok := result.Arguments["torrent-added"].(map[string]interface{}); ok {
		if id, ok := torrentAdded["id"].(float64); ok {
			c.JSON(http.StatusOK, gin.H{
				"message":   "Download started",
				"torrentId": int(id),
			})
			return
		}
	}

	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get torrent ID"})
}

func getTorrentStatus(c *gin.Context) {
	id := c.Param("id")
	var torrentId int
	fmt.Sscanf(id, "%d", &torrentId)

	args := map[string]interface{}{
		"ids": []int{torrentId},
		"fields": []string{
			"id",
			"name",
			"percentDone",
			"rateDownload",
			"status",
			"error",
			"errorString",
		},
	}

	result, err := client.sendRequest("torrent-get", args)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if torrents, ok := result.Arguments["torrents"].([]interface{}); ok && len(torrents) > 0 {
		if torrent, ok := torrents[0].(map[string]interface{}); ok {
			status := TorrentStatus{
				ID:           int(torrent["id"].(float64)),
				Name:         torrent["name"].(string),
				PercentDone:  torrent["percentDone"].(float64) * 100,
				RateDownload: int64(torrent["rateDownload"].(float64)),
				Status:       getStatusString(int(torrent["status"].(float64))),
			}

			if errVal, ok := torrent["error"]; ok {
				status.Error = int(errVal.(float64))
			}
			if errStr, ok := torrent["errorString"]; ok {
				status.ErrorString = errStr.(string)
			}

			c.JSON(http.StatusOK, status)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Torrent not found"})
}

func listTorrents(c *gin.Context) {
	id := c.Param("id")
	var torrentId int
	fmt.Sscanf(id, "%d", &torrentId)

	args := map[string]interface{}{
		"fields": []string{
			"id",
			"name",
			"percentDone",
			"rateDownload",
			"status",
			"error",
			"errorString",
		},
	}

	result, err := client.sendRequest("torrent-get", args)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	statuses := []TorrentStatus{}
	if torrents, ok := result.Arguments["torrents"].([]interface{}); ok && len(torrents) > 0 {
		for i := 0; i < len(torrents); i++ {
			if torrent, ok := torrents[i].(map[string]interface{}); ok {
				status := TorrentStatus{
					ID:           int(torrent["id"].(float64)),
					Name:         torrent["name"].(string),
					PercentDone:  torrent["percentDone"].(float64) * 100,
					RateDownload: int64(torrent["rateDownload"].(float64)),
					Status:       getStatusString(int(torrent["status"].(float64))),
				}

				if errVal, ok := torrent["error"]; ok {
					status.Error = int(errVal.(float64))
				}
				if errStr, ok := torrent["errorString"]; ok {
					status.ErrorString = errStr.(string)
				}

				statuses = append(statuses, status)
			}
		}

		c.JSON(http.StatusOK, statuses)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Torrent not found"})
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

var client *TransmissionRPC

func init() {
	host := os.Getenv("TRANSMISSION_HOST")
	if host == "" {
		host = "host.docker.internal"
	}

	port := os.Getenv("TRANSMISSION_PORT")
	if port == "" {
		port = "9091"
	}

	username := os.Getenv("TRANSMISSION_USERNAME")
	password := os.Getenv("TRANSMISSION_PASSWORD")

	client = &TransmissionRPC{
		URL:      fmt.Sprintf("http://%s:%s/transmission/rpc", host, port),
		Username: username,
		Password: password,
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

func main() {
	r := gin.Default()

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
	config.AllowHeaders = []string{"*"}

	r.Use(cors.New(config))

	r.Use(func(c *gin.Context) {
		log.Printf("[DEBUG] Request Headers:")
		for name, values := range c.Request.Header {
			log.Printf("[DEBUG] %s: %s", name, values)
		}
		c.Next()
	})

	r.POST("/download", handleDownload)
	r.GET("/status/:id", getTorrentStatus)
	r.GET("/torrents", listTorrents)

	r.Run(":8080")
}
