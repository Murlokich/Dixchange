package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	authHeader   = "Authorization"
	bearerPrefix = "Bearer "

	missingTokenResponseMessage        = "Access token is missing or format is incorrect"
	missingTokenSubjectResponseMessage = "Access token misses subject with user id"

	userIDCtxKey = "userID"
)

func authMiddleware(c *gin.Context, deps *Dependencies) {
	tokenStr, ok := extractAccessToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{errorField: missingTokenResponseMessage})
		// TODO: add user id into logger as a field
		deps.Logger.Errorf("missing access token")
		c.Abort()
		return
	}
	token, err := deps.TokenValidator.ValidateToken(tokenStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{errorField: missingTokenResponseMessage})
		deps.Logger.Errorf("authMiddleware validate token: %v", err)
		c.Abort()
		return
	}
	userID, err := token.Claims.GetSubject()
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{errorField: missingTokenSubjectResponseMessage})
		deps.Logger.Errorf("authMiddleware get subject token: %v", err)
		c.Abort()
		return
	}
	c.Set(userIDCtxKey, userID)
	c.Next()
}

func extractAccessToken(c *gin.Context) (string, bool) {
	requestAuthHeader := c.GetHeader(authHeader)
	if strings.HasPrefix(requestAuthHeader, bearerPrefix) {
		return strings.TrimPrefix(requestAuthHeader, bearerPrefix), true
	}

	tokenCookie, err := c.Cookie(accessTokenCookieName)
	if err != nil || tokenCookie == "" {
		return "", false
	}
	return tokenCookie, true
}
