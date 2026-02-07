package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/hasmikatom/torrent/scraper"
	"github.com/hasmikatom/torrent/transmission"
	"github.com/joho/godotenv"
)

var c *Config
var client *transmission.TransmissionRPC

func init() {
	godotenv.Load()

	c = SetConfigs()

	client = &transmission.TransmissionRPC{
		URL:      fmt.Sprintf("http://%s:%s/transmission/rpc", c.TransmissionHost, c.TransmissionPort),
		Username: c.TransmissionUsername,
		Password: c.TransmissionPassword,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	if err := scraper.GetPool().Init(); err != nil {
		log.Printf("Warning: Failed to initialize browser pool: %v", err)
	}

	// Load scraper config
	scraper.LoadScraperConfig()
}

func main() {
	r := gin.Default()

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
	config.AllowHeaders = []string{"*"}

	r.Use(cors.New(config))

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.POST("/download", handleDownload)
	r.POST("/download/file", handleFileDownload)
	r.POST("/download/batch", handleBatchDownload)
	r.POST("/download/file/batch", handleBatchFileDownload)

	// Prepare/finalize endpoints for name editing flow
	r.POST("/download/prepare", handlePrepareDownload)
	r.POST("/download/file/prepare", handleFilePrepareDownload)
	r.POST("/download/prepare/batch", handleBatchPrepareDownload)
	r.POST("/download/file/prepare/batch", handleBatchFilePrepareDownload)
	r.GET("/download/prepare/status/:id", handlePrepareStatus)
	r.POST("/download/finalize", handleFinalizeDownload)
	r.POST("/download/cancel", handleCancelDownload)
	r.GET("/status/:id", getTorrentStatus)
	r.GET("/torrents", listTorrents)
	r.DELETE("/torrents/:id", deleteTorrent)
	r.GET("/storage", getStorageInfo)

	r.POST("/scrape/piratebay/:name", scrapePirateBay)
	r.POST("/scrape/rutracker/:name", scrapeRuTracker)

	// SSE endpoints for real-time scrape progress
	r.GET("/scrape/piratebay/:name/stream", scrapePirateBaySSE)
	r.GET("/scrape/rutracker/:name/stream", scrapeRuTrackerSSE)
	r.GET("/scrape/sources", getScraperSources)

	// Create server with graceful shutdown
	srv := &http.Server{
		Addr:    ":" + c.AppPort,
		Handler: r,
	}

	// Start server in goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("Server started on port %s", c.AppPort)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Shutdown browser pool
	scraper.GetPool().Shutdown()

	// Give outstanding requests 5 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
