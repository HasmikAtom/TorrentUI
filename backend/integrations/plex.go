package integrations

import (
	"fmt"
	"net/http"
	"time"
)

const defaultPlexAPIURL = "https://plex.tv/api/v2/user"

var plexHTTPClient = &http.Client{Timeout: 10 * time.Second}

func ValidatePlexToken(token string, baseURL string) error {
	if token == "" {
		return fmt.Errorf("token is required")
	}

	url := baseURL
	if url == "" {
		url = defaultPlexAPIURL
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("X-Plex-Token", token)
	req.Header.Set("Accept", "application/json")

	resp, err := plexHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("could not verify token — try again later")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid Plex token")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected Plex API response: %d", resp.StatusCode)
	}

	return nil
}
