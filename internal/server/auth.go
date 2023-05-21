package server

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func (h handlers) checkAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		hasAuth := h.auth.Authenticate(c.Request)
		c.Set("has-auth", hasAuth)
		c.Next()
	}
}

func (h handlers) authPost() gin.HandlerFunc {
	return func(c *gin.Context) {
		h.auth.StartSession(c)
	}
}

func (h handlers) authDelete() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		h.auth.ClearSession(writer)
	}
}

func (h handlers) requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, ok := c.Get("has-auth")
		if !ok {
			h.auth.ClearSession(c.Writer)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Auth required",
			})
			return
		}
		c.Next()
	}
}

func RestrictIPAddresses(ipAddresses []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(ipAddresses) == 0 {
			c.Next()
			return
		}

		clientIP := c.ClientIP()
		for _, address := range ipAddresses {
			if strings.Contains(address, clientIP) {
				c.Next()
				return
			}
		}

		c.String(http.StatusUnauthorized, "Unauthorized access")
		c.Abort()
	}
}
