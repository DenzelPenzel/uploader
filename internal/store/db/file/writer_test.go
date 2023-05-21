package file_test

import (
	"database/sql"
	"errors"
	"github.com/denisschmidt/uploader/internal/store/db/file"
	"github.com/denisschmidt/uploader/internal/types"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

type (
	mockChunkRow struct {
		id         types.ID
		chunkIndex int
		chunk      []byte
	}

	mockSqlDB struct {
		rows []mockChunkRow
		err  error
	}
)

var errMockSqlFailure = errors.New("wrong SQL")

func (db *mockSqlDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	chunk := args[2].([]byte)
	chunkCopy := make([]byte, len(chunk))
	copy(chunkCopy, chunk)
	db.rows = append(db.rows, mockChunkRow{
		id:         args[0].(types.ID),
		chunkIndex: args[1].(int),
		chunk:      chunkCopy,
	})
	return nil, db.err
}

func TestWriteFile(t *testing.T) {
	for _, row := range []struct {
		description  string
		id           types.ID
		data         []byte
		chunkSize    int
		sqlExecErr   error
		errExpected  error
		rowsExpected []mockChunkRow
	}{
		{
			description: "data is smaller than chunk size",
			id:          types.ID("test_id"),
			data:        []byte("test test test"),
			chunkSize:   30,
			rowsExpected: []mockChunkRow{
				{
					id:         types.ID("test_id"),
					chunkIndex: 0,
					chunk:      []byte("test test test"),
				},
			},
		},
		{
			description: "data is split into single chunk",
			id:          types.ID("test_id"),
			data:        []byte("test"),
			chunkSize:   4,
			rowsExpected: []mockChunkRow{
				{
					id:         types.ID("test_id"),
					chunkIndex: 0,
					chunk:      []byte("test"),
				},
			},
		},
		{
			description: "data is split into two chunks",
			id:          types.ID("test_id"),
			data:        []byte("123456"),
			chunkSize:   5,
			rowsExpected: []mockChunkRow{
				{
					id:         types.ID("test_id"),
					chunkIndex: 0,
					chunk:      []byte("12345"),
				},
				{
					id:         types.ID("test_id"),
					chunkIndex: 1,
					chunk:      []byte("6"),
				},
			},
		},

		{
			description: "data is split exactly into two chunks",
			id:          types.ID("test_id"),
			data:        []byte("1234567890"),
			chunkSize:   5,
			rowsExpected: []mockChunkRow{
				{
					id:         types.ID("test_id"),
					chunkIndex: 0,
					chunk:      []byte("12345"),
				},
				{
					id:         types.ID("test_id"),
					chunkIndex: 1,
					chunk:      []byte("67890"),
				},
			},
		},

		{
			description: "write fails when SQL transaction returns error",
			id:          types.ID("test_id"),
			data:        []byte("1234567890"),
			chunkSize:   5,
			sqlExecErr:  errMockSqlFailure,
			errExpected: errMockSqlFailure,
		},
	} {
		t.Run(row.description, func(t *testing.T) {
			tx := mockSqlDB{
				err: row.sqlExecErr,
			}

			w := file.NewWriter(&tx, row.id, row.chunkSize)
			n, err := w.Write(row.data)

			require.Equal(t, err, row.errExpected)

			if err != nil {
				return
			}

			require.Equal(t, n, len(row.data))

			err = w.Close()
			require.Nil(t, err)

			res := reflect.DeepEqual(row.rowsExpected, tx.rows)
			require.True(t, res)
		})
	}
}
