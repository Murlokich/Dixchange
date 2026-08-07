package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	authHeader   = "Authorization"
	bearerPrefix = "Bearer "

	missingTokenResponseMessage        = "Access token is missing or format is incorrect"
	missingTokenSubjectResponseMessage = "Access token misses subject with user id"

	userIDCtxKey           = "userID"
	loggerWithUserIDCtxKey = "logger"
	userIDLoggerField      = "user_id"
)

var (
	ErrCtxMissingUserID = errors.New("gin ctx doesn't store user ID key")
	ErrCtxUserIDNotUID  = errors.New("gin ctx stores non uid string for user ID")
)

func GetUserIDFromCtx(c *gin.Context) (uuid.UUID, error) {
	userIDRaw, ok := c.Get(userIDCtxKey)
	if !ok {
		return uuid.Nil, ErrCtxMissingUserID
	}
	userID, ok := userIDRaw.(uuid.UUID)
	if !ok {
		return uuid.Nil, ErrCtxUserIDNotUID
	}
	return userID, nil
}

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
	userIDStr, err := token.Claims.GetSubject()
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{errorField: missingTokenSubjectResponseMessage})
		deps.Logger.Errorf("authMiddleware get subject token: %v", err)
		c.Abort()
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{errorField: missingTokenSubjectResponseMessage})
		deps.Logger.Errorf("authMiddleware parse subject token: %v", err)
		c.Abort()
		return
	}
	c.Set(userIDCtxKey, userID)
	c.Set(loggerWithUserIDCtxKey, deps.Logger.WithField(userIDLoggerField, userID))
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
