package operations

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/deployment"
	"moyuan-code/internal/serverresources"
)

type DecisionLedgerOptions struct {
	SourceType  string `json:"source_type,omitempty"`
	Status      string `json:"status,omitempty"`
	Decision    string `json:"decision,omitempty"`
	Environment string `json:"environment,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

type DecisionLedger struct {
	ID          string                `json:"id"`
	GeneratedAt string                `json:"generated_at"`
	Filters     DecisionLedgerOptions `json:"filters"`
	Summary     DecisionLedgerSummary `json:"summary"`
	Entries     []DecisionEntry       `json:"entries"`
}

type DecisionLedgerSummary struct {
	EntryCount       int            `json:"entry_count"`
	EvidenceRefCount int            `json:"evidence_ref_count"`
	BySourceType     map[string]int `json:"by_source_type,omitempty"`
	ByStatus         map[string]int `json:"by_status,omitempty"`
	ByDecision       map[string]int `json:"by_decision,omitempty"`
	ByEnvironment    map[string]int `json:"by_environment,omitempty"`
	AttentionCount   int            `json:"attention_count"`
	RedactionApplied bool           `json:"redaction_applied"`
}

type DecisionEntry struct {
	ID            string         `json:"id"`
	SourceType    string         `json:"source_type"`
	SourceID      string         `json:"source_id"`
	ParentRef     string         `json:"parent_ref,omitempty"`
	Environment   string         `json:"environment,omitempty"`
	Status        string         `json:"status"`
	Decision      string         `json:"decision"`
	Reasons       []string       `json:"reasons,omitempty"`
	PolicyID      string         `json:"policy_id,omitempty"`
	PolicyVersion string         `json:"policy_version,omitempty"`
	PolicySource  string         `json:"policy_source,omitempty"`
	RuleRefs      []string       `json:"rule_refs,omitempty"`
	EvidenceRefs  []string       `json:"evidence_refs,omitempty"`
	CreatedAt     string         `json:"created_at,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

func BuildDecisionLedger(rootDir string, options DecisionLedgerOptions) (DecisionLedger, error) {
	options = normalizeDecisionLedgerOptions(options)
	sourceLimit := options.Limit * 4
	if sourceLimit < 50 {
		sourceLimit = 50
	}
	if sourceLimit > 200 {
		sourceLimit = 200
	}
	entries := []DecisionEntry{}
	add := func(entry DecisionEntry) {
		if decisionEntryMatches(entry, options) {
			entries = append(entries, entry)
		}
	}

	if options.SourceType == "" || options.SourceType == "release_admission" {
		admissions, err := deployment.ListReleaseAdmissions(rootDir, sourceLimit)
		if err != nil {
			return DecisionLedger{}, err
		}
		for _, admission := range admissions {
			add(decisionEntryFromAdmission(admission))
		}
	}

	if options.SourceType == "" || options.SourceType == "maintenance_policy" {
		maintenanceEntries, err := maintenancePolicyDecisionEntries(rootDir, options)
		if err != nil {
			return DecisionLedger{}, err
		}
		for _, entry := range maintenanceEntries {
			add(entry)
		}
	}

	resources := []serverresources.Resource{}
	if options.SourceType == "" || options.SourceType == "resource_readiness" || options.SourceType == "maintenance_policy" {
		list, err := serverresources.List(rootDir)
		if err != nil {
			return DecisionLedger{}, err
		}
		resources = list
	}
	if options.SourceType == "" || options.SourceType == "resource_readiness" {
		for _, resource := range resources {
			add(decisionEntryFromResourceReadiness(serverresources.AssessDeploymentReadiness(resource, resource.Environment), resource))
		}
	}

	if options.SourceType == "" || options.SourceType == "post_deployment_verification" {
		verifications, err := deployment.ListPostDeploymentVerifications(rootDir, sourceLimit)
		if err != nil {
			return DecisionLedger{}, err
		}
		for _, verification := range verifications {
			add(decisionEntryFromPostDeploymentVerification(verification))
		}
	}

	if options.SourceType == "" || options.SourceType == "deployment_risk_handoff" {
		handoffs, err := listDeploymentRiskHandoffs(rootDir, sourceLimit)
		if err != nil {
			return DecisionLedger{}, err
		}
		for _, handoff := range handoffs {
			add(decisionEntryFromRiskHandoff(handoff))
		}
	}

	if options.SourceType == "" || options.SourceType == "deployment_risk_review" {
		reviews, err := listDeploymentRiskReviews(rootDir, sourceLimit)
		if err != nil {
			return DecisionLedger{}, err
		}
		for _, review := range reviews {
			add(decisionEntryFromRiskReview(review))
		}
	}

	entries = dedupeDecisionEntries(entries)
	sort.SliceStable(entries, func(i, j int) bool {
		left := parseTimelineTime(entries[i].CreatedAt)
		right := parseTimelineTime(entries[j].CreatedAt)
		if !left.Equal(right) {
			return left.After(right)
		}
		return entries[i].SourceType+"|"+entries[i].SourceID > entries[j].SourceType+"|"+entries[j].SourceID
	})
	if len(entries) > options.Limit {
		entries = entries[:options.Limit]
	}

	redacted := false
	entries, redacted = redactDecisionEntries(entries)
	now := time.Now().UTC()
	ledger := DecisionLedger{
		ID:          "decision-ledger-" + now.Format("20060102150405") + "-" + fmt.Sprintf("%09d", now.UnixNano()%1_000_000_000),
		GeneratedAt: now.Format(time.RFC3339Nano),
		Filters:     options,
		Entries:     entries,
	}
	ledger.Summary = buildDecisionLedgerSummary(entries)
	ledger.Summary.RedactionApplied = redacted
	return ledger, nil
}

func normalizeDecisionLedgerOptions(options DecisionLedgerOptions) DecisionLedgerOptions {
	options.SourceType = normalizeType(options.SourceType)
	options.Status = normalizeType(options.Status)
	options.Decision = normalizeType(options.Decision)
	options.Environment = normalizeType(options.Environment)
	if options.Limit <= 0 {
		options.Limit = 20
	}
	if options.Limit > 100 {
		options.Limit = 100
	}
	return options
}

func decisionEntryFromAdmission(admission deployment.ReleaseAdmission) DecisionEntry {
	return DecisionEntry{
		ID:            "decision-release-admission-" + admission.ID,
		SourceType:    "release_admission",
		SourceID:      admission.ID,
		ParentRef:     firstNonEmpty(admission.RehearsalID, admission.ExecutionID, admission.DeploymentID, admission.CandidateID),
		Environment:   admission.Environment,
		Status:        admission.Status,
		Decision:      admission.Decision,
		Reasons:       append([]string{}, admission.Reasons...),
		PolicyID:      admission.PolicyID,
		PolicyVersion: admission.PolicyVersion,
		PolicySource:  admission.PolicySource,
		RuleRefs:      admissionRuleRefs(admission.MatchedRules),
		EvidenceRefs:  append([]string{}, admission.EvidenceIDs...),
		CreatedAt:     admission.CreatedAt,
		Metadata: map[string]any{
			"signal_count":       len(admission.Signals),
			"matched_rule_count": len(admission.MatchedRules),
			"policy_blocked":     admission.PolicyDecision.Blocked,
			"manual_required":    admission.PolicyDecision.ManualRequired,
		},
	}
}

func maintenancePolicyDecisionEntries(rootDir string, options DecisionLedgerOptions) ([]DecisionEntry, error) {
	envs := map[string]bool{}
	if options.Environment != "" {
		envs[options.Environment] = true
	} else {
		envs["test_dev"] = true
		envs["production"] = true
		resources, err := serverresources.List(rootDir)
		if err != nil {
			return nil, err
		}
		for _, resource := range resources {
			if strings.TrimSpace(resource.Environment) != "" {
				envs[normalizeType(resource.Environment)] = true
			}
		}
	}
	out := []DecisionEntry{}
	for _, environment := range sortedBoolKeys(envs) {
		pack, err := serverresources.LoadMaintenancePolicyPack(rootDir, environment)
		if err != nil {
			return nil, err
		}
		decision := serverresources.EvaluateMaintenancePolicy(pack, serverresources.MaintenancePolicyContext{
			Environment: environment,
			Action:      "deploy",
		})
		out = append(out, DecisionEntry{
			ID:            "decision-maintenance-policy-" + decision.PolicyID + "-" + environment + "-deploy",
			SourceType:    "maintenance_policy",
			SourceID:      decision.PolicyID + ":" + environment + ":deploy",
			Environment:   decision.Environment,
			Status:        decision.Status,
			Decision:      decision.Decision,
			Reasons:       append([]string{}, decision.Reasons...),
			PolicyID:      decision.PolicyID,
			PolicyVersion: decision.PolicyVersion,
			PolicySource:  decision.PolicySource,
			RuleRefs:      maintenanceRuleRefs(decision.MatchedRules),
			CreatedAt:     decision.RequestedAt,
			Metadata: map[string]any{
				"action":                    decision.Action,
				"within_maintenance_window": decision.WithinMaintenanceWindow,
				"in_freeze_window":          decision.InFreezeWindow,
				"blocked":                   decision.Blocked,
				"manual_required":           decision.ManualRequired,
				"allowed":                   decision.Allowed,
			},
		})
	}
	return out, nil
}

func decisionEntryFromResourceReadiness(readiness serverresources.DeploymentReadiness, resource serverresources.Resource) DecisionEntry {
	return DecisionEntry{
		ID:          "decision-resource-readiness-" + readiness.ResourceID,
		SourceType:  "resource_readiness",
		SourceID:    readiness.ResourceID,
		ParentRef:   firstNonEmpty(resourceLastDeploymentID(resource), resource.Host),
		Environment: readiness.Environment,
		Status:      readiness.Status,
		Decision:    readiness.Decision,
		Reasons:     append([]string{}, readiness.Reasons...),
		CreatedAt:   firstNonEmpty(resource.UpdatedAt, resource.CreatedAt),
		Metadata: map[string]any{
			"target_environment": readiness.TargetEnvironment,
			"manual_required":    readiness.ManualRequired,
			"expiration_state":   readiness.ExpirationState,
			"health_status":      readiness.HealthStatus,
			"resource_status":    resource.Status,
		},
	}
}

func resourceLastDeploymentID(resource serverresources.Resource) string {
	if resource.LastDeployment == nil {
		return ""
	}
	return firstNonEmpty(resource.LastDeployment.ExecutionID, resource.LastDeployment.DeploymentID)
}

func decisionEntryFromPostDeploymentVerification(verification deployment.PostDeploymentVerification) DecisionEntry {
	return DecisionEntry{
		ID:           "decision-post-deployment-verification-" + verification.ID,
		SourceType:   "post_deployment_verification",
		SourceID:     verification.ID,
		ParentRef:    firstNonEmpty(verification.ExecutionID, verification.DeploymentID, verification.ReleaseID),
		Environment:  verification.Environment,
		Status:       verification.Status,
		Decision:     verification.Decision,
		Reasons:      append([]string{}, verification.Reasons...),
		EvidenceRefs: append([]string{}, verification.EvidenceIDs...),
		CreatedAt:    verification.CreatedAt,
		Metadata: map[string]any{
			"failure_class":              verification.FailureClass,
			"risk_handoff_recommended":   verification.RiskHandoffRecommended,
			"risk_source_type":           verification.RiskSourceType,
			"risk_source_id":             verification.RiskSourceID,
			"monitor_decision":           verification.MonitorDecision,
			"smoke_decision":             verification.SmokeDecision,
			"rollback_required":          verification.RollbackRequired,
			"post_deployment_history_id": verification.HistoryID,
		},
	}
}

func decisionEntryFromRiskHandoff(handoff deploymentRiskHandoffTimeline) DecisionEntry {
	return DecisionEntry{
		ID:           "decision-deployment-risk-handoff-" + handoff.ID,
		SourceType:   "deployment_risk_handoff",
		SourceID:     handoff.ID,
		ParentRef:    handoff.SourceID,
		Status:       handoff.Status,
		Decision:     handoff.Decision,
		Reasons:      append([]string{}, handoff.Reasons...),
		EvidenceRefs: append([]string{}, handoff.EvidenceRefs...),
		CreatedAt:    firstNonEmpty(handoff.ReviewedAt, handoff.CreatedAt),
		Metadata: map[string]any{
			"risk_source_type": handoff.SourceType,
			"failure_class":    handoff.FailureClass,
			"review_required":  handoff.ReviewRequired,
			"review_id":        handoff.ReviewID,
			"review_decision":  handoff.ReviewDecision,
			"repair_plan_id":   handoff.RepairPlanID,
			"bug_candidate_id": handoff.BugCandidateID,
		},
	}
}

func decisionEntryFromRiskReview(review deploymentRiskReviewTimeline) DecisionEntry {
	return DecisionEntry{
		ID:           "decision-deployment-risk-review-" + review.ID,
		SourceType:   "deployment_risk_review",
		SourceID:     review.ID,
		ParentRef:    firstNonEmpty(review.HandoffID, review.SourceID),
		Status:       review.Status,
		Decision:     review.Decision,
		Reasons:      compactStrings([]string{review.Reason, review.NextStep}),
		EvidenceRefs: append([]string{}, review.EvidenceRefs...),
		CreatedAt:    review.CreatedAt,
		Metadata: map[string]any{
			"risk_source_type": review.SourceType,
			"risk_source_id":   review.SourceID,
			"failure_class":    review.FailureClass,
			"reviewer_id":      review.ReviewerID,
			"next_step":        review.NextStep,
			"repair_plan_id":   review.RepairPlanID,
			"bug_candidate_id": review.BugCandidateID,
		},
	}
}

func decisionEntryMatches(entry DecisionEntry, options DecisionLedgerOptions) bool {
	if options.SourceType != "" && normalizeType(entry.SourceType) != options.SourceType {
		return false
	}
	if options.Status != "" && normalizeType(entry.Status) != options.Status {
		return false
	}
	if options.Decision != "" && normalizeType(entry.Decision) != options.Decision {
		return false
	}
	if options.Environment != "" && normalizeType(entry.Environment) != options.Environment {
		return false
	}
	return true
}

func dedupeDecisionEntries(entries []DecisionEntry) []DecisionEntry {
	seen := map[string]bool{}
	out := []DecisionEntry{}
	for _, entry := range entries {
		key := entry.SourceType + "|" + entry.SourceID
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, entry)
	}
	return out
}

func redactDecisionEntries(entries []DecisionEntry) ([]DecisionEntry, bool) {
	out := make([]DecisionEntry, 0, len(entries))
	redacted := false
	for _, entry := range entries {
		entry.ID = redactStringValue(entry.ID, &redacted)
		entry.SourceType = redactStringValue(entry.SourceType, &redacted)
		entry.SourceID = redactStringValue(entry.SourceID, &redacted)
		entry.ParentRef = redactStringValue(entry.ParentRef, &redacted)
		entry.Environment = redactStringValue(entry.Environment, &redacted)
		entry.Status = redactStringValue(entry.Status, &redacted)
		entry.Decision = redactStringValue(entry.Decision, &redacted)
		entry.PolicyID = redactStringValue(entry.PolicyID, &redacted)
		entry.PolicyVersion = redactStringValue(entry.PolicyVersion, &redacted)
		entry.PolicySource = redactStringValue(entry.PolicySource, &redacted)
		entry.CreatedAt = redactStringValue(entry.CreatedAt, &redacted)
		var changed bool
		entry.Reasons, changed = redactStrings(entry.Reasons)
		redacted = redacted || changed
		entry.RuleRefs, changed = redactStrings(entry.RuleRefs)
		redacted = redacted || changed
		entry.EvidenceRefs, changed = redactStrings(entry.EvidenceRefs)
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

func buildDecisionLedgerSummary(entries []DecisionEntry) DecisionLedgerSummary {
	summary := DecisionLedgerSummary{
		EntryCount:    len(entries),
		BySourceType:  map[string]int{},
		ByStatus:      map[string]int{},
		ByDecision:    map[string]int{},
		ByEnvironment: map[string]int{},
	}
	evidenceRefs := []string{}
	for _, entry := range entries {
		increment(summary.BySourceType, entry.SourceType)
		increment(summary.ByStatus, entry.Status)
		increment(summary.ByDecision, entry.Decision)
		increment(summary.ByEnvironment, environmentOrAll(entry.Environment))
		if isAttentionAuditItem(entry.Status, entry.Decision) {
			summary.AttentionCount++
		}
		for _, ref := range entry.EvidenceRefs {
			evidenceRefs = appendUnique(evidenceRefs, ref)
		}
	}
	summary.EvidenceRefCount = len(evidenceRefs)
	return summary
}

func admissionRuleRefs(rules []deployment.AdmissionRuleMatch) []string {
	refs := []string{}
	for _, rule := range rules {
		refs = appendUnique(refs, firstNonEmpty(rule.PolicyID, "policy")+":"+firstNonEmpty(rule.RuleID, "rule"))
	}
	return refs
}

func maintenanceRuleRefs(rules []serverresources.MaintenancePolicyRuleMatch) []string {
	refs := []string{}
	for _, rule := range rules {
		refs = appendUnique(refs, firstNonEmpty(rule.PolicyID, "policy")+":"+firstNonEmpty(rule.RuleID, "rule"))
	}
	return refs
}

func sortedBoolKeys(values map[string]bool) []string {
	keys := []string{}
	for key := range values {
		if strings.TrimSpace(key) != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}
