package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"moyuan-code/internal/controlplane"
	"moyuan-code/internal/deployment"
	"moyuan-code/internal/gitprovider"
	"moyuan-code/internal/issues"
	"moyuan-code/internal/memory"
	"moyuan-code/internal/orchestrator"
	"moyuan-code/internal/providers"
	"moyuan-code/internal/quality"
	"moyuan-code/internal/release"
	"moyuan-code/internal/repair"
	"moyuan-code/internal/requirement"
	"moyuan-code/internal/review"
	"moyuan-code/internal/scheduler"
	"moyuan-code/internal/serverresources"
	"moyuan-code/internal/store"
	"moyuan-code/internal/subagent"
)

const Version = "phase1-gin-gorm"

type Options struct {
	RootDir string
	Store   *store.Store
}

type requirementPlanRequest struct {
	Text string `json:"text"`
}

type routeRequest struct {
	Role                  string `json:"role"`
	TaskType              string `json:"task_type"`
	OutputType            string `json:"output_type"`
	RequiresRepoEdit      bool   `json:"requires_repo_edit"`
	IncludesSecrets       bool   `json:"includes_secrets"`
	IncludesSensitiveCode bool   `json:"includes_sensitive_code"`
	IncludesProjectMemory bool   `json:"includes_project_memory"`
}

type releaseSuggestRequest struct {
	Version   string `json:"version"`
	MinIssues int    `json:"min_issues"`
}

type deploymentPlanRequest struct {
	ReleaseID   string   `json:"release_id"`
	Environment string   `json:"environment"`
	ResourceIDs []string `json:"resource_ids"`
	Approved    bool     `json:"approved"`
}

type deploymentExecuteRequest struct {
	Mode     string   `json:"mode"`
	Approved bool     `json:"approved"`
	Commands []string `json:"commands"`
}

type resourceHealthScanRequest struct {
	Environment string   `json:"environment"`
	ResourceIDs []string `json:"resource_ids"`
	Approved    bool     `json:"approved"`
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
	router.GET("/v1/projects/:project_id/epics/:epic_id/issue-graph", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		graph, found, err := issues.LoadGraph(rootDir, c.Param("epic_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "issue graph not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"issue_graph": graph})
	})
	router.GET("/v1/projects/:project_id/epics/:epic_id/schedule", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		if _, found, err := issues.LoadGraph(rootDir, c.Param("epic_id")); err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		} else if !found {
			writeError(c, http.StatusNotFound, "schedule not found")
			return
		}
		plan, err := scheduler.Build(rootDir, c.Param("epic_id"), queryLimit(c, 1))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"schedule": plan})
	})
	router.GET("/v1/projects/:project_id/runs", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		states, err := orchestrator.ListRunStates(rootDir, queryLimit(c, 10))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"runs": states})
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
	router.GET("/v1/projects/:project_id/subagents", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		instances, err := subagent.List(rootDir, queryLimit(c, 10))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"subagents": instances})
	})
	router.GET("/v1/projects/:project_id/subagents/:subagent_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		instance, found, err := subagent.Load(rootDir, c.Param("subagent_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "subagent not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"subagent": instance})
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
	router.GET("/v1/projects/:project_id/quality-policy", func(c *gin.Context) {
		_, _, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"quality_policy": quality.CurrentPolicy()})
	})
	router.GET("/v1/projects/:project_id/quality-reports", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		reports, err := quality.ListReports(rootDir, queryLimit(c, 10))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"quality_reports": reports})
	})
	router.GET("/v1/projects/:project_id/quality/:report_id/explain", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		explanation, found, err := quality.Explain(rootDir, c.Param("report_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "quality report not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"quality_explanation": explanation})
	})
	router.POST("/v1/projects/:project_id/issues/:issue_id/merge-decision", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		decision, err := review.DecideMerge(rootDir, c.Param("issue_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusOK
		if decision.Status != "ready_to_merge" {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"merge_decision": decision})
	})
	router.POST("/v1/projects/:project_id/issues/:issue_id/git-provider-plan", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		plan, err := gitprovider.CreatePlan(c.Request.Context(), rootDir, c.Param("issue_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusOK
		if plan.Status == "blocked" {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"git_provider_plan": plan})
	})
	router.GET("/v1/projects/:project_id/git-provider-plans/:plan_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		plan, found, err := gitprovider.Load(rootDir, c.Param("plan_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "git provider plan not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"git_provider_plan": plan})
	})
	router.POST("/v1/projects/:project_id/releases/suggest", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req releaseSuggestRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		plan, err := release.Suggest(c.Request.Context(), rootDir, release.SuggestOptions{Version: req.Version, MinIssues: req.MinIssues})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusOK
		if plan.Status == "blocked" {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"release": plan})
	})
	router.GET("/v1/projects/:project_id/releases/:release_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		plan, found, err := release.Load(rootDir, c.Param("release_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "release not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"release": plan})
	})
	router.GET("/v1/projects/:project_id/resources", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		resources, err := serverresources.List(rootDir)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"resources": resources})
	})
	router.POST("/v1/projects/:project_id/resources", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req serverresources.Resource
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		resource, err := serverresources.Add(rootDir, req)
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusCreated, gin.H{"resource": resource})
	})
	router.GET("/v1/projects/:project_id/resources/expiration-scan", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		resources, err := serverresources.ExpirationScan(rootDir)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"resources": resources})
	})
	router.POST("/v1/projects/:project_id/resources/health-scan", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req resourceHealthScanRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		report, err := serverresources.HealthScan(c.Request.Context(), rootDir, serverresources.HealthScanOptions{
			Environment: req.Environment,
			ResourceIDs: req.ResourceIDs,
			Approved:    req.Approved,
		})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusOK
		if report.Status == "blocked" || report.Status == "attention_required" {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"health_scan": report})
	})
	router.GET("/v1/projects/:project_id/resources/:resource_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		resource, found, err := serverresources.Show(rootDir, c.Param("resource_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "resource not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"resource": resource})
	})
	router.POST("/v1/projects/:project_id/resources/:resource_id/disable", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		resource, found, err := serverresources.Disable(rootDir, c.Param("resource_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "resource not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"resource": resource})
	})
	router.POST("/v1/projects/:project_id/deployments/plan", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req deploymentPlanRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		plan, err := deployment.CreatePlan(rootDir, deployment.PlanOptions{
			ReleaseID:   req.ReleaseID,
			Environment: req.Environment,
			ResourceIDs: req.ResourceIDs,
			Approved:    req.Approved,
		})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusOK
		if plan.Status == "blocked" {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"deployment": plan})
	})
	router.GET("/v1/projects/:project_id/deployments", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		plans, err := deployment.ListPlans(rootDir, queryLimit(c, 10))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"deployments": plans})
	})
	router.GET("/v1/projects/:project_id/deployments/:deployment_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		plan, found, err := deployment.Load(rootDir, c.Param("deployment_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "deployment not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"deployment": plan})
	})
	router.POST("/v1/projects/:project_id/deployments/:deployment_id/execute", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req deploymentExecuteRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		execution, err := deployment.Execute(c.Request.Context(), rootDir, deployment.ExecuteOptions{
			DeploymentID: c.Param("deployment_id"),
			Mode:         req.Mode,
			Approved:     req.Approved,
			Commands:     req.Commands,
		})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusOK
		if execution.Status == "blocked" || execution.Status == "failed" {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"execution": execution})
	})
	router.GET("/v1/projects/:project_id/deployment-executions", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		executions, err := deployment.ListExecutions(rootDir, queryLimit(c, 10))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"executions": executions})
	})
	router.GET("/v1/projects/:project_id/deployment-executions/:execution_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		execution, found, err := deployment.LoadExecution(rootDir, c.Param("execution_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "deployment execution not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"execution": execution})
	})
	router.POST("/v1/projects/:project_id/requirements/plan", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req requirementPlanRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		plan, err := requirement.PlanFromText(rootDir, req.Text)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusCreated
		if plan.ClarificationDecision.Required {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"requirement": plan})
	})
	router.GET("/v1/projects/:project_id/providers", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		list, err := providers.List(rootDir)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"providers": list})
	})
	router.POST("/v1/projects/:project_id/providers", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req providers.Provider
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		provider, err := providers.Upsert(rootDir, req)
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusCreated, gin.H{"provider": provider})
	})
	router.GET("/v1/projects/:project_id/providers/:provider_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		provider, found, err := providers.Show(rootDir, c.Param("provider_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "provider not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"provider": provider})
	})
	router.POST("/v1/projects/:project_id/providers/:provider_id/disable", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		provider, found, err := providers.Disable(rootDir, c.Param("provider_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "provider not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"provider": provider})
	})
	router.POST("/v1/projects/:project_id/provider-route", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req routeRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		decision, err := providers.Route(rootDir, providers.RouteRequest(req))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusOK
		if decision.Blocked {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"route": decision})
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
	router.GET("/v1/projects/:project_id/requirements/:requirement_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		plan, found, err := requirement.Load(rootDir, c.Param("requirement_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "requirement plan not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"requirement": plan})
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
