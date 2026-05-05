package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"moyuan-code/internal/approvals"
	"moyuan-code/internal/auth"
	"moyuan-code/internal/batch"
	"moyuan-code/internal/controlloop"
	"moyuan-code/internal/controlplane"
	"moyuan-code/internal/deployment"
	"moyuan-code/internal/evidence"
	"moyuan-code/internal/gitprovider"
	"moyuan-code/internal/issues"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/memory"
	"moyuan-code/internal/operations"
	"moyuan-code/internal/orchestrator"
	"moyuan-code/internal/providers"
	"moyuan-code/internal/quality"
	"moyuan-code/internal/release"
	"moyuan-code/internal/repair"
	"moyuan-code/internal/requirement"
	"moyuan-code/internal/review"
	runtimemgr "moyuan-code/internal/runtime"
	"moyuan-code/internal/scheduler"
	"moyuan-code/internal/serverresources"
	"moyuan-code/internal/skills"
	"moyuan-code/internal/store"
	"moyuan-code/internal/subagent"
	"moyuan-code/internal/visuals"
	issueworktree "moyuan-code/internal/worktree"
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
	ModelStrategy         string `json:"model_strategy"`
	TaskType              string `json:"task_type"`
	OutputType            string `json:"output_type"`
	RequiresRepoEdit      bool   `json:"requires_repo_edit"`
	IncludesSecrets       bool   `json:"includes_secrets"`
	IncludesSensitiveCode bool   `json:"includes_sensitive_code"`
	IncludesProjectMemory bool   `json:"includes_project_memory"`
}

type providerOpsRefreshRequest struct {
	ProviderID      string `json:"provider_id"`
	IncludeDisabled bool   `json:"include_disabled"`
	Probe           bool   `json:"probe"`
	ProbeTimeoutMS  int    `json:"probe_timeout_ms"`
	Approved        bool   `json:"approved"`
}

type controlLoopRunRequest struct {
	Trigger            string   `json:"trigger"`
	RequestedBy        string   `json:"requested_by"`
	Steps              []string `json:"steps"`
	MaxSteps           int      `json:"max_steps"`
	StepTimeoutMS      int      `json:"step_timeout_ms"`
	ProviderID         string   `json:"provider_id"`
	IncludeDisabled    bool     `json:"include_disabled"`
	Probe              bool     `json:"probe"`
	ProbeApproved      bool     `json:"probe_approved"`
	ProbeTimeoutMS     int      `json:"probe_timeout_ms"`
	ComprehensionSince string   `json:"comprehension_since"`
}

type batchPlanRequest struct {
	Mode        string `json:"mode"`
	MaxParallel int    `json:"max_parallel"`
	RequestedBy string `json:"requested_by"`
}

type batchRunRequest struct {
	Mode              string `json:"mode"`
	Approved          bool   `json:"approved"`
	MaxIssues         int    `json:"max_issues"`
	RequestedBy       string `json:"requested_by"`
	Prompt            string `json:"prompt"`
	ContinueOnFailure bool   `json:"continue_on_failure"`
}

type operationRepairReviewRequest struct {
	Decision   string `json:"decision"`
	ReviewerID string `json:"reviewer_id"`
	Reason     string `json:"reason"`
	NextStep   string `json:"next_step"`
	RuntimeID  string `json:"runtime_id"`
}

type deploymentRiskHandoffRequest struct {
	AdmissionID      string `json:"admission_id"`
	MonitorSummaryID string `json:"monitor_summary_id"`
}

type deploymentRiskReviewRequest struct {
	Decision   string `json:"decision"`
	ReviewerID string `json:"reviewer_id"`
	Reason     string `json:"reason"`
	NextStep   string `json:"next_step"`
}

type gitProviderCreateRequest struct {
	Approved   bool   `json:"approved"`
	ApprovalID string `json:"approval_id"`
}

type releaseSuggestRequest struct {
	Version   string `json:"version"`
	MinIssues int    `json:"min_issues"`
}

type releaseProviderPublishRequest struct {
	Approved   bool   `json:"approved"`
	ApprovalID string `json:"approval_id"`
}

type deploymentPlanRequest struct {
	ReleaseID   string   `json:"release_id"`
	Environment string   `json:"environment"`
	ResourceIDs []string `json:"resource_ids"`
	Approved    bool     `json:"approved"`
}

type deploymentExecuteRequest struct {
	DeploymentID string   `json:"deployment_id"`
	Environment  string   `json:"environment"`
	Mode         string   `json:"mode"`
	Approved     bool     `json:"approved"`
	ApprovalID   string   `json:"approval_id"`
	Commands     []string `json:"commands"`
}

type rollbackExecuteRequest struct {
	Mode       string   `json:"mode"`
	Approved   bool     `json:"approved"`
	ApprovalID string   `json:"approval_id"`
	Commands   []string `json:"commands"`
}

type monitorSummaryRequest struct {
	Environment string `json:"environment"`
	Limit       int    `json:"limit"`
}

type deploymentRehearsalRequest struct {
	CandidateID  string `json:"candidate_id"`
	DeploymentID string `json:"deployment_id"`
	ExecutionID  string `json:"execution_id"`
	Environment  string `json:"environment"`
	MonitorLimit int    `json:"monitor_limit"`
}

type postDeploymentVerificationRequest struct {
	ExecutionID  string `json:"execution_id"`
	Environment  string `json:"environment"`
	MonitorLimit int    `json:"monitor_limit"`
}

type rehearsalSchedulerRequest struct {
	Trigger       string `json:"trigger"`
	CandidateID   string `json:"candidate_id"`
	DeploymentID  string `json:"deployment_id"`
	ExecutionID   string `json:"execution_id"`
	Environment   string `json:"environment"`
	MonitorLimit  int    `json:"monitor_limit"`
	MaxTargets    int    `json:"max_targets"`
	SkipAdmission bool   `json:"skip_admission"`
	RequestedBy   string `json:"requested_by"`
}

type releaseAdmissionRequest struct {
	RehearsalID  string `json:"rehearsal_id"`
	CandidateID  string `json:"candidate_id"`
	DeploymentID string `json:"deployment_id"`
	ExecutionID  string `json:"execution_id"`
	Environment  string `json:"environment"`
	MonitorLimit int    `json:"monitor_limit"`
}

type resourceHealthScanRequest struct {
	Environment string   `json:"environment"`
	ResourceIDs []string `json:"resource_ids"`
	Approved    bool     `json:"approved"`
}

type resourceRenewRequest struct {
	ExpiresAt string `json:"expires_at"`
	ActorID   string `json:"actor_id"`
	Reason    string `json:"reason"`
}

type resourceRetireRequest struct {
	ActorID string `json:"actor_id"`
	Reason  string `json:"reason"`
}

type approvalRequest struct {
	TargetType  string         `json:"target_type"`
	TargetID    string         `json:"target_id"`
	Action      string         `json:"action"`
	RiskLevel   string         `json:"risk_level"`
	RequestedBy string         `json:"requested_by"`
	Reason      string         `json:"reason"`
	Metadata    map[string]any `json:"metadata"`
}

type approvalDecisionRequest struct {
	Decision  string `json:"decision"`
	DecidedBy string `json:"decided_by"`
	Reason    string `json:"reason"`
}

type authSessionRequest struct {
	UserID      string   `json:"user_id"`
	DisplayName string   `json:"display_name"`
	Roles       []string `json:"roles"`
}

type authTokenRequest struct {
	Name    string   `json:"name"`
	ActorID string   `json:"actor_id"`
	Scopes  []string `json:"scopes"`
}

type serviceAccountRequest struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
}

type revokeRequest struct {
	ActorID string `json:"actor_id"`
	Reason  string `json:"reason"`
}

type visualDiagramPlanRequest struct {
	DiagramType string `json:"diagram_type"`
	Title       string `json:"title"`
	Scope       string `json:"scope"`
	Size        string `json:"size"`
}

type visualRenderRequest struct {
	Mode     string `json:"mode"`
	Approved bool   `json:"approved"`
}

func NewRouter(options Options) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(authzMiddleware(options))
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
	router.POST("/v1/projects/:project_id/epics/:epic_id/batches/plan", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req batchPlanRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		plan, err := batch.CreatePlan(rootDir, batch.PlanOptions{
			EpicID:      c.Param("epic_id"),
			Mode:        req.Mode,
			MaxParallel: req.MaxParallel,
			RequestedBy: req.RequestedBy,
		})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusCreated
		if plan.Status != "planned" {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"batch_plan": plan})
	})
	router.GET("/v1/projects/:project_id/epics/:epic_id/batches", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		plans, err := batch.List(rootDir, c.Param("epic_id"), queryLimit(c, 20))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"batch_plans": plans})
	})
	router.GET("/v1/projects/:project_id/batches", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		plans, err := batch.List(rootDir, c.Query("epic_id"), queryLimit(c, 20))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"batch_plans": plans})
	})
	router.GET("/v1/projects/:project_id/batches/:batch_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		plan, found, err := batch.Load(rootDir, c.Param("batch_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "batch plan not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"batch_plan": plan})
	})
	router.POST("/v1/projects/:project_id/batches/:batch_id/run", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		if _, found, err := batch.Load(rootDir, c.Param("batch_id")); err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		} else if !found {
			writeError(c, http.StatusNotFound, "batch plan not found")
			return
		}
		var req batchRunRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		run, err := batch.Run(c.Request.Context(), rootDir, batch.RunOptions{
			BatchID:           c.Param("batch_id"),
			Mode:              req.Mode,
			Approved:          req.Approved,
			MaxIssues:         req.MaxIssues,
			RequestedBy:       req.RequestedBy,
			Prompt:            req.Prompt,
			ContinueOnFailure: req.ContinueOnFailure,
		})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusAccepted
		if run.Status == "completed" {
			status = http.StatusOK
		}
		c.JSON(status, gin.H{"batch_run": run})
	})
	router.GET("/v1/projects/:project_id/batch-runs", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		runs, err := batch.ListRuns(rootDir, c.Query("batch_id"), queryLimit(c, 20))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"batch_runs": runs})
	})
	router.GET("/v1/projects/:project_id/batch-runs/:run_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		run, found, err := batch.LoadRun(rootDir, c.Param("run_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "batch run not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"batch_run": run})
	})
	router.POST("/v1/projects/:project_id/batches/:batch_id/merge-queue", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		queue, err := review.BuildMergeQueue(rootDir, c.Param("batch_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusAccepted
		if queue.Status == "ready_to_merge" {
			status = http.StatusOK
		}
		c.JSON(status, gin.H{"merge_queue": queue})
	})
	router.GET("/v1/projects/:project_id/merge-queues", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		queues, err := review.ListMergeQueues(rootDir, c.Query("batch_id"), queryLimit(c, 20))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"merge_queues": queues})
	})
	router.POST("/v1/projects/:project_id/merge-queues/:queue_id/integration-preview", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		preview, err := review.BuildIntegrationPreview(c.Request.Context(), rootDir, c.Param("queue_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusAccepted
		if preview.Status == "ready" {
			status = http.StatusOK
		}
		c.JSON(status, gin.H{"integration_preview": preview})
	})
	router.GET("/v1/projects/:project_id/merge-queues/:queue_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		queue, found, err := review.LoadMergeQueue(rootDir, c.Param("queue_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "merge queue not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"merge_queue": queue})
	})
	router.GET("/v1/projects/:project_id/integration-previews", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		previews, err := review.ListIntegrationPreviews(rootDir, c.Query("queue_id"), queryLimit(c, 20))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"integration_previews": previews})
	})
	router.GET("/v1/projects/:project_id/integration-previews/:preview_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		preview, found, err := review.LoadIntegrationPreview(rootDir, c.Param("preview_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "integration preview not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"integration_preview": preview})
	})
	router.POST("/v1/projects/:project_id/integration-previews/:preview_id/apply", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var request review.IntegrationApplyOptions
		if err := c.BindJSON(&request); err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		request.PreviewID = c.Param("preview_id")
		apply, err := review.ApplyIntegrationPreview(c.Request.Context(), rootDir, request)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusAccepted
		if apply.Status == "planned" || apply.Status == "applied" {
			status = http.StatusOK
		}
		c.JSON(status, gin.H{"integration_apply": apply})
	})
	router.POST("/v1/projects/:project_id/integration-applies/:apply_id/release-batch", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var request release.BatchOptions
		if err := c.BindJSON(&request); err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		request.IntegrationApplyID = c.Param("apply_id")
		plan, err := release.PlanBatch(rootDir, request)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusAccepted
		if plan.Status == "suggested" || plan.Status == "not_ready" {
			status = http.StatusOK
		}
		c.JSON(status, gin.H{"release_batch": plan})
	})
	router.GET("/v1/projects/:project_id/integration-applies", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		applies, err := review.ListIntegrationApplies(rootDir, c.Query("preview_id"), queryLimit(c, 20))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"integration_applies": applies})
	})
	router.GET("/v1/projects/:project_id/integration-applies/:apply_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		apply, found, err := review.LoadIntegrationApply(rootDir, c.Param("apply_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "integration apply not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"integration_apply": apply})
	})
	router.GET("/v1/projects/:project_id/release-batches", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		plans, err := release.ListBatchPlans(rootDir, c.Query("integration_apply_id"), queryLimit(c, 20))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"release_batches": plans})
	})
	router.GET("/v1/projects/:project_id/release-batches/:batch_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		plan, found, err := release.LoadBatchPlan(rootDir, c.Param("batch_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "release batch not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"release_batch": plan})
	})
	router.POST("/v1/projects/:project_id/release-batches/:batch_id/candidate", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var request release.CandidateOptions
		if err := c.BindJSON(&request); err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		request.ReleaseBatchID = c.Param("batch_id")
		candidate, err := release.PlanCandidate(c.Request.Context(), rootDir, request)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusAccepted
		if candidate.Status == "ready" {
			status = http.StatusOK
		}
		c.JSON(status, gin.H{"release_candidate": candidate})
	})
	router.GET("/v1/projects/:project_id/release-candidates", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		candidates, err := release.ListCandidates(rootDir, c.Query("release_batch_id"), queryLimit(c, 20))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"release_candidates": candidates})
	})
	router.GET("/v1/projects/:project_id/release-candidates/:candidate_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		candidate, found, err := release.LoadCandidate(rootDir, c.Param("candidate_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "release candidate not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"release_candidate": candidate})
	})
	router.POST("/v1/projects/:project_id/release-candidates/:candidate_id/apply", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var request release.CandidateApplyOptions
		if err := c.BindJSON(&request); err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		request.CandidateID = c.Param("candidate_id")
		apply, err := release.ApplyCandidate(c.Request.Context(), rootDir, request)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusAccepted
		if apply.Status == "planned" || apply.Status == "applied" {
			status = http.StatusOK
		}
		c.JSON(status, gin.H{"release_candidate_apply": apply})
	})
	router.GET("/v1/projects/:project_id/release-candidate-applies", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		applies, err := release.ListCandidateApplies(rootDir, c.Query("candidate_id"), queryLimit(c, 20))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"release_candidate_applies": applies})
	})
	router.GET("/v1/projects/:project_id/release-candidate-applies/:apply_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		apply, found, err := release.LoadCandidateApply(rootDir, c.Param("apply_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "release candidate apply not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"release_candidate_apply": apply})
	})
	router.POST("/v1/projects/:project_id/release-candidates/:candidate_id/provider-preview", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		preview, found, err := release.ProviderPreviewForCandidate(rootDir, c.Param("candidate_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "release candidate not found")
			return
		}
		status := http.StatusOK
		if preview.Status == "blocked" {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"release_candidate_provider_preview": preview})
	})
	router.GET("/v1/projects/:project_id/release-candidate-provider-previews", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		previews, err := release.ListCandidateProviderPreviews(rootDir, c.Query("candidate_id"), queryLimit(c, 20))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"release_candidate_provider_previews": previews})
	})
	router.GET("/v1/projects/:project_id/release-candidate-provider-previews/:preview_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		preview, found, err := release.LoadCandidateProviderPreview(rootDir, c.Param("preview_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "release candidate provider preview not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"release_candidate_provider_preview": preview})
	})
	router.POST("/v1/projects/:project_id/release-candidates/:candidate_id/provider-publish", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req releaseProviderPublishRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		execution, found, err := release.ProviderPublishForCandidate(rootDir, release.CandidateProviderPublishOptions{
			CandidateID: c.Param("candidate_id"),
			Approved:    req.Approved,
			ApprovalID:  req.ApprovalID,
		})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "release candidate not found")
			return
		}
		status := http.StatusOK
		if execution.Status == "blocked" || execution.Decision == "RELEASE_PROVIDER_PUBLISH_PREVIEW_ONLY" {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"release_provider_execution": execution})
	})
	router.POST("/v1/projects/:project_id/release-candidates/:candidate_id/pr-mr-plan", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		candidate, found, err := release.LoadCandidate(rootDir, c.Param("candidate_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "release candidate not found")
			return
		}
		candidateReady := candidate.Status == "ready" && candidate.Decision == "RELEASE_CANDIDATE_READY"
		reasons := []string{}
		if !candidateReady {
			reasons = append(reasons, "release_candidate_not_ready:"+candidate.Decision)
		} else if !release.CandidateReleaseBranchApplied(rootDir, candidate.ID) {
			candidateReady = false
			reasons = append(reasons, "release_branch_apply_missing")
		}
		plan, err := gitprovider.PlanReleaseCandidate(rootDir, gitprovider.ReleaseCandidatePlanOptions{
			CandidateID:    candidate.ID,
			CandidateReady: candidateReady,
			Version:        candidate.Version,
			Provider:       candidate.Provider,
			RemoteName:     candidate.RemoteName,
			RemoteURL:      candidate.RemoteURL,
			ReleaseBranch:  candidate.ReleaseBranch,
			Reasons:        reasons,
		})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusOK
		if plan.Status != "pr_mr_plan_ready" {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"git_provider_plan": plan})
	})
	router.POST("/v1/projects/:project_id/release-candidates/:candidate_id/deployment-plan", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var request deployment.CandidatePlanOptions
		if err := c.BindJSON(&request); err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		request.CandidateID = c.Param("candidate_id")
		plan, err := deployment.CreatePlanFromCandidate(rootDir, request)
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
	router.POST("/v1/projects/:project_id/release-candidates/:candidate_id/deployment-execution", func(c *gin.Context) {
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
		execution, err := deployment.ExecuteFromCandidate(c.Request.Context(), rootDir, deployment.CandidateExecuteOptions{
			CandidateID:  c.Param("candidate_id"),
			DeploymentID: req.DeploymentID,
			Environment:  req.Environment,
			Mode:         req.Mode,
			Approved:     req.Approved,
			ApprovalID:   req.ApprovalID,
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
	router.GET("/v1/projects/:project_id/release-candidates/:candidate_id/deployment-feedback", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		feedback, found, err := deployment.FeedbackForCandidate(rootDir, c.Param("candidate_id"), queryLimit(c, 10))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "release candidate not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"deployment_feedback": feedback})
	})
	router.GET("/v1/projects/:project_id/worktrees", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		records, err := issueworktree.List(rootDir, c.Query("issue_id"), queryLimit(c, 20))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"worktrees": records})
	})
	router.GET("/v1/projects/:project_id/worktrees/:worktree_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		record, found, err := issueworktree.Load(rootDir, c.Param("worktree_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "worktree not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"worktree": record})
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
	router.GET("/v1/projects/:project_id/audit-events", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		stream := c.Query("channel")
		if stream == "" {
			stream = c.Query("stream")
		}
		events, err := logging.List(rootDir, logging.Query{
			Stream:  stream,
			IssueID: c.Query("issue_id"),
			RunID:   c.Query("run_id"),
			Event:   c.Query("event"),
			Limit:   queryLimit(c, 20),
		})
		if err != nil {
			if logging.IsInvalidStreamError(err) {
				writeError(c, http.StatusBadRequest, err.Error())
				return
			}
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"audit_events": events})
	})
	router.GET("/v1/projects/:project_id/approvals", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		records, err := approvals.List(rootDir, approvals.ListOptions{Status: c.Query("status"), Limit: queryLimit(c, 20)})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"approvals": records})
	})
	router.POST("/v1/projects/:project_id/approvals", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req approvalRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		record, err := approvals.Request(rootDir, approvals.RequestOptions{
			TargetType:  req.TargetType,
			TargetID:    req.TargetID,
			Action:      req.Action,
			RiskLevel:   req.RiskLevel,
			RequestedBy: req.RequestedBy,
			Reason:      req.Reason,
			Metadata:    req.Metadata,
		})
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusCreated, gin.H{"approval": record})
	})
	router.GET("/v1/projects/:project_id/approvals/:approval_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		record, found, err := approvals.Load(rootDir, c.Param("approval_id"))
		if err != nil {
			if approvals.IsInvalidIDError(err) {
				writeError(c, http.StatusBadRequest, err.Error())
				return
			}
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "approval not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"approval": record})
	})
	router.POST("/v1/projects/:project_id/approvals/:approval_id/decide", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req approvalDecisionRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		record, found, err := approvals.Decide(rootDir, c.Param("approval_id"), approvals.DecisionOptions{
			Decision:  req.Decision,
			DecidedBy: req.DecidedBy,
			Reason:    req.Reason,
		})
		if err != nil {
			if approvals.IsInvalidIDError(err) {
				writeError(c, http.StatusBadRequest, err.Error())
				return
			}
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "approval not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"approval": record})
	})
	router.GET("/v1/projects/:project_id/auth/sessions", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		sessions, err := auth.ListSessions(rootDir)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"sessions": sessions})
	})
	router.POST("/v1/projects/:project_id/auth/sessions", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req authSessionRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		session, err := auth.CreateSession(rootDir, auth.CreateSessionOptions{UserID: req.UserID, DisplayName: req.DisplayName, Roles: req.Roles})
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusCreated, gin.H{"session": session})
	})
	router.POST("/v1/projects/:project_id/auth/sessions/:session_id/revoke", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req revokeRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		session, found, err := auth.RevokeSession(rootDir, c.Param("session_id"), auth.RevokeOptions{ActorID: req.ActorID, Reason: req.Reason})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "session not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"session": session})
	})
	router.GET("/v1/projects/:project_id/auth/api-tokens", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		tokens, err := auth.ListAPITokens(rootDir)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"api_tokens": tokens})
	})
	router.POST("/v1/projects/:project_id/auth/api-tokens", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req authTokenRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		token, err := auth.CreateAPIToken(rootDir, auth.CreateTokenOptions{Name: req.Name, ActorID: req.ActorID, Scopes: req.Scopes})
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusCreated, gin.H{"api_token": token.Token, "token_value": token.TokenValue})
	})
	router.POST("/v1/projects/:project_id/auth/api-tokens/:token_id/revoke", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req revokeRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		token, found, err := auth.RevokeAPIToken(rootDir, c.Param("token_id"), auth.RevokeOptions{ActorID: req.ActorID, Reason: req.Reason})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "api token not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"api_token": token})
	})
	router.GET("/v1/projects/:project_id/auth/service-accounts", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		accounts, err := auth.ListServiceAccounts(rootDir)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"service_accounts": accounts})
	})
	router.POST("/v1/projects/:project_id/auth/service-accounts", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req serviceAccountRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		account, err := auth.CreateServiceAccount(rootDir, auth.CreateServiceAccountOptions{ID: req.ID, Name: req.Name, Roles: req.Roles})
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusCreated, gin.H{"service_account": account})
	})
	router.GET("/v1/projects/:project_id/runtime-recoveries", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		records, err := runtimemgr.ListRecoveries(rootDir, queryLimit(c, 10))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"runtime_recoveries": records})
	})
	router.GET("/v1/projects/:project_id/runtime-recoveries/:recovery_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		record, found, err := runtimemgr.LoadRecovery(rootDir, c.Param("recovery_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "runtime recovery not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"runtime_recovery": record})
	})
	router.GET("/v1/projects/:project_id/runtime-recoveries/:recovery_id/artifacts", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		artifacts, found, err := runtimemgr.LoadRecoveryArtifacts(rootDir, c.Param("recovery_id"), 6000)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "runtime recovery not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"runtime_recovery_artifacts": artifacts})
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
	router.POST("/v1/projects/:project_id/visuals/diagrams/plan", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req visualDiagramPlanRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		plan, err := visuals.GeneratePlan(rootDir, visuals.DiagramOptions{
			DiagramType: req.DiagramType,
			Title:       req.Title,
			Scope:       req.Scope,
			Size:        req.Size,
		})
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusCreated, gin.H{"visual_plan": plan})
	})
	router.GET("/v1/projects/:project_id/visuals/assets", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		assets, err := visuals.ListAssets(rootDir, queryLimit(c, 10))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"visual_assets": assets})
	})
	router.POST("/v1/projects/:project_id/visuals/assets/:asset_id/render", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req visualRenderRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		execution, err := visuals.RenderAsset(c.Request.Context(), rootDir, visuals.RenderOptions{
			AssetID:  c.Param("asset_id"),
			Mode:     req.Mode,
			Approved: req.Approved,
		})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusOK
		if execution.Status == "blocked" || execution.Status == "failed" {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"visual_render_execution": execution})
	})
	router.GET("/v1/projects/:project_id/visuals/render-executions", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		executions, err := visuals.ListRenderExecutions(rootDir, queryLimit(c, 10))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"visual_render_executions": executions})
	})
	router.GET("/v1/projects/:project_id/visuals/render-executions/:execution_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		execution, found, err := visuals.LoadRenderExecution(rootDir, c.Param("execution_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "visual render execution not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"visual_render_execution": execution})
	})
	router.GET("/v1/projects/:project_id/visuals/assets/:asset_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		asset, found, err := visuals.LoadAsset(rootDir, c.Param("asset_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "visual asset not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"visual_asset": asset})
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
	router.GET("/v1/projects/:project_id/git-provider-plans", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		plans, err := gitprovider.List(rootDir, queryLimit(c, 20))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"git_provider_plans": plans})
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
	router.POST("/v1/projects/:project_id/git-provider-plans/:plan_id/sync", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		plan, found, err := gitprovider.SyncStatus(c.Request.Context(), rootDir, c.Param("plan_id"))
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
	router.POST("/v1/projects/:project_id/git-provider-plans/:plan_id/preview", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		plan, found, err := gitprovider.Preview(rootDir, c.Param("plan_id"))
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
	router.POST("/v1/projects/:project_id/git-provider-plans/:plan_id/create", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req gitProviderCreateRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		plan, found, err := gitprovider.Create(c.Request.Context(), rootDir, c.Param("plan_id"), gitprovider.CreateOptions{Approved: req.Approved, ApprovalID: req.ApprovalID})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "git provider plan not found")
			return
		}
		status := http.StatusOK
		if plan.PRMR.CreateDecision == "PR_MR_CREATE_APPROVAL_REQUIRED" || plan.PRMR.CreateDecision == "PR_MR_CREATE_PREVIEW_ONLY" {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"git_provider_plan": plan})
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
	router.POST("/v1/projects/:project_id/releases/:release_id/provider-preview", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		execution, found, err := release.ProviderPreview(rootDir, c.Param("release_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "release not found")
			return
		}
		status := http.StatusOK
		if execution.Status == "blocked" {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"release_provider_execution": execution})
	})
	router.POST("/v1/projects/:project_id/releases/:release_id/provider-publish", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req releaseProviderPublishRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		execution, found, err := release.ProviderPublish(rootDir, release.ProviderOptions{
			ReleaseID:  c.Param("release_id"),
			Approved:   req.Approved,
			ApprovalID: req.ApprovalID,
		})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "release not found")
			return
		}
		status := http.StatusOK
		if execution.Status == "blocked" || execution.Decision == "RELEASE_PROVIDER_PUBLISH_PREVIEW_ONLY" {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"release_provider_execution": execution})
	})
	router.GET("/v1/projects/:project_id/release-provider-executions", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		executions, err := release.ListProviderExecutions(rootDir, queryLimit(c, 10))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"release_provider_executions": executions})
	})
	router.GET("/v1/projects/:project_id/release-provider-executions/:execution_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		execution, found, err := release.LoadProviderExecution(rootDir, c.Param("execution_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "release provider execution not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"release_provider_execution": execution})
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
	router.GET("/v1/projects/:project_id/resources/maintenance", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		records, err := serverresources.ListMaintenance(rootDir, queryLimit(c, 20))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"maintenance_records": records})
	})
	router.GET("/v1/projects/:project_id/resources/lifecycle-alerts", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		alerts, err := serverresources.ListLifecycleAlerts(rootDir, queryLimit(c, 20))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"lifecycle_alerts": alerts})
	})
	router.POST("/v1/projects/:project_id/resources/maintenance/scan", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		records, err := serverresources.MaintenanceScan(rootDir)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"maintenance_records": records})
	})
	router.POST("/v1/projects/:project_id/resources/lifecycle/scan", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		report, err := serverresources.LifecycleScan(rootDir)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusOK
		if report.Status == "attention_required" {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"lifecycle_scan": report})
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
	router.GET("/v1/projects/:project_id/resources/maintenance-policy", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		pack, err := serverresources.LoadMaintenancePolicyPack(rootDir, c.Query("environment"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		payload := gin.H{"maintenance_policy_pack": pack}
		if c.Query("action") != "" {
			payload["maintenance_policy_decision"] = serverresources.EvaluateMaintenancePolicy(pack, serverresources.MaintenancePolicyContext{
				Environment: c.Query("environment"),
				Action:      c.Query("action"),
				ResourceID:  c.Query("resource_id"),
				RequestedAt: c.Query("requested_at"),
			})
		}
		c.JSON(http.StatusOK, payload)
	})
	router.GET("/v1/projects/:project_id/resources/deployment-refs", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		refs, err := serverresources.ListDeploymentReferences(rootDir, queryLimit(c, 20))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"resource_deployment_refs": refs})
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
	router.POST("/v1/projects/:project_id/resources/:resource_id/renew", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req resourceRenewRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		resource, record, found, err := serverresources.Renew(rootDir, serverresources.RenewalOptions{
			ResourceID: c.Param("resource_id"),
			ExpiresAt:  req.ExpiresAt,
			ActorID:    req.ActorID,
			Reason:     req.Reason,
		})
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "resource not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"resource": resource, "maintenance_record": record})
	})
	router.POST("/v1/projects/:project_id/resources/:resource_id/retire", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req resourceRetireRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		resource, record, found, err := serverresources.Retire(rootDir, serverresources.RetireOptions{
			ResourceID: c.Param("resource_id"),
			ActorID:    req.ActorID,
			Reason:     req.Reason,
		})
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "resource not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"resource": resource, "maintenance_record": record})
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
			ApprovalID:   req.ApprovalID,
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
	router.GET("/v1/projects/:project_id/deployment-monitor-history", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		histories, err := deployment.ListPostDeploymentHistories(rootDir, queryLimit(c, 10))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"post_deployment_histories": histories})
	})
	router.POST("/v1/projects/:project_id/deployment-monitor-summary", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req monitorSummaryRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		summary, err := deployment.BuildMonitorSummary(rootDir, deployment.MonitorSummaryOptions{Environment: req.Environment, Limit: req.Limit})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusOK
		if summary.Status == "critical" || summary.Status == "attention_required" || summary.Status == "unknown" {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"monitor_summary": summary})
	})
	router.GET("/v1/projects/:project_id/deployment-monitor-summaries", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		summaries, err := deployment.ListMonitorSummaries(rootDir, queryLimit(c, 10))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"monitor_summaries": summaries})
	})
	router.POST("/v1/projects/:project_id/post-deployment-verifications", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req postDeploymentVerificationRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		verification, err := deployment.BuildPostDeploymentVerification(rootDir, deployment.PostDeploymentVerificationOptions{
			ExecutionID:  req.ExecutionID,
			Environment:  req.Environment,
			MonitorLimit: req.MonitorLimit,
		})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusCreated
		if verification.Status == "blocked" || verification.Status == "attention_required" || verification.Status == "failed" {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"post_deployment_verification": verification})
	})
	router.GET("/v1/projects/:project_id/post-deployment-verifications", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		verifications, err := deployment.ListPostDeploymentVerifications(rootDir, queryLimit(c, 10))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"post_deployment_verifications": verifications})
	})
	router.GET("/v1/projects/:project_id/post-deployment-verifications/:verification_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		verification, found, err := deployment.LoadPostDeploymentVerification(rootDir, c.Param("verification_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "post deployment verification not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"post_deployment_verification": verification})
	})
	router.POST("/v1/projects/:project_id/deployment-rehearsals", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req deploymentRehearsalRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		rehearsal, err := deployment.BuildRehearsal(c.Request.Context(), rootDir, deployment.RehearsalOptions{
			CandidateID:  req.CandidateID,
			DeploymentID: req.DeploymentID,
			ExecutionID:  req.ExecutionID,
			Environment:  req.Environment,
			MonitorLimit: req.MonitorLimit,
		})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusCreated
		if rehearsal.Status == "blocked" || rehearsal.Status == "attention_required" {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"deployment_rehearsal": rehearsal})
	})
	router.GET("/v1/projects/:project_id/deployment-rehearsals", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		rehearsals, err := deployment.ListRehearsals(rootDir, queryLimit(c, 10))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"deployment_rehearsals": rehearsals})
	})
	router.GET("/v1/projects/:project_id/deployment-rehearsals/:rehearsal_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		rehearsal, found, err := deployment.LoadRehearsal(rootDir, c.Param("rehearsal_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "deployment rehearsal not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"deployment_rehearsal": rehearsal})
	})
	router.POST("/v1/projects/:project_id/deployment-rehearsal-scheduler-runs", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req rehearsalSchedulerRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		run, err := deployment.RunRehearsalScheduler(c.Request.Context(), rootDir, deployment.RehearsalSchedulerOptions{
			Trigger:       req.Trigger,
			CandidateID:   req.CandidateID,
			DeploymentID:  req.DeploymentID,
			ExecutionID:   req.ExecutionID,
			Environment:   req.Environment,
			MonitorLimit:  req.MonitorLimit,
			MaxTargets:    req.MaxTargets,
			SkipAdmission: req.SkipAdmission,
			RequestedBy:   req.RequestedBy,
		})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusCreated
		if run.Status == "blocked" || run.Status == "attention_required" {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"rehearsal_scheduler_run": run})
	})
	router.GET("/v1/projects/:project_id/deployment-rehearsal-scheduler-runs", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		runs, err := deployment.ListRehearsalSchedulerRuns(rootDir, queryLimit(c, 10))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"rehearsal_scheduler_runs": runs})
	})
	router.GET("/v1/projects/:project_id/deployment-rehearsal-scheduler-runs/:run_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		run, found, err := deployment.LoadRehearsalSchedulerRun(rootDir, c.Param("run_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "rehearsal scheduler run not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"rehearsal_scheduler_run": run})
	})
	router.POST("/v1/projects/:project_id/release-admissions", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req releaseAdmissionRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		admission, err := deployment.BuildReleaseAdmission(c.Request.Context(), rootDir, deployment.ReleaseAdmissionOptions{
			RehearsalID:  req.RehearsalID,
			CandidateID:  req.CandidateID,
			DeploymentID: req.DeploymentID,
			ExecutionID:  req.ExecutionID,
			Environment:  req.Environment,
			MonitorLimit: req.MonitorLimit,
		})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusCreated
		if admission.Status == "blocked" || admission.Status == "manual_required" {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"release_admission": admission})
	})
	router.GET("/v1/projects/:project_id/release-admission-policy", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		policy, err := deployment.LoadReleaseAdmissionPolicyPack(rootDir, c.Query("environment"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"release_admission_policy_pack": policy})
	})
	router.GET("/v1/projects/:project_id/release-admissions", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		admissions, err := deployment.ListReleaseAdmissions(rootDir, queryLimit(c, 10))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"release_admissions": admissions})
	})
	router.GET("/v1/projects/:project_id/release-admissions/:admission_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		admission, found, err := deployment.LoadReleaseAdmission(rootDir, c.Param("admission_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "release admission not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"release_admission": admission})
	})
	router.GET("/v1/projects/:project_id/deployment-executions/:execution_id/post-deployment-history", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		history, found, err := deployment.LoadPostDeploymentHistory(rootDir, c.Param("execution_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "post deployment history not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"post_deployment_history": history})
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
	router.POST("/v1/projects/:project_id/deployment-executions/:execution_id/rollback", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req rollbackExecuteRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		rollback, err := deployment.ExecuteRollback(c.Request.Context(), rootDir, deployment.RollbackExecuteOptions{
			ExecutionID: c.Param("execution_id"),
			Mode:        req.Mode,
			Approved:    req.Approved,
			ApprovalID:  req.ApprovalID,
			Commands:    req.Commands,
		})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusOK
		if rollback.Status == "blocked" || rollback.Status == "failed" {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"rollback_execution": rollback})
	})
	router.GET("/v1/projects/:project_id/deployment-rollback-executions", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		executions, err := deployment.ListRollbackExecutions(rootDir, queryLimit(c, 10))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"rollback_executions": executions})
	})
	router.GET("/v1/projects/:project_id/deployment-rollback-executions/:rollback_execution_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		rollback, found, err := deployment.LoadRollbackExecution(rootDir, c.Param("rollback_execution_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "deployment rollback execution not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"rollback_execution": rollback})
	})
	router.GET("/v1/projects/:project_id/evidence", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		records, err := evidence.List(rootDir, evidence.ListOptions{
			ParentType:  c.Query("parent_type"),
			ParentID:    c.Query("parent_id"),
			SubjectType: c.Query("subject_type"),
			SubjectID:   c.Query("subject_id"),
			Limit:       queryLimit(c, 20),
		})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"evidence": records})
	})
	router.GET("/v1/projects/:project_id/evidence/:evidence_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		record, found, err := evidence.Load(rootDir, c.Param("evidence_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "evidence not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"evidence": record})
	})
	router.GET("/v1/projects/:project_id/operations/timeline", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		items, err := operations.Timeline(rootDir, operations.TimelineOptions{
			Type:        c.Query("type"),
			Status:      c.Query("status"),
			Decision:    c.Query("decision"),
			Environment: c.Query("environment"),
			Limit:       queryLimit(c, 20),
		})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"operations_timeline": items})
	})
	router.GET("/v1/projects/:project_id/operations/audit-export", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		report, err := operations.ExportAudit(rootDir, operations.AuditExportOptions{
			Type:        c.Query("type"),
			Status:      c.Query("status"),
			Decision:    c.Query("decision"),
			Environment: c.Query("environment"),
			Limit:       queryLimit(c, 20),
			Format:      c.Query("format"),
		})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"operations_audit_export": report})
	})
	router.GET("/v1/projects/:project_id/operations/:operation_type/:operation_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		detail, found, err := operations.Load(rootDir, c.Param("operation_type"), c.Param("operation_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "operation not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"operation_detail": detail})
	})
	router.POST("/v1/projects/:project_id/operations/:operation_type/:operation_id/repair-candidate", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		candidate, found, err := repair.CandidateFromOperation(rootDir, c.Param("operation_type"), c.Param("operation_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "operation not found")
			return
		}
		status := http.StatusCreated
		if candidate.Decision != "REPAIR_CANDIDATE_CREATED" {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"operation_repair_candidate": candidate})
	})
	router.POST("/v1/projects/:project_id/control-loop/run", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req controlLoopRunRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		run, err := controlloop.Run(c.Request.Context(), rootDir, controlloop.RunOptions{
			Trigger:            req.Trigger,
			RequestedBy:        req.RequestedBy,
			Steps:              req.Steps,
			MaxSteps:           req.MaxSteps,
			StepTimeoutMS:      req.StepTimeoutMS,
			ProviderID:         req.ProviderID,
			IncludeDisabled:    req.IncludeDisabled,
			Probe:              req.Probe,
			ProbeApproved:      req.ProbeApproved,
			ProbeTimeoutMS:     req.ProbeTimeoutMS,
			ComprehensionSince: req.ComprehensionSince,
		})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusOK
		if run.Status == "failed" || run.Decision == "CONTROL_LOOP_COMPLETED_WITH_ATTENTION" {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"control_loop_run": run})
	})
	router.GET("/v1/projects/:project_id/control-loop/runs", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		runs, err := controlloop.List(rootDir, queryLimit(c, 20))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"control_loop_runs": runs})
	})
	router.GET("/v1/projects/:project_id/control-loop/runs/:run_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		run, found, err := controlloop.Load(rootDir, c.Param("run_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "control loop run not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"control_loop_run": run})
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
	router.GET("/v1/projects/:project_id/providers/telemetry", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		records, err := providers.ListTelemetry(rootDir, c.Query("provider_id"), queryLimit(c, 20))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"provider_telemetry": records})
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
	router.POST("/v1/projects/:project_id/providers/ops/refresh", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req providerOpsRefreshRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		result, err := providers.RefreshOps(rootDir, providers.OpsRefreshOptions{
			ProviderID:      req.ProviderID,
			IncludeDisabled: req.IncludeDisabled,
			Probe:           req.Probe,
			ProbeTimeoutMS:  req.ProbeTimeoutMS,
			Approved:        req.Approved,
		})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"provider_ops_refresh": result})
	})
	router.POST("/v1/projects/:project_id/providers/:provider_id/ops", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req providers.OpsSnapshot
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		provider, found, err := providers.UpdateOps(rootDir, c.Param("provider_id"), req)
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
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
	router.GET("/v1/projects/:project_id/skills", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		list, err := skills.List(rootDir)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"skills": list})
	})
	router.POST("/v1/projects/:project_id/skills", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req skills.Definition
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		skill, err := skills.Upsert(rootDir, req)
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusCreated, gin.H{"skill": skill})
	})
	router.POST("/v1/projects/:project_id/skills/recommend", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req skills.RecommendOptions
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		report, err := skills.Recommend(rootDir, req)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusCreated, gin.H{"skill_recommendation": report})
	})
	router.GET("/v1/projects/:project_id/skills/bindings", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		bindings, err := skills.ListBindings(rootDir)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"skill_bindings": bindings})
	})
	router.POST("/v1/projects/:project_id/skills/bindings", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req skills.Binding
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		binding, err := skills.UpsertBinding(rootDir, req)
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusCreated, gin.H{"skill_binding": binding})
	})
	router.POST("/v1/projects/:project_id/skills/bindings/:binding_id/disable", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		binding, found, err := skills.DisableBinding(rootDir, c.Param("binding_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "skill binding not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"skill_binding": binding})
	})
	router.GET("/v1/projects/:project_id/skills/effectiveness", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		records, err := skills.ListEffectiveness(rootDir, c.Query("skill_id"), queryLimit(c, 20))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"skill_effectiveness": records})
	})
	router.POST("/v1/projects/:project_id/skills/effectiveness", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req skills.Effectiveness
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		record, err := skills.RecordEffectiveness(rootDir, req)
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusCreated, gin.H{"skill_effectiveness": record})
	})
	router.POST("/v1/projects/:project_id/skills/:skill_id/disable", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		skill, found, err := skills.Disable(rootDir, c.Param("skill_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "skill not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"skill": skill})
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
	router.GET("/v1/projects/:project_id/repair/operation-candidates", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		candidates, err := repair.ListOperationRepairCandidates(rootDir, queryLimit(c, 20))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"operation_repair_candidates": candidates})
	})
	router.POST("/v1/projects/:project_id/repair/deployment-risk-handoffs", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req deploymentRiskHandoffRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		handoff, err := repair.CreateDeploymentRiskHandoff(rootDir, repair.DeploymentRiskHandoffOptions{
			AdmissionID:      req.AdmissionID,
			MonitorSummaryID: req.MonitorSummaryID,
		})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		status := http.StatusCreated
		if handoff.Status == "blocked" || handoff.Status == "review_required" {
			status = http.StatusAccepted
		}
		c.JSON(status, gin.H{"deployment_risk_handoff": handoff})
	})
	router.GET("/v1/projects/:project_id/repair/deployment-risk-handoffs", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		handoffs, err := repair.ListDeploymentRiskHandoffs(rootDir, queryLimit(c, 20))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"deployment_risk_handoffs": handoffs})
	})
	router.GET("/v1/projects/:project_id/repair/deployment-risk-handoffs/:handoff_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		handoff, found, err := repair.LoadDeploymentRiskHandoff(rootDir, c.Param("handoff_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "deployment risk handoff not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"deployment_risk_handoff": handoff})
	})
	router.GET("/v1/projects/:project_id/repair/deployment-risk-review-queue", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		items, err := repair.ListDeploymentRiskReviewQueue(rootDir, c.Query("status"), queryLimit(c, 20))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"deployment_risk_review_queue": items})
	})
	router.POST("/v1/projects/:project_id/repair/deployment-risk-handoffs/:handoff_id/review", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req deploymentRiskReviewRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		review, handoff, found, err := repair.ReviewDeploymentRiskHandoff(rootDir, c.Param("handoff_id"), repair.DeploymentRiskReviewOptions{
			Decision:   req.Decision,
			ReviewerID: req.ReviewerID,
			Reason:     req.Reason,
			NextStep:   req.NextStep,
		})
		if err != nil {
			status := http.StatusInternalServerError
			if err.Error() == "deployment_risk_review_decision_required" || err.Error() == "deployment_risk_handoff_not_reviewable" {
				status = http.StatusBadRequest
			}
			writeError(c, status, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "deployment risk handoff not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"deployment_risk_review": review, "deployment_risk_handoff": handoff})
	})
	router.GET("/v1/projects/:project_id/repair/deployment-risk-reviews", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		reviews, err := repair.ListDeploymentRiskReviews(rootDir, queryLimit(c, 20))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"deployment_risk_reviews": reviews})
	})
	router.GET("/v1/projects/:project_id/repair/deployment-risk-reviews/:review_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		review, found, err := repair.LoadDeploymentRiskReview(rootDir, c.Param("review_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "deployment risk review not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"deployment_risk_review": review})
	})
	router.GET("/v1/projects/:project_id/repair/operation-candidates/:candidate_id", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		candidate, found, err := repair.LoadOperationRepairCandidate(rootDir, c.Param("candidate_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "operation repair candidate not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"operation_repair_candidate": candidate})
	})
	router.POST("/v1/projects/:project_id/repair/operation-candidates/:candidate_id/review", func(c *gin.Context) {
		_, rootDir, ok, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(c, http.StatusNotFound, "project not found")
			return
		}
		var req operationRepairReviewRequest
		if err := c.BindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		review, candidate, attempt, found, err := repair.ReviewOperationRepairCandidate(c.Request.Context(), rootDir, c.Param("candidate_id"), repair.OperationRepairReviewOptions{
			Decision:   req.Decision,
			ReviewerID: req.ReviewerID,
			Reason:     req.Reason,
			NextStep:   req.NextStep,
			RuntimeID:  req.RuntimeID,
		})
		if err != nil {
			status := http.StatusInternalServerError
			if err.Error() == "review_decision_required" || err.Error() == "operation_repair_candidate_not_reviewable" || err.Error() == "repair_plan_required" {
				status = http.StatusBadRequest
			}
			writeError(c, status, err.Error())
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "operation repair candidate not found")
			return
		}
		payload := gin.H{"operation_repair_review": review, "operation_repair_candidate": candidate}
		if attempt != nil {
			payload["repair_attempt"] = attempt
		}
		c.JSON(http.StatusOK, payload)
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

type authzRule struct {
	Action string
	Risk   string
	Scopes []string
}

func authzMiddleware(options Options) gin.HandlerFunc {
	return func(c *gin.Context) {
		rule, ok := protectedAuthzRule(c.Request.Method, c.FullPath(), c.Request.URL.Path)
		if !ok {
			c.Next()
			return
		}
		_, rootDir, found, err := findProject(options, c.Param("project_id"))
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			c.Abort()
			return
		}
		if !found {
			writeError(c, http.StatusNotFound, "project not found")
			c.Abort()
			return
		}
		ctx, resolved, err := resolveRequestContext(rootDir, c)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			c.Abort()
			return
		}
		result := auth.Authorize(ctx, rule.Action, rule.Risk, rule.Scopes)
		if !resolved && result.Reason == "AUTH_PERMISSION_DENIED" {
			result.Reason = "AUTH_MISSING_CREDENTIAL"
		}
		_ = logging.Log(rootDir, "audit", "auth.decision."+strings.ToLower(result.Decision), map[string]any{
			"actor_id":    ctx.ActorID,
			"auth_method": ctx.AuthMethod,
			"action":      rule.Action,
			"decision":    result.Decision,
			"reason":      result.Reason,
			"risk":        rule.Risk,
		})
		if result.Decision != "ALLOW" {
			c.JSON(http.StatusForbidden, gin.H{"error": result.Reason, "authz": result})
			c.Abort()
			return
		}
		c.Set("auth_context", ctx)
		c.Next()
	}
}

func resolveRequestContext(rootDir string, c *gin.Context) (auth.RequestContext, bool, error) {
	if strings.TrimSpace(c.GetHeader("Authorization")) != "" {
		return auth.ResolveBearer(rootDir, c.GetHeader("Authorization"))
	}
	if strings.TrimSpace(c.GetHeader("X-Moyuan-Session")) != "" {
		return auth.ResolveSession(rootDir, c.GetHeader("X-Moyuan-Session"))
	}
	return auth.ResolveLocalOwner(rootDir)
}

func protectedAuthzRule(method string, fullPath string, rawPath string) (authzRule, bool) {
	if method != http.MethodPost {
		return authzRule{}, false
	}
	path := fullPath
	if path == "" {
		path = rawPath
	}
	switch path {
	case "/v1/projects/:project_id/epics/:epic_id/batches/plan":
		return authzRule{Action: "batch.plan", Risk: "normal", Scopes: []string{"project:read"}}, true
	case "/v1/projects/:project_id/batches/:batch_id/run":
		return authzRule{Action: "batch.run", Risk: "high", Scopes: []string{"run:write"}}, true
	case "/v1/projects/:project_id/batches/:batch_id/merge-queue":
		return authzRule{Action: "merge.queue", Risk: "normal", Scopes: []string{"review:write"}}, true
	case "/v1/projects/:project_id/merge-queues/:queue_id/integration-preview":
		return authzRule{Action: "merge.integration_preview", Risk: "normal", Scopes: []string{"review:write", "git:read"}}, true
	case "/v1/projects/:project_id/integration-previews/:preview_id/apply":
		return authzRule{Action: "merge.integration_apply", Risk: "high", Scopes: []string{"review:write", "git:write"}}, true
	case "/v1/projects/:project_id/integration-applies/:apply_id/release-batch":
		return authzRule{Action: "release.batch.plan", Risk: "normal", Scopes: []string{"release:write"}}, true
	case "/v1/projects/:project_id/release-batches/:batch_id/candidate":
		return authzRule{Action: "release.candidate.plan", Risk: "normal", Scopes: []string{"release:write"}}, true
	case "/v1/projects/:project_id/release-candidates/:candidate_id/apply":
		return authzRule{Action: "release.candidate.apply", Risk: "high", Scopes: []string{"release:write", "git:write"}}, true
	case "/v1/projects/:project_id/release-candidates/:candidate_id/provider-preview":
		return authzRule{Action: "release.candidate.provider_preview", Risk: "normal", Scopes: []string{"release:write"}}, true
	case "/v1/projects/:project_id/release-candidates/:candidate_id/provider-publish":
		return authzRule{Action: "release.candidate.provider_publish", Risk: "high", Scopes: []string{"release:write"}}, true
	case "/v1/projects/:project_id/release-candidates/:candidate_id/pr-mr-plan":
		return authzRule{Action: "release.candidate.pr_mr_plan", Risk: "normal", Scopes: []string{"release:write", "git:read"}}, true
	case "/v1/projects/:project_id/release-candidates/:candidate_id/deployment-plan":
		return authzRule{Action: "release.candidate.deployment_plan", Risk: "normal", Scopes: []string{"deploy:write"}}, true
	case "/v1/projects/:project_id/release-candidates/:candidate_id/deployment-execution":
		return authzRule{Action: "release.candidate.deployment_execute", Risk: "critical", Scopes: []string{"deploy:execute"}}, true
	case "/v1/projects/:project_id/providers/ops/refresh":
		return authzRule{Action: "provider.refresh", Risk: "high", Scopes: []string{"provider:write"}}, true
	case "/v1/projects/:project_id/control-loop/run":
		return authzRule{Action: "control_loop.run", Risk: "high", Scopes: []string{"control:write"}}, true
	case "/v1/projects/:project_id/repair/operation-candidates/:candidate_id/review":
		return authzRule{Action: "repair.candidate.review", Risk: "high", Scopes: []string{"repair:write"}}, true
	case "/v1/projects/:project_id/repair/deployment-risk-handoffs":
		return authzRule{Action: "repair.deployment_risk_handoff", Risk: "normal", Scopes: []string{"repair:write"}}, true
	case "/v1/projects/:project_id/repair/deployment-risk-handoffs/:handoff_id/review":
		return authzRule{Action: "repair.deployment_risk_review", Risk: "high", Scopes: []string{"repair:write", "review:write"}}, true
	case "/v1/projects/:project_id/approvals/:approval_id/decide":
		return authzRule{Action: "approval.decide", Risk: "high", Scopes: []string{"approval:decide"}}, true
	case "/v1/projects/:project_id/auth/sessions":
		return authzRule{Action: "auth.session.create", Risk: "high", Scopes: []string{"auth:write"}}, true
	case "/v1/projects/:project_id/auth/api-tokens":
		return authzRule{Action: "auth.token.create", Risk: "critical", Scopes: []string{"auth:write"}}, true
	case "/v1/projects/:project_id/auth/service-accounts":
		return authzRule{Action: "auth.service_account.upsert", Risk: "high", Scopes: []string{"auth:write"}}, true
	case "/v1/projects/:project_id/auth/sessions/:session_id/revoke":
		return authzRule{Action: "auth.session.revoke", Risk: "high", Scopes: []string{"auth:write"}}, true
	case "/v1/projects/:project_id/auth/api-tokens/:token_id/revoke":
		return authzRule{Action: "auth.token.revoke", Risk: "high", Scopes: []string{"auth:write"}}, true
	case "/v1/projects/:project_id/deployments/:deployment_id/execute":
		return authzRule{Action: "deployment.execute", Risk: "critical", Scopes: []string{"deploy:execute"}}, true
	case "/v1/projects/:project_id/deployment-executions/:execution_id/rollback":
		return authzRule{Action: "deployment.rollback.execute", Risk: "critical", Scopes: []string{"deploy:execute"}}, true
	case "/v1/projects/:project_id/deployment-monitor-summary":
		return authzRule{Action: "deployment.monitor.summary", Risk: "normal", Scopes: []string{"deploy:write"}}, true
	case "/v1/projects/:project_id/deployment-rehearsals":
		return authzRule{Action: "deployment.rehearsal", Risk: "normal", Scopes: []string{"deploy:write"}}, true
	case "/v1/projects/:project_id/deployment-rehearsal-scheduler-runs":
		return authzRule{Action: "deployment.rehearsal.scheduler", Risk: "normal", Scopes: []string{"deploy:write"}}, true
	case "/v1/projects/:project_id/release-admissions":
		return authzRule{Action: "release.admission", Risk: "normal", Scopes: []string{"release:write", "deploy:write"}}, true
	case "/v1/projects/:project_id/visuals/assets/:asset_id/render":
		return authzRule{Action: "visual.render", Risk: "high", Scopes: []string{"visual:render"}}, true
	case "/v1/projects/:project_id/resources/:resource_id/renew":
		return authzRule{Action: "resource.renew", Risk: "high", Scopes: []string{"resource:write"}}, true
	case "/v1/projects/:project_id/resources/:resource_id/retire":
		return authzRule{Action: "resource.retire", Risk: "high", Scopes: []string{"resource:write"}}, true
	case "/v1/projects/:project_id/git-provider-plans/:plan_id/sync":
		return authzRule{Action: "git.provider.sync", Risk: "high", Scopes: []string{"git:write"}}, true
	case "/v1/projects/:project_id/git-provider-plans/:plan_id/create":
		return authzRule{Action: "git.provider.create", Risk: "high", Scopes: []string{"git:write"}}, true
	case "/v1/projects/:project_id/releases/:release_id/provider-publish":
		return authzRule{Action: "release.provider.publish", Risk: "high", Scopes: []string{"release:write"}}, true
	default:
		return protectedAuthzRuleByRawPath(method, rawPath)
	}
}

func protectedAuthzRuleByRawPath(method string, rawPath string) (authzRule, bool) {
	if method != http.MethodPost {
		return authzRule{}, false
	}
	switch {
	case strings.Contains(rawPath, "/epics/") && strings.HasSuffix(rawPath, "/batches/plan"):
		return authzRule{Action: "batch.plan", Risk: "normal", Scopes: []string{"project:read"}}, true
	case strings.Contains(rawPath, "/batches/") && strings.HasSuffix(rawPath, "/run"):
		return authzRule{Action: "batch.run", Risk: "high", Scopes: []string{"run:write"}}, true
	case strings.Contains(rawPath, "/batches/") && strings.HasSuffix(rawPath, "/merge-queue"):
		return authzRule{Action: "merge.queue", Risk: "normal", Scopes: []string{"review:write"}}, true
	case strings.Contains(rawPath, "/merge-queues/") && strings.HasSuffix(rawPath, "/integration-preview"):
		return authzRule{Action: "merge.integration_preview", Risk: "normal", Scopes: []string{"review:write", "git:read"}}, true
	case strings.Contains(rawPath, "/integration-previews/") && strings.HasSuffix(rawPath, "/apply"):
		return authzRule{Action: "merge.integration_apply", Risk: "high", Scopes: []string{"review:write", "git:write"}}, true
	case strings.Contains(rawPath, "/integration-applies/") && strings.HasSuffix(rawPath, "/release-batch"):
		return authzRule{Action: "release.batch.plan", Risk: "normal", Scopes: []string{"release:write"}}, true
	case strings.Contains(rawPath, "/release-batches/") && strings.HasSuffix(rawPath, "/candidate"):
		return authzRule{Action: "release.candidate.plan", Risk: "normal", Scopes: []string{"release:write"}}, true
	case strings.Contains(rawPath, "/release-candidates/") && strings.HasSuffix(rawPath, "/apply"):
		return authzRule{Action: "release.candidate.apply", Risk: "high", Scopes: []string{"release:write", "git:write"}}, true
	case strings.Contains(rawPath, "/release-candidates/") && strings.HasSuffix(rawPath, "/provider-preview"):
		return authzRule{Action: "release.candidate.provider_preview", Risk: "normal", Scopes: []string{"release:write"}}, true
	case strings.Contains(rawPath, "/release-candidates/") && strings.HasSuffix(rawPath, "/provider-publish"):
		return authzRule{Action: "release.candidate.provider_publish", Risk: "high", Scopes: []string{"release:write"}}, true
	case strings.Contains(rawPath, "/release-candidates/") && strings.HasSuffix(rawPath, "/pr-mr-plan"):
		return authzRule{Action: "release.candidate.pr_mr_plan", Risk: "normal", Scopes: []string{"release:write", "git:read"}}, true
	case strings.Contains(rawPath, "/release-candidates/") && strings.HasSuffix(rawPath, "/deployment-plan"):
		return authzRule{Action: "release.candidate.deployment_plan", Risk: "normal", Scopes: []string{"deploy:write"}}, true
	case strings.Contains(rawPath, "/release-candidates/") && strings.HasSuffix(rawPath, "/deployment-execution"):
		return authzRule{Action: "release.candidate.deployment_execute", Risk: "critical", Scopes: []string{"deploy:execute"}}, true
	case strings.Contains(rawPath, "/providers/ops/refresh"):
		return authzRule{Action: "provider.refresh", Risk: "high", Scopes: []string{"provider:write"}}, true
	case strings.Contains(rawPath, "/control-loop/run"):
		return authzRule{Action: "control_loop.run", Risk: "high", Scopes: []string{"control:write"}}, true
	case strings.Contains(rawPath, "/repair/operation-candidates/") && strings.HasSuffix(rawPath, "/review"):
		return authzRule{Action: "repair.candidate.review", Risk: "high", Scopes: []string{"repair:write"}}, true
	case strings.HasSuffix(rawPath, "/repair/deployment-risk-handoffs"):
		return authzRule{Action: "repair.deployment_risk_handoff", Risk: "normal", Scopes: []string{"repair:write"}}, true
	case strings.Contains(rawPath, "/repair/deployment-risk-handoffs/") && strings.HasSuffix(rawPath, "/review"):
		return authzRule{Action: "repair.deployment_risk_review", Risk: "high", Scopes: []string{"repair:write", "review:write"}}, true
	case strings.Contains(rawPath, "/approvals/") && strings.HasSuffix(rawPath, "/decide"):
		return authzRule{Action: "approval.decide", Risk: "high", Scopes: []string{"approval:decide"}}, true
	case strings.HasSuffix(rawPath, "/auth/sessions"):
		return authzRule{Action: "auth.session.create", Risk: "high", Scopes: []string{"auth:write"}}, true
	case strings.HasSuffix(rawPath, "/auth/api-tokens"):
		return authzRule{Action: "auth.token.create", Risk: "critical", Scopes: []string{"auth:write"}}, true
	case strings.HasSuffix(rawPath, "/auth/service-accounts"):
		return authzRule{Action: "auth.service_account.upsert", Risk: "high", Scopes: []string{"auth:write"}}, true
	case strings.Contains(rawPath, "/auth/sessions/") && strings.HasSuffix(rawPath, "/revoke"):
		return authzRule{Action: "auth.session.revoke", Risk: "high", Scopes: []string{"auth:write"}}, true
	case strings.Contains(rawPath, "/auth/api-tokens/") && strings.HasSuffix(rawPath, "/revoke"):
		return authzRule{Action: "auth.token.revoke", Risk: "high", Scopes: []string{"auth:write"}}, true
	case strings.Contains(rawPath, "/deployments/") && strings.HasSuffix(rawPath, "/execute"):
		return authzRule{Action: "deployment.execute", Risk: "critical", Scopes: []string{"deploy:execute"}}, true
	case strings.Contains(rawPath, "/deployment-executions/") && strings.HasSuffix(rawPath, "/rollback"):
		return authzRule{Action: "deployment.rollback.execute", Risk: "critical", Scopes: []string{"deploy:execute"}}, true
	case strings.HasSuffix(rawPath, "/deployment-monitor-summary"):
		return authzRule{Action: "deployment.monitor.summary", Risk: "normal", Scopes: []string{"deploy:write"}}, true
	case strings.HasSuffix(rawPath, "/deployment-rehearsals"):
		return authzRule{Action: "deployment.rehearsal", Risk: "normal", Scopes: []string{"deploy:write"}}, true
	case strings.HasSuffix(rawPath, "/deployment-rehearsal-scheduler-runs"):
		return authzRule{Action: "deployment.rehearsal.scheduler", Risk: "normal", Scopes: []string{"deploy:write"}}, true
	case strings.HasSuffix(rawPath, "/release-admissions"):
		return authzRule{Action: "release.admission", Risk: "normal", Scopes: []string{"release:write", "deploy:write"}}, true
	case strings.Contains(rawPath, "/visuals/assets/") && strings.HasSuffix(rawPath, "/render"):
		return authzRule{Action: "visual.render", Risk: "high", Scopes: []string{"visual:render"}}, true
	case strings.Contains(rawPath, "/resources/") && strings.HasSuffix(rawPath, "/renew"):
		return authzRule{Action: "resource.renew", Risk: "high", Scopes: []string{"resource:write"}}, true
	case strings.Contains(rawPath, "/resources/") && strings.HasSuffix(rawPath, "/retire"):
		return authzRule{Action: "resource.retire", Risk: "high", Scopes: []string{"resource:write"}}, true
	case strings.Contains(rawPath, "/git-provider-plans/") && strings.HasSuffix(rawPath, "/sync"):
		return authzRule{Action: "git.provider.sync", Risk: "high", Scopes: []string{"git:write"}}, true
	case strings.Contains(rawPath, "/git-provider-plans/") && strings.HasSuffix(rawPath, "/create"):
		return authzRule{Action: "git.provider.create", Risk: "high", Scopes: []string{"git:write"}}, true
	case strings.Contains(rawPath, "/releases/") && strings.HasSuffix(rawPath, "/provider-publish"):
		return authzRule{Action: "release.provider.publish", Risk: "high", Scopes: []string{"release:write"}}, true
	default:
		return authzRule{}, false
	}
}
