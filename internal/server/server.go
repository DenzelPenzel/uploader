package server

import (
	"github.com/denisschmidt/uploader/config"
	"github.com/denisschmidt/uploader/internal/auth"
	"github.com/denisschmidt/uploader/internal/store"
	"github.com/denisschmidt/uploader/internal/store/db"
	"github.com/denisschmidt/uploader/internal/types"
	"net/http"
	"os"
	"path/filepath"
)

type Server struct {
	http *HttpServer
}

func New(config *config.Config, database store.Store, authenticator types.Authorizer) (*Server, error) {
	httpServer, err := NewHTTPServer(config, database, authenticator)
	if err != nil {
		return nil, err
	}

	server := &Server{
		http: httpServer,
	}
	return server, err
}

func (s *Server) Run() error {
	return s.http.Run()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.http.engine.ServeHTTP(w, req)
}

func initDatabase(cfg *config.Config) (store.Store, error) {
	if _, err := os.Stat(filepath.Dir(cfg.DBPath)); os.IsNotExist(err) {
		if err := os.Mkdir(filepath.Dir(cfg.DBPath), os.ModePerm); err != nil {
			return nil, err
		}
	}
	database := db.New(cfg.DBPath, cfg.DBChunkSize, true)

	return database, nil
}

func Run(path string) error {
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}

	database, err := initDatabase(cfg)
	if err != nil {
		return err
	}

	authenticator, err := auth.New(cfg.SecretKey)
	if err != nil {
		return err
	}

	server, err := New(cfg, database, &authenticator)
	if err != nil {
		return err
	}

	return server.Run()
}
