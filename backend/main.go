package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var client *TransmissionRPC

func init() {
	host := os.Getenv("TRANSMISSION_HOST")
	if host == "" {
		host = "host.docker.internal"
	}

	port := os.Getenv("TRANSMISSION_PORT")
	if port == "" {
		port = "9091"
	}

	username := os.Getenv("TRANSMISSION_USERNAME")
	password := os.Getenv("TRANSMISSION_PASSWORD")

	client = &TransmissionRPC{
		URL:      fmt.Sprintf("http://%s:%s/transmission/rpc", host, port),
		Username: username,
		Password: password,
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

	r.Run(":8080")
}
