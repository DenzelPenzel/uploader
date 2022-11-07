package main

import (
	"fmt"
	"github.com/denisschmidt/uploader/internal/auth"
	"github.com/denisschmidt/uploader/internal/server"
	"log"
	"os"
)

func main() {
	log.Print("Starting uploader server")
	port := getEnvKey("PORT")
	log.Printf("Listening on %s", port)

	authenticator, err := auth.New(getEnvKey("PS_SHARED_SECRET"))
	if err != nil {
		log.Fatalf("invalid shared secret: %v", err)
	}

	s := server.NewHTTPServer(fmt.Sprintf(":%s", port), authenticator)
	log.Fatal(s.ListenAndServe())
}

func getEnvKey(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("missing required env key: %s", val))
	}
	return val
}
