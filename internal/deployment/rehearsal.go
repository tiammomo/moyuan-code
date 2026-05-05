package deployment

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"moyuan-code/internal/evidence"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/release"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type RehearsalOptions struct {
	CandidateID  string `json:"candidate_id,omitempty"`
	DeploymentID string `json:"deployment_id,omitempty"`
	ExecutionID  string `json:"execution_id,omitempty"`
	Environment  string `json:"environment,omitempty"`
	MonitorLimit int    `json:"monitor_limit,omitempty"`
}

type DeploymentRehearsal struct {
	ID                  string                  `json:"id"`
	CandidateID         string                  `json:"candidate_id,omitempty"`
	DeploymentID        string                  `json:"deployment_id,omitempty"`
	ExecutionID         string                  `json:"execution_id,omitempty"`
	ReleaseID           string                  `json:"release_id,omitempty"`
	Environment         string                  `json:"environment,omitempty"`
	Status              string                  `json:"status"`
	Decision            string                  `json:"decision"`
	Reasons             []string                `json:"reasons"`
	Timeline            []RehearsalTimelineItem `json:"timeline"`
	MonitorSummaryID    string                  `json:"monitor_summary_id,omitempty"`
	MonitorStatus       string                  `json:"monitor_status,omitempty"`
	MonitorDecision     string                  `json:"monitor_decision,omitempty"`
	RollbackExecutionID string                  `json:"rollback_execution_id,omitempty"`
	RollbackStatus      string                  `json:"rollback_status,omitempty"`
	RollbackDecision    string                  `json:"rollback_decision,omitempty"`
	EvidenceIDs         []string                `json:"evidence_ids,omitempty"`
	CreatedAt           string                  `json:"created_at"`
	FinishedAt          string                  `json:"finished_at,omitempty"`
}

type RehearsalTimelineItem struct {
	Type        string   `json:"type"`
	ID          string   `json:"id"`
	Status      string   `json:"status"`
	Decision    string   `json:"decision"`
	Detail      string   `json:"detail,omitempty"`
	EvidenceIDs []string `json:"evidence_ids,omitempty"`
	CreatedAt   string   `json:"created_at,omitempty"`
}

func BuildRehearsal(ctx context.Context, rootDir string, options RehearsalOptions) (DeploymentRehearsal, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return DeploymentRehearsal{}, err
	}
	options.CandidateID = strings.TrimSpace(options.CandidateID)
	options.DeploymentID = strings.TrimSpace(options.DeploymentID)
	options.ExecutionID = strings.TrimSpace(options.ExecutionID)
	options.Environment = normalizeToken(options.Environment)
	if options.MonitorLimit <= 0 {
		options.MonitorLimit = 10
	}
	now := time.Now().UTC()
	rehearsal := DeploymentRehearsal{
		ID:        "deployment-rehearsal-" + textutil.Slugify(rehearsalIDSeed(options)) + "-" + now.Format("20060102150405") + "-" + strconv.FormatInt(now.UnixNano()%1_000_000_000, 10),
		Status:    "blocked",
		Decision:  "DEPLOYMENT_REHEARSAL_BLOCKED",
		Reasons:   []string{},
		Timeline:  []RehearsalTimelineItem{},
		CreatedAt: now.Format(time.RFC3339Nano),
	}

	var plan Plan
	var planFound bool
	if options.CandidateID != "" {
		rehearsal.CandidateID = options.CandidateID
		candidate, found, err := release.LoadCandidate(rootDir, options.CandidateID)
		if err != nil {
			return DeploymentRehearsal{}, err
		}
		if !found {
			rehearsal.Reasons = append(rehearsal.Reasons, "release_candidate_not_found")
			return finishRehearsal(rootDir, rehearsal)
		}
		rehearsal.Timeline = append(rehearsal.Timeline, RehearsalTimelineItem{
			Type:      "release_candidate",
			ID:        candidate.ID,
			Status:    candidate.Status,
			Decision:  candidate.Decision,
			Detail:    candidate.Version,
			CreatedAt: candidate.CreatedAt,
		})
		if options.DeploymentID == "" {
			latestPlan, found, err := LatestPlanForCandidate(rootDir, options.CandidateID, options.Environment)
			if err != nil {
				return DeploymentRehearsal{}, err
			}
			if found {
				plan = latestPlan
				planFound = true
				options.DeploymentID = latestPlan.ID
			}
		}
	}

	if options.DeploymentID != "" && !planFound {
		loaded, found, err := Load(rootDir, options.DeploymentID)
		if err != nil {
			return DeploymentRehearsal{}, err
		}
		if !found {
			rehearsal.DeploymentID = options.DeploymentID
			rehearsal.Reasons = append(rehearsal.Reasons, "deployment_not_found")
			return finishRehearsal(rootDir, rehearsal)
		}
		plan = loaded
		planFound = true
	}

	if planFound {
		rehearsal.DeploymentID = plan.ID
		rehearsal.ReleaseID = plan.ReleaseID
		rehearsal.Environment = plan.Environment
		rehearsal.Timeline = append(rehearsal.Timeline, RehearsalTimelineItem{
			Type:      "deployment_plan",
			ID:        plan.ID,
			Status:    plan.Status,
			Decision:  plan.Decision,
			Detail:    plan.Environment,
			CreatedAt: plan.CreatedAt,
		})
		if rehearsal.CandidateID == "" && strings.HasPrefix(plan.ReleaseID, "release-candidate-") {
			rehearsal.CandidateID = plan.ReleaseID
		}
		if options.Environment == "" {
			options.Environment = plan.Environment
		}
	}

	execution, executionFound, err := resolveRehearsalExecution(rootDir, options)
	if err != nil {
		return DeploymentRehearsal{}, err
	}
	if !executionFound {
		if rehearsal.DeploymentID == "" && options.ExecutionID == "" {
			rehearsal.Reasons = append(rehearsal.Reasons, "deployment_or_execution_required")
		} else {
			rehearsal.Reasons = append(rehearsal.Reasons, "deployment_execution_missing")
		}
		return finishRehearsal(rootDir, rehearsal)
	}
	if !planFound && execution.DeploymentID != "" {
		loaded, found, err := Load(rootDir, execution.DeploymentID)
		if err != nil {
			return DeploymentRehearsal{}, err
		}
		if found {
			plan = loaded
			planFound = true
			rehearsal.Timeline = append(rehearsal.Timeline, RehearsalTimelineItem{
				Type:      "deployment_plan",
				ID:        plan.ID,
				Status:    plan.Status,
				Decision:  plan.Decision,
				Detail:    plan.Environment,
				CreatedAt: plan.CreatedAt,
			})
		} else {
			rehearsal.Reasons = append(rehearsal.Reasons, "deployment_plan_missing")
		}
	}
	if rehearsal.DeploymentID == "" {
		rehearsal.DeploymentID = execution.DeploymentID
	}
	if rehearsal.ReleaseID == "" {
		rehearsal.ReleaseID = execution.ReleaseID
	}
	if rehearsal.Environment == "" {
		rehearsal.Environment = execution.Environment
	}
	rehearsal.ExecutionID = execution.ID
	executionEvidence, err := evidenceIDs(rootDir, "deployment_execution", execution.ID)
	if err != nil {
		return DeploymentRehearsal{}, err
	}
	rehearsal.EvidenceIDs = appendUnique(rehearsal.EvidenceIDs, executionEvidence...)
	rehearsal.Timeline = append(rehearsal.Timeline, RehearsalTimelineItem{
		Type:        "deployment_execution",
		ID:          execution.ID,
		Status:      execution.Status,
		Decision:    execution.Decision,
		Detail:      execution.Mode,
		EvidenceIDs: executionEvidence,
		CreatedAt:   execution.StartedAt,
	})

	history, found, err := LoadPostDeploymentHistory(rootDir, execution.ID)
	if err != nil {
		return DeploymentRehearsal{}, err
	}
	if found {
		rehearsal.EvidenceIDs = appendUnique(rehearsal.EvidenceIDs, history.EvidenceIDs...)
		rehearsal.Timeline = append(rehearsal.Timeline, RehearsalTimelineItem{
			Type:        "post_deployment_history",
			ID:          history.ID,
			Status:      history.Status,
			Decision:    history.Decision,
			Detail:      history.FailureClass,
			EvidenceIDs: history.EvidenceIDs,
			CreatedAt:   history.CreatedAt,
		})
	}

	monitor, err := BuildMonitorSummary(rootDir, MonitorSummaryOptions{Environment: rehearsal.Environment, Limit: options.MonitorLimit})
	if err != nil {
		return DeploymentRehearsal{}, err
	}
	rehearsal.MonitorSummaryID = monitor.ID
	rehearsal.MonitorStatus = monitor.Status
	rehearsal.MonitorDecision = monitor.Decision
	rehearsal.EvidenceIDs = appendUnique(rehearsal.EvidenceIDs, monitor.EvidenceIDs...)
	rehearsal.Timeline = append(rehearsal.Timeline, RehearsalTimelineItem{
		Type:        "monitor_summary",
		ID:          monitor.ID,
		Status:      monitor.Status,
		Decision:    monitor.Decision,
		Detail:      "history_count:" + strconv.Itoa(monitor.HistoryCount),
		EvidenceIDs: monitor.EvidenceIDs,
		CreatedAt:   monitor.CreatedAt,
	})

	if execution.RollbackSuggestion.Required {
		rollback, err := latestRollbackPreviewForExecution(rootDir, execution.ID)
		if err != nil {
			return DeploymentRehearsal{}, err
		}
		if rollback.ID == "" {
			rollback, err = ExecuteRollback(ctx, rootDir, RollbackExecuteOptions{ExecutionID: execution.ID, Mode: "preview"})
			if err != nil {
				return DeploymentRehearsal{}, err
			}
		}
		rehearsal.RollbackExecutionID = rollback.ID
		rehearsal.RollbackStatus = rollback.Status
		rehearsal.RollbackDecision = rollback.Decision
		rollbackEvidence, err := evidenceIDs(rootDir, "deployment_rollback_execution", rollback.ID)
		if err != nil {
			return DeploymentRehearsal{}, err
		}
		rehearsal.EvidenceIDs = appendUnique(rehearsal.EvidenceIDs, rollbackEvidence...)
		rehearsal.Timeline = append(rehearsal.Timeline, RehearsalTimelineItem{
			Type:        "rollback_preview",
			ID:          rollback.ID,
			Status:      rollback.Status,
			Decision:    rollback.Decision,
			Detail:      rollback.Mode,
			EvidenceIDs: rollbackEvidence,
			CreatedAt:   rollback.StartedAt,
		})
	}

	rehearsal.Status, rehearsal.Decision, rehearsal.Reasons = rehearsalDecision(execution, monitor, rehearsal.Reasons)
	return finishRehearsal(rootDir, rehearsal)
}

func LoadRehearsal(rootDir string, id string) (DeploymentRehearsal, bool, error) {
	var rehearsal DeploymentRehearsal
	found, err := fsutil.ReadJSON(rehearsalPath(rootDir, id), &rehearsal)
	return rehearsal, found, err
}

func ListRehearsals(rootDir string, limit int) ([]DeploymentRehearsal, error) {
	if err := fsutil.EnsureDir(rehearsalDir(rootDir)); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(rehearsalDir(rootDir))
	if err != nil {
		return nil, err
	}
	rehearsals := []DeploymentRehearsal{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var rehearsal DeploymentRehearsal
		found, err := fsutil.ReadJSON(filepath.Join(rehearsalDir(rootDir), entry.Name()), &rehearsal)
		if err != nil {
			return nil, err
		}
		if found && rehearsal.ID != "" {
			rehearsals = append(rehearsals, rehearsal)
		}
	}
	sort.SliceStable(rehearsals, func(i, j int) bool {
		return rehearsals[i].CreatedAt > rehearsals[j].CreatedAt
	})
	if limit <= 0 {
		limit = 20
	}
	if len(rehearsals) > limit {
		return rehearsals[:limit], nil
	}
	return rehearsals, nil
}

func resolveRehearsalExecution(rootDir string, options RehearsalOptions) (Execution, bool, error) {
	if options.ExecutionID != "" {
		execution, found, err := LoadExecution(rootDir, options.ExecutionID)
		if err != nil || !found {
			return Execution{}, found, err
		}
		if options.DeploymentID != "" && execution.DeploymentID != options.DeploymentID {
			return Execution{}, false, nil
		}
		return execution, true, nil
	}
	executions, err := ListExecutions(rootDir, 100)
	if err != nil {
		return Execution{}, false, err
	}
	for _, execution := range executions {
		if options.DeploymentID != "" && execution.DeploymentID != options.DeploymentID {
			continue
		}
		if options.Environment != "" && execution.Environment != options.Environment {
			continue
		}
		return execution, true, nil
	}
	return Execution{}, false, nil
}

func latestRollbackPreviewForExecution(rootDir string, executionID string) (RollbackExecution, error) {
	rollbacks, err := ListRollbackExecutions(rootDir, 100)
	if err != nil {
		return RollbackExecution{}, err
	}
	for _, rollback := range rollbacks {
		if rollback.ExecutionID == executionID && rollback.Mode == "preview" {
			return rollback, nil
		}
	}
	return RollbackExecution{}, nil
}

func rehearsalDecision(execution Execution, monitor MonitorSummary, reasons []string) (string, string, []string) {
	reasons = append(reasons, "deployment_execution:"+execution.Status)
	reasons = append(reasons, "monitor_summary:"+monitor.Status)
	if execution.Status == "blocked" {
		return "blocked", "DEPLOYMENT_REHEARSAL_BLOCKED", append(reasons, "deployment_execution_blocked")
	}
	if execution.Status == "failed" {
		return "attention_required", "DEPLOYMENT_REHEARSAL_ATTENTION_REQUIRED", append(reasons, "deployment_execution_failed")
	}
	if execution.RollbackSuggestion.Required {
		return "attention_required", "DEPLOYMENT_REHEARSAL_ATTENTION_REQUIRED", append(reasons, "rollback_required")
	}
	if monitor.Status == "critical" || monitor.Status == "attention_required" || monitor.Status == "unknown" {
		return "attention_required", "DEPLOYMENT_REHEARSAL_ATTENTION_REQUIRED", append(reasons, "monitor_attention_required")
	}
	return "completed", "DEPLOYMENT_REHEARSAL_READY", append(reasons, "rehearsal_ready")
}

func finishRehearsal(rootDir string, rehearsal DeploymentRehearsal) (DeploymentRehearsal, error) {
	rehearsal.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := fsutil.EnsureDir(rehearsalDir(rootDir)); err != nil {
		return DeploymentRehearsal{}, err
	}
	if err := fsutil.WriteJSON(rehearsalPath(rootDir, rehearsal.ID), rehearsal); err != nil {
		return DeploymentRehearsal{}, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "rehearsals.jsonl"), rehearsal); err != nil {
		return DeploymentRehearsal{}, err
	}
	_ = logging.Log(rootDir, "release", "deployment.rehearsal.created", map[string]any{
		"rehearsal_id":  rehearsal.ID,
		"candidate_id":  rehearsal.CandidateID,
		"deployment_id": rehearsal.DeploymentID,
		"execution_id":  rehearsal.ExecutionID,
		"decision":      rehearsal.Decision,
		"status":        rehearsal.Status,
		"environment":   rehearsal.Environment,
	})
	if _, err := evidence.Add(rootDir, evidence.AddOptions{
		ParentType:  "deployment_rehearsal",
		ParentID:    rehearsal.ID,
		SubjectType: "deployment",
		SubjectID:   rehearsalSubjectID(rehearsal),
		Operation:   "deployment.rehearsal",
		Status:      rehearsal.Status,
		Decision:    rehearsal.Decision,
		Reasons:     rehearsal.Reasons,
		Source:      "deployment",
		Artifacts: []evidence.ArtifactRef{{
			Kind: "deployment_rehearsal",
			ID:   rehearsal.ID,
			Path: filepath.ToSlash(filepath.Join(".moyuan", "lifecycle", "deployments", "rehearsals", rehearsal.ID+".json")),
		}},
	}); err != nil {
		return DeploymentRehearsal{}, err
	}
	return rehearsal, nil
}

func evidenceIDs(rootDir string, parentType string, parentID string) ([]string, error) {
	records, err := evidence.List(rootDir, evidence.ListOptions{ParentType: parentType, ParentID: parentID, Limit: 10})
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for _, record := range records {
		ids = append(ids, record.ID)
	}
	return ids, nil
}

func appendUnique(values []string, next ...string) []string {
	seen := map[string]bool{}
	for _, value := range values {
		seen[value] = true
	}
	for _, value := range next {
		if value == "" || seen[value] {
			continue
		}
		values = append(values, value)
		seen[value] = true
	}
	return values
}

func rehearsalIDSeed(options RehearsalOptions) string {
	for _, value := range []string{options.ExecutionID, options.DeploymentID, options.CandidateID, options.Environment} {
		if value != "" {
			return value
		}
	}
	return "manual"
}

func rehearsalSubjectID(rehearsal DeploymentRehearsal) string {
	for _, value := range []string{rehearsal.DeploymentID, rehearsal.ExecutionID, rehearsal.CandidateID, rehearsal.Environment} {
		if value != "" {
			return value
		}
	}
	return "unknown"
}

func rehearsalDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "rehearsals")
}

func rehearsalPath(rootDir string, id string) string {
	return filepath.Join(rehearsalDir(rootDir), id+".json")
}
