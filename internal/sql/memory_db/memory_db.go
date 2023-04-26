package memory_db

import (
	"database/sql"
	"fmt"
	"github.com/denisschmidt/uploader/internal/sql/sqlite"
	"github.com/denisschmidt/uploader/internal/sql/store"
	"math/rand"
	"time"
)

type Store interface {
	New() (*sql.DB, error)
	NewWithChunkSize(chunkSize int) (*sql.DB, error)
}

func NewSQLiteStore() store.Store {
	return sqlite.New()
}

func (s *SQLiteStore) New() (*sql.DB, error) {
	return s.NewWithChunkSize(0)
}

func (s *SQLiteStore) NewWithChunkSize(chunkSize int) (*sql.DB, error) {
	uri := ephemeralDbURI()
	if chunkSize > 0 {
		uri = fmt.Sprintf("%s&cache_size=%d", uri, chunkSize)
	}
	return sql.Open("sqlite3", uri)
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
