package main

import (
	"fmt"
	"log"
	"net/http"

	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/hasmikatom/torrent/scraper"
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

	c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Torrent not found => %s", err)})
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

func scrapePirateBay(c *gin.Context) {
	name := c.Param("name")

	u := &url.URL{
		Scheme: "https",
		Host:   "thepiratebay.org",
		Path:   "/search.php",
	}

	q := u.Query()
	q.Add("q", name)
	q.Add("all", "on")
	q.Add("search", "Pirate + Search")
	q.Add("page", "0")
	q.Add("orderby", "")

	u.RawQuery = q.Encode()

	gin.DefaultWriter.Write([]byte(u.String()))

	results, err := scraper.ScrapePirateBay(u.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Scraping didnt work: %v", err.Error())})
	}
	c.JSON(http.StatusOK, results)
}

func scrapeRuTracker(c *gin.Context) {
	name := c.Param("name")

	log.Println(name)

	u := &url.URL{
		Scheme: "https",
		Host:   "rutracker.org",
		Path:   "/forum/index.php",
	}

	log.Println(u.String())
	results, err := scraper.ScrapeRuTracker(u.String(), name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Scraping didnt work: %v", err.Error())})
	}
	c.JSON(http.StatusOK, results)
}

func ruTrackerFileDownload(c *gin.Context) {
	// url := c.Param("torrentFileUrl")

	// downloadTorrentFile(url)
	// torrentData, err := downloadTorrentFile(url)
	// if err != nil {
	// 	log.Printf("failed to download torrent file: %w", err)
	// }

	return
}

// func downloadTorrentFile(url string) ([]byte, error) {
// 	req, err := http.NewRequest("GET", url, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Add common headers to mimic a browser request
// 	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
// 	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

// 	resp, err := t.client.Do(req)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		return nil, fmt.Errorf("failed to download torrent file: HTTP %d", resp.StatusCode)
// 	}

// 	data, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return data, nil
// }

// func (t *TransmissionRPCClient) addTorrentToTransmission(torrentData []byte, downloadDir ...string) error {
// 	// Encode torrent data to base64
// 	encodedTorrent := base64.StdEncoding.EncodeToString(torrentData)

// 	// Prepare arguments
// 	args := map[string]interface{}{
// 		"metainfo": encodedTorrent,
// 	}

// 	// Add download directory if specified
// 	if len(downloadDir) > 0 && downloadDir[0] != "" {
// 		args["download-dir"] = downloadDir[0]
// 	}

// 	// Create RPC request
// 	request := TransmissionRequest{
// 		Method:    "torrent-add",
// 		Arguments: args,
// 		Tag:       1,
// 	}

// 	// Send request
// 	response, err := t.sendRPCRequest(request)
// 	if err != nil {
// 		return err
// 	}

// 	if response.Result != "success" {
// 		return fmt.Errorf("transmission RPC error: %s", response.Result)
// 	}

// 	fmt.Println("Torrent added successfully to Transmission")
// 	return nil
// }
