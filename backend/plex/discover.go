package plex

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const defaultResourcesURL = "https://plex.tv/api/v2/resources?includeHttps=1&includeRelay=1"

type plexConnection struct {
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
	URI      string `json:"uri"`
	Local    bool   `json:"local"`
	Relay    bool   `json:"relay"`
}

type plexResource struct {
	Name             string           `json:"name"`
	ClientIdentifier string           `json:"clientIdentifier"`
	Provides         string           `json:"provides"`
	Owned            bool             `json:"owned"`
	Connections      []plexConnection `json:"connections"`
}

type discoverer struct {
	resourcesURL string
	httpClient   *http.Client
	cache        *discoveryCache
}

func newDiscoverer(resourcesURL string, httpClient *http.Client, cache *discoveryCache) *discoverer {
	if resourcesURL == "" {
		resourcesURL = defaultResourcesURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &discoverer{
		resourcesURL: resourcesURL,
		httpClient:   httpClient,
		cache:        cache,
	}
}

func (d *discoverer) resolve(userID, token string) (ServerConn, error) {
	if conn, ok := d.cache.get(userID); ok {
		return conn, nil
	}

	req, err := http.NewRequest(http.MethodGet, d.resourcesURL, nil)
	if err != nil {
		return ServerConn{}, fmt.Errorf("build resources request: %w", err)
	}
	req.Header.Set("X-Plex-Token", token)
	req.Header.Set("Accept", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return ServerConn{}, ErrServerUnreachable
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return ServerConn{}, ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		return ServerConn{}, ErrServerUnreachable
	}

	var resources []plexResource
	if err := json.NewDecoder(resp.Body).Decode(&resources); err != nil {
		return ServerConn{}, ErrServerUnreachable
	}

	for _, r := range resources {
		if !r.Owned || !containsServer(r.Provides) {
			continue
		}
		uri := pickConnection(r.Connections)
		if uri == "" {
			continue
		}
		conn := ServerConn{
			BaseURL:           uri,
			MachineIdentifier: r.ClientIdentifier,
			ResolvedAt:        time.Now(),
		}
		d.cache.set(userID, conn)
		return conn, nil
	}

	return ServerConn{}, ErrServerUnreachable
}

func containsServer(provides string) bool {
	for _, p := range splitCSV(provides) {
		if p == "server" {
			return true
		}
	}
	return false
}

func splitCSV(s string) []string {
	out := []string{}
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return out
}

// pickConnection applies the connection-selection priority:
//  1. local + https
//  2. https + !relay (plex.direct)
//  3. relay
func pickConnection(cs []plexConnection) string {
	for _, c := range cs {
		if c.Local && c.Protocol == "https" {
			return c.URI
		}
	}
	for _, c := range cs {
		if c.Protocol == "https" && !c.Relay {
			return c.URI
		}
	}
	for _, c := range cs {
		if c.Relay {
			return c.URI
		}
	}
	return ""
}
