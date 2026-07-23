package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	accessTokenCookieName = "access_token"

	sessionCookiePath   = "/"
	sessionCookieMaxAge = 7 * 24 * 60 * 60
)

func setAuthCookies(c *gin.Context, accessToken string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     accessTokenCookieName,
		Value:    accessToken,
		Path:     sessionCookiePath,
		MaxAge:   sessionCookieMaxAge,
		Expires:  time.Now().Add(sessionCookieMaxAge * time.Second),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}
