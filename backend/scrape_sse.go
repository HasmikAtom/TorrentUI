package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hasmikatom/torrent/scraper"
)

// SSE event types
type SSEEvent struct {
	Type    string `json:"type"`    // "trying", "success", "error", "complete"
	Message string `json:"message"` // Human readable message
	Host    string `json:"host"`    // Current host being tried
	Label   string `json:"label"`   // Label for the host (e.g., "Main Site", "Proxy 1")
	Data    any    `json:"data,omitempty"`
}

func sendSSEEvent(c *gin.Context, event SSEEvent) {
	data, _ := json.Marshal(event)
	fmt.Fprintf(c.Writer, "data: %s\n\n", data)
	c.Writer.Flush()
}

func scrapePirateBaySSE(c *gin.Context) {
	name := c.Param("name")
	config := scraper.GetScraperConfig()
	source := config.ThePirateBay

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	var results []scraper.PirateBayTorrent
	var lastError error
	success := false

urlLoop:
	for i, urlConfig := range source.URLs {
		// Send "trying" event
		sendSSEEvent(c, SSEEvent{
			Type:    "trying",
			Message: fmt.Sprintf("Trying %s (%d/%d)...", urlConfig.Label, i+1, len(source.URLs)),
			Host:    urlConfig.Host,
			Label:   urlConfig.Label,
		})

		u := &url.URL{
			Scheme: "https",
			Host:   urlConfig.Host,
			Path:   "/search.php",
		}

		q := u.Query()
		q.Add("q", name)
		q.Add("all", "on")
		q.Add("search", "Pirate + Search")
		q.Add("page", "0")
		q.Add("orderby", "")
		u.RawQuery = q.Encode()

		// Try scraping with timeout
		done := make(chan struct{})
		var scrapeErr error
		var scrapeResults []scraper.PirateBayTorrent

		go func() {
			scrapeResults, scrapeErr = scraper.ScrapePirateBayWithTimeout(u.String(), time.Duration(source.TimeoutSeconds)*time.Second)
			close(done)
		}()

		select {
		case <-done:
			if scrapeErr == nil && len(scrapeResults) > 0 {
				results = scrapeResults
				success = true
				sendSSEEvent(c, SSEEvent{
					Type:    "success",
					Message: fmt.Sprintf("Found %d results on %s", len(results), urlConfig.Label),
					Host:    urlConfig.Host,
					Label:   urlConfig.Label,
				})
				break urlLoop
			} else {
				lastError = scrapeErr
				errMsg := "No results found"
				if scrapeErr != nil {
					errMsg = scrapeErr.Error()
				}
				sendSSEEvent(c, SSEEvent{
					Type:    "error",
					Message: fmt.Sprintf("%s failed: %s", urlConfig.Label, errMsg),
					Host:    urlConfig.Host,
					Label:   urlConfig.Label,
				})
			}
		case <-time.After(time.Duration(source.TimeoutSeconds+5) * time.Second):
			sendSSEEvent(c, SSEEvent{
				Type:    "error",
				Message: fmt.Sprintf("%s timed out", urlConfig.Label),
				Host:    urlConfig.Host,
				Label:   urlConfig.Label,
			})
		}
	}

	// Send final result
	if success {
		sendSSEEvent(c, SSEEvent{
			Type:    "complete",
			Message: fmt.Sprintf("Search complete - found %d results", len(results)),
			Data:    results,
		})
	} else {
		errMsg := "All sources failed"
		if lastError != nil {
			errMsg = lastError.Error()
		}
		sendSSEEvent(c, SSEEvent{
			Type:    "complete",
			Message: errMsg,
			Data:    nil,
		})
	}
}

func scrapeRuTrackerSSE(ctx *gin.Context) {
	name := ctx.Param("name")
	config := scraper.GetScraperConfig()
	source := config.RuTracker

	// Set SSE headers
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("Access-Control-Allow-Origin", "*")

	creds := scraper.RutrackerCredentials{
		Username: c.RutrackerUsername,
		Password: c.RutrackerPassword,
	}

	var results []scraper.RutrackerTorrent
	var lastError error
	success := false

ruTrackerLoop:
	for i, urlConfig := range source.URLs {
		// Send "trying" event
		sendSSEEvent(ctx, SSEEvent{
			Type:    "trying",
			Message: fmt.Sprintf("Trying %s (%d/%d)...", urlConfig.Label, i+1, len(source.URLs)),
			Host:    urlConfig.Host,
			Label:   urlConfig.Label,
		})

		u := &url.URL{
			Scheme: "https",
			Host:   urlConfig.Host,
			Path:   "/forum/index.php",
		}

		// Try scraping with timeout
		done := make(chan struct{})
		var scrapeErr error
		var scrapeResults []scraper.RutrackerTorrent

		go func() {
			scrapeResults, scrapeErr = scraper.ScrapeRuTrackerWithTimeout(u.String(), name, creds, time.Duration(source.TimeoutSeconds)*time.Second)
			close(done)
		}()

		select {
		case <-done:
			if scrapeErr == nil && len(scrapeResults) > 0 {
				results = scrapeResults
				success = true
				sendSSEEvent(ctx, SSEEvent{
					Type:    "success",
					Message: fmt.Sprintf("Found %d results on %s", len(results), urlConfig.Label),
					Host:    urlConfig.Host,
					Label:   urlConfig.Label,
				})
				break ruTrackerLoop
			} else {
				lastError = scrapeErr
				errMsg := "No results found"
				if scrapeErr != nil {
					errMsg = scrapeErr.Error()
				}
				sendSSEEvent(ctx, SSEEvent{
					Type:    "error",
					Message: fmt.Sprintf("%s failed: %s", urlConfig.Label, errMsg),
					Host:    urlConfig.Host,
					Label:   urlConfig.Label,
				})
			}
		case <-time.After(time.Duration(source.TimeoutSeconds+5) * time.Second):
			sendSSEEvent(ctx, SSEEvent{
				Type:    "error",
				Message: fmt.Sprintf("%s timed out", urlConfig.Label),
				Host:    urlConfig.Host,
				Label:   urlConfig.Label,
			})
		}
	}

	// Send final result
	if success {
		sendSSEEvent(ctx, SSEEvent{
			Type:    "complete",
			Message: fmt.Sprintf("Search complete - found %d results", len(results)),
			Data:    results,
		})
	} else {
		errMsg := "All sources failed"
		if lastError != nil {
			errMsg = lastError.Error()
		}
		sendSSEEvent(ctx, SSEEvent{
			Type:    "complete",
			Message: errMsg,
			Data:    nil,
		})
	}
}

// GetScraperSources returns available scraper sources for the frontend
func getScraperSources(c *gin.Context) {
	config := scraper.GetScraperConfig()

	response := map[string]any{
		"thepiratebay": map[string]any{
			"name": config.ThePirateBay.Name,
			"urls": config.ThePirateBay.URLs,
		},
		"rutracker": map[string]any{
			"name": config.RuTracker.Name,
			"urls": config.RuTracker.URLs,
		},
	}

	c.JSON(http.StatusOK, response)
}
