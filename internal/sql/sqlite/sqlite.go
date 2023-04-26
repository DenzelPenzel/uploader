package sqlite

import (
	"database/sql"
	"github.com/denisschmidt/uploader/internal/sql/store"
	"github.com/denisschmidt/uploader/internal/types"
	"io"
	"log"
)

type (
	database struct {
		ctx       *sql.DB
		chunkSize int
	}
)

func (d database) InsertRecord(reader io.Reader, metadata types.RecordMetadata) error {
	//TODO implement me
	panic("implement me")
}

func New(path string, chunkSize int, optimizeForLitestream bool) store.Store {
	log.Printf("reading DB from %s", path)
	ctx, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatalln(err)
	}

	if _, err := ctx.Exec(`PRAGMA temp_store = FILE; PRAGMA journal_mode = WAL;`); err != nil {
		log.Fatalf("failed to set pragmas: %v", err)
	}

	if optimizeForLitestream {
		if _, err := ctx.Exec(`
			-- Apply Litestream recommendations: https://litestream.io/tips/
			PRAGMA busy_timeout = 5000;
			PRAGMA synchronous = NORMAL;
			PRAGMA wal_autocheckpoint = 0;
		`); err != nil {
			log.Fatalf("failed to set Litestream compatibility pragmas: %v", err)
		}
	}

	return &database{
		ctx:       ctx,
		chunkSize: chunkSize,
	}
}
