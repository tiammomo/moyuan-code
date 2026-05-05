package deployment

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"moyuan-code/internal/evidence"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type PostDeploymentVerificationOptions struct {
	ExecutionID  string `json:"execution_id,omitempty"`
	Environment  string `json:"environment,omitempty"`
	MonitorLimit int    `json:"monitor_limit,omitempty"`
}

type PostDeploymentVerification struct {
	ID                     string   `json:"id"`
	ExecutionID            string   `json:"execution_id,omitempty"`
	DeploymentID           string   `json:"deployment_id,omitempty"`
	ReleaseID              string   `json:"release_id,omitempty"`
	Environment            string   `json:"environment,omitempty"`
	Status                 string   `json:"status"`
	Decision               string   `json:"decision"`
	Reasons                []string `json:"reasons"`
	HistoryID              string   `json:"history_id,omitempty"`
	HistoryStatus          string   `json:"history_status,omitempty"`
	HistoryDecision        string   `json:"history_decision,omitempty"`
	MonitorSummaryID       string   `json:"monitor_summary_id,omitempty"`
	MonitorStatus          string   `json:"monitor_status,omitempty"`
	MonitorDecision        string   `json:"monitor_decision,omitempty"`
	SmokeStatus            string   `json:"smoke_status,omitempty"`
	SmokeDecision          string   `json:"smoke_decision,omitempty"`
	RollbackRequired       bool     `json:"rollback_required"`
	RollbackDecision       string   `json:"rollback_decision,omitempty"`
	FailureClass           string   `json:"failure_class,omitempty"`
	RiskHandoffRecommended bool     `json:"risk_handoff_recommended"`
	RiskSourceType         string   `json:"risk_source_type,omitempty"`
	RiskSourceID           string   `json:"risk_source_id,omitempty"`
	EvidenceIDs            []string `json:"evidence_ids,omitempty"`
	CreatedAt              string   `json:"created_at"`
}

func BuildPostDeploymentVerification(rootDir string, options PostDeploymentVerificationOptions) (PostDeploymentVerification, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return PostDeploymentVerification{}, err
	}
	options.ExecutionID = strings.TrimSpace(options.ExecutionID)
	options.Environment = normalizeToken(options.Environment)
	if options.MonitorLimit <= 0 {
		options.MonitorLimit = 20
	}
	now := time.Now().UTC()
	verification := PostDeploymentVerification{
		ID:        "post-deployment-verification-" + textutil.Slugify(firstNonEmpty(options.ExecutionID, options.Environment, "latest")) + "-" + now.Format("20060102150405") + "-" + strconv.FormatInt(now.UnixNano()%1_000_000_000, 10),
		Status:    "blocked",
		Decision:  "POST_DEPLOYMENT_VERIFICATION_BLOCKED",
		Reasons:   []string{},
		CreatedAt: now.Format(time.RFC3339Nano),
	}
	if options.ExecutionID == "" {
		verification.Reasons = append(verification.Reasons, "execution_id_required")
		return finishPostDeploymentVerification(rootDir, verification)
	}
	execution, found, err := LoadExecution(rootDir, options.ExecutionID)
	if err != nil {
		return PostDeploymentVerification{}, err
	}
	if !found {
		verification.ExecutionID = options.ExecutionID
		verification.Reasons = append(verification.Reasons, "deployment_execution_not_found")
		return finishPostDeploymentVerification(rootDir, verification)
	}
	verification.ExecutionID = execution.ID
	verification.DeploymentID = execution.DeploymentID
	verification.ReleaseID = execution.ReleaseID
	verification.Environment = firstNonEmpty(options.Environment, execution.Environment)
	verification.SmokeStatus = execution.SmokeReport.Status
	verification.SmokeDecision = execution.SmokeReport.Decision
	verification.RollbackRequired = execution.RollbackSuggestion.Required
	verification.RollbackDecision = execution.RollbackSuggestion.Decision

	history, historyFound, err := LoadPostDeploymentHistory(rootDir, execution.ID)
	if err != nil {
		return PostDeploymentVerification{}, err
	}
	if historyFound {
		verification.HistoryID = history.ID
		verification.HistoryStatus = history.Status
		verification.HistoryDecision = history.Decision
		verification.FailureClass = history.FailureClass
		verification.EvidenceIDs = appendUnique(verification.EvidenceIDs, history.EvidenceIDs...)
		verification.Reasons = appendUnique(verification.Reasons, history.Reasons...)
		if verification.Environment == "" {
			verification.Environment = history.Environment
		}
	}
	summary, err := BuildMonitorSummary(rootDir, MonitorSummaryOptions{Environment: verification.Environment, Limit: options.MonitorLimit})
	if err != nil {
		return PostDeploymentVerification{}, err
	}
	verification.MonitorSummaryID = summary.ID
	verification.MonitorStatus = summary.Status
	verification.MonitorDecision = summary.Decision
	verification.EvidenceIDs = appendUnique(verification.EvidenceIDs, summary.EvidenceIDs...)
	verification.Reasons = appendUnique(verification.Reasons, summary.Reasons...)

	return finishPostDeploymentVerification(rootDir, finalizePostDeploymentVerification(verification))
}

func LoadPostDeploymentVerification(rootDir string, id string) (PostDeploymentVerification, bool, error) {
	var verification PostDeploymentVerification
	found, err := fsutil.ReadJSON(postDeploymentVerificationPath(rootDir, id), &verification)
	return verification, found, err
}

func ListPostDeploymentVerifications(rootDir string, limit int) ([]PostDeploymentVerification, error) {
	if err := fsutil.EnsureDir(postDeploymentVerificationDir(rootDir)); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(postDeploymentVerificationDir(rootDir))
	if err != nil {
		return nil, err
	}
	verifications := []PostDeploymentVerification{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var verification PostDeploymentVerification
		found, err := fsutil.ReadJSON(filepath.Join(postDeploymentVerificationDir(rootDir), entry.Name()), &verification)
		if err != nil {
			return nil, err
		}
		if found && verification.ID != "" {
			verifications = append(verifications, verification)
		}
	}
	sort.SliceStable(verifications, func(i, j int) bool {
		return verifications[i].CreatedAt > verifications[j].CreatedAt
	})
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if len(verifications) > limit {
		return verifications[:limit], nil
	}
	return verifications, nil
}

func finalizePostDeploymentVerification(verification PostDeploymentVerification) PostDeploymentVerification {
	verification.Status = "completed"
	verification.Decision = "POST_DEPLOYMENT_VERIFICATION_PASSED"
	if verification.HistoryStatus == "failed" {
		verification.Status = "failed"
		verification.Decision = "POST_DEPLOYMENT_VERIFICATION_FAILED"
		verification.RiskHandoffRecommended = true
		verification.RiskSourceType = "post_deployment_history"
		verification.RiskSourceID = verification.HistoryID
		verification.Reasons = appendUnique(verification.Reasons, "post_deployment_history_failed")
		return verification
	}
	if verification.HistoryStatus == "blocked" || verification.HistoryStatus == "manual_required" {
		verification.Status = "attention_required"
		verification.Decision = "POST_DEPLOYMENT_VERIFICATION_ATTENTION_REQUIRED"
		verification.RiskHandoffRecommended = true
		verification.RiskSourceType = "post_deployment_history"
		verification.RiskSourceID = verification.HistoryID
		verification.Reasons = appendUnique(verification.Reasons, "post_deployment_history_attention_required")
		return verification
	}
	if verification.MonitorStatus == "critical" || verification.MonitorStatus == "attention_required" || verification.MonitorStatus == "unknown" {
		verification.Status = "attention_required"
		verification.Decision = "POST_DEPLOYMENT_VERIFICATION_ATTENTION_REQUIRED"
		verification.RiskHandoffRecommended = true
		verification.RiskSourceType = "monitor_summary"
		verification.RiskSourceID = verification.MonitorSummaryID
		verification.Reasons = appendUnique(verification.Reasons, "monitor_summary_attention_required")
		return verification
	}
	if verification.RollbackRequired {
		verification.Status = "attention_required"
		verification.Decision = "POST_DEPLOYMENT_VERIFICATION_ATTENTION_REQUIRED"
		verification.RiskHandoffRecommended = true
		verification.RiskSourceType = "post_deployment_history"
		verification.RiskSourceID = verification.HistoryID
		verification.Reasons = appendUnique(verification.Reasons, "rollback_required")
		return verification
	}
	verification.Reasons = appendUnique(verification.Reasons, "post_deployment_verification_passed")
	return verification
}

func finishPostDeploymentVerification(rootDir string, verification PostDeploymentVerification) (PostDeploymentVerification, error) {
	if err := fsutil.WriteJSON(postDeploymentVerificationPath(rootDir, verification.ID), verification); err != nil {
		return PostDeploymentVerification{}, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "post-deployment-verifications.jsonl"), verification); err != nil {
		return PostDeploymentVerification{}, err
	}
	_ = logging.Log(rootDir, "release", "deployment.post_deployment_verification.created", map[string]any{
		"verification_id":          verification.ID,
		"execution_id":             verification.ExecutionID,
		"deployment_id":            verification.DeploymentID,
		"decision":                 verification.Decision,
		"status":                   verification.Status,
		"environment":              verification.Environment,
		"risk_handoff_recommended": verification.RiskHandoffRecommended,
	})
	record, err := evidence.Add(rootDir, evidence.AddOptions{
		ParentType:  "post_deployment_verification",
		ParentID:    verification.ID,
		SubjectType: "deployment_execution",
		SubjectID:   verification.ExecutionID,
		Operation:   "deployment.post_deployment.verify",
		Status:      verification.Status,
		Decision:    verification.Decision,
		Reasons:     verification.Reasons,
		Source:      "deployment",
		Artifacts: []evidence.ArtifactRef{{
			Kind: "post_deployment_verification",
			ID:   verification.ID,
			Path: filepath.ToSlash(filepath.Join(".moyuan", "lifecycle", "deployments", "post-deployment-verifications", verification.ID+".json")),
		}},
	})
	if err != nil {
		return PostDeploymentVerification{}, err
	}
	verification.EvidenceIDs = appendUnique(verification.EvidenceIDs, record.ID)
	if err := fsutil.WriteJSON(postDeploymentVerificationPath(rootDir, verification.ID), verification); err != nil {
		return PostDeploymentVerification{}, err
	}
	return verification, nil
}

func postDeploymentVerificationDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "post-deployment-verifications")
}

func postDeploymentVerificationPath(rootDir string, id string) string {
	return filepath.Join(postDeploymentVerificationDir(rootDir), id+".json")
}
