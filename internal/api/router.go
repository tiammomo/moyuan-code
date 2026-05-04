package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"moyuan-code/internal/controlplane"
	"moyuan-code/internal/memory"
	"moyuan-code/internal/orchestrator"
	"moyuan-code/internal/quality"
	"moyuan-code/internal/repair"
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
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"projects": projects})
	})
	router.GET("/v1/projects/:project_id", func(c *gin.Context) {
		project, _, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"project": project})
	})
	router.GET("/v1/projects/:project_id/issues/:issue_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		state, found, err := orchestrator.LoadIssueState(rootDir, c.Param("issue_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "issue state not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"issue": state})
	})
	router.GET("/v1/projects/:project_id/runs/:run_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		state, found, err := orchestrator.LoadRunState(rootDir, c.Param("run_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "run state not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"run": state})
	})
	router.GET("/v1/projects/:project_id/quality/:report_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		report, found, err := quality.Read(rootDir, c.Param("report_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "quality report not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"quality_report": report})
	})
	router.GET("/v1/projects/:project_id/memory/search", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		records, err := memory.Search(rootDir, c.Query("q"), queryLimit(c, 10))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"records": records})
	})
	router.GET("/v1/projects/:project_id/memory/candidates", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		candidates, err := memory.ListCandidates(rootDir, queryLimit(c, 20))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"candidates": candidates})
	})
	router.GET("/v1/projects/:project_id/repair/attempts/:attempt_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		attempt, found, err := repair.LoadAttempt(rootDir, c.Param("attempt_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "repair attempt not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"repair_attempt": attempt})
	})
	return router
}

func projects(options Options) (any, error) {
	if options.Store != nil {
		return options.Store.ListProjects()
	}
	return controlplane.List(options.RootDir)
}

func findProject(options Options, projectID string) (any, string, bool, error) {
	projectID = strings.TrimSpace(projectID)
	if options.Store != nil {
		project, ok, err := options.Store.FindProject(projectID)
		if err != nil {
			return nil, "", false, err
		}
		if ok {
			return project, project.Root, true, nil
		}
	}
	projects, err := controlplane.List(options.RootDir)
	if err != nil {
		return nil, "", false, err
	}
	for _, project := range projects {
		if project.ID == projectID {
			return project, project.Root, true, nil
		}
	}
	return nil, "", false, nil
}

func queryLimit(c *gin.Context, fallback int) int {
	limit, err := strconv.Atoi(c.Query("limit"))
	if err != nil || limit <= 0 {
		return fallback
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func writeError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}
