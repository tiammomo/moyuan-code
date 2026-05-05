package operations

import (
	"fmt"
	"sort"
	"time"
)

const (
	defaultWriteAdmissionPolicyID      = "write-admission-default-v1"
	defaultWriteAdmissionPolicyVersion = "2026-05-05"
)

type WriteAdmissionOptions struct {
	Provider      string `json:"provider,omitempty"`
	OperationType string `json:"operation_type,omitempty"`
	Environment   string `json:"environment,omitempty"`
	Status        string `json:"status,omitempty"`
	Decision      string `json:"decision,omitempty"`
	Target        string `json:"target,omitempty"`
	Limit         int    `json:"limit,omitempty"`
}

type WriteAdmissionReport struct {
	ID            string                `json:"id"`
	GeneratedAt   string                `json:"generated_at"`
	PolicyID      string                `json:"policy_id"`
	PolicyVersion string                `json:"policy_version"`
	Target        string                `json:"target"`
	Filters       WriteAdmissionOptions `json:"filters"`
	Summary       WriteAdmissionSummary `json:"summary"`
	Entries       []WriteAdmissionEntry `json:"entries"`
}

type WriteAdmissionSummary struct {
	EntryCount          int            `json:"entry_count"`
	ReadyCount          int            `json:"ready_count"`
	BlockedCount        int            `json:"blocked_count"`
	ManualRequiredCount int            `json:"manual_required_count"`
	RehearsalOnlyCount  int            `json:"rehearsal_only_count"`
	ByOperationType     map[string]int `json:"by_operation_type,omitempty"`
	ByProvider          map[string]int `json:"by_provider,omitempty"`
	ByEnvironment       map[string]int `json:"by_environment,omitempty"`
	ByStatus            map[string]int `json:"by_status,omitempty"`
	ByDecision          map[string]int `json:"by_decision,omitempty"`
	RedactionApplied    bool           `json:"redaction_applied"`
}

type WriteAdmissionEntry struct {
	ID                   string         `json:"id"`
	ProofID              string         `json:"proof_id"`
	ProofDecision        string         `json:"proof_decision"`
	OperationType        string         `json:"operation_type"`
	OperationID          string         `json:"operation_id"`
	Provider             string         `json:"provider,omitempty"`
	Environment          string         `json:"environment,omitempty"`
	Mode                 string         `json:"mode,omitempty"`
	Status               string         `json:"status"`
	Decision             string         `json:"decision"`
	Reasons              []string       `json:"reasons,omitempty"`
	RuleRefs             []string       `json:"rule_refs,omitempty"`
	SourceRef            string         `json:"source_ref,omitempty"`
	DryRun               bool           `json:"dry_run"`
	WriteEnabled         bool           `json:"write_enabled"`
	RehearsalAllowed     bool           `json:"rehearsal_allowed"`
	ApprovalRequired     bool           `json:"approval_required"`
	ApprovalSatisfied    bool           `json:"approval_satisfied"`
	ApprovalID           string         `json:"approval_id,omitempty"`
	SecretRefStatus      string         `json:"secret_ref_status,omitempty"`
	ProviderEvidenceRefs []string       `json:"provider_evidence_refs,omitempty"`
	LeastPrivilege       string         `json:"least_privilege,omitempty"`
	ReplayGuard          string         `json:"replay_guard,omitempty"`
	CreatedAt            string         `json:"created_at,omitempty"`
	Metadata             map[string]any `json:"metadata,omitempty"`
}

func BuildWriteAdmissions(rootDir string, options WriteAdmissionOptions) (WriteAdmissionReport, error) {
	options = normalizeWriteAdmissionOptions(options)
	proofReport, err := BuildWriteProofs(rootDir, WriteProofOptions{
		Provider:      options.Provider,
		OperationType: options.OperationType,
		Environment:   options.Environment,
		Limit:         options.Limit,
	})
	if err != nil {
		return WriteAdmissionReport{}, err
	}
	entries := []WriteAdmissionEntry{}
	for _, proof := range proofReport.Proofs {
		entry := writeAdmissionFromProof(proof, options.Target)
		if writeAdmissionMatches(entry, options) {
			entries = append(entries, entry)
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		left := parseTimelineTime(entries[i].CreatedAt)
		right := parseTimelineTime(entries[j].CreatedAt)
		if !left.Equal(right) {
			return left.After(right)
		}
		return entries[i].OperationType+"|"+entries[i].OperationID > entries[j].OperationType+"|"+entries[j].OperationID
	})
	if len(entries) > options.Limit {
		entries = entries[:options.Limit]
	}

	redacted := false
	entries, redacted = redactWriteAdmissionEntries(entries)
	now := time.Now().UTC()
	report := WriteAdmissionReport{
		ID:            "write-admission-report-" + now.Format("20060102150405") + "-" + fmt.Sprintf("%09d", now.UnixNano()%1_000_000_000),
		GeneratedAt:   now.Format(time.RFC3339Nano),
		PolicyID:      defaultWriteAdmissionPolicyID,
		PolicyVersion: defaultWriteAdmissionPolicyVersion,
		Target:        options.Target,
		Filters:       options,
		Entries:       entries,
	}
	report.Summary = buildWriteAdmissionSummary(entries)
	report.Summary.RedactionApplied = redacted || proofReport.Summary.RedactionApplied
	return report, nil
}

func normalizeWriteAdmissionOptions(options WriteAdmissionOptions) WriteAdmissionOptions {
	options.Provider = normalizeType(options.Provider)
	options.OperationType = normalizeType(options.OperationType)
	options.Environment = normalizeType(options.Environment)
	options.Status = normalizeType(options.Status)
	options.Decision = normalizeType(options.Decision)
	options.Target = normalizeType(options.Target)
	if options.Target == "" {
		options.Target = "real_write"
	}
	if options.Limit <= 0 {
		options.Limit = 20
	}
	if options.Limit > 100 {
		options.Limit = 100
	}
	return options
}

func writeAdmissionFromProof(proof WriteProof, target string) WriteAdmissionEntry {
	entry := WriteAdmissionEntry{
		ID:                   "write-admission-" + proof.OperationType + "-" + proof.OperationID,
		ProofID:              proof.ID,
		ProofDecision:        proof.Decision,
		OperationType:        proof.OperationType,
		OperationID:          proof.OperationID,
		Provider:             proof.Provider,
		Environment:          proof.Environment,
		Mode:                 proof.Mode,
		Status:               "ready",
		Decision:             "WRITE_ADMISSION_READY",
		Reasons:              []string{},
		RuleRefs:             []string{},
		SourceRef:            proof.SourceRef,
		DryRun:               proof.DryRun,
		WriteEnabled:         proof.WriteEnabled,
		RehearsalAllowed:     proof.DryRun || proof.Status == "completed" || proof.Status == "blocked",
		ApprovalRequired:     proof.ApprovalRequired,
		ApprovalSatisfied:    proof.ApprovalSatisfied,
		ApprovalID:           proof.ApprovalID,
		SecretRefStatus:      proof.SecretRefStatus,
		ProviderEvidenceRefs: append([]string{}, proof.ProviderEvidenceRefs...),
		LeastPrivilege:       proof.LeastPrivilege,
		ReplayGuard:          proof.ReplayGuard,
		CreatedAt:            proof.CreatedAt,
		Metadata: map[string]any{
			"target":             target,
			"proof_status":       proof.Status,
			"proof_decision":     proof.Decision,
			"proof_reason_count": len(proof.Reasons),
		},
	}
	block := func(decision string, reason string, rule string) {
		entry.Status = "blocked"
		entry.Decision = decision
		entry.Reasons = appendUnique(entry.Reasons, reason)
		entry.RuleRefs = appendUnique(entry.RuleRefs, rule)
	}
	manual := func(decision string, reason string, rule string) {
		if entry.Status != "blocked" {
			entry.Status = "manual_required"
			entry.Decision = decision
		}
		entry.Reasons = appendUnique(entry.Reasons, reason)
		entry.RuleRefs = appendUnique(entry.RuleRefs, rule)
	}
	rehearsalOnly := func(decision string, reason string, rule string) {
		if entry.Status != "blocked" && entry.Status != "manual_required" {
			entry.Status = "rehearsal_only"
			entry.Decision = decision
		}
		entry.Reasons = appendUnique(entry.Reasons, reason)
		entry.RuleRefs = appendUnique(entry.RuleRefs, rule)
		entry.RehearsalAllowed = true
	}

	if proof.Status == "failed" {
		block("WRITE_ADMISSION_SOURCE_FAILED", "write_proof_source_failed", "source_proof_must_not_fail")
	}
	if proof.SecretRefStatus == "missing_or_invalid" {
		block("WRITE_ADMISSION_SECRET_REF_MISSING", "secret_or_auth_ref_missing_or_invalid", "secret_ref_required")
	}
	if writeAdmissionRequiresEvidence(proof) && len(proof.ProviderEvidenceRefs) == 0 {
		block("WRITE_ADMISSION_EVIDENCE_REQUIRED", "provider_evidence_required", "provider_evidence_required")
	}
	if proof.ApprovalRequired && !proof.ApprovalSatisfied {
		manual("WRITE_ADMISSION_APPROVAL_REQUIRED", "approval_required_before_real_write", "approval_required")
	}
	if proof.Environment == "production" {
		manual("WRITE_ADMISSION_PRODUCTION_REVIEW_REQUIRED", "production_real_write_requires_manual_review", "production_review_required")
	}
	if !proof.WriteEnabled || proof.DryRun {
		rehearsalOnly("WRITE_ADMISSION_WRITE_DISABLED", "real_write_requires_write_enabled_non_dry_run_proof", "write_switch_required")
	}
	if proof.Status == "blocked" && entry.Status == "ready" {
		block("WRITE_ADMISSION_SOURCE_BLOCKED", "write_proof_source_blocked:"+proof.Decision, "source_proof_must_be_ready")
	}
	if proof.Status == "manual_required" && entry.Status == "ready" {
		manual("WRITE_ADMISSION_MANUAL_REVIEW_REQUIRED", "write_proof_manual_review_required", "manual_review_required")
	}
	if entry.Status == "ready" {
		entry.Reasons = appendUnique(entry.Reasons, "write_admission_policy_satisfied")
		entry.RuleRefs = appendUnique(entry.RuleRefs, "all_write_gates_satisfied")
	}
	entry.Reasons = compactStrings(entry.Reasons)
	entry.RuleRefs = compactStrings(entry.RuleRefs)
	entry.ProviderEvidenceRefs = compactStrings(entry.ProviderEvidenceRefs)
	return entry
}

func writeAdmissionRequiresEvidence(proof WriteProof) bool {
	switch proof.OperationType {
	case "release_provider_execution", "deployment_execution":
		return true
	default:
		return false
	}
}

func writeAdmissionMatches(entry WriteAdmissionEntry, options WriteAdmissionOptions) bool {
	if options.Provider != "" && normalizeType(entry.Provider) != options.Provider {
		return false
	}
	if options.OperationType != "" && normalizeType(entry.OperationType) != options.OperationType {
		return false
	}
	if options.Environment != "" && normalizeType(entry.Environment) != options.Environment {
		return false
	}
	if options.Status != "" && normalizeType(entry.Status) != options.Status {
		return false
	}
	if options.Decision != "" && normalizeType(entry.Decision) != options.Decision {
		return false
	}
	return true
}

func buildWriteAdmissionSummary(entries []WriteAdmissionEntry) WriteAdmissionSummary {
	summary := WriteAdmissionSummary{
		EntryCount:      len(entries),
		ByOperationType: map[string]int{},
		ByProvider:      map[string]int{},
		ByEnvironment:   map[string]int{},
		ByStatus:        map[string]int{},
		ByDecision:      map[string]int{},
	}
	for _, entry := range entries {
		summary.ByOperationType[entry.OperationType]++
		if entry.Provider != "" {
			summary.ByProvider[entry.Provider]++
		}
		if entry.Environment != "" {
			summary.ByEnvironment[entry.Environment]++
		}
		summary.ByStatus[entry.Status]++
		summary.ByDecision[entry.Decision]++
		switch entry.Status {
		case "ready":
			summary.ReadyCount++
		case "blocked":
			summary.BlockedCount++
		case "manual_required":
			summary.ManualRequiredCount++
		case "rehearsal_only":
			summary.RehearsalOnlyCount++
		}
	}
	return summary
}

func redactWriteAdmissionEntries(entries []WriteAdmissionEntry) ([]WriteAdmissionEntry, bool) {
	out := make([]WriteAdmissionEntry, 0, len(entries))
	redacted := false
	for _, entry := range entries {
		entry.ID = redactStringValue(entry.ID, &redacted)
		entry.ProofID = redactStringValue(entry.ProofID, &redacted)
		entry.OperationID = redactStringValue(entry.OperationID, &redacted)
		entry.SourceRef = redactStringValue(entry.SourceRef, &redacted)
		entry.ApprovalID = redactStringValue(entry.ApprovalID, &redacted)
		var changed bool
		entry.Reasons, changed = redactStrings(entry.Reasons)
		redacted = redacted || changed
		entry.ProviderEvidenceRefs, changed = redactStrings(entry.ProviderEvidenceRefs)
		redacted = redacted || changed
		if entry.Metadata != nil {
			var metadata any
			metadata, changed = redactAny(entry.Metadata)
			redacted = redacted || changed
			if cast, ok := metadata.(map[string]any); ok {
				entry.Metadata = cast
			}
		}
		out = append(out, entry)
	}
	return out, redacted
}
