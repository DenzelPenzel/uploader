package main

import (
	"flag"
	"fmt"
	"github.com/denisschmidt/uploader/internal/auth"
	"github.com/denisschmidt/uploader/internal/db"
	"github.com/denisschmidt/uploader/internal/server"
	db2 "github.com/denisschmidt/uploader/internal/sql/db"
	"log"
	"os"
	"path/filepath"
)

func main() {
	log.Print("Starting uploader server")
	port := getEnvKey("PORT")
	log.Printf("Listening on %s", port)
	dbPath := flag.String("db", "data/database.db", "path to sql database")

	flag.Parse()

	isLiteStream := os.Getenv("LITESTREAM_BUCKET") != ""
	authenticator, err := auth.New(getEnvKey("secretKey"))

	if err != nil {
		log.Fatalf("invalid shared secret: %v", err)
	}

	// check if path to database exists, if not create a dir
	if _, err := os.Stat(filepath.Dir(*dbPath)); os.IsNotExist(err) {
		if err := os.Mkdir(filepath.Dir(*dbPath), os.ModePerm); err != nil {
			panic(err)
		}
	}

	db := db2.New(*dbPath, isLiteStream)

	s := server.NewHTTPServer(authenticator, db)
	s.Router.Run(fmt.Sprintf(":%s", port))
}

func getEnvKey(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("missing required env key: %s", key))
	}
	return val
}
