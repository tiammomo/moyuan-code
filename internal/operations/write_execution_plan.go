package operations

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/evidence"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/workspace"
)

type WriteExecutionPlanOptions struct {
	ReviewPacketID string `json:"review_packet_id,omitempty"`
	Mode           string `json:"mode,omitempty"`
	ApprovalID     string `json:"approval_id,omitempty"`
	RequestedBy    string `json:"requested_by,omitempty"`
	Status         string `json:"status,omitempty"`
	Decision       string `json:"decision,omitempty"`
	Limit          int    `json:"limit,omitempty"`
}

type WriteExecutionPlanReport struct {
	ID          string                    `json:"id"`
	GeneratedAt string                    `json:"generated_at"`
	Filters     WriteExecutionPlanOptions `json:"filters"`
	Summary     WriteExecutionPlanSummary `json:"summary"`
	Plans       []WriteExecutionPlan      `json:"plans"`
}

type WriteExecutionPlanSummary struct {
	PlanCount           int            `json:"plan_count"`
	ReadyCount          int            `json:"ready_count"`
	PlannedCount        int            `json:"planned_count"`
	BlockedCount        int            `json:"blocked_count"`
	ManualRequiredCount int            `json:"manual_required_count"`
	ExternalWriteCount  int            `json:"external_write_count"`
	ByMode              map[string]int `json:"by_mode,omitempty"`
	ByStatus            map[string]int `json:"by_status,omitempty"`
	ByDecision          map[string]int `json:"by_decision,omitempty"`
}

type WriteExecutionPlan struct {
	ID                     string         `json:"id"`
	ReviewPacketID         string         `json:"review_packet_id,omitempty"`
	OperationType          string         `json:"operation_type,omitempty"`
	OperationID            string         `json:"operation_id,omitempty"`
	Provider               string         `json:"provider,omitempty"`
	Environment            string         `json:"environment,omitempty"`
	Mode                   string         `json:"mode"`
	Status                 string         `json:"status"`
	Decision               string         `json:"decision"`
	Reasons                []string       `json:"reasons,omitempty"`
	RuleRefs               []string       `json:"rule_refs,omitempty"`
	EvidenceRefs           []string       `json:"evidence_refs,omitempty"`
	ApprovalID             string         `json:"approval_id,omitempty"`
	RequestedBy            string         `json:"requested_by,omitempty"`
	ApplyAllowed           bool           `json:"apply_allowed"`
	ExternalWritePerformed bool           `json:"external_write_performed"`
	CreatedAt              string         `json:"created_at"`
	Metadata               map[string]any `json:"metadata,omitempty"`
}

func CreateWriteExecutionPlans(rootDir string, options WriteExecutionPlanOptions) (WriteExecutionPlanReport, error) {
	options = normalizeWriteExecutionPlanOptions(options)
	if options.Mode == "" {
		options.Mode = "preview"
	}
	plan, err := writeExecutionPlanFromReviewPacket(rootDir, options)
	if err != nil {
		return WriteExecutionPlanReport{}, err
	}
	if writeExecutionPlanMatches(plan, options) {
		plan, err = finishWriteExecutionPlan(rootDir, plan)
		if err != nil {
			return WriteExecutionPlanReport{}, err
		}
	} else {
		plan = WriteExecutionPlan{}
	}
	plans := []WriteExecutionPlan{}
	if plan.ID != "" {
		plans = append(plans, plan)
	}
	now := time.Now().UTC()
	report := WriteExecutionPlanReport{
		ID:          "write-execution-plan-report-" + now.Format("20060102150405") + "-" + fmt.Sprintf("%09d", now.UnixNano()%1_000_000_000),
		GeneratedAt: now.Format(time.RFC3339Nano),
		Filters:     options,
		Plans:       plans,
	}
	report.Summary = buildWriteExecutionPlanSummary(plans)
	return report, nil
}

func ListWriteExecutionPlans(rootDir string, options WriteExecutionPlanOptions) (WriteExecutionPlanReport, error) {
	options = normalizeWriteExecutionPlanOptions(options)
	if err := fsutil.EnsureDir(writeExecutionPlanDir(rootDir)); err != nil {
		return WriteExecutionPlanReport{}, err
	}
	entries, err := os.ReadDir(writeExecutionPlanDir(rootDir))
	if err != nil {
		return WriteExecutionPlanReport{}, err
	}
	plans := []WriteExecutionPlan{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var plan WriteExecutionPlan
		found, err := fsutil.ReadJSON(filepath.Join(writeExecutionPlanDir(rootDir), entry.Name()), &plan)
		if err != nil {
			return WriteExecutionPlanReport{}, err
		}
		if found && plan.ID != "" && writeExecutionPlanMatches(plan, options) {
			plans = append(plans, plan)
		}
	}
	sortWriteExecutionPlans(plans)
	if len(plans) > options.Limit {
		plans = plans[:options.Limit]
	}
	now := time.Now().UTC()
	report := WriteExecutionPlanReport{
		ID:          "write-execution-plan-list-" + now.Format("20060102150405") + "-" + fmt.Sprintf("%09d", now.UnixNano()%1_000_000_000),
		GeneratedAt: now.Format(time.RFC3339Nano),
		Filters:     options,
		Plans:       plans,
	}
	report.Summary = buildWriteExecutionPlanSummary(plans)
	return report, nil
}

func LoadWriteExecutionPlan(rootDir string, id string) (WriteExecutionPlan, bool, error) {
	var plan WriteExecutionPlan
	found, err := fsutil.ReadJSON(writeExecutionPlanPath(rootDir, strings.TrimSpace(id)), &plan)
	return plan, found, err
}

func writeExecutionPlanFromReviewPacket(rootDir string, options WriteExecutionPlanOptions) (WriteExecutionPlan, error) {
	now := time.Now().UTC()
	plan := WriteExecutionPlan{
		ID:                     "write-execution-plan-" + firstNonEmpty(options.ReviewPacketID, "missing-review-packet") + "-" + options.Mode + "-" + now.Format("20060102150405") + "-" + fmt.Sprintf("%09d", now.UnixNano()%1_000_000_000),
		ReviewPacketID:         options.ReviewPacketID,
		Mode:                   options.Mode,
		Status:                 "planned",
		Decision:               "WRITE_EXECUTION_PREVIEW_READY",
		Reasons:                []string{"write_execution_plan_created"},
		RuleRefs:               []string{"write_execution_plan_no_external_write_in_phase22"},
		EvidenceRefs:           []string{},
		ApprovalID:             options.ApprovalID,
		RequestedBy:            options.RequestedBy,
		ApplyAllowed:           false,
		ExternalWritePerformed: false,
		CreatedAt:              now.Format(time.RFC3339Nano),
		Metadata: map[string]any{
			"external_write_performed": false,
			"write_switch_env":         "MOYUAN_ALLOW_REAL_WRITE",
		},
	}
	blockPlan := func(decision string, reason string, rule string) {
		plan.Status = "blocked"
		plan.Decision = decision
		plan.Reasons = appendUnique(plan.Reasons, reason)
		plan.RuleRefs = appendUnique(plan.RuleRefs, rule)
	}
	manualPlan := func(decision string, reason string, rule string) {
		if plan.Status != "blocked" {
			plan.Status = "manual_required"
			plan.Decision = decision
		}
		plan.Reasons = appendUnique(plan.Reasons, reason)
		plan.RuleRefs = appendUnique(plan.RuleRefs, rule)
	}

	if options.Mode != "preview" && options.Mode != "apply" {
		blockPlan("WRITE_EXECUTION_MODE_UNSUPPORTED", "write_execution_mode_unsupported:"+options.Mode, "write_execution_mode_must_be_preview_or_apply")
		return normalizeWriteExecutionPlan(plan), nil
	}
	if options.ReviewPacketID == "" {
		blockPlan("WRITE_EXECUTION_REVIEW_PACKET_REQUIRED", "write_review_packet_id_required", "write_review_packet_required")
		return normalizeWriteExecutionPlan(plan), nil
	}
	packet, found, err := LoadWriteReviewPacket(rootDir, options.ReviewPacketID)
	if err != nil {
		return WriteExecutionPlan{}, err
	}
	if !found {
		blockPlan("WRITE_EXECUTION_REVIEW_PACKET_MISSING", "write_review_packet_missing:"+options.ReviewPacketID, "write_review_packet_must_exist")
		return normalizeWriteExecutionPlan(plan), nil
	}

	plan.OperationType = packet.OperationType
	plan.OperationID = packet.OperationID
	plan.Provider = packet.Provider
	plan.Environment = packet.Environment
	plan.RuleRefs = appendUniqueStrings(plan.RuleRefs, packet.RuleRefs...)
	plan.EvidenceRefs = appendUniqueStrings(plan.EvidenceRefs, packet.EvidenceRefs...)
	plan.Metadata["review_packet_status"] = packet.Status
	plan.Metadata["review_packet_decision"] = packet.Decision
	plan.Metadata["provider_requirement_id"] = packet.ProviderRequirementID
	plan.Metadata["remote_rehearsal_id"] = packet.RemoteRehearsalID
	plan.Metadata["queue_item_count"] = len(packet.QueueItems)

	if packet.Status == "blocked" {
		blockPlan("WRITE_EXECUTION_REVIEW_PACKET_BLOCKED", "write_review_packet_blocked:"+packet.Decision, "write_review_packet_must_be_ready")
		return normalizeWriteExecutionPlan(plan), nil
	}
	if packet.Status != "ready" {
		manualPlan("WRITE_EXECUTION_REVIEW_PACKET_NOT_READY", "write_review_packet_not_ready:"+packet.Decision, "write_review_packet_must_be_ready")
		return normalizeWriteExecutionPlan(plan), nil
	}
	if options.Mode == "preview" {
		plan.Status = "planned"
		plan.Decision = "WRITE_EXECUTION_PREVIEW_READY"
		plan.Reasons = appendUnique(plan.Reasons, "write_execution_preview_ready")
		plan.RuleRefs = appendUnique(plan.RuleRefs, "preview_does_not_require_real_write_switch")
		return normalizeWriteExecutionPlan(plan), nil
	}

	if options.ApprovalID == "" {
		manualPlan("WRITE_EXECUTION_APPROVAL_REQUIRED", "approval_id_required_for_apply", "apply_requires_approval_id")
		return normalizeWriteExecutionPlan(plan), nil
	}
	if os.Getenv("MOYUAN_ALLOW_REAL_WRITE") != "1" {
		blockPlan("WRITE_EXECUTION_APPLY_SWITCH_DISABLED", "real_write_switch_disabled", "apply_requires_moyuan_allow_real_write")
		return normalizeWriteExecutionPlan(plan), nil
	}
	plan.Status = "ready"
	plan.Decision = "WRITE_EXECUTION_APPLY_READY"
	plan.ApplyAllowed = true
	plan.Reasons = appendUnique(plan.Reasons, "write_execution_apply_ready")
	plan.RuleRefs = appendUnique(plan.RuleRefs, "apply_ready_but_external_write_not_executed_in_phase22")
	return normalizeWriteExecutionPlan(plan), nil
}

func normalizeWriteExecutionPlanOptions(options WriteExecutionPlanOptions) WriteExecutionPlanOptions {
	options.ReviewPacketID = strings.TrimSpace(options.ReviewPacketID)
	options.Mode = normalizeType(options.Mode)
	options.ApprovalID = strings.TrimSpace(options.ApprovalID)
	options.RequestedBy = strings.TrimSpace(options.RequestedBy)
	options.Status = normalizeType(options.Status)
	options.Decision = normalizeType(options.Decision)
	if options.Limit <= 0 {
		options.Limit = 20
	}
	if options.Limit > 100 {
		options.Limit = 100
	}
	return options
}

func normalizeWriteExecutionPlan(plan WriteExecutionPlan) WriteExecutionPlan {
	plan.OperationType = normalizeType(plan.OperationType)
	plan.Provider = normalizeType(plan.Provider)
	plan.Environment = normalizeType(plan.Environment)
	plan.Mode = normalizeType(plan.Mode)
	plan.Status = normalizeType(plan.Status)
	plan.RuleRefs = compactStrings(plan.RuleRefs)
	plan.EvidenceRefs = compactStrings(plan.EvidenceRefs)
	return plan
}

func writeExecutionPlanMatches(plan WriteExecutionPlan, options WriteExecutionPlanOptions) bool {
	if options.ReviewPacketID != "" && plan.ReviewPacketID != options.ReviewPacketID {
		return false
	}
	if options.Mode != "" && normalizeType(plan.Mode) != options.Mode {
		return false
	}
	if options.Status != "" && normalizeType(plan.Status) != options.Status {
		return false
	}
	if options.Decision != "" && normalizeType(plan.Decision) != options.Decision {
		return false
	}
	return true
}

func buildWriteExecutionPlanSummary(plans []WriteExecutionPlan) WriteExecutionPlanSummary {
	summary := WriteExecutionPlanSummary{
		PlanCount:  len(plans),
		ByMode:     map[string]int{},
		ByStatus:   map[string]int{},
		ByDecision: map[string]int{},
	}
	for _, plan := range plans {
		summary.ByMode[plan.Mode]++
		summary.ByStatus[plan.Status]++
		summary.ByDecision[plan.Decision]++
		if plan.ExternalWritePerformed {
			summary.ExternalWriteCount++
		}
		switch plan.Status {
		case "ready":
			summary.ReadyCount++
		case "planned":
			summary.PlannedCount++
		case "blocked":
			summary.BlockedCount++
		case "manual_required":
			summary.ManualRequiredCount++
		}
	}
	return summary
}

func sortWriteExecutionPlans(plans []WriteExecutionPlan) {
	sort.SliceStable(plans, func(i, j int) bool {
		left := parseTimelineTime(plans[i].CreatedAt)
		right := parseTimelineTime(plans[j].CreatedAt)
		if !left.Equal(right) {
			return left.After(right)
		}
		return plans[i].ID > plans[j].ID
	})
}

func finishWriteExecutionPlan(rootDir string, plan WriteExecutionPlan) (WriteExecutionPlan, error) {
	plan = normalizeWriteExecutionPlan(plan)
	if err := fsutil.EnsureDir(writeExecutionPlanDir(rootDir)); err != nil {
		return WriteExecutionPlan{}, err
	}
	if err := fsutil.WriteJSON(writeExecutionPlanPath(rootDir, plan.ID), plan); err != nil {
		return WriteExecutionPlan{}, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "write-execution-plans.jsonl"), plan); err != nil {
		return WriteExecutionPlan{}, err
	}
	_ = logging.Log(rootDir, "release", "operations.write_execution_plan.created", map[string]any{
		"plan_id":                  plan.ID,
		"review_packet_id":         plan.ReviewPacketID,
		"operation_id":             plan.OperationID,
		"mode":                     plan.Mode,
		"status":                   plan.Status,
		"decision":                 plan.Decision,
		"external_write_performed": plan.ExternalWritePerformed,
	})
	if _, err := evidence.Add(rootDir, evidence.AddOptions{
		ParentType:  "write_execution_plan",
		ParentID:    plan.ID,
		SubjectType: firstNonEmpty(plan.OperationType, "write_execution_plan"),
		SubjectID:   firstNonEmpty(plan.OperationID, plan.ID),
		Operation:   "operations.write_execution_plan.create",
		Status:      plan.Status,
		Decision:    plan.Decision,
		Reasons:     plan.Reasons,
		Source:      "operations",
		Artifacts: []evidence.ArtifactRef{{
			Kind: "write_execution_plan",
			ID:   plan.ID,
			Path: filepath.ToSlash(filepath.Join(".moyuan", "lifecycle", "deployments", "write-execution-plans", plan.ID+".json")),
		}},
	}); err != nil {
		return WriteExecutionPlan{}, err
	}
	return plan, nil
}

func writeExecutionPlanDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "write-execution-plans")
}

func writeExecutionPlanPath(rootDir string, id string) string {
	return filepath.Join(writeExecutionPlanDir(rootDir), id+".json")
}
