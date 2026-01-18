package scraper

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

type ScraperURL struct {
	Host  string `json:"host"`
	Label string `json:"label"`
}

type ScraperSource struct {
	Name           string       `json:"name"`
	TimeoutSeconds int          `json:"timeout_seconds"`
	URLs           []ScraperURL `json:"urls"`
}

type ScraperConfig struct {
	ThePirateBay ScraperSource `json:"thepiratebay"`
	RuTracker    ScraperSource `json:"rutracker"`
}

var (
	scraperConfig     *ScraperConfig
	scraperConfigOnce sync.Once
)

func LoadScraperConfig() *ScraperConfig {
	scraperConfigOnce.Do(func() {
		// Try multiple paths for the config file
		paths := []string{
			"config/scrapers.json",
			"../config/scrapers.json",
			"/app/config/scrapers.json",
		}

		var data []byte
		var err error
		for _, path := range paths {
			data, err = os.ReadFile(path)
			if err == nil {
				log.Printf("Loaded scraper config from: %s", path)
				break
			}
		}

		if err != nil {
			log.Printf("Warning: Could not load scrapers.json, using defaults: %v", err)
			scraperConfig = getDefaultConfig()
			return
		}

		var config ScraperConfig
		if err := json.Unmarshal(data, &config); err != nil {
			log.Printf("Warning: Could not parse scrapers.json, using defaults: %v", err)
			scraperConfig = getDefaultConfig()
			return
		}

		scraperConfig = &config
	})

	return scraperConfig
}

func getDefaultConfig() *ScraperConfig {
	return &ScraperConfig{
		ThePirateBay: ScraperSource{
			Name:           "ThePirateBay",
			TimeoutSeconds: 15,
			URLs: []ScraperURL{
				{Host: "thepiratebay.org", Label: "Main Site"},
			},
		},
		RuTracker: ScraperSource{
			Name:           "RuTracker",
			TimeoutSeconds: 20,
			URLs: []ScraperURL{
				{Host: "rutracker.org", Label: "Main Site"},
			},
		},
	}
}

func GetScraperConfig() *ScraperConfig {
	return LoadScraperConfig()
}
