package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RequireUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-User-Id")
		email := c.GetHeader("X-User-Email")
		if id == "" || email == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no user header"})
			return
		}
		c.Set("userId", id)
		c.Set("userEmail", email)
		c.Set("userRole", c.GetHeader("X-User-Role"))
		c.Next()
	}
}
