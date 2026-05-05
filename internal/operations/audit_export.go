package operations

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/deployment"
	"moyuan-code/internal/secrets"
	"moyuan-code/internal/serverresources"
)

type AuditExportOptions struct {
	Type        string `json:"type,omitempty"`
	Status      string `json:"status,omitempty"`
	Decision    string `json:"decision,omitempty"`
	Environment string `json:"environment,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Format      string `json:"format,omitempty"`
}

type AuditExport struct {
	ID                          string                          `json:"id"`
	Format                      string                          `json:"format"`
	GeneratedAt                 string                          `json:"generated_at"`
	Filters                     TimelineOptions                 `json:"filters"`
	Summary                     AuditSummary                    `json:"summary"`
	Timeline                    []TimelineItem                  `json:"timeline"`
	PostDeploymentVerifications []AuditVerification             `json:"post_deployment_verifications,omitempty"`
	ResourceDeploymentRefs      []serverresources.DeploymentRef `json:"resource_deployment_refs,omitempty"`
	EvidenceRefs                []string                        `json:"evidence_refs,omitempty"`
	Markdown                    string                          `json:"markdown,omitempty"`
}

type AuditSummary struct {
	TimelineItemCount               int            `json:"timeline_item_count"`
	PostDeploymentVerificationCount int            `json:"post_deployment_verification_count"`
	ResourceDeploymentRefCount      int            `json:"resource_deployment_ref_count"`
	EvidenceRefCount                int            `json:"evidence_ref_count"`
	RiskHandoffRecommendedCount     int            `json:"risk_handoff_recommended_count"`
	AttentionItemCount              int            `json:"attention_item_count"`
	ByType                          map[string]int `json:"by_type,omitempty"`
	ByStatus                        map[string]int `json:"by_status,omitempty"`
	ByDecision                      map[string]int `json:"by_decision,omitempty"`
	ByEnvironment                   map[string]int `json:"by_environment,omitempty"`
	RedactionApplied                bool           `json:"redaction_applied"`
}

type AuditVerification struct {
	ID                     string   `json:"id"`
	ExecutionID            string   `json:"execution_id,omitempty"`
	DeploymentID           string   `json:"deployment_id,omitempty"`
	ReleaseID              string   `json:"release_id,omitempty"`
	Environment            string   `json:"environment,omitempty"`
	Status                 string   `json:"status"`
	Decision               string   `json:"decision"`
	Reasons                []string `json:"reasons,omitempty"`
	FailureClass           string   `json:"failure_class,omitempty"`
	RiskHandoffRecommended bool     `json:"risk_handoff_recommended"`
	RiskSourceType         string   `json:"risk_source_type,omitempty"`
	RiskSourceID           string   `json:"risk_source_id,omitempty"`
	EvidenceIDs            []string `json:"evidence_ids,omitempty"`
	CreatedAt              string   `json:"created_at"`
}

func ExportAudit(rootDir string, options AuditExportOptions) (AuditExport, error) {
	options = normalizeAuditExportOptions(options)
	filters := TimelineOptions{
		Type:        options.Type,
		Status:      options.Status,
		Decision:    options.Decision,
		Environment: options.Environment,
		Limit:       options.Limit,
	}
	items, err := Timeline(rootDir, filters)
	if err != nil {
		return AuditExport{}, err
	}
	var redacted bool
	items, redacted = redactTimelineItems(items)

	sourceLimit := options.Limit * 4
	if sourceLimit < 50 {
		sourceLimit = 50
	}
	if sourceLimit > 200 {
		sourceLimit = 200
	}
	verifications, verificationRedacted, err := auditVerifications(rootDir, options, sourceLimit)
	if err != nil {
		return AuditExport{}, err
	}
	refs, refRedacted, err := auditDeploymentRefs(rootDir, options, sourceLimit)
	if err != nil {
		return AuditExport{}, err
	}
	evidenceRefs := collectAuditEvidenceRefs(items, verifications)
	evidenceRefs, evidenceRedacted := redactStrings(evidenceRefs)

	now := time.Now().UTC()
	report := AuditExport{
		ID:                          "operations-audit-export-" + now.Format("20060102150405") + "-" + fmt.Sprintf("%09d", now.UnixNano()%1_000_000_000),
		Format:                      options.Format,
		GeneratedAt:                 now.Format(time.RFC3339Nano),
		Filters:                     filters,
		Timeline:                    items,
		PostDeploymentVerifications: verifications,
		ResourceDeploymentRefs:      refs,
		EvidenceRefs:                evidenceRefs,
	}
	report.Summary = buildAuditSummary(report)
	report.Summary.RedactionApplied = redacted || verificationRedacted || refRedacted || evidenceRedacted
	if options.Format == "markdown" {
		report.Markdown = renderAuditMarkdown(report)
	}
	return report, nil
}

func normalizeAuditExportOptions(options AuditExportOptions) AuditExportOptions {
	timeline := normalizeTimelineOptions(TimelineOptions{
		Type:        options.Type,
		Status:      options.Status,
		Decision:    options.Decision,
		Environment: options.Environment,
		Limit:       options.Limit,
	})
	options.Type = timeline.Type
	options.Status = timeline.Status
	options.Decision = timeline.Decision
	options.Environment = timeline.Environment
	options.Limit = timeline.Limit
	options.Format = normalizeType(options.Format)
	if options.Format == "" {
		options.Format = "json"
	}
	if options.Format != "json" && options.Format != "markdown" {
		options.Format = "json"
	}
	return options
}

func auditVerifications(rootDir string, options AuditExportOptions, limit int) ([]AuditVerification, bool, error) {
	if options.Type != "" && options.Type != "post_deployment_verification" {
		return nil, false, nil
	}
	raw, err := deployment.ListPostDeploymentVerifications(rootDir, limit)
	if err != nil {
		return nil, false, err
	}
	out := []AuditVerification{}
	redacted := false
	for _, item := range raw {
		if !auditMatches("post_deployment_verification", item.Status, item.Decision, item.Environment, options) {
			continue
		}
		reasons, r := redactStrings(item.Reasons)
		evidenceIDs, e := redactStrings(item.EvidenceIDs)
		out = append(out, AuditVerification{
			ID:                     redactStringValue(item.ID, &redacted),
			ExecutionID:            redactStringValue(item.ExecutionID, &redacted),
			DeploymentID:           redactStringValue(item.DeploymentID, &redacted),
			ReleaseID:              redactStringValue(item.ReleaseID, &redacted),
			Environment:            redactStringValue(item.Environment, &redacted),
			Status:                 redactStringValue(item.Status, &redacted),
			Decision:               redactStringValue(item.Decision, &redacted),
			Reasons:                reasons,
			FailureClass:           redactStringValue(item.FailureClass, &redacted),
			RiskHandoffRecommended: item.RiskHandoffRecommended,
			RiskSourceType:         redactStringValue(item.RiskSourceType, &redacted),
			RiskSourceID:           redactStringValue(item.RiskSourceID, &redacted),
			EvidenceIDs:            evidenceIDs,
			CreatedAt:              redactStringValue(item.CreatedAt, &redacted),
		})
		redacted = redacted || r || e
	}
	return out, redacted, nil
}

func auditDeploymentRefs(rootDir string, options AuditExportOptions, limit int) ([]serverresources.DeploymentRef, bool, error) {
	if options.Type != "" && options.Type != "resource_deployment_ref" {
		return nil, false, nil
	}
	raw, err := serverresources.ListDeploymentReferences(rootDir, limit)
	if err != nil {
		return nil, false, err
	}
	out := []serverresources.DeploymentRef{}
	redacted := false
	for _, item := range raw {
		if !auditMatches("resource_deployment_ref", item.Status, item.Decision, item.Environment, options) {
			continue
		}
		item.ID = redactStringValue(item.ID, &redacted)
		item.ResourceID = redactStringValue(item.ResourceID, &redacted)
		item.Kind = redactStringValue(item.Kind, &redacted)
		item.DeploymentID = redactStringValue(item.DeploymentID, &redacted)
		item.ExecutionID = redactStringValue(item.ExecutionID, &redacted)
		item.ReleaseID = redactStringValue(item.ReleaseID, &redacted)
		item.Environment = redactStringValue(item.Environment, &redacted)
		item.Mode = redactStringValue(item.Mode, &redacted)
		item.Status = redactStringValue(item.Status, &redacted)
		item.Decision = redactStringValue(item.Decision, &redacted)
		item.RecordedAt = redactStringValue(item.RecordedAt, &redacted)
		out = append(out, item)
	}
	return out, redacted, nil
}

func auditMatches(itemType string, status string, decision string, environment string, options AuditExportOptions) bool {
	if options.Type != "" && normalizeType(itemType) != options.Type {
		return false
	}
	if options.Status != "" && normalizeType(status) != options.Status {
		return false
	}
	if options.Decision != "" && normalizeType(decision) != options.Decision {
		return false
	}
	if options.Environment != "" && normalizeType(environment) != options.Environment {
		return false
	}
	return true
}

func redactTimelineItems(items []TimelineItem) ([]TimelineItem, bool) {
	out := make([]TimelineItem, 0, len(items))
	redacted := false
	for _, item := range items {
		item.ID = redactStringValue(item.ID, &redacted)
		item.Type = redactStringValue(item.Type, &redacted)
		item.Operation = redactStringValue(item.Operation, &redacted)
		item.Status = redactStringValue(item.Status, &redacted)
		item.Decision = redactStringValue(item.Decision, &redacted)
		item.PrimaryRef = redactStringValue(item.PrimaryRef, &redacted)
		item.SecondaryRef = redactStringValue(item.SecondaryRef, &redacted)
		item.Environment = redactStringValue(item.Environment, &redacted)
		item.Timestamp = redactStringValue(item.Timestamp, &redacted)
		var changed bool
		item.Reasons, changed = redactStrings(item.Reasons)
		redacted = redacted || changed
		item.EvidenceRefs, changed = redactStrings(item.EvidenceRefs)
		redacted = redacted || changed
		if item.Metadata != nil {
			var metadata any
			metadata, changed = redactAny(item.Metadata)
			redacted = redacted || changed
			if cast, ok := metadata.(map[string]any); ok {
				item.Metadata = cast
			}
		}
		out = append(out, item)
	}
	return out, redacted
}

func collectAuditEvidenceRefs(items []TimelineItem, verifications []AuditVerification) []string {
	refs := []string{}
	for _, item := range items {
		for _, ref := range item.EvidenceRefs {
			refs = appendUnique(refs, ref)
		}
	}
	for _, verification := range verifications {
		for _, ref := range verification.EvidenceIDs {
			refs = appendUnique(refs, ref)
		}
	}
	sort.Strings(refs)
	return refs
}

func buildAuditSummary(report AuditExport) AuditSummary {
	summary := AuditSummary{
		TimelineItemCount:               len(report.Timeline),
		PostDeploymentVerificationCount: len(report.PostDeploymentVerifications),
		ResourceDeploymentRefCount:      len(report.ResourceDeploymentRefs),
		EvidenceRefCount:                len(report.EvidenceRefs),
		ByType:                          map[string]int{},
		ByStatus:                        map[string]int{},
		ByDecision:                      map[string]int{},
		ByEnvironment:                   map[string]int{},
	}
	for _, item := range report.Timeline {
		increment(summary.ByType, item.Type)
		increment(summary.ByStatus, item.Status)
		increment(summary.ByDecision, item.Decision)
		increment(summary.ByEnvironment, environmentOrAll(item.Environment))
		if isAttentionAuditItem(item.Status, item.Decision) {
			summary.AttentionItemCount++
		}
	}
	for _, verification := range report.PostDeploymentVerifications {
		if verification.RiskHandoffRecommended {
			summary.RiskHandoffRecommendedCount++
		}
	}
	return summary
}

func renderAuditMarkdown(report AuditExport) string {
	var b strings.Builder
	b.WriteString("# Operations Audit Export\n\n")
	b.WriteString("- Export ID: " + md(report.ID) + "\n")
	b.WriteString("- Generated At: " + md(report.GeneratedAt) + "\n")
	b.WriteString("- Format: " + md(report.Format) + "\n")
	b.WriteString("- Filters: " + md(renderFilters(report.Filters)) + "\n")
	b.WriteString("- Timeline Items: " + fmt.Sprintf("%d", report.Summary.TimelineItemCount) + "\n")
	b.WriteString("- Evidence Refs: " + fmt.Sprintf("%d", report.Summary.EvidenceRefCount) + "\n")
	b.WriteString("- Attention Items: " + fmt.Sprintf("%d", report.Summary.AttentionItemCount) + "\n")
	b.WriteString("- Risk Handoff Recommended: " + fmt.Sprintf("%d", report.Summary.RiskHandoffRecommendedCount) + "\n\n")

	b.WriteString("## Timeline\n\n")
	if len(report.Timeline) == 0 {
		b.WriteString("No timeline items matched filters.\n\n")
	} else {
		b.WriteString("| Time | Type | Status | Decision | Primary Ref | Environment | Reasons |\n")
		b.WriteString("| --- | --- | --- | --- | --- | --- | --- |\n")
		for _, item := range report.Timeline {
			b.WriteString("| " + md(item.Timestamp) + " | " + md(item.Type) + " | " + md(item.Status) + " | " + md(item.Decision) + " | " + md(item.PrimaryRef) + " | " + md(item.Environment) + " | " + md(strings.Join(item.Reasons, "; ")) + " |\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("## Post Deployment Verifications\n\n")
	if len(report.PostDeploymentVerifications) == 0 {
		b.WriteString("No post-deployment verifications matched filters.\n\n")
	} else {
		b.WriteString("| Created At | Verification | Status | Decision | Execution | Risk Handoff |\n")
		b.WriteString("| --- | --- | --- | --- | --- | --- |\n")
		for _, item := range report.PostDeploymentVerifications {
			b.WriteString("| " + md(item.CreatedAt) + " | " + md(item.ID) + " | " + md(item.Status) + " | " + md(item.Decision) + " | " + md(item.ExecutionID) + " | " + md(fmt.Sprintf("%v", item.RiskHandoffRecommended)) + " |\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("## Resource Deployment References\n\n")
	if len(report.ResourceDeploymentRefs) == 0 {
		b.WriteString("No resource deployment refs matched filters.\n\n")
	} else {
		b.WriteString("| Recorded At | Resource | Kind | Status | Decision | Deployment | Execution |\n")
		b.WriteString("| --- | --- | --- | --- | --- | --- | --- |\n")
		for _, item := range report.ResourceDeploymentRefs {
			b.WriteString("| " + md(item.RecordedAt) + " | " + md(item.ResourceID) + " | " + md(item.Kind) + " | " + md(item.Status) + " | " + md(item.Decision) + " | " + md(item.DeploymentID) + " | " + md(item.ExecutionID) + " |\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("## Evidence References\n\n")
	if len(report.EvidenceRefs) == 0 {
		b.WriteString("No evidence refs matched filters.\n")
	} else {
		for _, ref := range report.EvidenceRefs {
			b.WriteString("- " + md(ref) + "\n")
		}
	}
	return b.String()
}

func renderFilters(filters TimelineOptions) string {
	parts := []string{}
	if filters.Type != "" {
		parts = append(parts, "type="+filters.Type)
	}
	if filters.Status != "" {
		parts = append(parts, "status="+filters.Status)
	}
	if filters.Decision != "" {
		parts = append(parts, "decision="+filters.Decision)
	}
	if filters.Environment != "" {
		parts = append(parts, "environment="+filters.Environment)
	}
	parts = append(parts, fmt.Sprintf("limit=%d", filters.Limit))
	return strings.Join(parts, ", ")
}

func md(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	if len(value) > 160 {
		return value[:160] + "..."
	}
	return value
}

func redactAny(value any) (any, bool) {
	switch typed := value.(type) {
	case string:
		redacted := secrets.Redact(typed)
		return redacted, redacted != typed
	case []string:
		return redactStrings(typed)
	case []any:
		out := make([]any, 0, len(typed))
		changed := false
		for _, item := range typed {
			redacted, itemChanged := redactAny(item)
			changed = changed || itemChanged
			out = append(out, redacted)
		}
		return out, changed
	case map[string]any:
		out := map[string]any{}
		changed := false
		for key, item := range typed {
			redactedKey := secrets.Redact(key)
			redactedValue, itemChanged := redactAny(item)
			if redactedKey != key {
				changed = true
			}
			changed = changed || itemChanged
			out[redactedKey] = redactedValue
		}
		return out, changed
	default:
		return value, false
	}
}

func redactStrings(values []string) ([]string, bool) {
	out := make([]string, 0, len(values))
	changed := false
	for _, value := range values {
		redacted := secrets.Redact(value)
		if redacted != value {
			changed = true
		}
		out = append(out, redacted)
	}
	return out, changed
}

func redactStringValue(value string, changed *bool) string {
	redacted := secrets.Redact(value)
	if redacted != value {
		*changed = true
	}
	return redacted
}

func increment(values map[string]int, key string) {
	key = strings.TrimSpace(key)
	if key == "" {
		key = "unknown"
	}
	values[key]++
}

func isAttentionAuditItem(status string, decision string) bool {
	text := normalizeType(status + "_" + decision)
	return strings.Contains(text, "blocked") ||
		strings.Contains(text, "failed") ||
		strings.Contains(text, "attention") ||
		strings.Contains(text, "manual") ||
		strings.Contains(text, "review_required") ||
		strings.Contains(text, "rollback")
}
