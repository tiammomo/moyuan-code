package repair

import (
	"path/filepath"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/workspace"
)

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
	ID                     string   `json:"id"`
	BugCandidateID         string   `json:"bug_candidate_id"`
	IssueID                string   `json:"issue_id,omitempty"`
	WriteScope             []string `json:"write_scope"`
	Strategy               string   `json:"strategy"`
	RegressionTestRequired bool     `json:"regression_test_required"`
	Commands               []string `json:"commands"`
	RequiresApproval       bool     `json:"requires_approval"`
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
	_ = logging.Log(rootDir, "run", "self_repair.signal.captured", map[string]any{"signal_id": signal.ID, "signal_type": signal.SignalType})
	return signal, nil
}

func Classify(rootDir string, signal Signal) (Candidate, error) {
	classification := "NEEDS_EVIDENCE"
	confidence := 0.4
	status := "needs_evidence"
	if signal.SignalType == "test_failure" || signal.SignalType == "runtime_error" {
		classification = "CONFIRMED_BUG"
		confidence = 0.75
		status = "confirmed"
	}
	candidate := Candidate{
		ID:                   "bug-" + time.Now().UTC().Format("20060102150405.000000000"),
		SignalIDs:            []string{signal.ID},
		Title:                signal.Summary,
		Classification:       classification,
		Confidence:           confidence,
		RiskLevel:            "low",
		Status:               status,
		Reproducible:         classification == "CONFIRMED_BUG",
		ReproductionCommands: []string{},
		CreatedAt:            time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).RepairDir, "bug-candidates.jsonl"), candidate); err != nil {
		return Candidate{}, err
	}
	_ = logging.Log(rootDir, "run", "self_repair.bug.classified", map[string]any{"bug_candidate_id": candidate.ID, "decision": candidate.Classification})
	return candidate, nil
}

func PlanRepair(rootDir string, candidate Candidate) (Plan, error) {
	plan := Plan{
		ID:                     "repair-plan-" + time.Now().UTC().Format("20060102150405.000000000"),
		BugCandidateID:         candidate.ID,
		IssueID:                candidate.ID,
		WriteScope:             []string{"."},
		Strategy:               "minimal_fix",
		RegressionTestRequired: true,
		Commands:               []string{},
		RequiresApproval:       false,
	}
	if candidate.Classification != "CONFIRMED_BUG" {
		plan.Strategy = "issue_only"
		plan.RequiresApproval = true
	}
	if err := fsutil.WriteJSON(filepath.Join(workspace.ForRoot(rootDir).RepairDir, plan.ID+".json"), plan); err != nil {
		return Plan{}, err
	}
	_ = logging.Log(rootDir, "run", "self_repair.repair.planned", map[string]any{"bug_candidate_id": candidate.ID, "repair_plan_id": plan.ID, "decision": plan.Strategy})
	return plan, nil
}
