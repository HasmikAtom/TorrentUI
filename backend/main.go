package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/hasmikatom/torrent/scraper"
	"github.com/hasmikatom/torrent/transmission"
	"github.com/joho/godotenv"
)

var c *Config
var client *transmission.TransmissionRPC
var s *scraper.Scraper

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

	s = scraper.InitScraper()
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
	r.POST("/download/file/", handleFileDownload)
	r.GET("/status/:id", getTorrentStatus)
	r.GET("/torrents", listTorrents)

	r.POST("/scrape/piratebay/:name", scrapePirateBay)
	r.POST("/scrape/rutracker/:name", scrapeRuTracker)

	r.Run(":" + c.AppPort)
}
