package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"moyuan-code/internal/controlplane"
	"moyuan-code/internal/store"
)

const Version = "phase1-gin-gorm"

type Options struct {
	RootDir string
	Store   *store.Store
}

func NewRouter(options Options) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true, "version": Version})
	})
	router.GET("/v1/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": Version})
	})
	router.GET("/v1/projects", func(c *gin.Context) {
		projects, err := projects(options)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"projects": projects})
	})
	return router
}

func projects(options Options) (any, error) {
	if options.Store != nil {
		return options.Store.ListProjects()
	}
	return controlplane.List(options.RootDir)
}
