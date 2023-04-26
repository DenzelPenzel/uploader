package server

import (
	"errors"
	"github.com/denisschmidt/uploader/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"log"
	"net/http"
	"time"
)

func (s *HttpServer) filePost() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := s.insertFileFromRequest(c.Request)
		if err != nil {
			var de *dbError
			if errors.As(err, &de) {
				log.Printf("failed to insert uploaded file into data store: %v", err)
				c.AbortWithStatus(http.StatusInternalServerError)
			} else {
				log.Printf("invalid upload: %v", err)
				c.AbortWithStatus(http.StatusBadRequest)
			}
			return
		}

		c.Header("Content-Type", "application/json")
		c.JSON(http.StatusOK, gin.H{
			"ID": string(id),
		})
	}
}

func (s *HttpServer) insertFileFromRequest(r *http.Request) (types.ID, error) {
	if err := r.ParseMultipartForm(MULTI_PART_MAX_MEMORY); err != nil {
		return types.ID(""), err
	}

	defer func() {
		if err := r.MultipartForm.RemoveAll(); err != nil {
			log.Printf("filed to free multipart form resources: %v", err)
		}
	}()

	reader, metadata, err := r.FormFile("file")
	if err != nil {
		return types.ID(""), err
	}

	if metadata.Size == 0 {
		return types.ID(""), errors.New("file is empty")
	}

	note, err := parseFileNote(r.FormValue("note"))
	if err != nil {
		return types.ID(""), err
	}

	id := types.ID(uuid.New().String())

	err = s.db.InsertRecord(reader, types.RecordMetadata{
		ID:          id,
		Filename:    types.Filename(metadata.Filename),
		ContentType: types.ContentType(metadata.Header.Get("Content-Type")),
		Note:        note,
		CreateAt:    time.Now(),
	})

	if err != nil {
		log.Printf("failed to insert new record in db: %v", err)
		return types.ID(""), dbError{err}
	}

	return id, nil
}
