package repair

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/deployment"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type DeploymentRiskHandoffOptions struct {
	AdmissionID      string `json:"admission_id,omitempty"`
	MonitorSummaryID string `json:"monitor_summary_id,omitempty"`
}

type DeploymentRiskHandoff struct {
	ID             string     `json:"id"`
	SourceType     string     `json:"source_type"`
	SourceID       string     `json:"source_id"`
	Status         string     `json:"status"`
	Decision       string     `json:"decision"`
	FailureClass   string     `json:"failure_class"`
	SignalID       string     `json:"signal_id,omitempty"`
	BugCandidateID string     `json:"bug_candidate_id,omitempty"`
	RepairPlanID   string     `json:"repair_plan_id,omitempty"`
	EvidenceRefs   []string   `json:"evidence_refs,omitempty"`
	Reasons        []string   `json:"reasons"`
	ReviewRequired bool       `json:"review_required"`
	CreatedAt      string     `json:"created_at"`
	Signal         *Signal    `json:"signal,omitempty"`
	Candidate      *Candidate `json:"candidate,omitempty"`
	Plan           *Plan      `json:"plan,omitempty"`
}

func CreateDeploymentRiskHandoff(rootDir string, options DeploymentRiskHandoffOptions) (DeploymentRiskHandoff, error) {
	options.AdmissionID = strings.TrimSpace(options.AdmissionID)
	options.MonitorSummaryID = strings.TrimSpace(options.MonitorSummaryID)
	handoff := DeploymentRiskHandoff{
		ID:             "deployment-risk-handoff-" + textutil.Slugify(firstDeploymentRiskSource(options)) + "-" + time.Now().UTC().Format("20060102150405.000000000"),
		Status:         "ignored",
		Decision:       "DEPLOYMENT_RISK_HANDOFF_NOT_REQUIRED",
		FailureClass:   "none",
		EvidenceRefs:   []string{},
		Reasons:        []string{},
		ReviewRequired: false,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339Nano),
	}
	if options.AdmissionID != "" {
		admission, found, err := deployment.LoadReleaseAdmission(rootDir, options.AdmissionID)
		if err != nil {
			return DeploymentRiskHandoff{}, err
		}
		if !found {
			handoff.SourceType = "release_admission"
			handoff.SourceID = options.AdmissionID
			handoff.Status = "blocked"
			handoff.Decision = "DEPLOYMENT_RISK_HANDOFF_BLOCKED"
			handoff.FailureClass = "release_admission_missing"
			handoff.Reasons = append(handoff.Reasons, "release_admission_not_found")
			return saveDeploymentRiskHandoff(rootDir, handoff)
		}
		handoff.SourceType = "release_admission"
		handoff.SourceID = admission.ID
		handoff.EvidenceRefs = append([]string{}, admission.EvidenceIDs...)
		handoff.FailureClass = admissionFailureClass(admission)
		handoff.Reasons = append(handoff.Reasons, admission.Reasons...)
		if admission.Status == "allowed" {
			handoff.Reasons = append(handoff.Reasons, "release_admission_allowed")
			return saveDeploymentRiskHandoff(rootDir, handoff)
		}
		return createDeploymentRiskRepair(rootDir, handoff, "release admission risk: "+admission.Decision)
	}
	if options.MonitorSummaryID != "" {
		summary, found, err := deployment.LoadMonitorSummary(rootDir, options.MonitorSummaryID)
		if err != nil {
			return DeploymentRiskHandoff{}, err
		}
		if !found {
			handoff.SourceType = "monitor_summary"
			handoff.SourceID = options.MonitorSummaryID
			handoff.Status = "blocked"
			handoff.Decision = "DEPLOYMENT_RISK_HANDOFF_BLOCKED"
			handoff.FailureClass = "monitor_summary_missing"
			handoff.Reasons = append(handoff.Reasons, "monitor_summary_not_found")
			return saveDeploymentRiskHandoff(rootDir, handoff)
		}
		handoff.SourceType = "monitor_summary"
		handoff.SourceID = summary.ID
		handoff.EvidenceRefs = append([]string{}, summary.EvidenceIDs...)
		handoff.FailureClass = "monitor_" + summary.Status
		handoff.Reasons = append(handoff.Reasons, summary.Reasons...)
		if summary.Status == "healthy" {
			handoff.Reasons = append(handoff.Reasons, "monitor_summary_healthy")
			return saveDeploymentRiskHandoff(rootDir, handoff)
		}
		return createDeploymentRiskRepair(rootDir, handoff, "deployment monitor risk: "+summary.Decision)
	}
	handoff.Status = "blocked"
	handoff.Decision = "DEPLOYMENT_RISK_HANDOFF_BLOCKED"
	handoff.FailureClass = "risk_source_missing"
	handoff.Reasons = append(handoff.Reasons, "admission_or_monitor_summary_required")
	return saveDeploymentRiskHandoff(rootDir, handoff)
}

func LoadDeploymentRiskHandoff(rootDir string, id string) (DeploymentRiskHandoff, bool, error) {
	var handoff DeploymentRiskHandoff
	found, err := fsutil.ReadJSON(deploymentRiskHandoffPath(rootDir, id), &handoff)
	return handoff, found, err
}

func ListDeploymentRiskHandoffs(rootDir string, limit int) ([]DeploymentRiskHandoff, error) {
	if err := fsutil.EnsureDir(deploymentRiskHandoffDir(rootDir)); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(deploymentRiskHandoffDir(rootDir))
	if err != nil {
		return nil, err
	}
	handoffs := []DeploymentRiskHandoff{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var handoff DeploymentRiskHandoff
		found, err := fsutil.ReadJSON(filepath.Join(deploymentRiskHandoffDir(rootDir), entry.Name()), &handoff)
		if err != nil {
			return nil, err
		}
		if found && handoff.ID != "" {
			handoffs = append(handoffs, handoff)
		}
	}
	sort.SliceStable(handoffs, func(i, j int) bool {
		return handoffs[i].CreatedAt > handoffs[j].CreatedAt
	})
	if limit <= 0 {
		limit = 20
	}
	if len(handoffs) > limit {
		return handoffs[:limit], nil
	}
	return handoffs, nil
}

func createDeploymentRiskRepair(rootDir string, handoff DeploymentRiskHandoff, summary string) (DeploymentRiskHandoff, error) {
	signal, err := captureSignal(rootDir, "monitor_alert", summary, handoff.SourceType, handoff.SourceID, handoff.EvidenceRefs)
	if err != nil {
		return DeploymentRiskHandoff{}, err
	}
	candidate, err := Classify(rootDir, signal)
	if err != nil {
		return DeploymentRiskHandoff{}, err
	}
	plan, err := PlanRepair(rootDir, candidate)
	if err != nil {
		return DeploymentRiskHandoff{}, err
	}
	handoff.Status = "review_required"
	handoff.Decision = "DEPLOYMENT_RISK_HANDOFF_REVIEW_REQUIRED"
	handoff.SignalID = signal.ID
	handoff.BugCandidateID = candidate.ID
	handoff.RepairPlanID = plan.ID
	handoff.ReviewRequired = true
	handoff.Signal = &signal
	handoff.Candidate = &candidate
	handoff.Plan = &plan
	handoff.Reasons = append(handoff.Reasons, "repair_review_required")
	return saveDeploymentRiskHandoff(rootDir, handoff)
}

func saveDeploymentRiskHandoff(rootDir string, handoff DeploymentRiskHandoff) (DeploymentRiskHandoff, error) {
	if err := fsutil.EnsureDir(deploymentRiskHandoffDir(rootDir)); err != nil {
		return DeploymentRiskHandoff{}, err
	}
	if err := fsutil.WriteJSON(deploymentRiskHandoffPath(rootDir, handoff.ID), handoff); err != nil {
		return DeploymentRiskHandoff{}, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).RepairDir, "deployment-risk-handoffs.jsonl"), handoff); err != nil {
		return DeploymentRiskHandoff{}, err
	}
	_ = logging.Log(rootDir, "run", "self_repair.deployment_risk_handoff.created", map[string]any{
		"handoff_id":     handoff.ID,
		"source_type":    handoff.SourceType,
		"source_id":      handoff.SourceID,
		"decision":       handoff.Decision,
		"status":         handoff.Status,
		"repair_plan_id": handoff.RepairPlanID,
	})
	return handoff, nil
}

func admissionFailureClass(admission deployment.ReleaseAdmission) string {
	if admission.Status == "blocked" {
		return "release_admission_blocked"
	}
	if admission.Status == "manual_required" {
		return "release_admission_manual_required"
	}
	return "none"
}

func firstDeploymentRiskSource(options DeploymentRiskHandoffOptions) string {
	if options.AdmissionID != "" {
		return options.AdmissionID
	}
	if options.MonitorSummaryID != "" {
		return options.MonitorSummaryID
	}
	return "manual"
}

func deploymentRiskHandoffDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).RepairDir, "deployment-risk-handoffs")
}

func deploymentRiskHandoffPath(rootDir string, id string) string {
	return filepath.Join(deploymentRiskHandoffDir(rootDir), id+".json")
}
