package server

import (
	"fmt"
	"github.com/denisschmidt/uploader/internal/auth"
	"github.com/denisschmidt/uploader/internal/sql/store"
	"github.com/gin-gonic/gin"
	"net/http"
)

type (
	dbError struct {
		Err error
	}
)

func (dbe dbError) Error() string {
	return fmt.Sprintf("database error: %s", dbe.Err)
}

func (dbe dbError) Unwrap() error {
	return dbe.Err
}

func NewHTTPServer(authorizer auth.Authorizer, db store.Store) *HttpServer {
	router := gin.Default()
	s := &HttpServer{
		auth:   authorizer,
		Router: router,
		db:     db,
	}
	s.routes()
	return s
}

type HttpServer struct {
	auth   auth.Authorizer
	Router *gin.Engine
	db     store.Store
}

func (s *HttpServer) routes() {
	s.Router.POST("/api/auth", s.authPost())

	s.Router.Use(s.checkAuth())

	protectedApi := s.Router.Group("api")

	protectedApi.Use(s.requireAuth())
	{
		protectedApi.POST("/file", s.filePost())
	}
}

func (s *HttpServer) requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, ok := c.Get("has-auth")
		if !ok {
			s.auth.ClearSession(c.Writer)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Auth required",
			})
			return
		}
		c.Next()
	}
}

func (s *HttpServer) checkAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		hasAuth := s.auth.Authenticate(c.Request)
		c.Set("has-auth", hasAuth)
		c.Next()
	}
}

func (s *HttpServer) authPost() gin.HandlerFunc {
	return func(c *gin.Context) {
		s.auth.StartSession(c)
	}
}

func (s *HttpServer) authDelete() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		s.auth.ClearSession(writer)
	}
}
