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
	setCookie(c, accessTokenCookieName, accessToken, sessionCookieMaxAge, true, false)
}

func setCookie(c *gin.Context, name, value string, maxAge int, httpOnly bool, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     sessionCookiePath,
		MaxAge:   maxAge,
		Expires:  time.Now().Add(time.Duration(maxAge) * time.Second),
		HttpOnly: httpOnly,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
