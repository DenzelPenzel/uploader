package fake_db

import (
	"fmt"
	"github.com/denisschmidt/uploader/internal/store"
	"github.com/denisschmidt/uploader/internal/store/db"
	"math/rand"
	"time"
)

const optimizeForLitestream = false

func NewSqlWithChunk(chunkSize int) store.Store {
	uri := ephemeralDbURI()
	return db.NewWithChunkSize(uri, chunkSize, optimizeForLitestream)
}

func New(chunkSize int) store.Store {
	uri := ephemeralDbURI()
	return db.New(uri, chunkSize, optimizeForLitestream)
}

func ephemeralDbURI() string {
	rand.Seed(time.Now().UnixNano())
	letters := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
	name := make([]rune, 10)
	for i := range name {
		name[i] = letters[rand.Intn(len(letters))]
	}
	return fmt.Sprintf("file:%s?mode=memory&cache=shared", string(name))
}
