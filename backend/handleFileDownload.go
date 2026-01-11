package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/gin-gonic/gin"
)

type Request struct {
	URL       string `json:"url"`
	MediaType string `json:"mediaType"`
}

func handleFileDownload(c *gin.Context) {
	torrentFileSaveLocation := "/mediastorage/torrent-files"

	var req Request

	req.URL = c.PostForm("url")
	req.MediaType = c.PostForm("contentType")

	fileURL := req.URL

	if req.URL == "" || req.MediaType == "" {
		c.JSON(400, gin.H{"error": "url or contentType required"})
		return
	}

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	RutrackerLogin(ctx, fileURL)

	n, err := downloadFile(ctx, req.URL, torrentFileSaveLocation)
	if err != nil {
		log.Println("error downloading file ===> ", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	torrentLocation := filepath.Join(torrentFileSaveLocation, n)

	torrentData, err := os.ReadFile(torrentLocation)
	if err != nil {
		log.Printf("failed to read torrent file: %v", err)
	}

	base64Data := base64.StdEncoding.EncodeToString(torrentData)

	args := map[string]interface{}{
		"download-dir": "/mediastorage/" + req.MediaType,
		"metainfo":     base64Data,
	}
	result, err := client.SendRequest("torrent-add", args)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result.Result != "success" {
		log.Println("arguments ==> ", result.Arguments)
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

type BatchFileDownloadRequest struct {
	URLs        []string `json:"urls"`
	ContentType string   `json:"contentType"`
}

type BatchFileDownloadResponse struct {
	TorrentIds []int    `json:"torrentIds"`
	Errors     []string `json:"errors,omitempty"`
}

func handleBatchFileDownload(c *gin.Context) {
	torrentFileSaveLocation := "/mediastorage/torrent-files"

	var req BatchFileDownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}

	if len(req.URLs) == 0 {
		c.JSON(400, gin.H{"error": "At least one URL is required"})
		return
	}

	if req.ContentType == "" {
		c.JSON(400, gin.H{"error": "contentType is required"})
		return
	}

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Login once using the first URL
	if err := RutrackerLogin(ctx, req.URLs[0]); err != nil {
		c.JSON(500, gin.H{"error": "Failed to login to RuTracker"})
		return
	}

	var torrentIds []int
	var errors []string

	for _, fileURL := range req.URLs {
		filename, err := downloadFile(ctx, fileURL, torrentFileSaveLocation)
		if err != nil {
			log.Printf("Error downloading file %s: %v", fileURL, err)
			errors = append(errors, fmt.Sprintf("Failed to download: %v", err))
			continue
		}

		torrentLocation := filepath.Join(torrentFileSaveLocation, filename)

		torrentData, err := os.ReadFile(torrentLocation)
		if err != nil {
			log.Printf("Failed to read torrent file: %v", err)
			errors = append(errors, fmt.Sprintf("Failed to read torrent file: %v", err))
			continue
		}

		base64Data := base64.StdEncoding.EncodeToString(torrentData)

		args := map[string]interface{}{
			"download-dir": "/mediastorage/" + req.ContentType,
			"metainfo":     base64Data,
		}

		result, err := client.SendRequest("torrent-add", args)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to add torrent: %v", err))
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

	c.JSON(200, response)
}

func RutrackerLogin(ctx context.Context, url string) error {

	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),

		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("Extracting data...")
			return nil
		}),

		chromedp.WaitVisible(`a[onclick*="BB.toggle_top_login"]`, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),

		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("Login button visible...")
			return nil
		}),

		chromedp.Click(`//b[contains(text(), "Вход")]/parent::a`, chromedp.BySearch),

		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("Login button clicked...")
			return nil
		}),

		chromedp.WaitEnabled(`#top-login-uname`, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("Login fields available...")
			return nil
		}),
		chromedp.SendKeys(`#top-login-uname`, "HasmikAtom", chromedp.ByQuery),
		chromedp.SendKeys(`#top-login-pwd`, "57666777", chromedp.ByQuery),

		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("Login credentials filled...")
			return nil
		}),

		chromedp.Click(`#top-login-btn`, chromedp.ByQuery),

		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("Login button clicked...")
			return nil
		}),

		chromedp.Sleep(3*time.Second),

		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("waiting for the search bar and button...")
			return nil
		}),

		chromedp.WaitVisible(`#search-text`, chromedp.ByQuery),

		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("search button visible... all set for downloading torrent file...")
			return nil
		}),
	)

	if err != nil {
		log.Println("something happened while logging in ==> ", err)
	}

	return nil
}
