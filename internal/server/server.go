package server

import (
	"context"
	"github.com/denisschmidt/uploader/internal/auth"
	"github.com/gorilla/mux"
	"net/http"
)

func NewHTTPServer(addr string, authorizer auth.Authorizer) *http.Server {
	httpsrv := newHTTPServer(authorizer)
	httpsrv.routes()
	return &http.Server{
		Addr:    addr,
		Handler: httpsrv.router,
	}
}

type httpServer struct {
	auth   auth.Authorizer
	router *mux.Router
}

func newHTTPServer(authorizer auth.Authorizer) *httpServer {
	s := &httpServer{
		auth:   authorizer,
		router: mux.NewRouter(),
	}
	return s
}

func (s *httpServer) routes() {
	s.router.HandleFunc("/api/auth", s.authPost()).Methods(http.MethodPost)
	s.router.HandleFunc("/api/auth", s.authDelete()).Methods(http.MethodDelete)
	s.router.Use(s.checkAuth)
}

func (s *httpServer) authPost() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		s.auth.StartSession(writer, request)
	}
}

func (s *httpServer) authDelete() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		s.auth.ClearSession(writer)
	}
}

type authContextKey struct {
	name string
}

func (s *httpServer) checkAuth(h http.Handler) http.Handler {
	return http.HandlerFunc(func(write http.ResponseWriter, request *http.Request) {
		hasAuth := s.auth.Authenticate((request))
		ctx := context.WithValue(request.Context(), authContextKey{"is-authenticated"}, hasAuth)
		h.ServeHTTP(write, request.WithContext(ctx))
	})
}
