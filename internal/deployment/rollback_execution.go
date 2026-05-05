package deployment

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/approvals"
	"moyuan-code/internal/evidence"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type RollbackExecuteOptions struct {
	ExecutionID string   `json:"execution_id"`
	Mode        string   `json:"mode,omitempty"`
	Approved    bool     `json:"approved,omitempty"`
	ApprovalID  string   `json:"approval_id,omitempty"`
	Commands    []string `json:"commands,omitempty"`
}

type RollbackExecution struct {
	ID               string           `json:"id"`
	ExecutionID      string           `json:"execution_id"`
	DeploymentID     string           `json:"deployment_id,omitempty"`
	ReleaseID        string           `json:"release_id,omitempty"`
	Environment      string           `json:"environment,omitempty"`
	Mode             string           `json:"mode"`
	Status           string           `json:"status"`
	Decision         string           `json:"decision"`
	Reasons          []string         `json:"reasons"`
	Steps            []ExecutionStep  `json:"steps"`
	Runbook          *RollbackRunbook `json:"runbook,omitempty"`
	ApprovalID       string           `json:"approval_id,omitempty"`
	ApprovalConsumed bool             `json:"approval_consumed"`
	ExecutionEnabled bool             `json:"execution_enabled"`
	StartedAt        string           `json:"started_at"`
	FinishedAt       string           `json:"finished_at,omitempty"`
}

func ExecuteRollback(ctx context.Context, rootDir string, options RollbackExecuteOptions) (RollbackExecution, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return RollbackExecution{}, err
	}
	options.ExecutionID = strings.TrimSpace(options.ExecutionID)
	options.Mode = normalizeToken(options.Mode)
	options.ApprovalID = strings.TrimSpace(options.ApprovalID)
	options.Commands = normalizeCommands(options.Commands)
	if options.ExecutionID == "" {
		return RollbackExecution{}, errors.New("execution_id_required")
	}
	if options.Mode == "" {
		options.Mode = "preview"
	}
	now := time.Now().UTC()
	rollback := RollbackExecution{
		ID:          "rollback-exec-" + textutil.Slugify(options.ExecutionID+"-"+options.Mode) + "-" + now.Format("20060102150405"),
		ExecutionID: options.ExecutionID,
		Mode:        options.Mode,
		Status:      "blocked",
		Decision:    "ROLLBACK_EXECUTION_BLOCKED",
		Reasons:     []string{},
		Steps:       []ExecutionStep{},
		StartedAt:   now.Format(time.RFC3339Nano),
	}
	execution, found, err := LoadExecution(rootDir, options.ExecutionID)
	if err != nil {
		return RollbackExecution{}, err
	}
	if !found {
		rollback.Reasons = append(rollback.Reasons, "deployment_execution_not_found")
		return finishRollbackExecution(rootDir, rollback)
	}
	rollback.DeploymentID = execution.DeploymentID
	rollback.ReleaseID = execution.ReleaseID
	rollback.Environment = execution.Environment
	if !execution.RollbackSuggestion.Required {
		rollback.Reasons = append(rollback.Reasons, "rollback_not_required")
		return finishRollbackExecution(rootDir, rollback)
	}
	if execution.RollbackSuggestion.Runbook == nil {
		rollback.Reasons = append(rollback.Reasons, "rollback_runbook_missing")
		return finishRollbackExecution(rootDir, rollback)
	}
	rollback.Runbook = execution.RollbackSuggestion.Runbook
	switch options.Mode {
	case "preview":
		rollback.Status = "completed"
		rollback.Decision = "ROLLBACK_PREVIEW_READY"
		rollback.Reasons = append(rollback.Reasons, "rollback_preview_only")
		rollback.Steps = rollbackPreviewSteps(rollback.Runbook)
	case "local_shell":
		executeRollbackLocalShell(ctx, rootDir, execution, options, &rollback)
	default:
		rollback.Reasons = append(rollback.Reasons, "rollback_mode_not_allowed:"+options.Mode)
	}
	return finishRollbackExecution(rootDir, rollback)
}

func executeRollbackLocalShell(ctx context.Context, rootDir string, execution Execution, options RollbackExecuteOptions, rollback *RollbackExecution) {
	approvalID := ""
	if !options.Approved {
		approval, err := requestRollbackApproval(rootDir, execution, options.Mode)
		if err != nil {
			rollback.Reasons = append(rollback.Reasons, err.Error())
			return
		}
		rollback.ApprovalID = approval.ID
		rollback.Decision = "ROLLBACK_EXECUTION_APPROVAL_REQUIRED"
		rollback.Reasons = append(rollback.Reasons, "approval_required_before_rollback_execution")
		return
	}
	if options.ApprovalID == "" {
		rollback.Decision = "ROLLBACK_EXECUTION_APPROVAL_REQUIRED"
		rollback.Reasons = append(rollback.Reasons, "approval_id_required_before_rollback_execution")
		return
	}
	approval, found, err := approvals.VerifyApproved(rootDir, options.ApprovalID, rollbackApprovalScope(execution, options.Mode))
	rollback.ApprovalID = options.ApprovalID
	if err != nil {
		rollback.Decision = "ROLLBACK_EXECUTION_APPROVAL_REQUIRED"
		rollback.Reasons = append(rollback.Reasons, err.Error())
		return
	}
	if !found {
		rollback.Decision = "ROLLBACK_EXECUTION_APPROVAL_REQUIRED"
		rollback.Reasons = append(rollback.Reasons, "approval_not_found")
		return
	}
	approvalID = approval.ID
	rollback.ApprovalID = approval.ID
	rollback.ExecutionEnabled = rollbackExecutionEnabled()
	if !rollback.ExecutionEnabled {
		rollback.Decision = "ROLLBACK_EXECUTION_PREVIEW_ONLY"
		rollback.Reasons = append(rollback.Reasons, "rollback_execution_not_enabled")
		rollback.Steps = rollbackPreviewSteps(rollback.Runbook)
		return
	}
	if len(options.Commands) == 0 {
		rollback.Reasons = append(rollback.Reasons, "commands_required")
		return
	}
	if !allSafeShellCommands(options.Commands) {
		steps, _, reasons := runLocalShell(ctx, rootDir, options.Commands)
		rollback.Steps = steps
		rollback.Reasons = append(rollback.Reasons, reasons...)
		return
	}
	if !consumeRollbackApproval(rootDir, execution, options.Mode, approvalID, rollback) {
		return
	}
	steps, ok, reasons := runLocalShell(ctx, rootDir, options.Commands)
	rollback.Steps = steps
	rollback.Reasons = append(rollback.Reasons, reasons...)
	if ok {
		rollback.Status = "completed"
		rollback.Decision = "ROLLBACK_EXECUTION_COMPLETED"
		return
	}
	rollback.Status = "failed"
	rollback.Decision = "ROLLBACK_EXECUTION_FAILED"
}

func requestRollbackApproval(rootDir string, execution Execution, mode string) (approvals.Record, error) {
	return approvals.Request(rootDir, approvals.RequestOptions{
		TargetType:  "deployment_rollback_execution",
		TargetID:    execution.ID,
		Action:      "deployment.rollback.execute." + mode,
		RiskLevel:   riskForExecution(execution.Environment),
		RequestedBy: "system",
		Reason:      "rollback execution requires approval",
		Metadata: map[string]any{
			"execution_id":  execution.ID,
			"deployment_id": execution.DeploymentID,
			"release_id":    execution.ReleaseID,
			"environment":   execution.Environment,
			"mode":          mode,
		},
	})
}

func rollbackApprovalScope(execution Execution, mode string) approvals.RequestOptions {
	return approvals.RequestOptions{
		TargetType: "deployment_rollback_execution",
		TargetID:   execution.ID,
		Action:     "deployment.rollback.execute." + mode,
	}
}

func consumeRollbackApproval(rootDir string, execution Execution, mode string, approvalID string, rollback *RollbackExecution) bool {
	consumed, found, err := approvals.ConsumeApproved(rootDir, approvalID, approvals.ConsumeOptions{
		TargetType: "deployment_rollback_execution",
		TargetID:   execution.ID,
		Action:     "deployment.rollback.execute." + mode,
		ConsumedBy: "rollback-executor",
		Reason:     "deployment rollback execution " + mode,
	})
	if err != nil {
		rollback.Decision = "ROLLBACK_EXECUTION_APPROVAL_REQUIRED"
		rollback.Reasons = append(rollback.Reasons, err.Error())
		return false
	}
	if !found {
		rollback.Decision = "ROLLBACK_EXECUTION_APPROVAL_REQUIRED"
		rollback.Reasons = append(rollback.Reasons, "approval_not_found")
		return false
	}
	rollback.ApprovalID = consumed.ID
	rollback.ApprovalConsumed = true
	rollback.Reasons = append(rollback.Reasons, "approval_consumed_before_rollback_execution")
	return true
}

func LoadRollbackExecution(rootDir string, id string) (RollbackExecution, bool, error) {
	var execution RollbackExecution
	found, err := fsutil.ReadJSON(rollbackExecutionPath(rootDir, id), &execution)
	return execution, found, err
}

func ListRollbackExecutions(rootDir string, limit int) ([]RollbackExecution, error) {
	if err := fsutil.EnsureDir(rollbackExecutionDir(rootDir)); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(rollbackExecutionDir(rootDir))
	if err != nil {
		return nil, err
	}
	executions := []RollbackExecution{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var execution RollbackExecution
		found, err := fsutil.ReadJSON(filepath.Join(rollbackExecutionDir(rootDir), entry.Name()), &execution)
		if err != nil {
			return nil, err
		}
		if found && execution.ID != "" {
			executions = append(executions, execution)
		}
	}
	sort.SliceStable(executions, func(i, j int) bool {
		return executions[i].StartedAt > executions[j].StartedAt
	})
	if limit > 0 && len(executions) > limit {
		return executions[:limit], nil
	}
	return executions, nil
}

func finishRollbackExecution(rootDir string, rollback RollbackExecution) (RollbackExecution, error) {
	rollback.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := fsutil.EnsureDir(rollbackExecutionDir(rootDir)); err != nil {
		return RollbackExecution{}, err
	}
	if err := fsutil.WriteJSON(rollbackExecutionPath(rootDir, rollback.ID), rollback); err != nil {
		return RollbackExecution{}, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "rollback-executions.jsonl"), rollback); err != nil {
		return RollbackExecution{}, err
	}
	_ = logging.Log(rootDir, "release", "deployment.rollback.execution.created", map[string]any{
		"rollback_execution_id": rollback.ID,
		"execution_id":          rollback.ExecutionID,
		"deployment_id":         rollback.DeploymentID,
		"decision":              rollback.Decision,
		"status":                rollback.Status,
		"environment":           rollback.Environment,
		"mode":                  rollback.Mode,
	})
	if _, err := evidence.Add(rootDir, evidence.AddOptions{
		ParentType:  "deployment_rollback_execution",
		ParentID:    rollback.ID,
		SubjectType: "deployment_execution",
		SubjectID:   rollback.ExecutionID,
		Operation:   "deployment.rollback.execute." + rollback.Mode,
		Status:      rollback.Status,
		Decision:    rollback.Decision,
		Reasons:     rollback.Reasons,
		Source:      "deployment",
		Artifacts: []evidence.ArtifactRef{{
			Kind: "deployment_rollback_execution",
			ID:   rollback.ID,
			Path: filepath.ToSlash(filepath.Join(".moyuan", "lifecycle", "deployments", "rollback-executions", rollback.ID+".json")),
		}},
	}); err != nil {
		return RollbackExecution{}, err
	}
	return rollback, nil
}

func rollbackPreviewSteps(runbook *RollbackRunbook) []ExecutionStep {
	steps := []ExecutionStep{}
	if runbook == nil {
		return steps
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, item := range runbook.Steps {
		steps = append(steps, ExecutionStep{
			Name:       item.Name,
			Status:     "planned",
			Command:    item.Action,
			Output:     item.Verification,
			Allowlist:  []string{"preview_only", "manual_or_approved_execution"},
			StartedAt:  now,
			FinishedAt: now,
		})
	}
	return steps
}

func rollbackExecutionDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "rollback-executions")
}

func rollbackExecutionPath(rootDir string, id string) string {
	return filepath.Join(rollbackExecutionDir(rootDir), id+".json")
}

func rollbackExecutionEnabled() bool {
	return os.Getenv("MOYUAN_ALLOW_ROLLBACK_EXECUTE") == "1"
}
