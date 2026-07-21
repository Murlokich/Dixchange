package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	authHeader   = "Authorization"
	bearerPrefix = "Bearer "

	missingTokenResponseMessage = "Access token is missing or format is incorrect"
)

func authMiddleware(c *gin.Context, deps *Dependencies) {
	tokenStr, ok := extractBearerToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{errorField: missingTokenResponseMessage})
		// TODO: add user id into logger as a field
		deps.Logger.Errorf("missing access token")
		c.Abort()
		return
	}
	_, err := deps.TokenValidator.ValidateToken(tokenStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{errorField: missingTokenResponseMessage})
		deps.Logger.Errorf("authMiddleware validate token: %v", err)
		c.Abort()
		return
	}
}

func extractBearerToken(c *gin.Context) (string, bool) {
	authHeader := c.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		return "", false
	}
	return strings.TrimPrefix(authHeader, bearerPrefix), true
}
