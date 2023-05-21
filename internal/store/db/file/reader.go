package file

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/denisschmidt/uploader/internal/types"
	"io"
	"log"
)

type (
	reader struct {
		db         *sql.DB
		ID         types.ID
		fileLength int64
		offset     int64
		chunkSize  int64
		buf        *bytes.Buffer
	}
)

func NewReader(db *sql.DB, id types.ID) (io.ReadSeeker, error) {
	chunkSize, err := getChunkSize(db, id)
	if err != nil {
		return nil, err
	}

	fileLength, err := getFileLength(db, id, chunkSize)
	if err != nil {
		return nil, err
	}

	return &reader{
		db:         db,
		ID:         id,
		fileLength: fileLength,
		offset:     0,
		chunkSize:  chunkSize,
		buf:        bytes.NewBuffer([]byte{}),
	}, nil
}

func (r *reader) Read(p []byte) (n int, err error) {
	read := 0
	for {
		n, err := r.buf.Read(p[read:])
		read += n
		// If buf is empty, check if we've read the entire file
		if err == io.EOF {
			if r.offset == r.fileLength {
				// Return EOF if we've reached the end of the file
				return read, io.EOF
			}
			// Repopulate the buf with data from the SQLite DB and continue reading
			err := r.populateBuffer()
			if err != nil {
				return read, err
			}
			continue
		}
		if read >= len(p) {
			break
		}
	}

	return read, nil
}

func (r *reader) Seek(offset int64, whence int) (int64, error) {
	// Reset the buf since seeking to a new position invalidates its content
	r.buf = bytes.NewBuffer([]byte{})

	// Update the file offset based on the 'whence' parameter
	switch whence {
	case io.SeekStart:
		r.offset = offset
	case io.SeekCurrent:
		r.offset += offset
	case io.SeekEnd:
		r.offset = int64(r.fileLength) - offset
	default:
		return r.offset, fmt.Errorf("invalid whence value: %d", whence)
	}

	return r.offset, nil
}

func (r *reader) populateBuffer() error {
	// Return EOF if we've reached the end of the file
	if r.offset == int64(r.fileLength) {
		return io.EOF
	}
	// Calculate the current chunk index based on the file offset and chunk size
	chunkIndex := r.offset / int64(r.chunkSize)

	// Query the database to retrieve the chunk data for the given ID and chunkIndex
	var chunk []byte

	err := r.db.QueryRow(`
		SELECT chunk
		FROM metadata
		WHERE id=? AND chunk_index=?
		ORDER BY
		    chunk_index ASC 
	`, r.ID, chunkIndex).Scan(&chunk)
	if err != nil {
		log.Printf("reading chunk failed: %v", err)
		return err
	}

	// Determine the start index within the chunk to read from based on the file offset
	readStart := r.offset % int64(r.chunkSize)

	// Update the buf with the chunk data starting from the readStart index
	r.buf = bytes.NewBuffer(chunk[readStart:])

	// Update the file offset by adding the remaining length of the chunk
	r.offset += int64(len(chunk)) - readStart

	return nil
}

func getChunkSize(db *sql.DB, id types.ID) (int64, error) {
	var chunkSize int64
	if err := db.QueryRow(`
		SELECT
		LENGTH(chunk) AS chunk_size
		FROM
			metadata
		WHERE
			id=?
		ORDER BY
			chunk_index ASC
		LIMIT 1
	`, id).Scan(&chunkSize); err != nil {
		return 0, err
	}

	return chunkSize, nil
}

func getFileLength(db *sql.DB, id types.ID, chunkSize int64) (int64, error) {
	var chunkIndex int64
	var chunkLen int64
	// retrieve the last chunk index and the length of that chunk
	if err := db.QueryRow(`
		SELECT
			chunk_index,
			LENGTH(chunk) AS chunk_size
		FROM
			metadata
		WHERE
			id=?
		ORDER BY
			chunk_index DESC
		LIMIT 1
	`, id).Scan(&chunkIndex, &chunkLen); err != nil {
		return 0, err
	}

	return (chunkSize * chunkIndex) + chunkLen, nil
}
