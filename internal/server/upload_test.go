package server_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"github.com/denisschmidt/uploader/config"
	"github.com/denisschmidt/uploader/internal/auth/fake_auth"
	"github.com/denisschmidt/uploader/internal/server"
	"github.com/denisschmidt/uploader/internal/store/db/fake_db"
	"github.com/denisschmidt/uploader/internal/types"
	"github.com/stretchr/testify/require"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestInsertRecord(t *testing.T) {
	for _, row := range []struct {
		description string
		filename    string
		contents    string
		note        string
		status      int
	}{
		{
			description: "valid file with no note",
			filename:    "test1.png",
			contents:    "file content",
			status:      http.StatusOK,
		},
		{
			description: "valid file with a note",
			filename:    "test2.png",
			contents:    "file content",
			note:        "test note",
			status:      http.StatusOK,
		},
		{
			description: "valid file with a too-long note",
			filename:    "test3.png",
			contents:    "file content",
			note:        strings.Repeat("X", 501),
			status:      http.StatusBadRequest,
		},
		{
			description: "filename error",
			filename:    ".",
			contents:    "file content",
			status:      http.StatusBadRequest,
		},
		{
			description: "empty file content",
			filename:    "test.jpg",
			contents:    "",
			status:      http.StatusBadRequest,
		},
	} {
		t.Run(row.description, func(t *testing.T) {
			chunkSize := 5
			defaultConfig := config.DefaultConfig()
			defaultConfig.SecretKey = "hello"
			database := fake_db.New(chunkSize)
			authenticator := fake_auth.FakeAuth{}
			s, err := server.New(defaultConfig, database, authenticator)
			require.NoError(t, err)

			formData, contentType := createMultipartFormBody(row.filename, row.note, bytes.NewBuffer([]byte(row.contents)))

			req, err := http.NewRequest("POST", "/api/file", formData)
			require.NoError(t, err)
			req.Header.Add("Content-Type", contentType)

			rec := httptest.NewRecorder()
			s.ServeHTTP(rec, req)

			// check if statuses are the same
			require.Equal(t, row.status, rec.Code)

			if rec.Code != http.StatusOK {
				return
			}

			var response types.RecordPostResponse
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			require.NoError(t, err)

			record, err := database.GetRecord(types.ID(response.ID))
			require.NoError(t, err)

			got, err := io.ReadAll(record.Reader)
			require.NoError(t, err)
			require.True(t, reflect.DeepEqual(got, []byte(row.contents)))

			require.Equal(t, types.Filename(row.filename), record.Filename)
		})
	}
}

func TestUpdateRecord(t *testing.T) {
	mockRecord := types.Metadata{
		ID:       types.ID(strings.Repeat("X", 10)),
		Filename: types.Filename("test_init_name.png"),
		Note:     types.Note("test init note"),
	}

	for _, row := range []struct {
		description string
		id          string
		body        string
		newFilename string
		newNote     string
		status      int
	}{
		{
			description: "success update metadata",
			id:          strings.Repeat("X", 10),
			body: `{
				"filename": "test_1.png",
				"note":"this is test_1"
			}`,
			newFilename: "test_1.png",
			newNote:     "this is test_1",
			status:      http.StatusOK,
		},
		{
			description: "failed update due invalid filename",
			id:          strings.Repeat("X", 10),
			body: `{
				"filename": "",
				"note":"this is test_2"
			}`,
			newFilename: "test_init_name.png",
			newNote:     "test init note",
			status:      http.StatusBadRequest,
		},

		{
			description: "failed update due malicious code",
			id:          strings.Repeat("X", 10),
			body: `{
				"filename": "test_3.png",
				"note":"<script>alert(1)</script>"
			}`,
			newFilename: "test_init_name.png",
			newNote:     "test init note",
			status:      http.StatusBadRequest,
		},
		{
			description: "failed due not-exists record ID",
			id:          strings.Repeat("B", 10),
			body: `{
				"filename": "test_4.png",
				"note":"this is test_4"
			}`,
			newFilename: "test_init_name.png",
			newNote:     "test init note",
			status:      http.StatusNotFound,
		},
	} {
		defaultConfig := config.DefaultConfig()
		defaultConfig.SecretKey = "hello"
		database := fake_db.New(defaultConfig.DBChunkSize)

		err := database.InsertRecord(strings.NewReader("file data"), mockRecord)
		require.NoError(t, err)

		authenticator := fake_auth.FakeAuth{}
		s, err := server.New(defaultConfig, database, authenticator)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		req, err := http.NewRequest("PUT", "/api/file/"+row.id, strings.NewReader(row.body))
		require.NoError(t, err)
		req.Header.Add("Content-Type", "text/json")
		s.ServeHTTP(rec, req)
		// check if statuses are the same
		require.Equal(t, row.status, rec.Code)

		record, err := database.GetRecord(types.ID(mockRecord.ID))
		require.NoError(t, err)

		require.Equal(t, types.Filename(row.newFilename), record.Filename)
		require.Equal(t, types.Note(row.newNote), record.Note)
	}
}

func TestDeleteRecord(t *testing.T) {
	mockRecord := types.Metadata{
		ID:       types.ID(strings.Repeat("X", 10)),
		Filename: types.Filename("test_init_name.png"),
		Note:     types.Note("test init note"),
	}

	defaultConfig := config.DefaultConfig()
	defaultConfig.SecretKey = "hello"
	database := fake_db.New(defaultConfig.DBChunkSize)

	err := database.InsertRecord(strings.NewReader("file data"), mockRecord)
	require.NoError(t, err)

	authenticator := fake_auth.FakeAuth{}
	s, err := server.New(defaultConfig, database, authenticator)
	require.NoError(t, err)

	record, err := database.GetRecord(types.ID(mockRecord.ID))
	require.NoError(t, err)
	require.Equal(t, types.Filename("test_init_name.png"), record.Filename)

	got, err := io.ReadAll(record.Reader)
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(got, []byte("file data")))

	rec := httptest.NewRecorder()
	req, err := http.NewRequest("DELETE", "/api/file/"+string(record.ID), nil)
	require.NoError(t, err)
	s.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req, err = http.NewRequest("PUT", "/api/file/"+string(record.ID), strings.NewReader(`{
		"filename": "test.png",
		"note":"this is test"
	}`))
	require.NoError(t, err)
	s.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func createMultipartFormBody(filename, note string, r io.Reader) (io.Reader, string) {
	var b bytes.Buffer
	bw := bufio.NewWriter(&b)
	mw := multipart.NewWriter(bw)

	f, err := mw.CreateFormFile("file", filename)
	if err != nil {
		panic(err)
	}
	io.Copy(f, r)

	nf, err := mw.CreateFormField("note")
	if err != nil {
		panic(err)
	}
	nf.Write([]byte(note))

	mw.Close()
	bw.Flush()

	return bufio.NewReader(&b), mw.FormDataContentType()
}
