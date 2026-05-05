package operations

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/deployment"
	"moyuan-code/internal/release"
	"moyuan-code/internal/secrets"
	"moyuan-code/internal/serverresources"
)

type WriteProofOptions struct {
	Provider      string `json:"provider,omitempty"`
	OperationType string `json:"operation_type,omitempty"`
	Environment   string `json:"environment,omitempty"`
	Status        string `json:"status,omitempty"`
	Decision      string `json:"decision,omitempty"`
	Limit         int    `json:"limit,omitempty"`
}

type WriteProofReport struct {
	ID          string            `json:"id"`
	GeneratedAt string            `json:"generated_at"`
	Filters     WriteProofOptions `json:"filters"`
	Summary     WriteProofSummary `json:"summary"`
	Proofs      []WriteProof      `json:"proofs"`
}

type WriteProofSummary struct {
	ProofCount          int            `json:"proof_count"`
	BlockedCount        int            `json:"blocked_count"`
	ManualRequiredCount int            `json:"manual_required_count"`
	ByOperationType     map[string]int `json:"by_operation_type,omitempty"`
	ByProvider          map[string]int `json:"by_provider,omitempty"`
	ByEnvironment       map[string]int `json:"by_environment,omitempty"`
	ByStatus            map[string]int `json:"by_status,omitempty"`
	ByDecision          map[string]int `json:"by_decision,omitempty"`
	RedactionApplied    bool           `json:"redaction_applied"`
}

type WriteProof struct {
	ID                   string         `json:"id"`
	OperationType        string         `json:"operation_type"`
	OperationID          string         `json:"operation_id"`
	Provider             string         `json:"provider,omitempty"`
	Environment          string         `json:"environment,omitempty"`
	Mode                 string         `json:"mode,omitempty"`
	Status               string         `json:"status"`
	Decision             string         `json:"decision"`
	Reasons              []string       `json:"reasons,omitempty"`
	SourceRef            string         `json:"source_ref,omitempty"`
	DryRun               bool           `json:"dry_run"`
	WriteEnabled         bool           `json:"write_enabled"`
	ApprovalID           string         `json:"approval_id,omitempty"`
	ApprovalConsumed     bool           `json:"approval_consumed"`
	ApprovalRequired     bool           `json:"approval_required"`
	ApprovalSatisfied    bool           `json:"approval_satisfied"`
	SecretRefStatus      string         `json:"secret_ref_status,omitempty"`
	ProviderEvidenceRefs []string       `json:"provider_evidence_refs,omitempty"`
	LeastPrivilege       string         `json:"least_privilege,omitempty"`
	ReplayGuard          string         `json:"replay_guard,omitempty"`
	CreatedAt            string         `json:"created_at,omitempty"`
	Metadata             map[string]any `json:"metadata,omitempty"`
}

func BuildWriteProofs(rootDir string, options WriteProofOptions) (WriteProofReport, error) {
	options = normalizeWriteProofOptions(options)
	sourceLimit := options.Limit * 4
	if sourceLimit < 50 {
		sourceLimit = 50
	}
	if sourceLimit > 200 {
		sourceLimit = 200
	}

	resources, err := serverresources.List(rootDir)
	if err != nil {
		return WriteProofReport{}, err
	}
	resourceByID := map[string]serverresources.Resource{}
	for _, resource := range resources {
		resourceByID[resource.ID] = resource
	}

	proofs := []WriteProof{}
	add := func(proof WriteProof) {
		proof = normalizeWriteProof(proof)
		if writeProofMatches(proof, options) {
			proofs = append(proofs, proof)
		}
	}

	if options.OperationType == "" || options.OperationType == "release_provider_execution" {
		executions, err := release.ListProviderExecutions(rootDir, sourceLimit)
		if err != nil {
			return WriteProofReport{}, err
		}
		for _, execution := range executions {
			refs, err := evidenceRefs(rootDir, "release_provider_execution", execution.ID, nil)
			if err != nil {
				return WriteProofReport{}, err
			}
			add(writeProofFromReleaseProviderExecution(execution, refs))
		}
	}

	if options.OperationType == "" || options.OperationType == "deployment_execution" {
		executions, err := deployment.ListExecutions(rootDir, sourceLimit)
		if err != nil {
			return WriteProofReport{}, err
		}
		for _, execution := range executions {
			refs, err := evidenceRefs(rootDir, "deployment_execution", execution.ID, nil)
			if err != nil {
				return WriteProofReport{}, err
			}
			add(writeProofFromDeploymentExecution(execution, resourceByID, refs))
		}
	}

	if options.OperationType == "" || options.OperationType == "resource_maintenance" {
		records, err := serverresources.ListMaintenance(rootDir, sourceLimit)
		if err != nil {
			return WriteProofReport{}, err
		}
		for _, record := range records {
			add(writeProofFromResourceMaintenance(record, resourceByID))
		}
	}

	proofs = dedupeWriteProofs(proofs)
	sort.SliceStable(proofs, func(i, j int) bool {
		left := parseTimelineTime(proofs[i].CreatedAt)
		right := parseTimelineTime(proofs[j].CreatedAt)
		if !left.Equal(right) {
			return left.After(right)
		}
		return proofs[i].OperationType+"|"+proofs[i].OperationID > proofs[j].OperationType+"|"+proofs[j].OperationID
	})
	if len(proofs) > options.Limit {
		proofs = proofs[:options.Limit]
	}

	redacted := false
	proofs, redacted = redactWriteProofs(proofs)
	now := time.Now().UTC()
	report := WriteProofReport{
		ID:          "write-proof-report-" + now.Format("20060102150405") + "-" + fmt.Sprintf("%09d", now.UnixNano()%1_000_000_000),
		GeneratedAt: now.Format(time.RFC3339Nano),
		Filters:     options,
		Proofs:      proofs,
	}
	report.Summary = buildWriteProofSummary(proofs)
	report.Summary.RedactionApplied = redacted
	return report, nil
}

func normalizeWriteProofOptions(options WriteProofOptions) WriteProofOptions {
	options.Provider = normalizeType(options.Provider)
	options.OperationType = normalizeType(options.OperationType)
	options.Environment = normalizeType(options.Environment)
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

func writeProofFromReleaseProviderExecution(execution release.ProviderExecution, evidenceRefs []string) WriteProof {
	provider := firstNonEmpty(execution.Provider, execution.RemotePlan.Provider)
	dryRun := execution.Mode == "preview" || !execution.WriteEnabled
	approvalRequired := execution.Mode == "publish"
	proof := WriteProof{
		ID:                   "write-proof-release-provider-" + execution.ID,
		OperationType:        "release_provider_execution",
		OperationID:          execution.ID,
		Provider:             provider,
		Mode:                 execution.Mode,
		Status:               "completed",
		Decision:             "WRITE_PROOF_SATISFIED",
		Reasons:              append([]string{}, execution.Reasons...),
		SourceRef:            execution.ReleaseID,
		DryRun:               dryRun,
		WriteEnabled:         execution.WriteEnabled,
		ApprovalID:           execution.ApprovalID,
		ApprovalConsumed:     execution.ApprovalConsumed,
		ApprovalRequired:     approvalRequired,
		ApprovalSatisfied:    !approvalRequired || execution.ApprovalConsumed,
		SecretRefStatus:      "referenced_by_provider_config",
		ProviderEvidenceRefs: append([]string{}, evidenceRefs...),
		LeastPrivilege:       "git_provider_release_write_scope",
		ReplayGuard:          "release_provider_execution_id_and_approval_consumption",
		CreatedAt:            firstNonEmpty(execution.FinishedAt, execution.StartedAt),
		Metadata: map[string]any{
			"release_id":          execution.ReleaseID,
			"candidate_id":        execution.CandidateID,
			"version":             execution.Version,
			"adapter_status":      execution.AdapterStatus,
			"remote_action_count": len(execution.RemotePlan.Actions),
			"remote_result_count": len(execution.RemoteResults),
			"source_decision":     execution.Decision,
			"source_status":       execution.Status,
		},
	}
	switch {
	case !execution.WriteEnabled:
		proof.Status = "blocked"
		proof.Decision = "WRITE_PROOF_WRITE_DISABLED"
		proof.SecretRefStatus = "not_required_until_write_enabled"
		proof.Reasons = appendUnique(proof.Reasons, "release_provider_write_switch_required")
	case approvalRequired && !execution.ApprovalConsumed:
		proof.Status = "manual_required"
		proof.Decision = "WRITE_PROOF_APPROVAL_REQUIRED"
		proof.Reasons = appendUnique(proof.Reasons, "release_provider_publish_approval_required")
	case execution.Decision == "RELEASE_PROVIDER_PUBLISH_AUTH_REQUIRED":
		proof.Status = "blocked"
		proof.Decision = "WRITE_PROOF_SECRET_REF_MISSING"
		proof.SecretRefStatus = "missing_or_invalid"
	case strings.Contains(normalizeType(execution.Decision), "unsupported"):
		proof.Status = "blocked"
		proof.Decision = "WRITE_PROOF_PROVIDER_UNSUPPORTED"
	case execution.Status == "failed":
		proof.Status = "failed"
		proof.Decision = "WRITE_PROOF_PROVIDER_WRITE_FAILED"
	}
	return proof
}

func writeProofFromDeploymentExecution(execution deployment.Execution, resourceByID map[string]serverresources.Resource, evidenceRefs []string) WriteProof {
	provider := deploymentExecutionProvider(execution, resourceByID)
	dryRun := execution.Mode == "dry_run" || execution.Mode == "ssh_preview" || !deploymentExecutionWriteEnabled(execution)
	approvalRequired := deploymentExecutionApprovalRequired(execution)
	proof := WriteProof{
		ID:                   "write-proof-deployment-" + execution.ID,
		OperationType:        "deployment_execution",
		OperationID:          execution.ID,
		Provider:             provider,
		Environment:          execution.Environment,
		Mode:                 execution.Mode,
		Status:               "completed",
		Decision:             "WRITE_PROOF_SATISFIED",
		Reasons:              append([]string{}, execution.Reasons...),
		SourceRef:            execution.DeploymentID,
		DryRun:               dryRun,
		WriteEnabled:         deploymentExecutionWriteEnabled(execution),
		ApprovalID:           execution.ApprovalID,
		ApprovalConsumed:     execution.ApprovalConsumed,
		ApprovalRequired:     approvalRequired,
		ApprovalSatisfied:    !approvalRequired || execution.ApprovalConsumed,
		SecretRefStatus:      deploymentSecretRefStatus(execution),
		ProviderEvidenceRefs: append([]string{}, evidenceRefs...),
		LeastPrivilege:       deploymentLeastPrivilege(execution),
		ReplayGuard:          "deployment_execution_id_and_approval_consumption",
		CreatedAt:            firstNonEmpty(execution.FinishedAt, execution.StartedAt),
		Metadata: map[string]any{
			"deployment_id":       execution.DeploymentID,
			"release_id":          execution.ReleaseID,
			"resource_count":      len(execution.Resources),
			"step_count":          len(execution.Steps),
			"remote_exec_enabled": execution.RemoteExecEnabled,
			"source_decision":     execution.Decision,
			"source_status":       execution.Status,
		},
	}
	switch {
	case execution.Environment == "production" && proof.WriteEnabled:
		proof.Status = "manual_required"
		proof.Decision = "WRITE_PROOF_PRODUCTION_BLOCKED"
		proof.Reasons = appendUnique(proof.Reasons, "production_real_execution_requires_explicit_release_gate")
	case !proof.WriteEnabled:
		proof.Status = "blocked"
		proof.Decision = "WRITE_PROOF_WRITE_DISABLED"
		proof.Reasons = appendUnique(proof.Reasons, "deployment_write_not_enabled")
	case proof.SecretRefStatus == "missing_or_invalid":
		proof.Status = "blocked"
		proof.Decision = "WRITE_PROOF_SECRET_REF_MISSING"
		proof.Reasons = appendUnique(proof.Reasons, "deployment_remote_auth_ref_missing_or_invalid")
	case approvalRequired && !execution.ApprovalConsumed:
		proof.Status = "manual_required"
		proof.Decision = "WRITE_PROOF_APPROVAL_REQUIRED"
		proof.Reasons = appendUnique(proof.Reasons, "deployment_execution_approval_required")
	case execution.Status == "failed":
		proof.Status = "failed"
		proof.Decision = "WRITE_PROOF_PROVIDER_WRITE_FAILED"
	}
	return proof
}

func writeProofFromResourceMaintenance(record serverresources.MaintenanceRecord, resourceByID map[string]serverresources.Resource) WriteProof {
	resource, found := resourceByID[record.ResourceID]
	provider := "local_registry"
	secretStatus := "not_applicable"
	if found {
		provider = firstNonEmpty(resource.Provider, provider)
		secretStatus = secretReferenceStatus(resource.AuthRef, "not_applicable")
	}
	proof := WriteProof{
		ID:                   "write-proof-resource-maintenance-" + record.ID,
		OperationType:        "resource_maintenance",
		OperationID:          record.ID,
		Provider:             provider,
		Environment:          record.Environment,
		Mode:                 record.Type,
		Status:               "completed",
		Decision:             "WRITE_PROOF_SATISFIED",
		Reasons:              []string{},
		SourceRef:            record.ResourceID,
		DryRun:               false,
		WriteEnabled:         true,
		ApprovalRequired:     record.Environment == "production",
		ApprovalSatisfied:    record.Environment != "production",
		SecretRefStatus:      secretStatus,
		ProviderEvidenceRefs: []string{},
		LeastPrivilege:       "resource_registry_write_only",
		ReplayGuard:          "resource_id_and_maintenance_record_id",
		CreatedAt:            record.CreatedAt,
		Metadata: map[string]any{
			"resource_id":      record.ResourceID,
			"maintenance_type": record.Type,
			"source_decision":  record.Decision,
			"source_status":    record.Status,
		},
	}
	if record.Reason != "" {
		proof.Reasons = append(proof.Reasons, record.Reason)
	}
	if record.Environment == "production" {
		proof.Status = "manual_required"
		proof.Decision = "WRITE_PROOF_PRODUCTION_APPROVAL_REQUIRED"
		proof.Reasons = appendUnique(proof.Reasons, "production_resource_maintenance_requires_approval")
	}
	if record.Status == "blocked" {
		proof.Status = "blocked"
		proof.Decision = "WRITE_PROOF_SOURCE_BLOCKED"
	}
	return proof
}

func normalizeWriteProof(proof WriteProof) WriteProof {
	proof.Provider = normalizeType(proof.Provider)
	proof.OperationType = normalizeType(proof.OperationType)
	proof.Environment = normalizeType(proof.Environment)
	proof.Mode = normalizeType(proof.Mode)
	proof.Status = normalizeType(proof.Status)
	if proof.SecretRefStatus != "" {
		proof.SecretRefStatus = normalizeType(proof.SecretRefStatus)
	}
	proof.Reasons = compactStrings(proof.Reasons)
	proof.ProviderEvidenceRefs = compactStrings(proof.ProviderEvidenceRefs)
	return proof
}

func writeProofMatches(proof WriteProof, options WriteProofOptions) bool {
	if options.Provider != "" && normalizeType(proof.Provider) != options.Provider {
		return false
	}
	if options.OperationType != "" && normalizeType(proof.OperationType) != options.OperationType {
		return false
	}
	if options.Environment != "" && normalizeType(proof.Environment) != options.Environment {
		return false
	}
	if options.Status != "" && normalizeType(proof.Status) != options.Status {
		return false
	}
	if options.Decision != "" && normalizeType(proof.Decision) != options.Decision {
		return false
	}
	return true
}

func deploymentExecutionWriteEnabled(execution deployment.Execution) bool {
	switch execution.Mode {
	case "local_shell":
		return execution.ApprovalConsumed
	case "ssh_execute":
		return execution.RemoteExecEnabled && execution.ApprovalConsumed
	default:
		return false
	}
}

func deploymentExecutionApprovalRequired(execution deployment.Execution) bool {
	return execution.Mode == "local_shell" || execution.Mode == "ssh_execute"
}

func deploymentExecutionProvider(execution deployment.Execution, resourceByID map[string]serverresources.Resource) string {
	if execution.RemotePlan != nil {
		for _, target := range execution.RemotePlan.Targets {
			if target.Provider != "" {
				return target.Provider
			}
			if resource, ok := resourceByID[target.ResourceID]; ok && resource.Provider != "" {
				return resource.Provider
			}
		}
	}
	for _, summary := range execution.Resources {
		if resource, ok := resourceByID[summary.ID]; ok && resource.Provider != "" {
			return resource.Provider
		}
	}
	return "local"
}

func deploymentSecretRefStatus(execution deployment.Execution) string {
	switch execution.Mode {
	case "dry_run":
		return "not_required_for_dry_run"
	case "local_shell":
		return "not_applicable"
	}
	if execution.RemotePlan == nil || len(execution.RemotePlan.Targets) == 0 {
		return "missing_or_invalid"
	}
	for _, target := range execution.RemotePlan.Targets {
		if !secrets.IsSafeReference(target.AuthRef) {
			return "missing_or_invalid"
		}
	}
	return "referenced"
}

func deploymentLeastPrivilege(execution deployment.Execution) string {
	switch execution.Mode {
	case "ssh_execute", "ssh_preview":
		return "ssh_command_allowlist_and_resource_auth_ref"
	case "local_shell":
		return "local_shell_safe_command_allowlist"
	default:
		return "dry_run_no_write"
	}
}

func secretReferenceStatus(ref string, fallback string) string {
	if strings.TrimSpace(ref) == "" {
		return "missing_or_invalid"
	}
	if !secrets.IsSafeReference(ref) {
		return "missing_or_invalid"
	}
	if fallback != "" {
		return fallback
	}
	return "referenced"
}

func dedupeWriteProofs(proofs []WriteProof) []WriteProof {
	seen := map[string]bool{}
	out := []WriteProof{}
	for _, proof := range proofs {
		key := proof.OperationType + "|" + proof.OperationID
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, proof)
	}
	return out
}

func buildWriteProofSummary(proofs []WriteProof) WriteProofSummary {
	summary := WriteProofSummary{
		ProofCount:      len(proofs),
		ByOperationType: map[string]int{},
		ByProvider:      map[string]int{},
		ByEnvironment:   map[string]int{},
		ByStatus:        map[string]int{},
		ByDecision:      map[string]int{},
	}
	for _, proof := range proofs {
		summary.ByOperationType[proof.OperationType]++
		if proof.Provider != "" {
			summary.ByProvider[proof.Provider]++
		}
		if proof.Environment != "" {
			summary.ByEnvironment[proof.Environment]++
		}
		if proof.Status != "" {
			summary.ByStatus[proof.Status]++
		}
		if proof.Decision != "" {
			summary.ByDecision[proof.Decision]++
		}
		if proof.Status == "blocked" {
			summary.BlockedCount++
		}
		if proof.Status == "manual_required" {
			summary.ManualRequiredCount++
		}
	}
	return summary
}

func redactWriteProofs(proofs []WriteProof) ([]WriteProof, bool) {
	out := make([]WriteProof, 0, len(proofs))
	redacted := false
	for _, proof := range proofs {
		proof.ID = redactStringValue(proof.ID, &redacted)
		proof.OperationID = redactStringValue(proof.OperationID, &redacted)
		proof.SourceRef = redactStringValue(proof.SourceRef, &redacted)
		proof.ApprovalID = redactStringValue(proof.ApprovalID, &redacted)
		var changed bool
		proof.Reasons, changed = redactStrings(proof.Reasons)
		redacted = redacted || changed
		proof.ProviderEvidenceRefs, changed = redactStrings(proof.ProviderEvidenceRefs)
		redacted = redacted || changed
		if proof.Metadata != nil {
			var metadata any
			metadata, changed = redactAny(proof.Metadata)
			redacted = redacted || changed
			if cast, ok := metadata.(map[string]any); ok {
				proof.Metadata = cast
			}
		}
		out = append(out, proof)
	}
	return out, redacted
}
