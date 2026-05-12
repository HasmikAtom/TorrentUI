package plex

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	clientIdentifier = "torrent-ui"
	clientProduct    = "TorrentUI"
)

type PlexClient struct {
	httpClient *http.Client
	discoverer *discoverer
}

func newClient(httpClient *http.Client) *PlexClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &PlexClient{httpClient: httpClient}
}

// New constructs a fully-wired client with cached discovery.
func New(httpClient *http.Client) *PlexClient {
	c := newClient(httpClient)
	c.discoverer = newDiscoverer("", httpClient, newDiscoveryCache(5*time.Minute))
	return c
}

// ResolveServer returns the user's PMS connection. Used by handlers.
func (c *PlexClient) ResolveServer(userID, token string) (ServerConn, error) {
	if c.discoverer == nil {
		return ServerConn{}, ErrServerUnreachable
	}
	return c.discoverer.resolve(userID, token)
}

// InvalidateServer drops the cached PMS for a user (e.g. on 401).
func (c *PlexClient) InvalidateServer(userID string) {
	if c.discoverer != nil {
		c.discoverer.cache.invalidate(userID)
	}
}

type mediaContainer struct {
	Size      int             `json:"size"`
	TotalSize int             `json:"totalSize"`
	Offset    int             `json:"offset"`
	Directory []sectionEntry  `json:"Directory"`
	Metadata  []metadataEntry `json:"Metadata"`
}

type sectionsResponse struct {
	MediaContainer mediaContainer `json:"MediaContainer"`
}

type sectionEntry struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

type metadataEntry struct {
	RatingKey      string        `json:"ratingKey"`
	Title          string        `json:"title"`
	Year           int           `json:"year"`
	Thumb          string        `json:"thumb"`
	Art            string        `json:"art"`
	Rating         float64       `json:"rating"`
	AudienceRating float64       `json:"audienceRating"`
	Duration       int64         `json:"duration"`
	AddedAt        int64         `json:"addedAt"`
	Summary        string        `json:"summary"`
	ContentRating  string        `json:"contentRating"`
	Studio         string        `json:"studio"`
	OriginallyAt   string        `json:"originallyAvailableAt"`
	Genre          []taggedEntry `json:"Genre"`
	Director       []taggedEntry `json:"Director"`
	Writer         []taggedEntry `json:"Writer"`
	Role           []taggedEntry `json:"Role"`
}

type taggedEntry struct {
	Tag string `json:"tag"`
}

// ListMovies queries movie libraries on the user's PMS. v1: paginates within
// the first movie library; if exhausted, advances to the next library.
func (c *PlexClient) ListMovies(conn ServerConn, token string, start, size int, sort string) (ListMoviesResult, error) {
	libs, err := c.listMovieLibraries(conn, token)
	if err != nil {
		return ListMoviesResult{}, err
	}
	if len(libs) == 0 {
		return ListMoviesResult{Items: []Movie{}, Total: 0, Start: start, Size: 0}, nil
	}

	// Single library: just query it.
	if len(libs) == 1 {
		return c.queryLibraryPage(conn, token, libs[0].Key, start, size, sort)
	}
	return c.queryAcrossLibraries(conn, token, libs, start, size, sort)
}

func (c *PlexClient) listMovieLibraries(conn ServerConn, token string) ([]sectionEntry, error) {
	var resp sectionsResponse
	if err := c.getJSON(conn.BaseURL+"/library/sections", token, nil, &resp); err != nil {
		return nil, err
	}
	out := []sectionEntry{}
	for _, d := range resp.MediaContainer.Directory {
		if d.Type == "movie" {
			out = append(out, d)
		}
	}
	return out, nil
}

func (c *PlexClient) queryLibraryPage(conn ServerConn, token, libKey string, start, size int, sort string) (ListMoviesResult, error) {
	q := url.Values{}
	q.Set("type", "1")
	q.Set("X-Plex-Container-Start", strconv.Itoa(start))
	q.Set("X-Plex-Container-Size", strconv.Itoa(size))
	q.Set("sort", sort)

	var resp sectionsResponse
	endpoint := fmt.Sprintf("%s/library/sections/%s/all", conn.BaseURL, libKey)
	if err := c.getJSON(endpoint, token, q, &resp); err != nil {
		return ListMoviesResult{}, err
	}
	items := make([]Movie, 0, len(resp.MediaContainer.Metadata))
	for _, m := range resp.MediaContainer.Metadata {
		items = append(items, toMovie(m))
	}
	return ListMoviesResult{
		Items: items,
		Total: resp.MediaContainer.TotalSize,
		Start: start,
		Size:  len(items),
	}, nil
}

// queryAcrossLibraries paginates virtually across multiple movie libraries
// by treating them as a concatenated stream in the order Plex returned them.
// First it sums totals across libraries (one /all?size=0 request each, cheap),
// then walks libraries until `size` items have been gathered starting at `start`.
func (c *PlexClient) queryAcrossLibraries(conn ServerConn, token string, libs []sectionEntry, start, size int, sort string) (ListMoviesResult, error) {
	libTotals := make([]int, len(libs))
	grandTotal := 0
	for i, lib := range libs {
		t, err := c.libraryTotal(conn, token, lib.Key, sort)
		if err != nil {
			return ListMoviesResult{}, err
		}
		libTotals[i] = t
		grandTotal += t
	}

	items := make([]Movie, 0, size)
	cursor := 0 // virtual offset across the concatenation
	remaining := size

	for i, lib := range libs {
		libStart := cursor // virtual offset where this library begins
		libEnd := cursor + libTotals[i]
		cursor = libEnd

		if remaining <= 0 {
			break
		}
		if start >= libEnd {
			continue
		}
		// Translate virtual offsets to per-library offsets.
		localStart := 0
		if start > libStart {
			localStart = start - libStart
		}
		want := remaining
		available := libTotals[i] - localStart
		if want > available {
			want = available
		}
		if want <= 0 {
			continue
		}
		page, err := c.queryLibraryPage(conn, token, lib.Key, localStart, want, sort)
		if err != nil {
			return ListMoviesResult{}, err
		}
		items = append(items, page.Items...)
		remaining -= len(page.Items)
	}

	return ListMoviesResult{
		Items: items,
		Total: grandTotal,
		Start: start,
		Size:  len(items),
	}, nil
}

func (c *PlexClient) libraryTotal(conn ServerConn, token, libKey, sort string) (int, error) {
	q := url.Values{}
	q.Set("type", "1")
	q.Set("X-Plex-Container-Start", "0")
	q.Set("X-Plex-Container-Size", "0")
	q.Set("sort", sort)

	var resp sectionsResponse
	endpoint := fmt.Sprintf("%s/library/sections/%s/all", conn.BaseURL, libKey)
	if err := c.getJSON(endpoint, token, q, &resp); err != nil {
		return 0, err
	}
	return resp.MediaContainer.TotalSize, nil
}

func (c *PlexClient) getJSON(endpoint, token string, query url.Values, out interface{}) error {
	if query != nil && len(query) > 0 {
		endpoint = endpoint + "?" + query.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("X-Plex-Token", token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Plex-Client-Identifier", clientIdentifier)
	req.Header.Set("X-Plex-Product", clientProduct)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ErrServerUnreachable
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return ErrServerUnreachable
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return ErrServerUnreachable
	}
	return nil
}

// GetMovie returns the detail for a single movie by rating key.
func (c *PlexClient) GetMovie(conn ServerConn, token, ratingKey string) (MovieDetail, error) {
	var resp sectionsResponse
	endpoint := fmt.Sprintf("%s/library/metadata/%s", conn.BaseURL, ratingKey)
	if err := c.getJSON(endpoint, token, nil, &resp); err != nil {
		return MovieDetail{}, err
	}
	if len(resp.MediaContainer.Metadata) == 0 {
		return MovieDetail{}, ErrServerUnreachable
	}
	m := resp.MediaContainer.Metadata[0]

	d := MovieDetail{
		Movie:                 toMovie(m),
		ContentRating:         m.ContentRating,
		Studio:                m.Studio,
		OriginallyAvailableAt: m.OriginallyAt,
		Genres:                tagsOf(m.Genre),
		Directors:             tagsOf(m.Director),
		Writers:               tagsOf(m.Writer),
		Cast:                  tagsOf(m.Role),
	}
	if len(d.Cast) > 6 {
		d.Cast = d.Cast[:6]
	}
	return d, nil
}

func tagsOf(in []taggedEntry) []string {
	out := make([]string, 0, len(in))
	for _, t := range in {
		out = append(out, t.Tag)
	}
	return out
}

// FetchImage returns an open HTTP response streaming the image bytes.
// Caller MUST close resp.Body.
func (c *PlexClient) FetchImage(conn ServerConn, token, path string) (*http.Response, error) {
	endpoint := conn.BaseURL + path
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build image request: %w", err)
	}
	req.Header.Set("X-Plex-Token", token)
	req.Header.Set("X-Plex-Client-Identifier", clientIdentifier)
	req.Header.Set("X-Plex-Product", clientProduct)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, ErrServerUnreachable
	}
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		return nil, ErrUnauthorized
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, ErrServerUnreachable
	}
	return resp, nil
}

func toMovie(m metadataEntry) Movie {
	return Movie{
		RatingKey:      m.RatingKey,
		Title:          m.Title,
		Year:           m.Year,
		Thumb:          m.Thumb,
		Art:            m.Art,
		Rating:         m.Rating,
		AudienceRating: m.AudienceRating,
		Duration:       m.Duration,
		AddedAt:        m.AddedAt,
		Summary:        m.Summary,
	}
}
