package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hasmikatom/torrent/scraper"
)

// PrepareResponse is the response for prepare endpoints
type PrepareResponse struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Ready bool   `json:"ready"`
}

// PrepareStatusResponse is the response for status polling
type PrepareStatusResponse struct {
	ID                      int     `json:"id"`
	Name                    string  `json:"name"`
	Ready                   bool    `json:"ready"`
	MetadataPercentComplete float64 `json:"metadataPercentComplete"`
}

// FinalizeRequest is the request for finalize endpoint
type FinalizeRequest struct {
	Torrents    []TorrentFinalize `json:"torrents"`
	ContentType string            `json:"contentType"`
}

// TorrentFinalize contains the torrent ID and optional new name
type TorrentFinalize struct {
	ID      int    `json:"id"`
	NewName string `json:"newName,omitempty"`
}

// CancelRequest is the request for cancel endpoint
type CancelRequest struct {
	IDs []int `json:"ids"`
}

// BatchPrepareRequest is the request for batch prepare
type BatchPrepareRequest struct {
	MagnetLinks []string `json:"magnetLinks"`
}

// BatchFilePrepareRequest is the request for batch file prepare
type BatchFilePrepareRequest struct {
	URLs []string `json:"urls"`
}

// BatchPrepareResponse is the response for batch prepare
type BatchPrepareResponse struct {
	Torrents []PrepareResponse `json:"torrents"`
	Errors   []string          `json:"errors,omitempty"`
}

// handlePrepareDownload adds a torrent paused and returns its info
func handlePrepareDownload(gc *gin.Context) {
	magnetLink := gc.PostForm("magnetLink")
	torrentFile, _ := gc.FormFile("torrentFile")

	if magnetLink == "" && torrentFile == nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Either magnet link or torrent file is required"})
		return
	}

	var filename string
	var isTempFile bool
	if torrentFile != nil {
		filename = fmt.Sprintf("/tmp/%s", torrentFile.Filename)
		if err := gc.SaveUploadedFile(torrentFile, filename); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save torrent file"})
			return
		}
		isTempFile = true
		defer func() {
			if isTempFile {
				os.Remove(filename)
			}
		}()
	} else {
		filename = magnetLink
	}

	// Add torrent paused
	args := map[string]interface{}{
		"filename": filename,
		"paused":   true,
	}

	result, err := client.SendRequest("torrent-add", args)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result.Result != "success" {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add torrent"})
		return
	}

	// Check for torrent-added or torrent-duplicate
	var torrentInfo map[string]interface{}
	if added, ok := result.Arguments["torrent-added"].(map[string]interface{}); ok {
		torrentInfo = added
	} else if duplicate, ok := result.Arguments["torrent-duplicate"].(map[string]interface{}); ok {
		torrentInfo = duplicate
	}

	if torrentInfo == nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get torrent info"})
		return
	}

	id, idOk := GetInt(torrentInfo, "id")
	name, nameOk := GetString(torrentInfo, "name")

	if !idOk {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get torrent ID"})
		return
	}

	// For magnet links, metadata may not be ready yet
	// For torrent files, metadata is always ready
	ready := torrentFile != nil || nameOk

	gc.JSON(http.StatusOK, PrepareResponse{
		ID:    id,
		Name:  name,
		Ready: ready,
	})
}

// handleFilePrepareDownload handles RuTracker file prepare
func handleFilePrepareDownload(gc *gin.Context) {
	torrentFileSaveLocation := "/mediastorage/torrent-files"

	url := gc.PostForm("url")
	if url == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "url is required"})
		return
	}

	// Use browser pool for download
	ctx, cancel := scraper.GetPool().NewTabContext(120 * time.Second)
	defer cancel()

	creds := scraper.RutrackerCredentials{
		Username: c.RutrackerUsername,
		Password: c.RutrackerPassword,
	}

	if err := scraper.RutrackerLogin(ctx, url, creds); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to login"})
		return
	}

	filename, err := downloadFile(ctx, url, torrentFileSaveLocation)
	if err != nil {
		log.Printf("Error downloading file: %v", err)
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to download torrent file"})
		return
	}

	torrentLocation := filepath.Join(torrentFileSaveLocation, filename)
	defer os.Remove(torrentLocation)

	torrentData, err := os.ReadFile(torrentLocation)
	if err != nil {
		log.Printf("Failed to read torrent file: %v", err)
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read torrent file"})
		return
	}

	base64Data := base64.StdEncoding.EncodeToString(torrentData)

	// Add torrent paused
	args := map[string]interface{}{
		"metainfo": base64Data,
		"paused":   true,
	}

	result, err := client.SendRequest("torrent-add", args)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result.Result != "success" {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add torrent"})
		return
	}

	var torrentInfo map[string]interface{}
	if added, ok := result.Arguments["torrent-added"].(map[string]interface{}); ok {
		torrentInfo = added
	} else if duplicate, ok := result.Arguments["torrent-duplicate"].(map[string]interface{}); ok {
		torrentInfo = duplicate
	}

	if torrentInfo == nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get torrent info"})
		return
	}

	id, idOk := GetInt(torrentInfo, "id")
	name, _ := GetString(torrentInfo, "name")

	if !idOk {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get torrent ID"})
		return
	}

	// For torrent files, metadata is always ready
	gc.JSON(http.StatusOK, PrepareResponse{
		ID:    id,
		Name:  name,
		Ready: true,
	})
}

// handleBatchPrepareDownload handles batch magnet link prepare
func handleBatchPrepareDownload(gc *gin.Context) {
	var req BatchPrepareRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if len(req.MagnetLinks) == 0 {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "At least one magnet link is required"})
		return
	}

	var torrents []PrepareResponse
	var errors []string

	for _, magnetLink := range req.MagnetLinks {
		args := map[string]interface{}{
			"filename": magnetLink,
			"paused":   true,
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

		var torrentInfo map[string]interface{}
		if added, ok := result.Arguments["torrent-added"].(map[string]interface{}); ok {
			torrentInfo = added
		} else if duplicate, ok := result.Arguments["torrent-duplicate"].(map[string]interface{}); ok {
			torrentInfo = duplicate
		}

		if torrentInfo == nil {
			errors = append(errors, "Failed to get torrent info")
			continue
		}

		id, idOk := GetInt(torrentInfo, "id")
		name, nameOk := GetString(torrentInfo, "name")

		if !idOk {
			errors = append(errors, "Failed to get torrent ID")
			continue
		}

		torrents = append(torrents, PrepareResponse{
			ID:    id,
			Name:  name,
			Ready: nameOk,
		})
	}

	gc.JSON(http.StatusOK, BatchPrepareResponse{
		Torrents: torrents,
		Errors:   errors,
	})
}

// handleBatchFilePrepareDownload handles batch RuTracker file prepare
func handleBatchFilePrepareDownload(gc *gin.Context) {
	torrentFileSaveLocation := "/mediastorage/torrent-files"

	var req BatchFilePrepareRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if len(req.URLs) == 0 {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "At least one URL is required"})
		return
	}

	// Use browser pool for downloads
	ctx, cancel := scraper.GetPool().NewTabContext(180 * time.Second)
	defer cancel()

	creds := scraper.RutrackerCredentials{
		Username: c.RutrackerUsername,
		Password: c.RutrackerPassword,
	}

	// Login once
	if err := scraper.RutrackerLogin(ctx, req.URLs[0], creds); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to login to RuTracker"})
		return
	}

	var torrents []PrepareResponse
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
		os.Remove(torrentLocation)

		if err != nil {
			log.Printf("Failed to read torrent file: %v", err)
			errors = append(errors, "Failed to read torrent file")
			continue
		}

		base64Data := base64.StdEncoding.EncodeToString(torrentData)

		args := map[string]interface{}{
			"metainfo": base64Data,
			"paused":   true,
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

		var torrentInfo map[string]interface{}
		if added, ok := result.Arguments["torrent-added"].(map[string]interface{}); ok {
			torrentInfo = added
		} else if duplicate, ok := result.Arguments["torrent-duplicate"].(map[string]interface{}); ok {
			torrentInfo = duplicate
		}

		if torrentInfo == nil {
			errors = append(errors, "Failed to get torrent info")
			continue
		}

		id, idOk := GetInt(torrentInfo, "id")
		name, _ := GetString(torrentInfo, "name")

		if !idOk {
			errors = append(errors, "Failed to get torrent ID")
			continue
		}

		torrents = append(torrents, PrepareResponse{
			ID:    id,
			Name:  name,
			Ready: true,
		})
	}

	gc.JSON(http.StatusOK, BatchPrepareResponse{
		Torrents: torrents,
		Errors:   errors,
	})
}

// handlePrepareStatus checks metadata completion status
func handlePrepareStatus(gc *gin.Context) {
	id := gc.Param("id")
	var torrentId int
	fmt.Sscanf(id, "%d", &torrentId)

	args := map[string]interface{}{
		"ids": []int{torrentId},
		"fields": []string{
			"id",
			"name",
			"metadataPercentComplete",
		},
	}

	result, err := client.SendRequest("torrent-get", args)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if torrents, ok := result.Arguments["torrents"].([]interface{}); ok && len(torrents) > 0 {
		if torrent, ok := torrents[0].(map[string]interface{}); ok {
			id, _ := GetInt(torrent, "id")
			name, _ := GetString(torrent, "name")
			metadataPercent, _ := GetFloat64(torrent, "metadataPercentComplete")

			ready := metadataPercent >= 1.0

			gc.JSON(http.StatusOK, PrepareStatusResponse{
				ID:                      id,
				Name:                    name,
				Ready:                   ready,
				MetadataPercentComplete: metadataPercent,
			})
			return
		}
	}

	gc.JSON(http.StatusNotFound, gin.H{"error": "Torrent not found"})
}

// handleFinalizeDownload renames (if needed) and starts torrents
func handleFinalizeDownload(gc *gin.Context) {
	var req FinalizeRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if len(req.Torrents) == 0 {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "At least one torrent is required"})
		return
	}

	if req.ContentType == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "contentType is required"})
		return
	}

	downloadDir, err := GetDownloadDir(req.ContentType)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var torrentIds []int
	var errors []string

	for _, t := range req.Torrents {
		// First, set the download directory
		setLocationArgs := map[string]interface{}{
			"ids":      []int{t.ID},
			"location": downloadDir,
			"move":     false,
		}

		_, err := client.SendRequest("torrent-set-location", setLocationArgs)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to set location for torrent %d: %v", t.ID, err))
			continue
		}

		// Rename if new name provided
		if t.NewName != "" {
			// Get current name first
			getArgs := map[string]interface{}{
				"ids":    []int{t.ID},
				"fields": []string{"name"},
			}

			getResult, err := client.SendRequest("torrent-get", getArgs)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Failed to get torrent info: %v", err))
				continue
			}

			var currentName string
			if torrents, ok := getResult.Arguments["torrents"].([]interface{}); ok && len(torrents) > 0 {
				if torrent, ok := torrents[0].(map[string]interface{}); ok {
					currentName, _ = GetString(torrent, "name")
				}
			}

			if currentName != "" && currentName != t.NewName {
				renameArgs := map[string]interface{}{
					"ids":  []int{t.ID},
					"path": currentName,
					"name": t.NewName,
				}

				renameResult, err := client.SendRequest("torrent-rename-path", renameArgs)
				if err != nil {
					log.Printf("Failed to rename torrent %d: %v", t.ID, err)
					// Continue anyway, renaming is not critical
				} else if renameResult.Result != "success" {
					log.Printf("Failed to rename torrent %d: %s", t.ID, renameResult.Result)
				}
			}
		}

		// Start the torrent
		startArgs := map[string]interface{}{
			"ids": []int{t.ID},
		}

		startResult, err := client.SendRequest("torrent-start", startArgs)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to start torrent %d: %v", t.ID, err))
			continue
		}

		if startResult.Result != "success" {
			errors = append(errors, fmt.Sprintf("Failed to start torrent %d: %s", t.ID, startResult.Result))
			continue
		}

		torrentIds = append(torrentIds, t.ID)
	}

	gc.JSON(http.StatusOK, gin.H{
		"message":    "Torrents started",
		"torrentIds": torrentIds,
		"errors":     errors,
	})
}

// handleCancelDownload removes paused torrents
func handleCancelDownload(gc *gin.Context) {
	var req CancelRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if len(req.IDs) == 0 {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "At least one ID is required"})
		return
	}

	args := map[string]interface{}{
		"ids":               req.IDs,
		"delete-local-data": true,
	}

	result, err := client.SendRequest("torrent-remove", args)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result.Result != "success" {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove torrents"})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"message": "Torrents cancelled"})
}
