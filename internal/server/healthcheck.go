package server

import (
	"github.com/denisschmidt/uploader/constants"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

func (h handlers) healthCheck(startedAt time.Time) gin.HandlerFunc {
	return func(c *gin.Context) {
		now := time.Now().UTC()
		diff := now.Sub(startedAt)

		c.JSON(http.StatusOK, gin.H{
			"started_at":            startedAt.String(),
			"uptime":                diff.String(),
			"status":                "Ok",
			"version":               constants.Version,
			"revision":              constants.Revision,
			"build_time":            constants.BuildTime,
			"compiler":              constants.Compiler,
			"latest_commit_message": constants.LatestCommitMessage,
			"ip_address":            c.ClientIP(),
		})
	}
}
