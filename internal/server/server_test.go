package server_test

import (
	"github.com/denisschmidt/uploader/internal/auth"
	"github.com/denisschmidt/uploader/internal/db"
	"github.com/denisschmidt/uploader/internal/server"
	db2 "github.com/denisschmidt/uploader/internal/sql/db"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServer(t *testing.T) {
	for scenario, fn := range map[string]func(t *testing.T, server *server.HttpServer){
		"authorized success":     testSuccessAuthorized,
		"authorized failed":      testFailedAuthorized,
		"authorized bad request": testBadRequestAuthorized,
	} {
		t.Run(scenario, func(t *testing.T) {
			server, teardown := setupTest(t)
			defer teardown()
			fn(t, server)
		})
	}
}

func setupTest(t *testing.T) (*server.HttpServer, func()) {
	t.Helper()

	store := db2.New()
	authenticator, err := auth.New("hello")
	require.NoError(t, err)
	server := server.NewHTTPServer(authenticator, store)
	return server, func() {}
}

func testSuccessAuthorized(t *testing.T, server *server.HttpServer) {
	body := `{"secretKey": "hello"}`

	req, err := http.NewRequest("POST", "/api/auth", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Add("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)
	status := w.Code
	require.Equal(t, status, http.StatusOK)

	buf := w.Body
	require.Equal(t, buf.Len(), 0)
}

func testFailedAuthorized(t *testing.T, server *server.HttpServer) {
	body := `{"secretKey": "hello world"}`

	req, err := http.NewRequest("POST", "/api/auth", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Add("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)
	status := w.Code
	require.Equal(t, status, http.StatusUnauthorized)
}

func testBadRequestAuthorized(t *testing.T, server *server.HttpServer) {
	body := `{"secretKey_1": "hello world"}`

	req, err := http.NewRequest("POST", "/api/auth", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Add("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)
	status := w.Code
	require.Equal(t, status, http.StatusBadRequest)
}
