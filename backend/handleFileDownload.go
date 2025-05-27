package main

import (
	"context"
	"encoding/base64"
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

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	fileURL := req.URL
	log.Println("downloading ", fileURL, "as a ", req.MediaType)

	if req.URL == "" || req.MediaType == "" {
		c.JSON(400, gin.H{"error": "url or mediaType required"})
		return
	}

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	defer cancel()

	RutrackerLogin(ctx, fileURL)

	n, err := downloadFile(ctx, req.URL, torrentFileSaveLocation)
	if err != nil {
		log.Println("error downloading file ===> ", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	torrentLocation := filepath.Join(torrentFileSaveLocation, n)

	log.Println("transmission is about to receive the file ===>", torrentLocation)

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
