package db

import (
	"database/sql"
	"github.com/denisschmidt/uploader/internal/litewrapper"
	"github.com/denisschmidt/uploader/internal/sql/store"
	"github.com/denisschmidt/uploader/internal/types"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"log"
	"time"
)

const (
	timeFormat       = time.RFC3339
	defaultChunkSize = 327680
)

type sqldb struct {
	ctx       *sql.DB
	chunkSize int
}

type writer struct {
	ctx     litewrapper.SqlDB
	ID      types.ID
	buf     []byte
	written int
}

// Writer is the interface that wraps the basic Write method.
//
// Write writes len(p) bytes from p to the underlying data stream.
// It returns the number of bytes written from p (0 <= n <= len(p))
// and any error encountered that caused the write to stop early.
// Write must return a non-nil error if it returns n < len(p).
// Write must not modify the slice data, even temporarily.
func (w *writer) Write(p []byte) (int, error) {
	n := 0
	for {
		if n == len(p) {
			break
		}
		start := w.written % len(w.buf)
		copySize := min(len(w.buf)-start, len(p)-n)
		end := start + copySize
		copy(w.buf[start:end], p[n:n+copySize])
		if end == len(w.buf) {
			if err := w.flush(len(w.buf)); err != nil {
				return n, err
			}
		}
		w.written += copySize
		n += copySize
	}

	return n, nil
}

func (w writer) Close() error {
	unflushed := w.written % len(w.buf)
	if unflushed != 0 {
		return w.flush(unflushed)
	}
	return nil
}

func (w *writer) flush(n int) error {
	idx := w.written / len(w.buf)
	_, err := w.ctx.Exec(`
	INSERT INTO
		records_data
	(
		id,
		chunk_index,
		chunk
	)
	VALUES(?,?,?)
 	`, w.ID, idx, w.buf[0:n])
	return err
}

func NewWriter(ctx litewrapper.SqlDB, id types.ID, chunkLen int) io.WriteCloser {
	return &writer{
		ctx: ctx,
		ID:  id,
		buf: make([]byte, chunkLen),
	}
}

func New(path string, optimizeForLiteStream bool) store.Store {
	return NewWithChunkSize(path, defaultChunkSize, optimizeForLiteStream)
}

func NewWithChunkSize(path string, chunkSize int, optimizeForLiteStream bool) store.Store {
	ctx, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatalln(err)
	}

	if _, err := ctx.Exec(`
		PRAGMA temp_store = FILE;
		PRAGMA journal_mode = WAL;
	`); err != nil {
		log.Fatalf("failed to set up pragmas database: %v", err)
	}

	if optimizeForLiteStream {
		if _, err := ctx.Exec(`
			-- Apply Litestream recommendations: https://litestream.io/tips/
			PRAGMA busy_timeout = 5000;
			PRAGMA synchronous = NORMAL;
			PRAGMA wal_autocheckpoint = 0;
		`); err != nil {
			log.Fatalf("failed to set up Litestream pragmas %v", err)
		}
	}

	return &sqldb{
		ctx:       ctx,
		chunkSize: chunkSize,
	}
}

func (d sqldb) InsertRecord(reader io.Reader, metadata types.RecordMetadata) error {
	log.Printf("create a new record %s", metadata.ID)

	w := NewWriter(d.ctx, metadata.ID, d.chunkSize)

	if _, err := io.Copy(w, reader); err != nil {
		return err
	}

	if err := w.Close(); err != nil {
		return err
	}

	_, err := d.ctx.Exec(`
	INSERT INTO
		records
	(
		id,
		filename,
		note,
		content_type,
		create_at,
	)
	VALUES(?,?,?,?,?)`,
		metadata.ID,
		metadata.Filename,
		metadata.Note,
		metadata.ContentType,
		metadata.CreateAt.UTC().Format(timeFormat),
	)

	if err != nil {
		log.Printf("insert record into Records table failed: %v", err)
		return err
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
