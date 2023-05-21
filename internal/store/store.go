package store

import (
	"github.com/denisschmidt/uploader/internal/types"
	"io"
)

type Store interface {
	InsertRecord(reader io.Reader, metadata types.Metadata) error
	GetRecord(id types.ID) (types.UploadRecord, error)
	GetMetadata(id types.ID) (types.Metadata, error)

	UpdateRecordMetadata(id types.ID, metadata types.Metadata) error

	DeleteRecord(id types.ID) error
}
