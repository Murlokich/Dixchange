package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func loginHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"action": "login"})
}

func registerHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"action": "register"})
}
