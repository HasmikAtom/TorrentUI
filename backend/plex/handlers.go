package plex

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hasmikatom/torrent/integrations"
)

// Client is the surface the handlers depend on. Both *PlexClient and test
// fakes implement it.
type Client interface {
	ResolveServer(userID, token string) (ServerConn, error)
	InvalidateServer(userID string)
	ListMovies(conn ServerConn, token string, start, size int, sort string) (ListMoviesResult, error)
	GetMovie(conn ServerConn, token, ratingKey string) (MovieDetail, error)
	FetchImage(conn ServerConn, token, path string) (*http.Response, error)
}

var validSorts = map[string]bool{
	"addedAt:desc":  true,
	"titleSort:asc": true,
	"year:desc":     true,
	"rating:desc":   true,
}

const (
	defaultPageSize = 50
	maxPageSize     = 200
)

func RegisterHandlers(g *gin.RouterGroup, store *integrations.Store, client Client) {
	g.GET("/plex/movies", listMoviesHandler(store, client))
	g.GET("/plex/movies/:ratingKey", movieDetailHandler(store, client))
	// image handler added in next task
}

// resolveUserPlex pulls the user's token + resolves their PMS connection.
// Writes the appropriate error response and returns ok=false if anything
// is missing or unreachable.
func resolveUserPlex(c *gin.Context, store *integrations.Store, client Client) (string, ServerConn, bool) {
	userID := c.GetString("userId")
	row, err := store.GetIntegrations(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read integrations"})
		return "", ServerConn{}, false
	}
	if !row.PlexEnabled || row.PlexToken == "" {
		c.JSON(http.StatusPreconditionFailed, gin.H{"error": "plex_not_configured"})
		return "", ServerConn{}, false
	}
	conn, err := client.ResolveServer(userID, row.PlexToken)
	if err == ErrUnauthorized {
		client.InvalidateServer(userID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "plex_unauthorized"})
		return "", ServerConn{}, false
	}
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "plex_server_unreachable"})
		return "", ServerConn{}, false
	}
	return row.PlexToken, conn, true
}

func listMoviesHandler(store *integrations.Store, client Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, conn, ok := resolveUserPlex(c, store, client)
		if !ok {
			return
		}

		start, _ := strconv.Atoi(c.DefaultQuery("start", "0"))
		if start < 0 {
			start = 0
		}
		size, _ := strconv.Atoi(c.DefaultQuery("size", strconv.Itoa(defaultPageSize)))
		if size <= 0 {
			size = defaultPageSize
		}
		if size > maxPageSize {
			size = maxPageSize
		}
		sort := strings.TrimSpace(c.DefaultQuery("sort", "addedAt:desc"))
		if !validSorts[sort] {
			sort = "addedAt:desc"
		}

		res, err := client.ListMovies(conn, token, start, size, sort)
		if err == ErrUnauthorized {
			client.InvalidateServer(c.GetString("userId"))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "plex_unauthorized"})
			return
		}
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "plex_server_unreachable"})
			return
		}
		if res.Items == nil {
			res.Items = []Movie{}
		}
		c.JSON(http.StatusOK, res)
	}
}

func movieDetailHandler(store *integrations.Store, client Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, conn, ok := resolveUserPlex(c, store, client)
		if !ok {
			return
		}

		ratingKey := c.Param("ratingKey")
		if ratingKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ratingKey required"})
			return
		}

		d, err := client.GetMovie(conn, token, ratingKey)
		if err == ErrUnauthorized {
			client.InvalidateServer(c.GetString("userId"))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "plex_unauthorized"})
			return
		}
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "plex_server_unreachable"})
			return
		}
		c.JSON(http.StatusOK, d)
	}
}
