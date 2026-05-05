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
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type ReleaseAdmissionOptions struct {
	RehearsalID  string `json:"rehearsal_id,omitempty"`
	CandidateID  string `json:"candidate_id,omitempty"`
	DeploymentID string `json:"deployment_id,omitempty"`
	ExecutionID  string `json:"execution_id,omitempty"`
	Environment  string `json:"environment,omitempty"`
	MonitorLimit int    `json:"monitor_limit,omitempty"`
}

type ReleaseAdmission struct {
	ID             string                         `json:"id"`
	RehearsalID    string                         `json:"rehearsal_id,omitempty"`
	CandidateID    string                         `json:"candidate_id,omitempty"`
	DeploymentID   string                         `json:"deployment_id,omitempty"`
	ExecutionID    string                         `json:"execution_id,omitempty"`
	Environment    string                         `json:"environment,omitempty"`
	Status         string                         `json:"status"`
	Decision       string                         `json:"decision"`
	Reasons        []string                       `json:"reasons"`
	Signals        []AdmissionSignal              `json:"signals"`
	PolicyID       string                         `json:"policy_id,omitempty"`
	PolicyVersion  string                         `json:"policy_version,omitempty"`
	PolicySource   string                         `json:"policy_source,omitempty"`
	MatchedRules   []AdmissionRuleMatch           `json:"matched_rules,omitempty"`
	PolicyDecision ReleaseAdmissionPolicyDecision `json:"policy_decision,omitempty"`
	EvidenceIDs    []string                       `json:"evidence_ids,omitempty"`
	CreatedAt      string                         `json:"created_at"`
}

type AdmissionSignal struct {
	Type     string `json:"type"`
	ID       string `json:"id,omitempty"`
	Status   string `json:"status"`
	Decision string `json:"decision"`
	Severity string `json:"severity,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

func BuildReleaseAdmission(ctx context.Context, rootDir string, options ReleaseAdmissionOptions) (ReleaseAdmission, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return ReleaseAdmission{}, err
	}
	options.RehearsalID = strings.TrimSpace(options.RehearsalID)
	options.CandidateID = strings.TrimSpace(options.CandidateID)
	options.DeploymentID = strings.TrimSpace(options.DeploymentID)
	options.ExecutionID = strings.TrimSpace(options.ExecutionID)
	options.Environment = normalizeToken(options.Environment)
	if options.MonitorLimit <= 0 {
		options.MonitorLimit = 10
	}
	now := time.Now().UTC()
	admission := ReleaseAdmission{
		ID:        "release-admission-" + textutil.Slugify(admissionIDSeed(options)) + "-" + now.Format("20060102150405") + "-" + strconv.FormatInt(now.UnixNano()%1_000_000_000, 10),
		Status:    "blocked",
		Decision:  "RELEASE_ADMISSION_BLOCKED",
		Reasons:   []string{},
		Signals:   []AdmissionSignal{},
		CreatedAt: now.Format(time.RFC3339Nano),
	}
	rehearsal, found, err := resolveAdmissionRehearsal(ctx, rootDir, options)
	if err != nil {
		return ReleaseAdmission{}, err
	}
	if !found {
		admission.Reasons = append(admission.Reasons, "deployment_rehearsal_missing")
		return finalizeAndFinishReleaseAdmission(rootDir, admission)
	}
	admission.RehearsalID = rehearsal.ID
	admission.CandidateID = firstNonEmpty(options.CandidateID, rehearsal.CandidateID)
	admission.DeploymentID = firstNonEmpty(options.DeploymentID, rehearsal.DeploymentID)
	admission.ExecutionID = firstNonEmpty(options.ExecutionID, rehearsal.ExecutionID)
	admission.Environment = firstNonEmpty(options.Environment, rehearsal.Environment)
	admission.EvidenceIDs = appendUnique(admission.EvidenceIDs, rehearsal.EvidenceIDs...)
	rehearsalEvidence, err := evidenceIDs(rootDir, "deployment_rehearsal", rehearsal.ID)
	if err != nil {
		return ReleaseAdmission{}, err
	}
	admission.EvidenceIDs = appendUnique(admission.EvidenceIDs, rehearsalEvidence...)
	admission.Signals = append(admission.Signals, AdmissionSignal{
		Type:     "deployment_rehearsal",
		ID:       rehearsal.ID,
		Status:   rehearsal.Status,
		Decision: rehearsal.Decision,
		Reason:   firstReason(rehearsal.Reasons),
	})
	if rehearsal.MonitorSummaryID != "" {
		admission.Signals = append(admission.Signals, AdmissionSignal{
			Type:     "monitor_summary",
			ID:       rehearsal.MonitorSummaryID,
			Status:   rehearsal.MonitorStatus,
			Decision: rehearsal.MonitorDecision,
			Severity: monitorSeverity(rehearsal.MonitorStatus),
		})
	}
	if rehearsal.RollbackExecutionID != "" {
		admission.Signals = append(admission.Signals, AdmissionSignal{
			Type:     "rollback_preview",
			ID:       rehearsal.RollbackExecutionID,
			Status:   rehearsal.RollbackStatus,
			Decision: rehearsal.RollbackDecision,
			Severity: "warning",
			Reason:   "rollback_required",
		})
	}
	if admission.CandidateID != "" {
		feedback, feedbackFound, err := FeedbackForCandidate(rootDir, admission.CandidateID, options.MonitorLimit)
		if err != nil {
			return ReleaseAdmission{}, err
		}
		if feedbackFound {
			admission.EvidenceIDs = appendUnique(admission.EvidenceIDs, feedback.EvidenceIDs...)
			admission.Signals = append(admission.Signals, AdmissionSignal{
				Type:     "candidate_deployment_feedback",
				ID:       feedback.ID,
				Status:   feedback.Status,
				Decision: feedback.Decision,
				Severity: feedback.Severity,
				Reason:   firstReason(feedback.Reasons),
			})
		}
	}
	if admission.DeploymentID != "" {
		plan, planFound, err := Load(rootDir, admission.DeploymentID)
		if err != nil {
			return ReleaseAdmission{}, err
		}
		if planFound {
			for _, resource := range plan.Resources {
				admission.Signals = append(admission.Signals, AdmissionSignal{
					Type:     "resource_status",
					ID:       resource.ID,
					Status:   resource.Status,
					Decision: "RESOURCE_STATUS_RECORDED",
					Reason:   resource.Environment,
				})
			}
		}
	}
	return finalizeAndFinishReleaseAdmission(rootDir, admission)
}

func LoadReleaseAdmission(rootDir string, id string) (ReleaseAdmission, bool, error) {
	var admission ReleaseAdmission
	found, err := fsutil.ReadJSON(releaseAdmissionPath(rootDir, id), &admission)
	return admission, found, err
}

func ListReleaseAdmissions(rootDir string, limit int) ([]ReleaseAdmission, error) {
	if err := fsutil.EnsureDir(releaseAdmissionDir(rootDir)); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(releaseAdmissionDir(rootDir))
	if err != nil {
		return nil, err
	}
	admissions := []ReleaseAdmission{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var admission ReleaseAdmission
		found, err := fsutil.ReadJSON(filepath.Join(releaseAdmissionDir(rootDir), entry.Name()), &admission)
		if err != nil {
			return nil, err
		}
		if found && admission.ID != "" {
			admissions = append(admissions, admission)
		}
	}
	sort.SliceStable(admissions, func(i, j int) bool {
		return admissions[i].CreatedAt > admissions[j].CreatedAt
	})
	if limit <= 0 {
		limit = 20
	}
	if len(admissions) > limit {
		return admissions[:limit], nil
	}
	return admissions, nil
}

func resolveAdmissionRehearsal(ctx context.Context, rootDir string, options ReleaseAdmissionOptions) (DeploymentRehearsal, bool, error) {
	if options.RehearsalID != "" {
		return LoadRehearsal(rootDir, options.RehearsalID)
	}
	rehearsal, err := BuildRehearsal(ctx, rootDir, RehearsalOptions{
		CandidateID:  options.CandidateID,
		DeploymentID: options.DeploymentID,
		ExecutionID:  options.ExecutionID,
		Environment:  options.Environment,
		MonitorLimit: options.MonitorLimit,
	})
	return rehearsal, rehearsal.ID != "", err
}

func finalizeAndFinishReleaseAdmission(rootDir string, admission ReleaseAdmission) (ReleaseAdmission, error) {
	pack, err := LoadReleaseAdmissionPolicyPack(rootDir, admission.Environment)
	if err != nil {
		return ReleaseAdmission{}, err
	}
	return finishReleaseAdmission(rootDir, EvaluateReleaseAdmissionPolicy(pack, admission))
}

func finishReleaseAdmission(rootDir string, admission ReleaseAdmission) (ReleaseAdmission, error) {
	if err := fsutil.EnsureDir(releaseAdmissionDir(rootDir)); err != nil {
		return ReleaseAdmission{}, err
	}
	if err := fsutil.WriteJSON(releaseAdmissionPath(rootDir, admission.ID), admission); err != nil {
		return ReleaseAdmission{}, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "release-admissions.jsonl"), admission); err != nil {
		return ReleaseAdmission{}, err
	}
	_ = logging.Log(rootDir, "release", "release.admission.created", map[string]any{
		"admission_id":  admission.ID,
		"candidate_id":  admission.CandidateID,
		"deployment_id": admission.DeploymentID,
		"rehearsal_id":  admission.RehearsalID,
		"decision":      admission.Decision,
		"status":        admission.Status,
		"environment":   admission.Environment,
	})
	if _, err := evidence.Add(rootDir, evidence.AddOptions{
		ParentType:  "release_admission",
		ParentID:    admission.ID,
		SubjectType: "deployment",
		SubjectID:   admissionSubjectID(admission),
		Operation:   "release.admission",
		Status:      admission.Status,
		Decision:    admission.Decision,
		Reasons:     admission.Reasons,
		Source:      "deployment",
		Artifacts: []evidence.ArtifactRef{{
			Kind: "release_admission",
			ID:   admission.ID,
			Path: filepath.ToSlash(filepath.Join(".moyuan", "lifecycle", "deployments", "release-admissions", admission.ID+".json")),
		}},
	}); err != nil {
		return ReleaseAdmission{}, err
	}
	return admission, nil
}

func monitorSeverity(status string) string {
	switch status {
	case "critical":
		return "critical"
	case "attention_required", "unknown":
		return "warning"
	default:
		return ""
	}
}

func firstReason(reasons []string) string {
	if len(reasons) == 0 {
		return ""
	}
	return reasons[0]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func admissionIDSeed(options ReleaseAdmissionOptions) string {
	for _, value := range []string{options.RehearsalID, options.ExecutionID, options.DeploymentID, options.CandidateID, options.Environment} {
		if value != "" {
			return value
		}
	}
	return "manual"
}

func admissionSubjectID(admission ReleaseAdmission) string {
	for _, value := range []string{admission.CandidateID, admission.DeploymentID, admission.ExecutionID, admission.RehearsalID, admission.Environment} {
		if value != "" {
			return value
		}
	}
	return "unknown"
}

func releaseAdmissionDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "release-admissions")
}

func releaseAdmissionPath(rootDir string, id string) string {
	return filepath.Join(releaseAdmissionDir(rootDir), id+".json")
}
