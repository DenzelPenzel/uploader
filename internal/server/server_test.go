package server_test

import (
	"github.com/denisschmidt/uploader/config"
	"github.com/denisschmidt/uploader/internal/auth"
	"github.com/denisschmidt/uploader/internal/server"
	"github.com/denisschmidt/uploader/internal/store/db/fake_db"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServer(t *testing.T) {
	for scenario, fn := range map[string]func(t *testing.T, server *server.Server){
		"authorized success":     testSuccessAuthorized,
		"authorized failed":      testFailedAuthorized,
		"authorized bad request": testBadRequestAuthorized,
	} {
		t.Run(scenario, func(t *testing.T) {
			s, teardown := setupTest(t)
			defer teardown()
			fn(t, s)
		})
	}
}

func setupTest(t *testing.T) (*server.Server, func()) {
	t.Helper()
	chunkSize := 5
	defaultConfig := config.DefaultConfig()
	defaultConfig.SecretKey = "hello"
	database := fake_db.NewSqlWithChunk(chunkSize)
	authenticator, err := auth.New(defaultConfig.SecretKey)
	s, err := server.New(defaultConfig, database, &authenticator)
	require.NoError(t, err)

	return s, func() {}
}

func testSuccessAuthorized(t *testing.T, s *server.Server) {
	body := `{"secretKey": "hello"}`

	req, err := http.NewRequest("POST", "/api/auth", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Add("Content-Type", "application/json")

	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	status := w.Code
	require.Equal(t, status, http.StatusOK)

	buf := w.Body
	require.Equal(t, buf.Len(), 0)
}

func testFailedAuthorized(t *testing.T, s *server.Server) {
	body := `{"secretKey": "hello world"}`

	req, err := http.NewRequest("POST", "/api/auth", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Add("Content-Type", "application/json")

	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	status := w.Code
	require.Equal(t, status, http.StatusUnauthorized)
}

func testBadRequestAuthorized(t *testing.T, s *server.Server) {
	body := `{"secretKey_1": "hello world"}`

	req, err := http.NewRequest("POST", "/api/auth", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Add("Content-Type", "application/json")

	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	status := w.Code
	require.Equal(t, status, http.StatusBadRequest)
}
