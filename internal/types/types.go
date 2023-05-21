package types

import (
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"time"
)

type (
	ID          string
	Filename    string
	ContentType string
	Note        string

	Metadata struct {
		ID          ID
		Filename    Filename
		Note        Note
		ContentType ContentType
		CreateAt    time.Time
		Size        int64
	}

	MetadataRequest struct {
		Filename string `json:"filename"`
		Note     string `json:"note"`
	}

	RecordPostResponse struct {
		ID string `json:"id"`
	}

	UploadRecord struct {
		Metadata
		Reader io.ReadSeeker
	}

	Authorizer interface {
		Authenticate(r *http.Request) bool
		StartSession(c *gin.Context)
		ClearSession(w http.ResponseWriter)
	}
)
