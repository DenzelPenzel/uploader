package server

import (
	"errors"
	"fmt"
	"github.com/denisschmidt/uploader/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"log"
	"net/http"
	"time"
)

func (h handlers) filePost() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := h.insertFileFromRequest(c.Request)
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

func (h handlers) filePut() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseRecordId(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("bad record ID: %v", err),
			})
			return
		}

		var md types.MetadataRequest
		if err := c.ShouldBindJSON(&md); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Bad request: %v", err),
			})
			return
		}

		metadata, err := parseMetadataFromRequest(md)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Bad request: %v", err),
			})
			return
		}

		err = h.db.UpdateRecordMetadata(id, metadata)
		if err != nil {
			if _, ok := err.(types.ErrFileNotExists); ok {
				c.JSON(http.StatusNotFound, gin.H{
					"error": fmt.Sprintf("Record not found ID: %v", id),
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to update record: %v", err),
			})
		}
	}
}

func (h handlers) fileDelete() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseRecordId(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("bad record ID: %v", err),
			})
			return
		}

		err = h.db.DeleteRecord(types.ID(id))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to delete record %v: %v", id, err),
			})
		}
	}
}

func (h handlers) insertFileFromRequest(r *http.Request) (types.ID, error) {
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

	err = validateFilename(metadata.Filename)
	if err != nil {
		return types.ID(""), err
	}

	note := r.FormValue("note")
	err = validateFileNote(note)
	if err != nil {
		return types.ID(""), err
	}

	id := types.ID(uuid.New().String())

	err = h.db.InsertRecord(reader, types.Metadata{
		ID:          id,
		Filename:    types.Filename(metadata.Filename),
		ContentType: types.ContentType(metadata.Header.Get("Content-Type")),
		Note:        types.Note(note),
		CreateAt:    time.Now(),
	})
	if err != nil {
		log.Printf("failed to insert new record in db: %v", err)
		return types.ID(""), dbError{err}
	}

	return id, nil
}
