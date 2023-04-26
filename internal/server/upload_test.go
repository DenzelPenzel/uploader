package db_test

import (
	"github.com/denisschmidt/uploader/internal/sql/memory_db"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInsertRecord(t *testing.T) {
	store := memory_db.NewSQLiteStore()
	db, err := store.NewWithChunkSize(1024)
	require.NoError(t, err)

	defer db.Close()

}
