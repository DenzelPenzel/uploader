package db_test

import (
	"bytes"
	"github.com/denisschmidt/uploader/internal/store/db/fake_db"
	"github.com/denisschmidt/uploader/internal/types"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

func TestReadLastByteOfRecord(t *testing.T) {
	chunkSize := 5
	db := fake_db.NewSqlWithChunk(chunkSize)
	data := "test test test@"
	reader := bytes.NewBufferString(data)

	err := db.InsertRecord(reader, types.Metadata{
		ID:       types.ID("test"),
		Filename: "test.txt",
		Note:     "Hello world",
	})

	require.NoError(t, err)

	record, err := db.GetRecord(types.ID("test"))
	require.NoError(t, err)

	pos, err := record.Reader.Seek(1, io.SeekEnd)
	require.NoError(t, err)

	want := int64(len(data))
	require.Equal(t, pos, want-1)

	content, err := io.ReadAll(record.Reader)
	require.NoError(t, err)

	require.Equal(t, string(content), "@")
}
