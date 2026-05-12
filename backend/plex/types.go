package plex

import (
	"errors"
	"time"
)

// ErrUnauthorized indicates Plex rejected the user's token (401).
var ErrUnauthorized = errors.New("plex: unauthorized")

// ErrServerUnreachable indicates the user's PMS could not be reached or
// no eligible server was discovered.
var ErrServerUnreachable = errors.New("plex: server unreachable")

// ErrNotConfigured indicates the user has no Plex token or has disabled
// the Plex integration.
var ErrNotConfigured = errors.New("plex: not configured")

// ServerConn is a resolved connection to a user's Plex Media Server.
type ServerConn struct {
	BaseURL           string
	MachineIdentifier string
	ResolvedAt        time.Time
}

// Movie is the summary form returned in list responses.
type Movie struct {
	RatingKey      string  `json:"ratingKey"`
	Title          string  `json:"title"`
	Year           int     `json:"year"`
	Thumb          string  `json:"thumb"`
	Art            string  `json:"art"`
	Rating         float64 `json:"rating"`
	AudienceRating float64 `json:"audienceRating"`
	Duration       int64   `json:"duration"`
	AddedAt        int64   `json:"addedAt"`
	Summary        string  `json:"summary"`
}

// MovieDetail is the expanded form returned for a single movie.
type MovieDetail struct {
	Movie
	ContentRating         string   `json:"contentRating"`
	Studio                string   `json:"studio"`
	OriginallyAvailableAt string   `json:"originallyAvailableAt"`
	Genres                []string `json:"genres"`
	Directors             []string `json:"directors"`
	Writers               []string `json:"writers"`
	Cast                  []string `json:"cast"`
}

// ListMoviesResult is the paginated list response.
type ListMoviesResult struct {
	Items []Movie `json:"items"`
	Total int     `json:"total"`
	Start int     `json:"start"`
	Size  int     `json:"size"`
}
