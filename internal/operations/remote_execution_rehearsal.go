package operations

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/deployment"
	"moyuan-code/internal/evidence"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/secrets"
	"moyuan-code/internal/workspace"
)

type RemoteExecutionRehearsalOptions struct {
	AdmissionID string `json:"admission_id,omitempty"`
	ExecutionID string `json:"execution_id,omitempty"`
	Provider    string `json:"provider,omitempty"`
	Environment string `json:"environment,omitempty"`
	Status      string `json:"status,omitempty"`
	Decision    string `json:"decision,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

type RemoteExecutionRehearsalReport struct {
	ID          string                          `json:"id"`
	GeneratedAt string                          `json:"generated_at"`
	Filters     RemoteExecutionRehearsalOptions `json:"filters"`
	Summary     RemoteExecutionRehearsalSummary `json:"summary"`
	Rehearsals  []RemoteExecutionRehearsal      `json:"rehearsals"`
}

type RemoteExecutionRehearsalSummary struct {
	RehearsalCount int            `json:"rehearsal_count"`
	CompletedCount int            `json:"completed_count"`
	BlockedCount   int            `json:"blocked_count"`
	ManualCount    int            `json:"manual_count"`
	ByProvider     map[string]int `json:"by_provider,omitempty"`
	ByEnvironment  map[string]int `json:"by_environment,omitempty"`
	ByStatus       map[string]int `json:"by_status,omitempty"`
	ByDecision     map[string]int `json:"by_decision,omitempty"`
}

type RemoteExecutionRehearsal struct {
	ID                         string                        `json:"id"`
	SourceAdmissionID          string                        `json:"source_admission_id,omitempty"`
	SourceProofID              string                        `json:"source_proof_id,omitempty"`
	OperationType              string                        `json:"operation_type"`
	OperationID                string                        `json:"operation_id"`
	Provider                   string                        `json:"provider,omitempty"`
	Environment                string                        `json:"environment,omitempty"`
	Mode                       string                        `json:"mode,omitempty"`
	Status                     string                        `json:"status"`
	Decision                   string                        `json:"decision"`
	Reasons                    []string                      `json:"reasons,omitempty"`
	RuleRefs                   []string                      `json:"rule_refs,omitempty"`
	EvidenceRefs               []string                      `json:"evidence_refs,omitempty"`
	ProviderRequirementID      string                        `json:"provider_requirement_id,omitempty"`
	ProviderRequirementVersion string                        `json:"provider_requirement_version,omitempty"`
	ProviderRequirementRefs    []string                      `json:"provider_requirement_refs,omitempty"`
	TargetChecks               []RemoteExecutionTargetCheck  `json:"target_checks,omitempty"`
	CommandChecks              []RemoteExecutionCommandCheck `json:"command_checks,omitempty"`
	AuthRefChecks              []RemoteExecutionAuthRefCheck `json:"auth_ref_checks,omitempty"`
	RollbackCheck              RemoteExecutionRollbackCheck  `json:"rollback_check,omitempty"`
	CreatedAt                  string                        `json:"created_at"`
	FinishedAt                 string                        `json:"finished_at,omitempty"`
	Metadata                   map[string]any                `json:"metadata,omitempty"`
}

type RemoteExecutionTargetCheck struct {
	ResourceID  string `json:"resource_id,omitempty"`
	Environment string `json:"environment,omitempty"`
	Provider    string `json:"provider,omitempty"`
	HostStatus  string `json:"host_status"`
	Status      string `json:"status"`
	Decision    string `json:"decision"`
	Reason      string `json:"reason,omitempty"`
}

type RemoteExecutionCommandCheck struct {
	ResourceID    string   `json:"resource_id,omitempty"`
	Mode          string   `json:"mode,omitempty"`
	CommandCount  int      `json:"command_count"`
	Allowlist     []string `json:"allowlist,omitempty"`
	Status        string   `json:"status"`
	Decision      string   `json:"decision"`
	Reason        string   `json:"reason,omitempty"`
	PreviewOnly   bool     `json:"preview_only"`
	NoRemoteWrite bool     `json:"no_remote_write"`
}

type RemoteExecutionAuthRefCheck struct {
	ResourceID    string `json:"resource_id,omitempty"`
	Provider      string `json:"provider,omitempty"`
	AuthRefStatus string `json:"auth_ref_status"`
	Status        string `json:"status"`
	Decision      string `json:"decision"`
	Reason        string `json:"reason,omitempty"`
}

type RemoteExecutionRollbackCheck struct {
	Required bool   `json:"required"`
	Status   string `json:"status,omitempty"`
	Decision string `json:"decision,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

func RunRemoteExecutionRehearsals(rootDir string, options RemoteExecutionRehearsalOptions) (RemoteExecutionRehearsalReport, error) {
	options = normalizeRemoteExecutionRehearsalOptions(options)
	admissionReport, err := BuildWriteAdmissions(rootDir, WriteAdmissionOptions{
		OperationType: "deployment_execution",
		Environment:   options.Environment,
		Limit:         options.Limit,
	})
	if err != nil {
		return RemoteExecutionRehearsalReport{}, err
	}
	rehearsals := []RemoteExecutionRehearsal{}
	for _, admission := range admissionReport.Entries {
		if options.AdmissionID != "" && admission.ID != options.AdmissionID {
			continue
		}
		if options.ExecutionID != "" && admission.OperationID != options.ExecutionID {
			continue
		}
		rehearsal, err := remoteExecutionRehearsalFromAdmission(rootDir, admission)
		if err != nil {
			return RemoteExecutionRehearsalReport{}, err
		}
		if !remoteExecutionRehearsalMatches(rehearsal, options) {
			continue
		}
		rehearsal, err = finishRemoteExecutionRehearsal(rootDir, rehearsal)
		if err != nil {
			return RemoteExecutionRehearsalReport{}, err
		}
		rehearsals = append(rehearsals, rehearsal)
	}
	sort.SliceStable(rehearsals, func(i, j int) bool {
		return rehearsals[i].CreatedAt > rehearsals[j].CreatedAt
	})
	if len(rehearsals) > options.Limit {
		rehearsals = rehearsals[:options.Limit]
	}
	now := time.Now().UTC()
	report := RemoteExecutionRehearsalReport{
		ID:          "remote-execution-rehearsal-report-" + now.Format("20060102150405") + "-" + fmt.Sprintf("%09d", now.UnixNano()%1_000_000_000),
		GeneratedAt: now.Format(time.RFC3339Nano),
		Filters:     options,
		Rehearsals:  rehearsals,
	}
	report.Summary = buildRemoteExecutionRehearsalSummary(rehearsals)
	return report, nil
}

func ListRemoteExecutionRehearsals(rootDir string, options RemoteExecutionRehearsalOptions) (RemoteExecutionRehearsalReport, error) {
	options = normalizeRemoteExecutionRehearsalOptions(options)
	if err := fsutil.EnsureDir(remoteExecutionRehearsalDir(rootDir)); err != nil {
		return RemoteExecutionRehearsalReport{}, err
	}
	entries, err := os.ReadDir(remoteExecutionRehearsalDir(rootDir))
	if err != nil {
		return RemoteExecutionRehearsalReport{}, err
	}
	rehearsals := []RemoteExecutionRehearsal{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var rehearsal RemoteExecutionRehearsal
		found, err := fsutil.ReadJSON(filepath.Join(remoteExecutionRehearsalDir(rootDir), entry.Name()), &rehearsal)
		if err != nil {
			return RemoteExecutionRehearsalReport{}, err
		}
		if found && rehearsal.ID != "" && remoteExecutionRehearsalMatches(rehearsal, options) {
			rehearsals = append(rehearsals, rehearsal)
		}
	}
	sort.SliceStable(rehearsals, func(i, j int) bool {
		return rehearsals[i].CreatedAt > rehearsals[j].CreatedAt
	})
	if len(rehearsals) > options.Limit {
		rehearsals = rehearsals[:options.Limit]
	}
	now := time.Now().UTC()
	report := RemoteExecutionRehearsalReport{
		ID:          "remote-execution-rehearsal-list-" + now.Format("20060102150405") + "-" + fmt.Sprintf("%09d", now.UnixNano()%1_000_000_000),
		GeneratedAt: now.Format(time.RFC3339Nano),
		Filters:     options,
		Rehearsals:  rehearsals,
	}
	report.Summary = buildRemoteExecutionRehearsalSummary(rehearsals)
	return report, nil
}

func LoadRemoteExecutionRehearsal(rootDir string, id string) (RemoteExecutionRehearsal, bool, error) {
	var rehearsal RemoteExecutionRehearsal
	found, err := fsutil.ReadJSON(remoteExecutionRehearsalPath(rootDir, id), &rehearsal)
	return rehearsal, found, err
}

func remoteExecutionRehearsalFromAdmission(rootDir string, admission WriteAdmissionEntry) (RemoteExecutionRehearsal, error) {
	now := time.Now().UTC()
	rehearsal := RemoteExecutionRehearsal{
		ID:                         "remote-execution-rehearsal-" + admission.OperationID + "-" + now.Format("20060102150405") + "-" + fmt.Sprintf("%09d", now.UnixNano()%1_000_000_000),
		SourceAdmissionID:          admission.ID,
		SourceProofID:              admission.ProofID,
		OperationType:              admission.OperationType,
		OperationID:                admission.OperationID,
		Provider:                   admission.Provider,
		Environment:                admission.Environment,
		Mode:                       admission.Mode,
		Status:                     "completed",
		Decision:                   "REMOTE_EXECUTION_REHEARSAL_READY",
		Reasons:                    []string{"remote_execution_rehearsal_no_write_performed"},
		RuleRefs:                   append([]string{}, admission.RuleRefs...),
		EvidenceRefs:               append([]string{}, admission.ProviderEvidenceRefs...),
		ProviderRequirementID:      admission.ProviderRequirementID,
		ProviderRequirementVersion: admission.ProviderRequirementVersion,
		ProviderRequirementRefs:    append([]string{}, admission.ProviderRequirementRefs...),
		TargetChecks:               []RemoteExecutionTargetCheck{},
		CommandChecks:              []RemoteExecutionCommandCheck{},
		AuthRefChecks:              []RemoteExecutionAuthRefCheck{},
		CreatedAt:                  now.Format(time.RFC3339Nano),
		Metadata: map[string]any{
			"source_admission_status":     admission.Status,
			"source_admission_decision":   admission.Decision,
			"write_enabled":               admission.WriteEnabled,
			"dry_run":                     admission.DryRun,
			"remote_write_never_executed": true,
		},
	}
	block := func(decision string, reason string, rule string) {
		rehearsal.Status = "blocked"
		rehearsal.Decision = decision
		rehearsal.Reasons = appendUnique(rehearsal.Reasons, reason)
		rehearsal.RuleRefs = appendUnique(rehearsal.RuleRefs, rule)
	}
	manual := func(decision string, reason string, rule string) {
		if rehearsal.Status != "blocked" {
			rehearsal.Status = "manual_required"
			rehearsal.Decision = decision
		}
		rehearsal.Reasons = appendUnique(rehearsal.Reasons, reason)
		rehearsal.RuleRefs = appendUnique(rehearsal.RuleRefs, rule)
	}
	if admission.OperationType != "deployment_execution" {
		block("REMOTE_EXECUTION_REHEARSAL_OPERATION_UNSUPPORTED", "remote_rehearsal_requires_deployment_execution", "remote_rehearsal_supported_operation")
		return rehearsal, nil
	}
	if admission.Status == "blocked" {
		block("REMOTE_EXECUTION_REHEARSAL_SOURCE_BLOCKED", "source_write_admission_blocked:"+admission.Decision, "source_write_admission_must_allow_rehearsal")
	}
	if admission.Status == "manual_required" {
		manual("REMOTE_EXECUTION_REHEARSAL_MANUAL_REQUIRED", "source_write_admission_manual_required", "source_write_admission_manual_review")
	}
	if !admission.RehearsalAllowed {
		block("REMOTE_EXECUTION_REHEARSAL_NOT_ALLOWED", "source_write_admission_not_rehearsal_allowed", "source_write_admission_must_allow_rehearsal")
	}
	execution, found, err := deployment.LoadExecution(rootDir, admission.OperationID)
	if err != nil {
		return RemoteExecutionRehearsal{}, err
	}
	if !found {
		block("REMOTE_EXECUTION_REHEARSAL_EXECUTION_MISSING", "deployment_execution_missing", "deployment_execution_required")
		return rehearsal, nil
	}
	if rehearsal.Environment == "" {
		rehearsal.Environment = execution.Environment
	}
	if rehearsal.Mode == "" {
		rehearsal.Mode = execution.Mode
	}
	refs, err := evidenceRefs(rootDir, "deployment_execution", execution.ID, nil)
	if err != nil {
		return RemoteExecutionRehearsal{}, err
	}
	rehearsal.EvidenceRefs = appendUniqueStrings(rehearsal.EvidenceRefs, refs...)
	addExecutionRemoteChecks(&rehearsal, execution, block)
	addRollbackCheck(&rehearsal, execution, block, manual)
	if rehearsal.Status == "completed" {
		rehearsal.Reasons = appendUnique(rehearsal.Reasons, "remote_execution_rehearsal_ready")
		rehearsal.RuleRefs = appendUnique(rehearsal.RuleRefs, "remote_execution_rehearsal_all_checks_passed")
	}
	return normalizeRemoteExecutionRehearsal(rehearsal), nil
}

func addExecutionRemoteChecks(rehearsal *RemoteExecutionRehearsal, execution deployment.Execution, block func(string, string, string)) {
	if execution.RemotePlan == nil || len(execution.RemotePlan.Targets) == 0 {
		if len(execution.Resources) == 0 {
			block("REMOTE_EXECUTION_REHEARSAL_TARGET_MISSING", "remote_targets_missing", "remote_target_required")
			return
		}
		for _, resource := range execution.Resources {
			rehearsal.TargetChecks = append(rehearsal.TargetChecks, RemoteExecutionTargetCheck{
				ResourceID:  resource.ID,
				Environment: resource.Environment,
				HostStatus:  hostStatus(resource.Host),
				Status:      "blocked",
				Decision:    "REMOTE_TARGET_REMOTE_PLAN_MISSING",
				Reason:      "remote_plan_missing",
			})
		}
		block("REMOTE_EXECUTION_REHEARSAL_REMOTE_PLAN_MISSING", "remote_plan_missing", "remote_plan_required")
		return
	}
	for _, target := range execution.RemotePlan.Targets {
		targetStatus := "ready"
		targetDecision := "REMOTE_TARGET_READY"
		targetReason := firstNonEmpty(target.Reason, "remote_target_ready")
		if target.Status == "blocked" {
			targetStatus = "blocked"
			targetDecision = "REMOTE_TARGET_BLOCKED"
			block("REMOTE_EXECUTION_REHEARSAL_TARGET_BLOCKED", "remote_target_blocked:"+targetReason, "remote_target_must_be_ready")
		}
		if strings.TrimSpace(target.Host) == "" {
			targetStatus = "blocked"
			targetDecision = "REMOTE_TARGET_HOST_MISSING"
			targetReason = "remote_target_host_missing"
			block("REMOTE_EXECUTION_REHEARSAL_TARGET_MISSING", "remote_target_host_missing:"+target.ResourceID, "remote_target_host_required")
		}
		rehearsal.TargetChecks = append(rehearsal.TargetChecks, RemoteExecutionTargetCheck{
			ResourceID:  target.ResourceID,
			Environment: target.Environment,
			Provider:    normalizeProviderAlias(firstNonEmpty(target.Provider, rehearsal.Provider), "deployment_execution"),
			HostStatus:  hostStatus(target.Host),
			Status:      targetStatus,
			Decision:    targetDecision,
			Reason:      targetReason,
		})
		authStatus, authDecision, authReason := remoteExecutionAuthStatus(target)
		if authStatus == "blocked" {
			block("REMOTE_EXECUTION_REHEARSAL_AUTH_REF_BLOCKED", "remote_auth_ref_blocked:"+target.ResourceID, "remote_auth_ref_required")
		}
		rehearsal.AuthRefChecks = append(rehearsal.AuthRefChecks, RemoteExecutionAuthRefCheck{
			ResourceID:    target.ResourceID,
			Provider:      normalizeProviderAlias(firstNonEmpty(target.Provider, rehearsal.Provider), "deployment_execution"),
			AuthRefStatus: authReason,
			Status:        authStatus,
			Decision:      authDecision,
			Reason:        authReason,
		})
		commandStatus, commandDecision, commandReason := remoteExecutionCommandStatus(execution.Mode, target.Commands)
		if commandStatus == "blocked" {
			block("REMOTE_EXECUTION_REHEARSAL_COMMAND_BLOCKED", "remote_command_blocked:"+target.ResourceID, "remote_command_allowlist_required")
		}
		rehearsal.CommandChecks = append(rehearsal.CommandChecks, RemoteExecutionCommandCheck{
			ResourceID:    target.ResourceID,
			Mode:          execution.Mode,
			CommandCount:  len(target.Commands),
			Allowlist:     remoteExecutionCommandAllowlist(execution.Mode),
			Status:        commandStatus,
			Decision:      commandDecision,
			Reason:        commandReason,
			PreviewOnly:   execution.Mode == "ssh_preview" || execution.Mode == "dry_run",
			NoRemoteWrite: execution.Mode == "ssh_preview" || execution.Mode == "dry_run" || !execution.RemoteExecEnabled,
		})
	}
}

func addRollbackCheck(rehearsal *RemoteExecutionRehearsal, execution deployment.Execution, block func(string, string, string), manual func(string, string, string)) {
	check := RemoteExecutionRollbackCheck{
		Required: execution.RollbackSuggestion.Required,
		Status:   "ready",
		Decision: "REMOTE_ROLLBACK_READY",
		Reason:   "rollback_not_required",
	}
	if execution.RollbackSuggestion.Required {
		check.Reason = firstNonEmpty(execution.RollbackSuggestion.Reason, "rollback_required")
		if execution.RollbackSuggestion.Runbook == nil || len(execution.RollbackSuggestion.Runbook.Steps) == 0 {
			check.Status = "blocked"
			check.Decision = "REMOTE_ROLLBACK_RUNBOOK_MISSING"
			block("REMOTE_EXECUTION_REHEARSAL_ROLLBACK_BLOCKED", "rollback_runbook_missing", "rollback_runbook_required")
		} else if execution.Environment == "production" {
			check.Status = "manual_required"
			check.Decision = "REMOTE_ROLLBACK_PRODUCTION_REVIEW_REQUIRED"
			manual("REMOTE_EXECUTION_REHEARSAL_MANUAL_REQUIRED", "production_rollback_plan_requires_manual_review", "production_rollback_review_required")
		}
	}
	rehearsal.RollbackCheck = check
}

func remoteExecutionAuthStatus(target deployment.RemoteTarget) (string, string, string) {
	if strings.TrimSpace(target.AuthRef) == "" {
		return "blocked", "REMOTE_AUTH_REF_MISSING", "auth_ref_missing"
	}
	if !secrets.IsSafeReference(target.AuthRef) {
		return "blocked", "REMOTE_AUTH_REF_UNSAFE", "auth_ref_must_be_reference"
	}
	return "ready", "REMOTE_AUTH_REF_REFERENCED", "referenced"
}

func remoteExecutionCommandStatus(mode string, commands []string) (string, string, string) {
	mode = normalizeType(mode)
	if mode == "dry_run" || mode == "ssh_preview" {
		return "ready", "REMOTE_COMMAND_PREVIEW_ONLY", "preview_only_no_remote_command_executed"
	}
	if len(commands) == 0 {
		return "blocked", "REMOTE_COMMAND_REQUIRED", "remote_command_required"
	}
	for _, command := range commands {
		if !remoteExecutionCommandAllowed(command) {
			return "blocked", "REMOTE_COMMAND_NOT_ALLOWED", "remote_command_not_allowed"
		}
	}
	return "ready", "REMOTE_COMMAND_ALLOWLISTED", "remote_command_allowlisted"
}

func remoteExecutionCommandAllowed(command string) bool {
	if strings.ContainsAny(command, "\n\r") {
		return false
	}
	for _, token := range []string{";", "&&", "||", "`", "$(", ">", "<", "|"} {
		if strings.Contains(command, token) {
			return false
		}
	}
	for _, prefix := range remoteExecutionCommandAllowlist("ssh_execute") {
		if strings.HasSuffix(prefix, " ") {
			if strings.HasPrefix(command, prefix) {
				return true
			}
			continue
		}
		if command == prefix {
			return true
		}
	}
	return false
}

func remoteExecutionCommandAllowlist(mode string) []string {
	if normalizeType(mode) == "ssh_preview" || normalizeType(mode) == "dry_run" {
		return []string{"preview_only", "secret_ref_only", "no_remote_command_executed"}
	}
	return []string{"true", "echo ", "printf ", "curl -fsS http://127.0.0.1", "curl -fsS http://localhost", "systemctl status ", "docker ps", "docker compose ps"}
}

func hostStatus(host string) string {
	if strings.TrimSpace(host) == "" {
		return "missing"
	}
	return "referenced"
}

func normalizeRemoteExecutionRehearsalOptions(options RemoteExecutionRehearsalOptions) RemoteExecutionRehearsalOptions {
	options.AdmissionID = strings.TrimSpace(options.AdmissionID)
	options.ExecutionID = strings.TrimSpace(options.ExecutionID)
	options.Provider = normalizeType(options.Provider)
	if options.Provider != "" {
		options.Provider = normalizeProviderAlias(options.Provider, "deployment_execution")
	}
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

func normalizeRemoteExecutionRehearsal(rehearsal RemoteExecutionRehearsal) RemoteExecutionRehearsal {
	rehearsal.Provider = normalizeProviderAlias(normalizeType(rehearsal.Provider), "deployment_execution")
	rehearsal.Environment = normalizeType(rehearsal.Environment)
	rehearsal.Mode = normalizeType(rehearsal.Mode)
	rehearsal.Status = normalizeType(rehearsal.Status)
	rehearsal.EvidenceRefs = compactStrings(rehearsal.EvidenceRefs)
	rehearsal.RuleRefs = compactStrings(rehearsal.RuleRefs)
	rehearsal.ProviderRequirementRefs = compactStrings(rehearsal.ProviderRequirementRefs)
	return rehearsal
}

func remoteExecutionRehearsalMatches(rehearsal RemoteExecutionRehearsal, options RemoteExecutionRehearsalOptions) bool {
	if options.AdmissionID != "" && rehearsal.SourceAdmissionID != options.AdmissionID {
		return false
	}
	if options.ExecutionID != "" && rehearsal.OperationID != options.ExecutionID {
		return false
	}
	if options.Provider != "" && normalizeProviderAlias(rehearsal.Provider, "deployment_execution") != options.Provider {
		return false
	}
	if options.Environment != "" && normalizeType(rehearsal.Environment) != options.Environment {
		return false
	}
	if options.Status != "" && normalizeType(rehearsal.Status) != options.Status {
		return false
	}
	if options.Decision != "" && normalizeType(rehearsal.Decision) != options.Decision {
		return false
	}
	return true
}

func buildRemoteExecutionRehearsalSummary(rehearsals []RemoteExecutionRehearsal) RemoteExecutionRehearsalSummary {
	summary := RemoteExecutionRehearsalSummary{
		RehearsalCount: len(rehearsals),
		ByProvider:     map[string]int{},
		ByEnvironment:  map[string]int{},
		ByStatus:       map[string]int{},
		ByDecision:     map[string]int{},
	}
	for _, rehearsal := range rehearsals {
		if rehearsal.Provider != "" {
			summary.ByProvider[rehearsal.Provider]++
		}
		if rehearsal.Environment != "" {
			summary.ByEnvironment[rehearsal.Environment]++
		}
		summary.ByStatus[rehearsal.Status]++
		summary.ByDecision[rehearsal.Decision]++
		switch rehearsal.Status {
		case "completed":
			summary.CompletedCount++
		case "blocked":
			summary.BlockedCount++
		case "manual_required":
			summary.ManualCount++
		}
	}
	return summary
}

func finishRemoteExecutionRehearsal(rootDir string, rehearsal RemoteExecutionRehearsal) (RemoteExecutionRehearsal, error) {
	rehearsal.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	rehearsal = normalizeRemoteExecutionRehearsal(rehearsal)
	if err := fsutil.EnsureDir(remoteExecutionRehearsalDir(rootDir)); err != nil {
		return RemoteExecutionRehearsal{}, err
	}
	if err := fsutil.WriteJSON(remoteExecutionRehearsalPath(rootDir, rehearsal.ID), rehearsal); err != nil {
		return RemoteExecutionRehearsal{}, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "remote-execution-rehearsals.jsonl"), rehearsal); err != nil {
		return RemoteExecutionRehearsal{}, err
	}
	_ = logging.Log(rootDir, "release", "operations.remote_execution.rehearsal.created", map[string]any{
		"rehearsal_id": rehearsal.ID,
		"execution_id": rehearsal.OperationID,
		"provider":     rehearsal.Provider,
		"environment":  rehearsal.Environment,
		"status":       rehearsal.Status,
		"decision":     rehearsal.Decision,
	})
	if _, err := evidence.Add(rootDir, evidence.AddOptions{
		ParentType:  "remote_execution_rehearsal",
		ParentID:    rehearsal.ID,
		SubjectType: "deployment_execution",
		SubjectID:   rehearsal.OperationID,
		Operation:   "operations.remote_execution.rehearsal",
		Status:      rehearsal.Status,
		Decision:    rehearsal.Decision,
		Reasons:     rehearsal.Reasons,
		Source:      "operations",
		Artifacts: []evidence.ArtifactRef{{
			Kind: "remote_execution_rehearsal",
			ID:   rehearsal.ID,
			Path: filepath.ToSlash(filepath.Join(".moyuan", "lifecycle", "deployments", "remote-execution-rehearsals", rehearsal.ID+".json")),
		}},
	}); err != nil {
		return RemoteExecutionRehearsal{}, err
	}
	return rehearsal, nil
}

func remoteExecutionRehearsalDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "remote-execution-rehearsals")
}

func remoteExecutionRehearsalPath(rootDir string, id string) string {
	return filepath.Join(remoteExecutionRehearsalDir(rootDir), id+".json")
}

func appendUniqueStrings(values []string, additions ...string) []string {
	for _, addition := range additions {
		values = appendUnique(values, addition)
	}
	return values
}
