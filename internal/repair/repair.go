package repair

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/issues"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/memory"
	"moyuan-code/internal/operations"
	"moyuan-code/internal/quality"
	"moyuan-code/internal/runtime"
	"moyuan-code/internal/textutil"
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

type OperationRepairCandidate struct {
	ID                string     `json:"id"`
	Status            string     `json:"status"`
	Decision          string     `json:"decision"`
	OperationType     string     `json:"operation_type"`
	OperationID       string     `json:"operation_id"`
	Operation         string     `json:"operation,omitempty"`
	OperationStatus   string     `json:"operation_status,omitempty"`
	OperationDecision string     `json:"operation_decision,omitempty"`
	FailureClass      string     `json:"failure_class"`
	SignalType        string     `json:"signal_type"`
	SignalID          string     `json:"signal_id,omitempty"`
	BugCandidateID    string     `json:"bug_candidate_id,omitempty"`
	RepairPlanID      string     `json:"repair_plan_id,omitempty"`
	EvidenceRefs      []string   `json:"evidence_refs"`
	ArtifactRefs      []string   `json:"artifact_refs"`
	Reasons           []string   `json:"reasons"`
	ReviewRequired    bool       `json:"review_required"`
	ReviewedAt        string     `json:"reviewed_at,omitempty"`
	ReviewedBy        string     `json:"reviewed_by,omitempty"`
	ReviewDecision    string     `json:"review_decision,omitempty"`
	ReviewReason      string     `json:"review_reason,omitempty"`
	IssueID           string     `json:"issue_id,omitempty"`
	RepairAttemptID   string     `json:"repair_attempt_id,omitempty"`
	CreatedAt         string     `json:"created_at"`
	Signal            *Signal    `json:"signal,omitempty"`
	Candidate         *Candidate `json:"candidate,omitempty"`
	Plan              *Plan      `json:"plan,omitempty"`
}

type OperationRepairReviewOptions struct {
	Decision   string `json:"decision"`
	ReviewerID string `json:"reviewer_id,omitempty"`
	Reason     string `json:"reason,omitempty"`
	NextStep   string `json:"next_step,omitempty"`
	RuntimeID  string `json:"runtime_id,omitempty"`
}

type OperationRepairReview struct {
	ID              string `json:"id"`
	CandidateID     string `json:"candidate_id"`
	Decision        string `json:"decision"`
	ReviewerID      string `json:"reviewer_id,omitempty"`
	Reason          string `json:"reason,omitempty"`
	NextStep        string `json:"next_step,omitempty"`
	Status          string `json:"status"`
	IssueID         string `json:"issue_id,omitempty"`
	RepairAttemptID string `json:"repair_attempt_id,omitempty"`
	CreatedAt       string `json:"created_at"`
}

type RepairIssue struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Status          string   `json:"status"`
	SourceType      string   `json:"source_type"`
	SourceID        string   `json:"source_id"`
	OperationType   string   `json:"operation_type,omitempty"`
	OperationID     string   `json:"operation_id,omitempty"`
	BugCandidateID  string   `json:"bug_candidate_id,omitempty"`
	RepairPlanID    string   `json:"repair_plan_id,omitempty"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
	FailureClass    string   `json:"failure_class,omitempty"`
	Acceptance      []string `json:"acceptance,omitempty"`
	RecommendedRole string   `json:"recommended_role,omitempty"`
	CreatedAt       string   `json:"created_at"`
}

func CaptureSignal(rootDir string, signalType string, summary string, sourceID string) (Signal, error) {
	return captureSignal(rootDir, signalType, summary, "run", sourceID, []string{})
}

func captureSignal(rootDir string, signalType string, summary string, sourceType string, sourceID string, evidenceRefs []string) (Signal, error) {
	signal := Signal{
		ID:           "signal-" + time.Now().UTC().Format("20060102150405.000000000"),
		SignalType:   signalType,
		SourceType:   sourceType,
		SourceID:     sourceID,
		Summary:      summary,
		EvidenceRefs: append([]string{}, evidenceRefs...),
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
	if signal.SignalType == "smoke_failure" || signal.SignalType == "monitor_alert" {
		classification = "NEEDS_EVIDENCE"
		confidence = 0.55
		status = "review_required"
		riskLevel = "medium"
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
	return planRepair(rootDir, candidate, false)
}

func planReviewOnlyRepair(rootDir string, candidate Candidate) (Plan, error) {
	return planRepair(rootDir, candidate, true)
}

func planRepair(rootDir string, candidate Candidate, reviewOnly bool) (Plan, error) {
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
	if reviewOnly {
		plan.Strategy = "review_repair_candidate"
		plan.RequiresApproval = true
		plan.Status = "candidate_review_required"
	}
	if err := fsutil.WriteJSON(planPath(rootDir, plan.ID), plan); err != nil {
		return Plan{}, err
	}
	_ = logging.Log(rootDir, "run", "self_repair.repair.planned", map[string]any{"bug_candidate_id": candidate.ID, "repair_plan_id": plan.ID, "decision": plan.Strategy, "requires_approval": plan.RequiresApproval})
	return plan, nil
}

func CandidateFromOperation(rootDir string, operationType string, operationID string) (OperationRepairCandidate, bool, error) {
	detail, found, err := operations.Load(rootDir, operationType, operationID)
	if err != nil || !found {
		return OperationRepairCandidate{}, found, err
	}
	signalType, failureClass, repairable, reasons := classifyOperationDetail(detail)
	result := OperationRepairCandidate{
		ID:                "operation-repair-candidate-" + textutil.Slugify(detail.OperationType+"-"+detail.ID) + "-" + time.Now().UTC().Format("20060102150405.000000000"),
		Status:            "ignored",
		Decision:          "REPAIR_CANDIDATE_NOT_REQUIRED",
		OperationType:     detail.OperationType,
		OperationID:       detail.ID,
		Operation:         detail.Operation,
		OperationStatus:   detail.Status,
		OperationDecision: detail.Decision,
		FailureClass:      failureClass,
		SignalType:        signalType,
		EvidenceRefs:      operationEvidenceRefs(detail),
		ArtifactRefs:      operationArtifactRefs(detail),
		Reasons:           append([]string{}, reasons...),
		ReviewRequired:    false,
		CreatedAt:         time.Now().UTC().Format(time.RFC3339Nano),
	}
	if !repairable {
		if err := saveOperationRepairCandidate(rootDir, result); err != nil {
			return OperationRepairCandidate{}, true, err
		}
		_ = logging.Log(rootDir, "run", "self_repair.operation_candidate.skipped", map[string]any{"operation_type": result.OperationType, "operation_id": result.OperationID, "decision": result.Decision, "reason": strings.Join(result.Reasons, ",")})
		return result, true, nil
	}
	signal, err := captureSignal(rootDir, signalType, operationSignalSummary(detail), "operation", detail.ID, result.EvidenceRefs)
	if err != nil {
		return OperationRepairCandidate{}, true, err
	}
	candidate, err := Classify(rootDir, signal)
	if err != nil {
		return OperationRepairCandidate{}, true, err
	}
	plan, err := planReviewOnlyRepair(rootDir, candidate)
	if err != nil {
		return OperationRepairCandidate{}, true, err
	}
	result.Status = "review_required"
	result.Decision = "REPAIR_CANDIDATE_CREATED"
	result.SignalID = signal.ID
	result.BugCandidateID = candidate.ID
	result.RepairPlanID = plan.ID
	result.ReviewRequired = true
	result.Signal = &signal
	result.Candidate = &candidate
	result.Plan = &plan
	if err := saveOperationRepairCandidate(rootDir, result); err != nil {
		return OperationRepairCandidate{}, true, err
	}
	_ = logging.Log(rootDir, "run", "self_repair.operation_candidate.created", map[string]any{"operation_type": result.OperationType, "operation_id": result.OperationID, "repair_plan_id": result.RepairPlanID, "failure_class": result.FailureClass})
	return result, true, nil
}

func LoadOperationRepairCandidate(rootDir string, id string) (OperationRepairCandidate, bool, error) {
	id, ok := cleanRepairID(id)
	if !ok {
		return OperationRepairCandidate{}, false, nil
	}
	var candidate OperationRepairCandidate
	found, err := fsutil.ReadJSON(operationRepairCandidatePath(rootDir, id), &candidate)
	return candidate, found, err
}

func ListOperationRepairCandidates(rootDir string, limit int) ([]OperationRepairCandidate, error) {
	lines, err := fsutil.TailLines(filepath.Join(workspace.ForRoot(rootDir).RepairDir, "operation-candidates.jsonl"), limit)
	if err != nil {
		return nil, err
	}
	latest := map[string]OperationRepairCandidate{}
	for _, line := range lines {
		var candidate OperationRepairCandidate
		if err := json.Unmarshal([]byte(line), &candidate); err != nil {
			return nil, err
		}
		if candidate.ID != "" {
			latest[candidate.ID] = candidate
		}
	}
	candidates := []OperationRepairCandidate{}
	for _, candidate := range latest {
		candidates = append(candidates, candidate)
	}
	sortOperationRepairCandidates(candidates)
	if limit > 0 && len(candidates) > limit {
		return candidates[:limit], nil
	}
	return candidates, nil
}

func ReviewOperationRepairCandidate(ctx context.Context, rootDir string, candidateID string, options OperationRepairReviewOptions) (OperationRepairReview, OperationRepairCandidate, *Attempt, bool, error) {
	_ = ctx
	options.Decision = normalizeReviewDecision(options.Decision)
	options.NextStep = normalizeReviewNextStep(options.NextStep, options.Decision)
	options.ReviewerID = strings.TrimSpace(options.ReviewerID)
	options.Reason = strings.TrimSpace(options.Reason)
	if options.Decision == "" {
		return OperationRepairReview{}, OperationRepairCandidate{}, nil, false, errors.New("review_decision_required")
	}
	candidate, found, err := LoadOperationRepairCandidate(rootDir, candidateID)
	if err != nil || !found {
		return OperationRepairReview{}, OperationRepairCandidate{}, nil, found, err
	}
	if candidate.Status != "review_required" && candidate.Status != "approved" {
		return OperationRepairReview{}, OperationRepairCandidate{}, nil, true, errors.New("operation_repair_candidate_not_reviewable")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	review := OperationRepairReview{
		ID:          "operation-repair-review-" + textutil.Slugify(candidate.ID) + "-" + time.Now().UTC().Format("20060102150405.000000000"),
		CandidateID: candidate.ID,
		Decision:    options.Decision,
		ReviewerID:  options.ReviewerID,
		Reason:      options.Reason,
		NextStep:    options.NextStep,
		Status:      "completed",
		CreatedAt:   now,
	}
	candidate.ReviewRequired = false
	candidate.ReviewedAt = now
	candidate.ReviewedBy = options.ReviewerID
	candidate.ReviewDecision = options.Decision
	candidate.ReviewReason = options.Reason
	if options.Decision == "rejected" {
		candidate.Status = "rejected"
		candidate.Decision = "REPAIR_CANDIDATE_REJECTED"
		candidate.Reasons = append(candidate.Reasons, "review_rejected:"+options.Reason)
		if err := saveOperationRepairCandidate(rootDir, candidate); err != nil {
			return OperationRepairReview{}, OperationRepairCandidate{}, nil, true, err
		}
		if err := saveOperationRepairReview(rootDir, review); err != nil {
			return OperationRepairReview{}, OperationRepairCandidate{}, nil, true, err
		}
		_ = logging.Log(rootDir, "run", "self_repair.operation_candidate.rejected", map[string]any{"candidate_id": candidate.ID, "reviewer_id": options.ReviewerID})
		return review, candidate, nil, true, nil
	}
	candidate.Status = "approved"
	candidate.Decision = "REPAIR_CANDIDATE_APPROVED"
	var attempt *Attempt
	if options.NextStep == "issue" || options.NextStep == "repair_attempt" {
		repairIssue, err := createRepairIssue(rootDir, candidate)
		if err != nil {
			return OperationRepairReview{}, OperationRepairCandidate{}, nil, true, err
		}
		candidate.IssueID = repairIssue.ID
		review.IssueID = repairIssue.ID
		if candidate.RepairPlanID != "" {
			plan, err := LoadPlan(rootDir, candidate.RepairPlanID)
			if err != nil {
				return OperationRepairReview{}, OperationRepairCandidate{}, nil, true, err
			}
			plan.IssueID = repairIssue.ID
			plan.Status = "review_approved"
			if err := savePlan(rootDir, plan); err != nil {
				return OperationRepairReview{}, OperationRepairCandidate{}, nil, true, err
			}
			candidate.Plan = &plan
		}
	}
	if options.NextStep == "repair_attempt" {
		if candidate.RepairPlanID == "" {
			return OperationRepairReview{}, OperationRepairCandidate{}, nil, true, errors.New("repair_plan_required")
		}
		created, err := CreateReviewOnlyAttempt(rootDir, candidate.RepairPlanID, options.RuntimeID)
		if err != nil {
			return OperationRepairReview{}, OperationRepairCandidate{}, nil, true, err
		}
		attempt = &created
		candidate.RepairAttemptID = created.ID
		review.RepairAttemptID = created.ID
	}
	if err := saveOperationRepairCandidate(rootDir, candidate); err != nil {
		return OperationRepairReview{}, OperationRepairCandidate{}, nil, true, err
	}
	if err := saveOperationRepairReview(rootDir, review); err != nil {
		return OperationRepairReview{}, OperationRepairCandidate{}, nil, true, err
	}
	_ = logging.Log(rootDir, "run", "self_repair.operation_candidate.approved", map[string]any{"candidate_id": candidate.ID, "reviewer_id": options.ReviewerID, "next_step": options.NextStep})
	return review, candidate, attempt, true, nil
}

func CreateReviewOnlyAttempt(rootDir string, planID string, runtimeID string) (Attempt, error) {
	plan, err := LoadPlan(rootDir, planID)
	if err != nil {
		return Attempt{}, err
	}
	if runtimeID == "" {
		runtimeID = "review_only"
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
		Status:         "review_ready",
		AttemptNo:      attemptNo,
		MaxAttempts:    plan.MaxAttempts,
		RuntimeID:      runtimeID,
		FailureReason:  "manual_review_required_before_runtime_execution",
		StartedAt:      time.Now().UTC().Format(time.RFC3339Nano),
	}
	if attempt.MaxAttempts <= 0 {
		attempt.MaxAttempts = defaultMaxAttempts
	}
	return finishAttempt(rootDir, attempt)
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
	if attempt.Status == "review_ready" {
		event = "self_repair.repair.review_ready"
	}
	_ = logging.Log(rootDir, "run", event, map[string]any{"repair_attempt_id": attempt.ID, "repair_plan_id": attempt.PlanID, "status": attempt.Status, "reason": attempt.FailureReason})
	return attempt, nil
}

func savePlan(rootDir string, plan Plan) error {
	return fsutil.WriteJSON(planPath(rootDir, plan.ID), plan)
}

func saveOperationRepairCandidate(rootDir string, candidate OperationRepairCandidate) error {
	if err := fsutil.WriteJSON(operationRepairCandidatePath(rootDir, candidate.ID), candidate); err != nil {
		return err
	}
	return fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).RepairDir, "operation-candidates.jsonl"), candidate)
}

func saveOperationRepairReview(rootDir string, review OperationRepairReview) error {
	if err := fsutil.WriteJSON(operationRepairReviewPath(rootDir, review.ID), review); err != nil {
		return err
	}
	return fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).RepairDir, "operation-candidate-reviews.jsonl"), review)
}

func createRepairIssue(rootDir string, candidate OperationRepairCandidate) (RepairIssue, error) {
	issueID := "repair-" + textutil.Slugify(candidate.FailureClass+"-"+candidate.OperationType+"-"+candidate.OperationID)
	if issueID == "repair" {
		issueID = "repair-" + textutil.Slugify(candidate.ID)
	}
	issue := RepairIssue{
		ID:              issueID,
		Title:           repairIssueTitle(candidate),
		Status:          "ready",
		SourceType:      "operation_repair_candidate",
		SourceID:        candidate.ID,
		OperationType:   candidate.OperationType,
		OperationID:     candidate.OperationID,
		BugCandidateID:  candidate.BugCandidateID,
		RepairPlanID:    candidate.RepairPlanID,
		EvidenceRefs:    append([]string{}, candidate.EvidenceRefs...),
		FailureClass:    candidate.FailureClass,
		RecommendedRole: "repair_agent",
		Acceptance: []string{
			"reproduce or explain the failure evidence before code changes",
			"apply minimal fix within approved write scope",
			"add or update regression coverage",
			"pass quality gate and independent review before merge",
		},
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := fsutil.WriteJSON(repairIssuePath(rootDir, issue.ID), issue); err != nil {
		return RepairIssue{}, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).RepairDir, "repair-issues.jsonl"), issue); err != nil {
		return RepairIssue{}, err
	}
	if err := upsertRepairIssueGraph(rootDir, issue); err != nil {
		return RepairIssue{}, err
	}
	_ = logging.Log(rootDir, "run", "self_repair.issue.created", map[string]any{"issue_id": issue.ID, "candidate_id": candidate.ID})
	return issue, nil
}

func upsertRepairIssueGraph(rootDir string, repairIssue RepairIssue) error {
	graph, found, err := issues.LoadGraph(rootDir, "repair-epic")
	if err != nil {
		return err
	}
	if !found {
		graph = issues.Graph{
			Epic:  issues.Epic{ID: "repair-epic", Title: "operation-repair", Status: "active"},
			Nodes: []issues.Node{},
		}
	}
	exists := false
	for idx := range graph.Nodes {
		if graph.Nodes[idx].ID == repairIssue.ID {
			graph.Nodes[idx].Title = repairIssue.Title
			graph.Nodes[idx].Status = repairIssue.Status
			exists = true
			break
		}
	}
	if !exists {
		graph.Nodes = append(graph.Nodes, issues.Node{ID: repairIssue.ID, Title: repairIssue.Title, Status: repairIssue.Status, DependsOn: []string{}})
	}
	if err := issues.SaveGraph(rootDir, graph); err != nil {
		return err
	}
	return issues.SaveSchedule(rootDir, issues.Summarize(graph))
}

func normalizeReviewDecision(value string) string {
	value = strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "-", "_")
	switch value {
	case "approve", "approved":
		return "approved"
	case "reject", "rejected":
		return "rejected"
	default:
		return ""
	}
}

func normalizeReviewNextStep(value string, decision string) string {
	value = strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "-", "_")
	if decision == "rejected" {
		return "none"
	}
	switch value {
	case "", "issue":
		return "issue"
	case "repair_attempt", "attempt":
		return "repair_attempt"
	case "none":
		return "none"
	default:
		return "issue"
	}
}

func sortOperationRepairCandidates(candidates []OperationRepairCandidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		return operationCandidateSortTime(candidates[i]) > operationCandidateSortTime(candidates[j])
	})
}

func operationCandidateSortTime(candidate OperationRepairCandidate) string {
	if candidate.ReviewedAt != "" {
		return candidate.ReviewedAt
	}
	return candidate.CreatedAt
}

func repairIssueTitle(candidate OperationRepairCandidate) string {
	parts := []string{"repair", candidate.FailureClass, candidate.OperationType, candidate.OperationID}
	filtered := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, " / ")
}

func cleanRepairID(id string) (string, bool) {
	id = strings.TrimSpace(id)
	if id == "" || strings.Contains(id, "/") || strings.Contains(id, "\\") || strings.Contains(id, "..") {
		return "", false
	}
	return id, true
}

func classifyOperationDetail(detail operations.Detail) (string, string, bool, []string) {
	reasons := append([]string{}, detail.Reasons...)
	decision := strings.ToUpper(detail.Decision)
	status := strings.ToLower(detail.Status)
	switch {
	case strings.Contains(decision, "SMOKE_FAILED") || strings.Contains(detail.Summary.SmokeDecision, "FAILED"):
		return "smoke_failure", "smoke_failed", true, append(reasons, "operation_smoke_failed")
	case strings.Contains(decision, "MONITOR_FAILED") || strings.Contains(detail.Summary.MonitorDecision, "FAILED"):
		return "monitor_alert", "monitor_failed", true, append(reasons, "operation_monitor_failed")
	case status == "failed" || strings.Contains(decision, "FAILED") || strings.Contains(decision, "REJECTED"):
		return "runtime_error", "operation_failed", true, append(reasons, "operation_failed")
	case status == "blocked" || strings.Contains(decision, "BLOCKED"):
		return "operation_blocked", "operation_blocked", true, append(reasons, "operation_blocked")
	default:
		return "", "none", false, append(reasons, "operation_not_failed")
	}
}

func operationSignalSummary(detail operations.Detail) string {
	parts := []string{detail.OperationType, detail.Operation, detail.Decision}
	for _, reason := range detail.Reasons {
		if strings.TrimSpace(reason) != "" {
			parts = append(parts, strings.TrimSpace(reason))
			break
		}
	}
	return strings.Join(parts, " / ")
}

func operationEvidenceRefs(detail operations.Detail) []string {
	refs := []string{}
	for _, record := range detail.Evidence {
		if record.ID != "" {
			refs = append(refs, record.ID)
		}
	}
	return refs
}

func operationArtifactRefs(detail operations.Detail) []string {
	refs := []string{}
	for _, artifact := range detail.Artifacts {
		if artifact.Path != "" {
			refs = append(refs, artifact.Path)
			continue
		}
		if artifact.ID != "" {
			refs = append(refs, artifact.ID)
		}
	}
	return refs
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

func operationRepairCandidatePath(rootDir string, id string) string {
	return filepath.Join(workspace.ForRoot(rootDir).RepairDir, "operation-candidates", id+".json")
}

func operationRepairReviewPath(rootDir string, id string) string {
	return filepath.Join(workspace.ForRoot(rootDir).RepairDir, "operation-candidate-reviews", id+".json")
}

func repairIssuePath(rootDir string, id string) string {
	return filepath.Join(workspace.ForRoot(rootDir).RepairDir, "issues", id+".json")
}
