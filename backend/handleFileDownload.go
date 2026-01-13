package main

import (
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hasmikatom/torrent/scraper"
)

type Request struct {
	URL       string `json:"url"`
	MediaType string `json:"mediaType"`
}

func handleFileDownload(gc *gin.Context) {
	torrentFileSaveLocation := "/mediastorage/torrent-files"

	var req Request

	req.URL = gc.PostForm("url")
	req.MediaType = gc.PostForm("contentType")

	fileURL := req.URL

	if req.URL == "" || req.MediaType == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "url or contentType required"})
		return
	}

	// Validate mediaType to prevent path traversal
	downloadDir, err := GetDownloadDir(req.MediaType)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Use browser pool for better resource management
	ctx, cancel := scraper.GetPool().NewTabContext(120 * time.Second)
	defer cancel()

	creds := scraper.RutrackerCredentials{
		Username: c.RutrackerUsername,
		Password: c.RutrackerPassword,
	}
	if err := scraper.RutrackerLogin(ctx, fileURL, creds); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to login"})
		return
	}

	n, err := downloadFile(ctx, req.URL, torrentFileSaveLocation)
	if err != nil {
		log.Println("error downloading file ===> ", err)
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to download torrent file"})
		return
	}

	torrentLocation := filepath.Join(torrentFileSaveLocation, n)
	defer os.Remove(torrentLocation) // Clean up downloaded torrent file

	torrentData, err := os.ReadFile(torrentLocation)
	if err != nil {
		log.Printf("failed to read torrent file: %v", err)
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read torrent file"})
		return
	}

	base64Data := base64.StdEncoding.EncodeToString(torrentData)

	args := map[string]interface{}{
		"download-dir": downloadDir,
		"metainfo":     base64Data,
	}
	result, err := client.SendRequest("torrent-add", args)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add torrent"})
		return
	}

	if result.Result != "success" {
		log.Println("arguments ==> ", result.Arguments)
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add torrent"})
		return
	}

	if torrentAdded, ok := result.Arguments["torrent-added"].(map[string]interface{}); ok {
		if id, ok := torrentAdded["id"].(float64); ok {
			gc.JSON(http.StatusOK, gin.H{
				"message":   "Download started",
				"torrentId": int(id),
			})
			return
		}
	}

	gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get torrent ID"})
}

type BatchFileDownloadRequest struct {
	URLs        []string `json:"urls"`
	ContentType string   `json:"contentType"`
}

type BatchFileDownloadResponse struct {
	TorrentIds []int    `json:"torrentIds"`
	Errors     []string `json:"errors,omitempty"`
}

func handleBatchFileDownload(gc *gin.Context) {
	torrentFileSaveLocation := "/mediastorage/torrent-files"

	var req BatchFileDownloadRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if len(req.URLs) == 0 {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "At least one URL is required"})
		return
	}

	if req.ContentType == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "contentType is required"})
		return
	}

	// Validate contentType to prevent path traversal
	downloadDir, err := GetDownloadDir(req.ContentType)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Use browser pool for better resource management
	ctx, cancel := scraper.GetPool().NewTabContext(180 * time.Second)
	defer cancel()

	creds := scraper.RutrackerCredentials{
		Username: c.RutrackerUsername,
		Password: c.RutrackerPassword,
	}

	// Login once using the first URL
	if err := scraper.RutrackerLogin(ctx, req.URLs[0], creds); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to login to RuTracker"})
		return
	}

	var torrentIds []int
	var errors []string

	for _, fileURL := range req.URLs {
		filename, err := downloadFile(ctx, fileURL, torrentFileSaveLocation)
		if err != nil {
			log.Printf("Error downloading file %s: %v", fileURL, err)
			errors = append(errors, "Failed to download file")
			continue
		}

		torrentLocation := filepath.Join(torrentFileSaveLocation, filename)

		torrentData, err := os.ReadFile(torrentLocation)
		// Clean up downloaded file regardless of success
		os.Remove(torrentLocation)

		if err != nil {
			log.Printf("Failed to read torrent file: %v", err)
			errors = append(errors, "Failed to read torrent file")
			continue
		}

		base64Data := base64.StdEncoding.EncodeToString(torrentData)

		args := map[string]interface{}{
			"download-dir": downloadDir,
			"metainfo":     base64Data,
		}

		result, err := client.SendRequest("torrent-add", args)
		if err != nil {
			errors = append(errors, "Failed to add torrent")
			continue
		}

		if result.Result != "success" {
			errors = append(errors, "Failed to add torrent: unsuccessful result")
			continue
		}

		if torrentAdded, ok := result.Arguments["torrent-added"].(map[string]interface{}); ok {
			if id, ok := torrentAdded["id"].(float64); ok {
				torrentIds = append(torrentIds, int(id))
			}
		}
	}

	response := BatchFileDownloadResponse{
		TorrentIds: torrentIds,
		Errors:     errors,
	}

	gc.JSON(http.StatusOK, response)
}

