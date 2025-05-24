package main

import (
	"fmt"
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

func scrape(c *gin.Context) {
	// baseurl := "https://thepiratebay.org"
	name := c.Param("name")
	// // name := c.Query("name")
	// var torrentName = ""
	// fmt.Sscanf(name, "%d", &torrentName)
	// "https://thepiratebay.org/search.php?q=this+is+where+i+leave+you&all=on&search=Pirate+Search&page=0&orderby="

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

	// c.JSON(http.StatusOK, u.String())

	results, err := scraper.Scrape(u.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Scraping didnt work: "})
	}
	c.JSON(http.StatusOK, results)
}
