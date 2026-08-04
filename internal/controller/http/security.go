package http

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const (
	accessTokenCookieName = "access_token"

	sessionCookiePath   = "/"
	sessionCookieMaxAge = 7 * 24

	corsMaxAge = 12
)

func setAuthCookies(c *gin.Context, accessToken string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     accessTokenCookieName,
		Value:    accessToken,
		Path:     sessionCookiePath,
		MaxAge:   sessionCookieMaxAge,
		Expires:  time.Now().Add(sessionCookieMaxAge * time.Hour),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func SetCORSMiddleware(r *gin.Engine, frontendHost string) {
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{frontendHost},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
		// Expose response headers that the frontend is allowed to read
		ExposeHeaders: []string{"Content-Length"},
		// Allow cookies, authorization headers, or SSL client certificates
		AllowCredentials: true,
		// How long the browser should cache this preflight OPTIONS response (in hours)
		MaxAge: corsMaxAge * time.Hour,
	}))
}
