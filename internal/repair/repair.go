package repair

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/memory"
	"moyuan-code/internal/quality"
	"moyuan-code/internal/runtime"
	"moyuan-code/internal/workspace"
)

const defaultMaxAttempts = 2

type Signal struct {
	ID           string   `json:"id"`
	SignalType   string   `json:"signal_type"`
	SourceType   string   `json:"source_type"`
	SourceID     string   `json:"source_id,omitempty"`
	Summary      string   `json:"summary"`
	EvidenceRefs []string `json:"evidence_refs"`
	OccurredAt   string   `json:"occurred_at"`
	TraceID      string   `json:"trace_id"`
}

type Candidate struct {
	ID                   string   `json:"id"`
	SignalIDs            []string `json:"signal_ids"`
	Title                string   `json:"title"`
	Classification       string   `json:"classification"`
	Confidence           float64  `json:"confidence"`
	RiskLevel            string   `json:"risk_level"`
	Status               string   `json:"status"`
	Reproducible         bool     `json:"reproducible"`
	ReproductionCommands []string `json:"reproduction_commands"`
	SuspectedRootCause   string   `json:"suspected_root_cause,omitempty"`
	CreatedAt            string   `json:"created_at"`
}

type Plan struct {
	ID                      string   `json:"id"`
	BugCandidateID          string   `json:"bug_candidate_id"`
	IssueID                 string   `json:"issue_id,omitempty"`
	WriteScope              []string `json:"write_scope"`
	Strategy                string   `json:"strategy"`
	RegressionTestRequired  bool     `json:"regression_test_required"`
	QualityGateRequired     bool     `json:"quality_gate_required"`
	Commands                []string `json:"commands"`
	RequiresApproval        bool     `json:"requires_approval"`
	MaxAttempts             int      `json:"max_attempts"`
	CandidateClassification string   `json:"candidate_classification"`
	RiskLevel               string   `json:"risk_level"`
	Status                  string   `json:"status"`
}

type Attempt struct {
	ID              string          `json:"id"`
	PlanID          string          `json:"plan_id"`
	BugCandidateID  string          `json:"bug_candidate_id"`
	IssueID         string          `json:"issue_id"`
	Status          string          `json:"status"`
	AttemptNo       int             `json:"attempt_no"`
	MaxAttempts     int             `json:"max_attempts"`
	RuntimeID       string          `json:"runtime_id"`
	RuntimeStatus   string          `json:"runtime_status,omitempty"`
	QualityReportID string          `json:"quality_report_id,omitempty"`
	QualityStatus   string          `json:"quality_status,omitempty"`
	ReviewStatus    string          `json:"review_status,omitempty"`
	ChangedFiles    []string        `json:"changed_files"`
	FailureReason   string          `json:"failure_reason,omitempty"`
	StartedAt       string          `json:"started_at"`
	FinishedAt      string          `json:"finished_at"`
	RuntimeResult   *runtime.Result `json:"runtime_result,omitempty"`
	QualityReport   *quality.Report `json:"quality_report,omitempty"`
	MemoryStatus    string          `json:"memory_status,omitempty"`
	MemoryRecordID  string          `json:"memory_record_id,omitempty"`
}

func CaptureSignal(rootDir string, signalType string, summary string, sourceID string) (Signal, error) {
	signal := Signal{
		ID:           "signal-" + time.Now().UTC().Format("20060102150405.000000000"),
		SignalType:   signalType,
		SourceType:   "run",
		SourceID:     sourceID,
		Summary:      summary,
		EvidenceRefs: []string{},
		OccurredAt:   time.Now().UTC().Format(time.RFC3339Nano),
		TraceID:      "trace-" + time.Now().UTC().Format("20060102150405"),
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).RepairDir, "signals.jsonl"), signal); err != nil {
		return Signal{}, err
	}
	_ = logging.Log(rootDir, "run", "self_repair.signal.captured", map[string]any{"signal_id": signal.ID, "signal_type": signal.SignalType, "trace_id": signal.TraceID})
	return signal, nil
}

func Classify(rootDir string, signal Signal) (Candidate, error) {
	classification := "NEEDS_EVIDENCE"
	confidence := 0.4
	status := "needs_evidence"
	reproducible := false
	riskLevel := "medium"
	if signal.SignalType == "test_failure" || signal.SignalType == "runtime_error" {
		classification = "CONFIRMED_BUG"
		confidence = 0.75
		status = "confirmed"
		reproducible = true
		riskLevel = "low"
	}
	if signal.SignalType == "enhancement" {
		classification = "ENHANCEMENT"
		confidence = 0.6
		status = "issue_required"
	}
	candidate := Candidate{
		ID:                   "bug-" + time.Now().UTC().Format("20060102150405.000000000"),
		SignalIDs:            []string{signal.ID},
		Title:                signal.Summary,
		Classification:       classification,
		Confidence:           confidence,
		RiskLevel:            riskLevel,
		Status:               status,
		Reproducible:         reproducible,
		ReproductionCommands: []string{},
		CreatedAt:            time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).RepairDir, "bug-candidates.jsonl"), candidate); err != nil {
		return Candidate{}, err
	}
	_ = logging.Log(rootDir, "run", "self_repair.bug.classified", map[string]any{"bug_candidate_id": candidate.ID, "decision": candidate.Classification, "confidence": candidate.Confidence})
	return candidate, nil
}

func PlanRepair(rootDir string, candidate Candidate) (Plan, error) {
	plan := Plan{
		ID:                      "repair-plan-" + time.Now().UTC().Format("20060102150405.000000000"),
		BugCandidateID:          candidate.ID,
		IssueID:                 candidate.ID,
		WriteScope:              []string{"."},
		Strategy:                "minimal_fix",
		RegressionTestRequired:  true,
		QualityGateRequired:     true,
		Commands:                []string{},
		RequiresApproval:        false,
		MaxAttempts:             defaultMaxAttempts,
		CandidateClassification: candidate.Classification,
		RiskLevel:               candidate.RiskLevel,
		Status:                  "planned",
	}
	if candidate.Classification != "CONFIRMED_BUG" || candidate.RiskLevel != "low" || !candidate.Reproducible {
		plan.Strategy = "issue_only"
		plan.RequiresApproval = true
		plan.Status = "requires_approval"
	}
	if err := fsutil.WriteJSON(planPath(rootDir, plan.ID), plan); err != nil {
		return Plan{}, err
	}
	_ = logging.Log(rootDir, "run", "self_repair.repair.planned", map[string]any{"bug_candidate_id": candidate.ID, "repair_plan_id": plan.ID, "decision": plan.Strategy, "requires_approval": plan.RequiresApproval})
	return plan, nil
}

func RunAttempt(ctx context.Context, rootDir string, planID string, runtimeID string, prompt string) (Attempt, error) {
	plan, err := LoadPlan(rootDir, planID)
	if err != nil {
		return Attempt{}, err
	}
	if runtimeID == "" {
		runtimeID = "local_shell"
	}
	attemptNo, err := nextAttemptNo(rootDir, plan.ID)
	if err != nil {
		return Attempt{}, err
	}
	attempt := Attempt{
		ID:             "repair-attempt-" + time.Now().UTC().Format("20060102150405.000000000"),
		PlanID:         plan.ID,
		BugCandidateID: plan.BugCandidateID,
		IssueID:        plan.IssueID,
		Status:         "running",
		AttemptNo:      attemptNo,
		MaxAttempts:    plan.MaxAttempts,
		RuntimeID:      runtimeID,
		StartedAt:      time.Now().UTC().Format(time.RFC3339Nano),
	}
	if attempt.MaxAttempts <= 0 {
		attempt.MaxAttempts = defaultMaxAttempts
	}
	if attempt.AttemptNo > attempt.MaxAttempts {
		attempt.Status = "blocked"
		attempt.FailureReason = "max_attempts_exceeded"
		return finishAttempt(rootDir, attempt)
	}
	if plan.RequiresApproval || plan.CandidateClassification != "CONFIRMED_BUG" || plan.RiskLevel != "low" {
		attempt.Status = "blocked"
		attempt.FailureReason = "approval_required"
		return finishAttempt(rootDir, attempt)
	}
	if strings.TrimSpace(prompt) == "" {
		attempt.Status = "blocked"
		attempt.FailureReason = "missing_repair_prompt"
		return finishAttempt(rootDir, attempt)
	}
	_ = logging.Log(rootDir, "run", "self_repair.repair.started", map[string]any{"repair_attempt_id": attempt.ID, "repair_plan_id": plan.ID, "attempt_no": attempt.AttemptNo})
	rt, err := runtime.Invoke(ctx, rootDir, runtime.Invocation{
		RunID:     attempt.ID,
		IssueID:   plan.IssueID,
		Role:      "repair_agent",
		RuntimeID: runtimeID,
		Mode:      "code",
		Prompt:    prompt,
	})
	if err != nil {
		return Attempt{}, err
	}
	attempt.RuntimeResult = &rt
	attempt.RuntimeStatus = rt.Status
	attempt.ChangedFiles = rt.ChangedFiles
	if rt.Status != "completed" {
		attempt.Status = "failed"
		attempt.FailureReason = "runtime_" + rt.Status
		return finishAttempt(rootDir, attempt)
	}
	report, err := quality.RunWithReview(ctx, rootDir, plan.IssueID, quality.ReviewInput{
		ChangedFiles:    rt.ChangedFiles,
		DiffSummaryPath: rt.DiffSummaryPath,
		ProtectedFiles:  rt.Diff.ProtectedFiles,
		RuntimeRisks:    rt.Risks,
	})
	if err != nil {
		return Attempt{}, err
	}
	attempt.QualityReport = &report
	attempt.QualityReportID = report.ID
	attempt.QualityStatus = report.Status
	attempt.ReviewStatus = report.ReviewStatus
	if report.Status != "passed" || report.ReviewStatus == "rejected" {
		attempt.Status = "needs_rework"
		attempt.FailureReason = "quality_rejected"
		return finishAttempt(rootDir, attempt)
	}
	decision, err := memory.Submit(rootDir, "lesson", repairMemorySummary(plan, attempt), []string{"repair", "quality"}, "self_repair")
	if err == nil {
		attempt.MemoryStatus = decision.Status
		if decision.Record != nil {
			attempt.MemoryRecordID = decision.Record.ID
		}
	}
	attempt.Status = "repaired"
	return finishAttempt(rootDir, attempt)
}

func LoadPlan(rootDir string, planID string) (Plan, error) {
	var plan Plan
	found, err := fsutil.ReadJSON(planPath(rootDir, planID), &plan)
	if err != nil {
		return Plan{}, err
	}
	if !found {
		return Plan{}, errors.New("repair plan not found")
	}
	if plan.MaxAttempts <= 0 {
		plan.MaxAttempts = defaultMaxAttempts
	}
	return plan, nil
}

func LoadAttempt(rootDir string, attemptID string) (Attempt, bool, error) {
	var attempt Attempt
	found, err := fsutil.ReadJSON(attemptPath(rootDir, attemptID), &attempt)
	return attempt, found, err
}

func finishAttempt(rootDir string, attempt Attempt) (Attempt, error) {
	attempt.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := fsutil.WriteJSON(attemptPath(rootDir, attempt.ID), attempt); err != nil {
		return Attempt{}, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).RepairDir, "attempts.jsonl"), attempt); err != nil {
		return Attempt{}, err
	}
	event := "self_repair.repair.completed"
	if attempt.Status != "repaired" {
		event = "self_repair.repair.failed"
	}
	_ = logging.Log(rootDir, "run", event, map[string]any{"repair_attempt_id": attempt.ID, "repair_plan_id": attempt.PlanID, "status": attempt.Status, "reason": attempt.FailureReason})
	return attempt, nil
}

func nextAttemptNo(rootDir string, planID string) (int, error) {
	lines, err := fsutil.TailLines(filepath.Join(workspace.ForRoot(rootDir).RepairDir, "attempts.jsonl"), 1000)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, line := range lines {
		var attempt Attempt
		if err := json.Unmarshal([]byte(line), &attempt); err == nil && attempt.PlanID == planID {
			count++
		}
	}
	return count + 1, nil
}

func repairMemorySummary(plan Plan, attempt Attempt) string {
	changed := strings.Join(attempt.ChangedFiles, ", ")
	if changed == "" {
		changed = "no files"
	}
	return fmt.Sprintf("Repair succeeded for %s with quality report %s and changed files %s. Future similar bugs should run regression tests before acceptance.", plan.BugCandidateID, attempt.QualityReportID, changed)
}

func planPath(rootDir string, planID string) string {
	return filepath.Join(workspace.ForRoot(rootDir).RepairDir, planID+".json")
}

func attemptPath(rootDir string, attemptID string) string {
	return filepath.Join(workspace.ForRoot(rootDir).RepairDir, "attempts", attemptID+".json")
}
