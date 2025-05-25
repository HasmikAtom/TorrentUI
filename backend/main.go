package main

import (
	"fmt"
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var c *Config
var client *TransmissionRPC

func init() {
	godotenv.Load()

	c = SetConfigs()

	client = &TransmissionRPC{
		URL:      fmt.Sprintf("http://%s:%s/transmission/rpc", c.TransmissionHost, c.TransmissionPort),
		Username: c.TransmissionUsername,
		Password: c.TransmissionPassword,
	}
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
	r.POST("/scrape/:name", scrape)

	r.Run(":" + c.AppPort)
}
