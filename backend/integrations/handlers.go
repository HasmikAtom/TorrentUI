package integrations

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type plexRequest struct {
	Token   *string `json:"token"`
	Enabled *bool   `json:"enabled"`
}

func RegisterHandlers(g *gin.RouterGroup, store *Store, plexAPIURL string) {
	g.GET("/integrations", func(c *gin.Context) {
		userID := c.GetString("userId")
		row, err := store.GetIntegrations(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load integrations"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"userId":       row.UserID,
			"plexEnabled":  row.PlexEnabled,
			"plexHasToken": row.PlexToken != "",
		})
	})

	g.PUT("/integrations/plex", func(c *gin.Context) {
		userID := c.GetString("userId")
		var req plexRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		// Toggle-only: no token provided, just flip enabled
		if req.Token == nil && req.Enabled != nil {
			if err := store.SetPlexEnabled(userID, *req.Enabled); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update"})
				return
			}
			row, _ := store.GetIntegrations(userID)
			c.JSON(http.StatusOK, gin.H{
				"userId":       row.UserID,
				"plexEnabled":  row.PlexEnabled,
				"plexHasToken": row.PlexToken != "",
			})
			return
		}

		// Token save: validate first
		if req.Token == nil || *req.Token == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
			return
		}

		if err := ValidatePlexToken(*req.Token, plexAPIURL); err != nil {
			status := http.StatusBadRequest
			if err.Error() == "could not verify token — try again later" {
				status = http.StatusBadGateway
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}

		enabled := true
		if req.Enabled != nil {
			enabled = *req.Enabled
		}

		if err := store.UpsertPlex(userID, *req.Token, enabled); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save"})
			return
		}

		row, _ := store.GetIntegrations(userID)
		c.JSON(http.StatusOK, gin.H{
			"userId":       row.UserID,
			"plexEnabled":  row.PlexEnabled,
			"plexHasToken": row.PlexToken != "",
		})
	})

	g.DELETE("/integrations/plex", func(c *gin.Context) {
		userID := c.GetString("userId")
		if err := store.DeletePlex(userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete"})
			return
		}
		c.Status(http.StatusNoContent)
	})
}
