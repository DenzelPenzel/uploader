package server

import (
	"expvar"
	"fmt"
	"github.com/denisschmidt/uploader/config"
	"github.com/denisschmidt/uploader/internal/middleware"
	"github.com/denisschmidt/uploader/internal/stats"
	"github.com/denisschmidt/uploader/internal/store"
	"github.com/denisschmidt/uploader/internal/types"
	"github.com/gin-gonic/contrib/cors"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"time"
)

type dbError struct {
	Err error
}

type HttpServer struct {
	engine *gin.Engine
	config *config.Config
}

type handlers struct {
	auth types.Authorizer
	db   store.Store
}

func (dbe dbError) Error() string {
	return fmt.Sprintf("database error: %s", dbe.Err)
}

func (dbe dbError) Unwrap() error {
	return dbe.Err
}

func NewHTTPServer(config *config.Config, database store.Store, authenticator types.Authorizer) (*HttpServer, error) {
	server := &HttpServer{
		config: config,
	}
	if err := server.Init(database, authenticator); err != nil {
		return nil, err
	}

	return server, nil
}

func (s *HttpServer) Init(database store.Store, authenticator types.Authorizer) error {
	router := gin.Default()
	handlder := &handlers{
		auth: authenticator,
		db:   database,
	}

	if s.config.Debug {
		router.Use(gin.Recovery())
	}

	if s.config.AllowedOrigins != nil && s.config.AllowedMethods != nil {
		allowAllOrigins := len(s.config.AllowedOrigins) == 1 && s.config.AllowedOrigins[0] == "*"
		allowedOrigins := s.config.AllowedOrigins
		if allowAllOrigins {
			allowedOrigins = nil
		}

		router.Use(cors.New(cors.Config{
			AllowAllOrigins: allowAllOrigins,
			AllowedOrigins:  allowedOrigins,
			AllowedMethods:  s.config.AllowedMethods,
			AllowedHeaders:  s.config.AllowedHeaders,
		}))
	}

	router.GET("/healthcheck", handlder.healthCheck(time.Now().UTC()))

	if s.config.Options.EnableStats {
		stat := stats.NewStatistic()

		router.Use(func(c *gin.Context) {
			startTime, recorder := stat.StartRecording(c.Writer)
			c.Next()
			stat.EndRecording(startTime, recorder)
		})

		router.GET("/sys/stats", func(c *gin.Context) {
			c.JSON(http.StatusOK, stat.GatherData())
		})
	}

	restrictIPAddresses := RestrictIPAddresses(s.config.Options.AllowedIPAddresses)

	if s.config.Options.EnableHealth {
		router.GET("/sys/health", restrictIPAddresses, gin.WrapH(expvar.Handler()))
		router.GET("/sys/info", restrictIPAddresses, handlder.sysStats())
	}

	router.POST("/api/auth", handlder.authPost())
	router.Use(handlder.checkAuth())

	protectedApi := router.Group("api")
	protectedApi.Use(handlder.requireAuth())
	{
		protectedApi.POST("/file", handlder.filePost())
		protectedApi.PUT("/file/:id", handlder.filePut())
		protectedApi.DELETE("/file/:id", handlder.fileDelete())
	}

	view := router.Group("/")
	view.Use(middleware.UpgradeToHttps())

	s.engine = router

	return nil
}

func (s *HttpServer) Run() error {
	err := s.engine.Run(fmt.Sprintf(":%s", strconv.Itoa(s.config.Port)))
	if err != nil {
		return err
	}
	return nil
}
