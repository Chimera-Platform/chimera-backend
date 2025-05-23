package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ProfileHandler returns the authenticated user's profile info.
func ProfileHandler(c *gin.Context) {
	userToken, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User info missing in context"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"profile": userToken})
}
