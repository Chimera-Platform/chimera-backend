package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// PingHandler responds with a simple "pong" message.
func PingHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
}
