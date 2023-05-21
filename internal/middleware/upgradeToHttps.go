package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// upgradeToHttps - if client is connecting over plaintext HTTP, upgrade to HTTPS
func UpgradeToHttps() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Also we can check TLS field of the request, but TLS field check won't work
		// If your app is behind a reverse proxy that terminates the TLS connection before your app
		//In that case, you need to check for the X-Forwarded-Proto header
		if c.GetHeader("X-Forwarded-Proto") == "http" {
			c.Redirect(http.StatusMovedPermanently, "https://"+c.Request.Host+c.Request.RequestURI)
			c.Abort()
			return
		}
		c.Next()
	}
}
