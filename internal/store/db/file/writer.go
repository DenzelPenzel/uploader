package file

import (
	"github.com/denisschmidt/uploader/internal/store/db/wrapper"
	"github.com/denisschmidt/uploader/internal/types"
	"io"
)

type writer struct {
	ctx     wrapper.SqlDB
	ID      types.ID
	buf     []byte
	written int
}

// NewWriter creates a new Writer for the given ID using the specified SqlDB instance.
// The data will be split into separate rows in the database, with each row containing at most chunkSize bytes
func NewWriter(ctx wrapper.SqlDB, id types.ID, chunkLen int) io.WriteCloser {
	return &writer{
		ctx: ctx,
		ID:  id,
		buf: make([]byte, chunkLen),
	}
}

// Writer is the interface that wraps the basic Write method
// Write writes len(data) bytes from data to the underlying data stream
// It returns the number of bytes written from data (0 <= n <= len(data))
// and any error encountered that caused the write to stop early
// Write must return a non-nil error if it returns n < len(data)
// Write must not modify the slice data, even temporarily
func (w *writer) Write(data []byte) (int, error) {
	bytesWritten := 0

	for {
		if bytesWritten == len(data) {
			break
		}
		bufferStart := w.written % len(w.buf)
		copySize := min(len(w.buf)-bufferStart, len(data)-bytesWritten)
		bufferEnd := bufferStart + copySize
		copy(w.buf[bufferStart:bufferEnd], data[bytesWritten:bytesWritten+copySize])

		if bufferEnd == len(w.buf) {
			if err := w.flush(len(w.buf)); err != nil {
				return bytesWritten, err
			}
		}

		w.written += copySize
		bytesWritten += copySize
	}

	return bytesWritten, nil
}

func (w *writer) Close() error {
	unflushed := w.written % len(w.buf)
	if unflushed != 0 {
		return w.flush(unflushed)
	}
	return nil
}

// flush - responsible for writing a chunk of data from the buf to the SQLite database
// by inserting a new row in the `metadata` table with the associated entry ID, chunk_index, and data chunk
// =====================================================================================================================
// id: the unique identifier for the entry this data belongs to.
// chunk_index: the index of the current data chunk.
// chunk: the actual data chunk, which is a slice of the buf containing the first n bytes.
func (w *writer) flush(n int) error {
	idx := w.written / len(w.buf)
	_, err := w.ctx.Exec(`
	INSERT INTO
		metadata
	(
		id,
		chunk_index,
		chunk
	)
	VALUES(?,?,?)
 	`, w.ID, idx, w.buf[0:n])
	return err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
