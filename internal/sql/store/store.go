package store

import (
	"github.com/denisschmidt/uploader/internal/types"
	"io"
)

type Store interface {
	InsertRecord(reader io.Reader, metadata types.RecordMetadata) error
}
