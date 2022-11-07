package server

import (
	"fmt"
	"github.com/denisschmidt/uploader/internal/auth"
	"github.com/stretchr/testify/require"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestServer(t *testing.T) {
	for scenario, fn := range map[string]func(t *testing.T, server *http.Server){
		"authorized success": testAuthorized,
	} {
		t.Run(scenario, func(t *testing.T) {
			server, teardown := setupTest(t)
			defer teardown()
			fn(t, server)
		})
	}
}

func setupTest(t *testing.T) (*http.Server, func()) {
	t.Helper()

	// creating a listener on the local network address
	// the 0 port is useful for when we don’t care what port we use since 0 will automatically assign us a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	port := os.Getenv("PORT")
	if port == "" {
		port = "4001"
	}

	authenticator, err := auth.New("hello")
	server := NewHTTPServer(fmt.Sprintf(":%s", port), authenticator)

	go func() {
		// start serving requests in a goroutine because Serve method is a blocking call
		// if we didn’t run it in a goroutine our tests further down would never run
		server.Serve(listener)
	}()

	return server, func() {
		listener.Close()
		server.Close()
	}
}

func testAuthorized(t *testing.T, server *http.Server) {
	body := `{"secretKey": "hello"}`

	req, err := http.NewRequest("POST", "/api/auth", strings.NewReader(body))
	require.NoError(t, err)

	req.Header.Add("Content-Type", "text/plain")

	w := httptest.NewRecorder()
	server.Handler.ServeHTTP(w, req)

	status := w.Code
	require.Equal(t, status, http.StatusOK)

	buf := w.Body
	require.Equal(t, buf.Len(), 0)
}
