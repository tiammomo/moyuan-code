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
	"moyuan-code/internal/workspace"
)

type WriteAdapterExecutionOptions struct {
	ExecutionPlanID string `json:"execution_plan_id,omitempty"`
	Mode            string `json:"mode,omitempty"`
	AdapterID       string `json:"adapter_id,omitempty"`
	Status          string `json:"status,omitempty"`
	Decision        string `json:"decision,omitempty"`
	Limit           int    `json:"limit,omitempty"`
}

type WriteAdapterExecutionReport struct {
	ID          string                       `json:"id"`
	GeneratedAt string                       `json:"generated_at"`
	Filters     WriteAdapterExecutionOptions `json:"filters"`
	Summary     WriteAdapterExecutionSummary `json:"summary"`
	Executions  []WriteAdapterExecution      `json:"executions"`
}

type WriteAdapterExecutionSummary struct {
	ExecutionCount       int            `json:"execution_count"`
	CompletedCount       int            `json:"completed_count"`
	BlockedCount         int            `json:"blocked_count"`
	ManualRequiredCount  int            `json:"manual_required_count"`
	SandboxResultCount   int            `json:"sandbox_result_count"`
	RollbackBoundCount   int            `json:"rollback_bound_count"`
	ExternalAttemptCount int            `json:"external_attempt_count"`
	ExternalWriteCount   int            `json:"external_write_count"`
	ByAdapter            map[string]int `json:"by_adapter,omitempty"`
	ByMode               map[string]int `json:"by_mode,omitempty"`
	ByStatus             map[string]int `json:"by_status,omitempty"`
	ByDecision           map[string]int `json:"by_decision,omitempty"`
}

type WriteAdapterExecution struct {
	ID                     string                      `json:"id"`
	ExecutionPlanID        string                      `json:"execution_plan_id,omitempty"`
	ReviewPacketID         string                      `json:"review_packet_id,omitempty"`
	OperationType          string                      `json:"operation_type,omitempty"`
	OperationID            string                      `json:"operation_id,omitempty"`
	Provider               string                      `json:"provider,omitempty"`
	Environment            string                      `json:"environment,omitempty"`
	AdapterID              string                      `json:"adapter_id,omitempty"`
	Mode                   string                      `json:"mode"`
	Status                 string                      `json:"status"`
	Decision               string                      `json:"decision"`
	Reasons                []string                    `json:"reasons,omitempty"`
	RuleRefs               []string                    `json:"rule_refs,omitempty"`
	EvidenceRefs           []string                    `json:"evidence_refs,omitempty"`
	GuardResults           []WriteAdapterGuardResult   `json:"guard_results,omitempty"`
	SandboxResults         []WriteAdapterSandboxResult `json:"sandbox_results,omitempty"`
	RollbackBinding        WriteAdapterRollbackBinding `json:"rollback_binding,omitempty"`
	ApplyAllowed           bool                        `json:"apply_allowed"`
	ExternalWriteAttempted bool                        `json:"external_write_attempted"`
	ExternalWritePerformed bool                        `json:"external_write_performed"`
	CreatedAt              string                      `json:"created_at"`
	FinishedAt             string                      `json:"finished_at,omitempty"`
	Metadata               map[string]any              `json:"metadata,omitempty"`
}

type WriteAdapterGuardResult struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Decision string `json:"decision"`
	Reason   string `json:"reason,omitempty"`
}

type WriteAdapterSandboxResult struct {
	ResourceID    string   `json:"resource_id,omitempty"`
	Environment   string   `json:"environment,omitempty"`
	Provider      string   `json:"provider,omitempty"`
	HostStatus    string   `json:"host_status,omitempty"`
	Command       string   `json:"command,omitempty"`
	Allowlist     []string `json:"allowlist,omitempty"`
	Status        string   `json:"status"`
	Decision      string   `json:"decision"`
	Reason        string   `json:"reason,omitempty"`
	PreviewOnly   bool     `json:"preview_only"`
	NoRemoteWrite bool     `json:"no_remote_write"`
}

type WriteAdapterRollbackBinding struct {
	DeploymentID string `json:"deployment_id,omitempty"`
	Required     bool   `json:"required"`
	Status       string `json:"status,omitempty"`
	Decision     string `json:"decision,omitempty"`
	Reason       string `json:"reason,omitempty"`
	PlanRef      string `json:"plan_ref,omitempty"`
	RunbookRef   string `json:"runbook_ref,omitempty"`
	ActionCount  int    `json:"action_count,omitempty"`
	StepCount    int    `json:"step_count,omitempty"`
}

func CreateWriteAdapterExecutions(rootDir string, options WriteAdapterExecutionOptions) (WriteAdapterExecutionReport, error) {
	options = normalizeWriteAdapterExecutionOptions(options)
	if options.Mode == "" {
		options.Mode = "preview"
	}
	execution, err := writeAdapterExecutionFromPlan(rootDir, options)
	if err != nil {
		return WriteAdapterExecutionReport{}, err
	}
	if writeAdapterExecutionMatches(execution, options) {
		execution, err = finishWriteAdapterExecution(rootDir, execution)
		if err != nil {
			return WriteAdapterExecutionReport{}, err
		}
	} else {
		execution = WriteAdapterExecution{}
	}
	executions := []WriteAdapterExecution{}
	if execution.ID != "" {
		executions = append(executions, execution)
	}
	now := time.Now().UTC()
	report := WriteAdapterExecutionReport{
		ID:          "write-adapter-execution-report-" + now.Format("20060102150405") + "-" + fmt.Sprintf("%09d", now.UnixNano()%1_000_000_000),
		GeneratedAt: now.Format(time.RFC3339Nano),
		Filters:     options,
		Executions:  executions,
	}
	report.Summary = buildWriteAdapterExecutionSummary(executions)
	return report, nil
}

func ListWriteAdapterExecutions(rootDir string, options WriteAdapterExecutionOptions) (WriteAdapterExecutionReport, error) {
	options = normalizeWriteAdapterExecutionOptions(options)
	if err := fsutil.EnsureDir(writeAdapterExecutionDir(rootDir)); err != nil {
		return WriteAdapterExecutionReport{}, err
	}
	entries, err := os.ReadDir(writeAdapterExecutionDir(rootDir))
	if err != nil {
		return WriteAdapterExecutionReport{}, err
	}
	executions := []WriteAdapterExecution{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var execution WriteAdapterExecution
		found, err := fsutil.ReadJSON(filepath.Join(writeAdapterExecutionDir(rootDir), entry.Name()), &execution)
		if err != nil {
			return WriteAdapterExecutionReport{}, err
		}
		if found && execution.ID != "" && writeAdapterExecutionMatches(execution, options) {
			executions = append(executions, execution)
		}
	}
	sortWriteAdapterExecutions(executions)
	if len(executions) > options.Limit {
		executions = executions[:options.Limit]
	}
	now := time.Now().UTC()
	report := WriteAdapterExecutionReport{
		ID:          "write-adapter-execution-list-" + now.Format("20060102150405") + "-" + fmt.Sprintf("%09d", now.UnixNano()%1_000_000_000),
		GeneratedAt: now.Format(time.RFC3339Nano),
		Filters:     options,
		Executions:  executions,
	}
	report.Summary = buildWriteAdapterExecutionSummary(executions)
	return report, nil
}

func LoadWriteAdapterExecution(rootDir string, id string) (WriteAdapterExecution, bool, error) {
	var execution WriteAdapterExecution
	found, err := fsutil.ReadJSON(writeAdapterExecutionPath(rootDir, strings.TrimSpace(id)), &execution)
	return execution, found, err
}

func writeAdapterExecutionFromPlan(rootDir string, options WriteAdapterExecutionOptions) (WriteAdapterExecution, error) {
	now := time.Now().UTC()
	execution := WriteAdapterExecution{
		ID:                     "write-adapter-execution-" + firstNonEmpty(options.ExecutionPlanID, "missing-plan") + "-" + options.Mode + "-" + now.Format("20060102150405") + "-" + fmt.Sprintf("%09d", now.UnixNano()%1_000_000_000),
		ExecutionPlanID:        options.ExecutionPlanID,
		Mode:                   options.Mode,
		Status:                 "completed",
		Decision:               "WRITE_ADAPTER_PREVIEW_READY",
		Reasons:                []string{"write_adapter_execution_created"},
		RuleRefs:               []string{"write_adapter_dispatch_scaffold_no_external_write"},
		EvidenceRefs:           []string{},
		GuardResults:           []WriteAdapterGuardResult{},
		ApplyAllowed:           false,
		ExternalWriteAttempted: false,
		ExternalWritePerformed: false,
		CreatedAt:              now.Format(time.RFC3339Nano),
		Metadata: map[string]any{
			"external_write_attempted": false,
			"external_write_performed": false,
			"adapter_switch_env":       "MOYUAN_ENABLE_WRITE_ADAPTERS",
		},
	}
	block := func(decision string, reason string, rule string) {
		execution.Status = "blocked"
		execution.Decision = decision
		execution.Reasons = appendUnique(execution.Reasons, reason)
		execution.RuleRefs = appendUnique(execution.RuleRefs, rule)
	}
	manual := func(decision string, reason string, rule string) {
		if execution.Status != "blocked" {
			execution.Status = "manual_required"
			execution.Decision = decision
		}
		execution.Reasons = appendUnique(execution.Reasons, reason)
		execution.RuleRefs = appendUnique(execution.RuleRefs, rule)
	}

	if options.Mode != "preview" && options.Mode != "apply" {
		block("WRITE_ADAPTER_MODE_UNSUPPORTED", "write_adapter_mode_unsupported:"+options.Mode, "write_adapter_mode_must_be_preview_or_apply")
		execution.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		return normalizeWriteAdapterExecution(execution), nil
	}
	if options.ExecutionPlanID == "" {
		block("WRITE_ADAPTER_EXECUTION_PLAN_REQUIRED", "write_execution_plan_id_required", "write_execution_plan_required")
		execution.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		return normalizeWriteAdapterExecution(execution), nil
	}
	plan, found, err := LoadWriteExecutionPlan(rootDir, options.ExecutionPlanID)
	if err != nil {
		return WriteAdapterExecution{}, err
	}
	if !found {
		block("WRITE_ADAPTER_EXECUTION_PLAN_MISSING", "write_execution_plan_missing:"+options.ExecutionPlanID, "write_execution_plan_must_exist")
		execution.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		return normalizeWriteAdapterExecution(execution), nil
	}

	execution.ReviewPacketID = plan.ReviewPacketID
	execution.OperationType = plan.OperationType
	execution.OperationID = plan.OperationID
	execution.Provider = plan.Provider
	execution.Environment = plan.Environment
	execution.AdapterID = firstNonEmpty(options.AdapterID, deriveWriteAdapterID(plan))
	execution.RuleRefs = appendUniqueStrings(execution.RuleRefs, plan.RuleRefs...)
	execution.EvidenceRefs = appendUniqueStrings(execution.EvidenceRefs, plan.EvidenceRefs...)
	execution.Metadata["execution_plan_status"] = plan.Status
	execution.Metadata["execution_plan_decision"] = plan.Decision

	addGuard := func(name string, status string, decision string, reason string) {
		execution.GuardResults = append(execution.GuardResults, WriteAdapterGuardResult{Name: name, Status: status, Decision: decision, Reason: reason})
	}
	if execution.AdapterID == "" {
		addGuard("adapter_resolution", "blocked", "WRITE_ADAPTER_UNSUPPORTED", "adapter_not_resolved")
		block("WRITE_ADAPTER_UNSUPPORTED", "write_adapter_not_resolved", "write_adapter_supported_operation_required")
		execution.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		return normalizeWriteAdapterExecution(execution), nil
	}
	addGuard("adapter_resolution", "ready", "WRITE_ADAPTER_RESOLVED", execution.AdapterID)
	if execution.AdapterID == "ssh_deployment_adapter" {
		if err := addSSHAdapterSandbox(rootDir, &execution, plan, addGuard, block, manual); err != nil {
			return WriteAdapterExecution{}, err
		}
	}
	if plan.Status == "blocked" {
		addGuard("execution_plan_status", "blocked", "WRITE_ADAPTER_PLAN_BLOCKED", plan.Decision)
		block("WRITE_ADAPTER_PLAN_BLOCKED", "write_execution_plan_blocked:"+plan.Decision, "write_execution_plan_must_be_dispatchable")
	}
	if plan.Status == "manual_required" {
		addGuard("execution_plan_status", "manual_required", "WRITE_ADAPTER_PLAN_MANUAL_REQUIRED", plan.Decision)
		manual("WRITE_ADAPTER_PLAN_MANUAL_REQUIRED", "write_execution_plan_manual_required:"+plan.Decision, "write_execution_plan_must_be_dispatchable")
	}
	if options.Mode == "preview" && execution.Status == "completed" {
		addGuard("external_write", "ready", "WRITE_ADAPTER_PREVIEW_NO_EXTERNAL_WRITE", "preview_mode")
		execution.Reasons = appendUnique(execution.Reasons, "write_adapter_preview_ready")
		execution.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		return normalizeWriteAdapterExecution(execution), nil
	}
	if options.Mode == "apply" {
		if plan.Status != "ready" || !plan.ApplyAllowed {
			addGuard("apply_plan", "manual_required", "WRITE_ADAPTER_APPLY_PLAN_NOT_READY", plan.Decision)
			manual("WRITE_ADAPTER_APPLY_PLAN_NOT_READY", "write_execution_plan_not_apply_ready:"+plan.Decision, "apply_requires_ready_execution_plan")
		}
		if os.Getenv("MOYUAN_ENABLE_WRITE_ADAPTERS") != "1" {
			addGuard("adapter_switch", "blocked", "WRITE_ADAPTER_SWITCH_DISABLED", "MOYUAN_ENABLE_WRITE_ADAPTERS")
			block("WRITE_ADAPTER_SWITCH_DISABLED", "write_adapter_switch_disabled", "write_adapter_requires_explicit_switch")
		}
		if execution.Status == "completed" {
			if execution.AdapterID == "server_resource_registry_adapter" {
				addGuard("resource_registry_receipt", "ready", "WRITE_ADAPTER_RESOURCE_REGISTRY_RECEIPT_READY", "local_registry_receipt")
				execution.ApplyAllowed = true
				execution.Decision = "WRITE_ADAPTER_RESOURCE_REGISTRY_APPLIED"
				execution.Reasons = appendUnique(execution.Reasons, "server_resource_registry_apply_receipt_recorded")
				execution.RuleRefs = appendUnique(execution.RuleRefs, "server_resource_registry_local_receipt_only")
				execution.Metadata["adapter_mutation_scope"] = "local_registry_receipt"
			} else {
				addGuard("adapter_implementation", "manual_required", "WRITE_ADAPTER_IMPLEMENTATION_REQUIRED", execution.AdapterID)
				manual("WRITE_ADAPTER_IMPLEMENTATION_REQUIRED", "real_write_adapter_not_implemented:"+execution.AdapterID, "real_adapter_implementation_required")
			}
		}
	}
	execution.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return normalizeWriteAdapterExecution(execution), nil
}

func addSSHAdapterSandbox(rootDir string, execution *WriteAdapterExecution, plan WriteExecutionPlan, addGuard func(string, string, string, string), block func(string, string, string), manual func(string, string, string)) error {
	execution.RuleRefs = appendUnique(execution.RuleRefs, "ssh_adapter_sandbox_required")
	if strings.TrimSpace(plan.OperationID) == "" {
		addGuard("ssh_execution_load", "blocked", "WRITE_ADAPTER_SSH_EXECUTION_ID_REQUIRED", "deployment_execution_id_required")
		block("WRITE_ADAPTER_SSH_EXECUTION_ID_REQUIRED", "deployment_execution_id_required", "ssh_adapter_requires_deployment_execution")
		return nil
	}
	deploymentExecution, found, err := deployment.LoadExecution(rootDir, plan.OperationID)
	if err != nil {
		return err
	}
	if !found {
		addGuard("ssh_execution_load", "blocked", "WRITE_ADAPTER_SSH_EXECUTION_MISSING", plan.OperationID)
		block("WRITE_ADAPTER_SSH_EXECUTION_MISSING", "deployment_execution_missing:"+plan.OperationID, "ssh_adapter_requires_existing_deployment_execution")
		return nil
	}
	execution.Metadata["deployment_execution_status"] = deploymentExecution.Status
	execution.Metadata["deployment_execution_decision"] = deploymentExecution.Decision
	execution.Metadata["deployment_execution_mode"] = deploymentExecution.Mode
	execution.Metadata["deployment_remote_exec_enabled"] = deploymentExecution.RemoteExecEnabled
	addGuard("ssh_execution_load", "ready", "WRITE_ADAPTER_SSH_EXECUTION_LOADED", deploymentExecution.ID)

	if deploymentExecution.RemotePlan == nil || len(deploymentExecution.RemotePlan.Targets) == 0 {
		addGuard("ssh_remote_plan", "blocked", "WRITE_ADAPTER_SSH_REMOTE_PLAN_MISSING", "remote_plan_missing")
		block("WRITE_ADAPTER_SSH_REMOTE_PLAN_MISSING", "ssh_remote_plan_missing:"+deploymentExecution.ID, "ssh_adapter_requires_remote_plan")
		return bindSSHAdapterRollback(rootDir, execution, deploymentExecution, addGuard, block, manual)
	}
	execution.Metadata["ssh_remote_plan_status"] = deploymentExecution.RemotePlan.Status
	execution.Metadata["ssh_remote_plan_decision"] = deploymentExecution.RemotePlan.Decision
	execution.Metadata["ssh_remote_target_count"] = len(deploymentExecution.RemotePlan.Targets)
	addGuard("ssh_remote_plan", "ready", "WRITE_ADAPTER_SSH_REMOTE_PLAN_LOADED", deploymentExecution.RemotePlan.Decision)

	for _, target := range deploymentExecution.RemotePlan.Targets {
		targetReason := firstNonEmpty(target.Reason, "remote_target_ready")
		if strings.TrimSpace(target.Host) == "" {
			addGuard("ssh_target:"+target.ResourceID, "blocked", "WRITE_ADAPTER_SSH_TARGET_HOST_MISSING", "host_required")
			block("WRITE_ADAPTER_SSH_TARGET_BLOCKED", "ssh_target_host_missing:"+target.ResourceID, "ssh_target_host_required")
		} else if target.Status == "blocked" {
			addGuard("ssh_target:"+target.ResourceID, "blocked", "WRITE_ADAPTER_SSH_TARGET_BLOCKED", targetReason)
			block("WRITE_ADAPTER_SSH_TARGET_BLOCKED", "ssh_target_blocked:"+target.ResourceID+":"+targetReason, "ssh_target_must_be_ready")
		} else {
			addGuard("ssh_target:"+target.ResourceID, "ready", "WRITE_ADAPTER_SSH_TARGET_READY", targetReason)
		}
		authStatus, authDecision, authReason := remoteExecutionAuthStatus(target)
		addGuard("ssh_auth_ref:"+target.ResourceID, authStatus, strings.Replace(authDecision, "REMOTE_", "WRITE_ADAPTER_SSH_", 1), authReason)
		if authStatus == "blocked" {
			block("WRITE_ADAPTER_SSH_AUTH_REF_BLOCKED", "ssh_auth_ref_blocked:"+target.ResourceID+":"+authReason, "ssh_auth_ref_required")
		}
		if len(target.Commands) == 0 {
			execution.SandboxResults = append(execution.SandboxResults, WriteAdapterSandboxResult{
				ResourceID:    target.ResourceID,
				Environment:   target.Environment,
				Provider:      normalizeProviderAlias(firstNonEmpty(target.Provider, plan.Provider), "deployment_execution"),
				HostStatus:    hostStatus(target.Host),
				Allowlist:     sshAdapterSandboxAllowlist(deploymentExecution.Mode),
				Status:        "blocked",
				Decision:      "WRITE_ADAPTER_SSH_SANDBOX_COMMAND_REQUIRED",
				Reason:        "remote_command_required",
				PreviewOnly:   sshAdapterPreviewOnly(deploymentExecution.Mode),
				NoRemoteWrite: sshAdapterNoRemoteWrite(deploymentExecution.Mode, deploymentExecution.RemoteExecEnabled),
			})
			block("WRITE_ADAPTER_SSH_SANDBOX_COMMAND_BLOCKED", "ssh_command_required:"+target.ResourceID, "ssh_adapter_command_required")
			continue
		}
		for _, command := range target.Commands {
			status, decision, reason, previewOnly, noRemoteWrite := sshAdapterSandboxCommandStatus(deploymentExecution.Mode, deploymentExecution.RemoteExecEnabled, command)
			execution.SandboxResults = append(execution.SandboxResults, WriteAdapterSandboxResult{
				ResourceID:    target.ResourceID,
				Environment:   target.Environment,
				Provider:      normalizeProviderAlias(firstNonEmpty(target.Provider, plan.Provider), "deployment_execution"),
				HostStatus:    hostStatus(target.Host),
				Command:       strings.TrimSpace(command),
				Allowlist:     sshAdapterSandboxAllowlist(deploymentExecution.Mode),
				Status:        status,
				Decision:      decision,
				Reason:        reason,
				PreviewOnly:   previewOnly,
				NoRemoteWrite: noRemoteWrite,
			})
			switch status {
			case "blocked":
				block("WRITE_ADAPTER_SSH_SANDBOX_COMMAND_BLOCKED", "ssh_command_blocked:"+target.ResourceID+":"+reason, "ssh_adapter_command_allowlist_required")
			case "manual_required":
				manual("WRITE_ADAPTER_SSH_SANDBOX_MANUAL_REQUIRED", "ssh_command_manual_required:"+target.ResourceID+":"+reason, "ssh_adapter_command_manual_review")
			}
		}
	}
	execution.Metadata["ssh_sandbox_result_count"] = len(execution.SandboxResults)
	if execution.Status == "completed" && len(execution.SandboxResults) > 0 && !writeAdapterSandboxHasStatus(execution.SandboxResults, "blocked") && !writeAdapterSandboxHasStatus(execution.SandboxResults, "manual_required") {
		addGuard("ssh_command_sandbox", "ready", "WRITE_ADAPTER_SSH_SANDBOX_READY", fmt.Sprintf("%d_commands_checked", len(execution.SandboxResults)))
		execution.Reasons = appendUnique(execution.Reasons, "ssh_adapter_sandbox_ready")
	}
	return bindSSHAdapterRollback(rootDir, execution, deploymentExecution, addGuard, block, manual)
}

func bindSSHAdapterRollback(rootDir string, execution *WriteAdapterExecution, deploymentExecution deployment.Execution, addGuard func(string, string, string, string), block func(string, string, string), manual func(string, string, string)) error {
	binding := WriteAdapterRollbackBinding{
		DeploymentID: deploymentExecution.DeploymentID,
		Required:     false,
		Status:       "ready",
		Decision:     "WRITE_ADAPTER_ROLLBACK_NOT_REQUIRED",
		Reason:       "rollback_not_required",
	}
	deploymentPlan, found, err := deployment.Load(rootDir, deploymentExecution.DeploymentID)
	if err != nil {
		return err
	}
	if found {
		binding.PlanRef = filepath.ToSlash(filepath.Join(".moyuan", "lifecycle", "deployments", deploymentPlan.ID+".json"))
		if deploymentPlan.RollbackPlan.Required || len(deploymentPlan.RollbackPlan.Actions) > 0 {
			binding.Required = true
			binding.Status = "ready"
			binding.Decision = "WRITE_ADAPTER_ROLLBACK_PLAN_BOUND"
			binding.Reason = "deployment_rollback_plan_bound"
			binding.ActionCount = len(deploymentPlan.RollbackPlan.Actions)
			if len(deploymentPlan.RollbackPlan.Actions) == 0 {
				binding.Status = "blocked"
				binding.Decision = "WRITE_ADAPTER_ROLLBACK_PLAN_MISSING"
				binding.Reason = "rollback_actions_missing"
				block("WRITE_ADAPTER_ROLLBACK_BLOCKED", "rollback_actions_missing:"+deploymentPlan.ID, "rollback_actions_required")
			}
		}
	} else {
		binding.Required = true
		binding.Status = "blocked"
		binding.Decision = "WRITE_ADAPTER_ROLLBACK_PLAN_MISSING"
		binding.Reason = "deployment_plan_missing"
		block("WRITE_ADAPTER_ROLLBACK_BLOCKED", "deployment_plan_missing:"+deploymentExecution.DeploymentID, "rollback_plan_required")
	}
	if deploymentExecution.RollbackSuggestion.Required {
		binding.Required = true
		binding.Reason = firstNonEmpty(deploymentExecution.RollbackSuggestion.Reason, "rollback_suggestion_required")
		if deploymentExecution.RollbackSuggestion.Runbook == nil || len(deploymentExecution.RollbackSuggestion.Runbook.Steps) == 0 {
			binding.Status = "blocked"
			binding.Decision = "WRITE_ADAPTER_ROLLBACK_RUNBOOK_MISSING"
			block("WRITE_ADAPTER_ROLLBACK_BLOCKED", "rollback_runbook_missing:"+deploymentExecution.ID, "rollback_runbook_required")
		} else {
			binding.Status = "ready"
			binding.Decision = "WRITE_ADAPTER_ROLLBACK_RUNBOOK_BOUND"
			binding.StepCount = len(deploymentExecution.RollbackSuggestion.Runbook.Steps)
			binding.RunbookRef = filepath.ToSlash(filepath.Join(".moyuan", "lifecycle", "deployments", "rollback-runbooks", deploymentExecution.ID+".json"))
		}
	}
	if deploymentExecution.Environment == "production" && binding.Required && binding.Status == "ready" {
		binding.Status = "manual_required"
		binding.Decision = "WRITE_ADAPTER_ROLLBACK_PRODUCTION_REVIEW_REQUIRED"
		manual("WRITE_ADAPTER_ROLLBACK_PRODUCTION_REVIEW_REQUIRED", "production_rollback_binding_requires_manual_review", "production_rollback_review_required")
	}
	execution.RollbackBinding = binding
	addGuard("rollback_binding", binding.Status, binding.Decision, binding.Reason)
	if binding.Status == "ready" {
		execution.Reasons = appendUnique(execution.Reasons, "ssh_adapter_rollback_bound")
	}
	return nil
}

func sshAdapterSandboxCommandStatus(mode string, remoteExecEnabled bool, command string) (string, string, string, bool, bool) {
	command = strings.TrimSpace(command)
	previewOnly := sshAdapterPreviewOnly(mode)
	noRemoteWrite := sshAdapterNoRemoteWrite(mode, remoteExecEnabled)
	if command == "" {
		return "blocked", "WRITE_ADAPTER_SSH_SANDBOX_COMMAND_REQUIRED", "remote_command_required", previewOnly, noRemoteWrite
	}
	if previewOnly {
		if sshAdapterPreviewCommandUnsafe(command) {
			return "blocked", "WRITE_ADAPTER_SSH_SANDBOX_COMMAND_UNSAFE", "preview_command_contains_shell_control_token", previewOnly, noRemoteWrite
		}
		return "ready", "WRITE_ADAPTER_SSH_SANDBOX_PREVIEW_COMMAND_RECORDED", "preview_only_no_remote_command_executed", previewOnly, true
	}
	if !remoteExecutionCommandAllowed(command) {
		return "blocked", "WRITE_ADAPTER_SSH_SANDBOX_COMMAND_NOT_ALLOWED", "remote_command_not_allowed", previewOnly, noRemoteWrite
	}
	return "ready", "WRITE_ADAPTER_SSH_SANDBOX_COMMAND_ALLOWLISTED", "remote_command_allowlisted", previewOnly, noRemoteWrite
}

func sshAdapterPreviewOnly(mode string) bool {
	mode = normalizeType(mode)
	return mode == "ssh_preview" || mode == "dry_run"
}

func sshAdapterNoRemoteWrite(mode string, remoteExecEnabled bool) bool {
	return sshAdapterPreviewOnly(mode) || !remoteExecEnabled
}

func sshAdapterPreviewCommandUnsafe(command string) bool {
	if strings.ContainsAny(command, "\n\r") {
		return true
	}
	for _, token := range []string{";", "&&", "||", "`", "$(", ">", "<", "|"} {
		if strings.Contains(command, token) {
			return true
		}
	}
	return false
}

func sshAdapterSandboxAllowlist(mode string) []string {
	if sshAdapterPreviewOnly(mode) {
		return []string{"preview_only", "secret_ref_only", "no_remote_command_executed", "disallow_shell_control_tokens"}
	}
	return remoteExecutionCommandAllowlist("ssh_execute")
}

func writeAdapterSandboxHasStatus(results []WriteAdapterSandboxResult, status string) bool {
	for _, result := range results {
		if result.Status == status {
			return true
		}
	}
	return false
}

func deriveWriteAdapterID(plan WriteExecutionPlan) string {
	switch normalizeType(plan.OperationType) {
	case "release_provider_execution":
		switch normalizeType(plan.Provider) {
		case "github":
			return "github_release_provider_adapter"
		case "gitee":
			return "gitee_release_provider_adapter"
		case "gitlab":
			return "gitlab_release_provider_adapter"
		default:
			return "generic_release_provider_adapter"
		}
	case "deployment_execution":
		switch normalizeType(plan.Provider) {
		case "ssh", "local_vm":
			return "ssh_deployment_adapter"
		case "cloud", "aliyun", "tencent_cloud":
			return normalizeType(plan.Provider) + "_deployment_adapter"
		default:
			return "generic_deployment_adapter"
		}
	case "resource_maintenance":
		return "server_resource_registry_adapter"
	default:
		return ""
	}
}

func normalizeWriteAdapterExecutionOptions(options WriteAdapterExecutionOptions) WriteAdapterExecutionOptions {
	options.ExecutionPlanID = strings.TrimSpace(options.ExecutionPlanID)
	options.Mode = normalizeType(options.Mode)
	options.AdapterID = normalizeType(options.AdapterID)
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

func normalizeWriteAdapterExecution(execution WriteAdapterExecution) WriteAdapterExecution {
	execution.OperationType = normalizeType(execution.OperationType)
	execution.Provider = normalizeType(execution.Provider)
	execution.Environment = normalizeType(execution.Environment)
	execution.AdapterID = normalizeType(execution.AdapterID)
	execution.Mode = normalizeType(execution.Mode)
	execution.Status = normalizeType(execution.Status)
	execution.RuleRefs = compactStrings(execution.RuleRefs)
	execution.EvidenceRefs = compactStrings(execution.EvidenceRefs)
	return execution
}

func writeAdapterExecutionMatches(execution WriteAdapterExecution, options WriteAdapterExecutionOptions) bool {
	if options.ExecutionPlanID != "" && execution.ExecutionPlanID != options.ExecutionPlanID {
		return false
	}
	if options.Mode != "" && normalizeType(execution.Mode) != options.Mode {
		return false
	}
	if options.AdapterID != "" && normalizeType(execution.AdapterID) != options.AdapterID {
		return false
	}
	if options.Status != "" && normalizeType(execution.Status) != options.Status {
		return false
	}
	if options.Decision != "" && normalizeType(execution.Decision) != options.Decision {
		return false
	}
	return true
}

func buildWriteAdapterExecutionSummary(executions []WriteAdapterExecution) WriteAdapterExecutionSummary {
	summary := WriteAdapterExecutionSummary{
		ExecutionCount: len(executions),
		ByAdapter:      map[string]int{},
		ByMode:         map[string]int{},
		ByStatus:       map[string]int{},
		ByDecision:     map[string]int{},
	}
	for _, execution := range executions {
		if execution.AdapterID != "" {
			summary.ByAdapter[execution.AdapterID]++
		}
		summary.ByMode[execution.Mode]++
		summary.ByStatus[execution.Status]++
		summary.ByDecision[execution.Decision]++
		if execution.ExternalWriteAttempted {
			summary.ExternalAttemptCount++
		}
		if execution.ExternalWritePerformed {
			summary.ExternalWriteCount++
		}
		summary.SandboxResultCount += len(execution.SandboxResults)
		if execution.RollbackBinding.Decision != "" {
			summary.RollbackBoundCount++
		}
		switch execution.Status {
		case "completed":
			summary.CompletedCount++
		case "blocked":
			summary.BlockedCount++
		case "manual_required":
			summary.ManualRequiredCount++
		}
	}
	return summary
}

func sortWriteAdapterExecutions(executions []WriteAdapterExecution) {
	sort.SliceStable(executions, func(i, j int) bool {
		left := parseTimelineTime(executions[i].CreatedAt)
		right := parseTimelineTime(executions[j].CreatedAt)
		if !left.Equal(right) {
			return left.After(right)
		}
		return executions[i].ID > executions[j].ID
	})
}

func finishWriteAdapterExecution(rootDir string, execution WriteAdapterExecution) (WriteAdapterExecution, error) {
	execution = normalizeWriteAdapterExecution(execution)
	if err := fsutil.EnsureDir(writeAdapterExecutionDir(rootDir)); err != nil {
		return WriteAdapterExecution{}, err
	}
	if err := fsutil.WriteJSON(writeAdapterExecutionPath(rootDir, execution.ID), execution); err != nil {
		return WriteAdapterExecution{}, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "write-adapter-executions.jsonl"), execution); err != nil {
		return WriteAdapterExecution{}, err
	}
	_ = logging.Log(rootDir, "release", "operations.write_adapter_execution.created", map[string]any{
		"adapter_execution_id":     execution.ID,
		"execution_plan_id":        execution.ExecutionPlanID,
		"adapter_id":               execution.AdapterID,
		"mode":                     execution.Mode,
		"status":                   execution.Status,
		"decision":                 execution.Decision,
		"sandbox_results":          len(execution.SandboxResults),
		"rollback_binding":         execution.RollbackBinding.Decision,
		"external_write_attempted": execution.ExternalWriteAttempted,
		"external_write_performed": execution.ExternalWritePerformed,
	})
	if _, err := evidence.Add(rootDir, evidence.AddOptions{
		ParentType:  "write_adapter_execution",
		ParentID:    execution.ID,
		SubjectType: firstNonEmpty(execution.OperationType, "write_adapter_execution"),
		SubjectID:   firstNonEmpty(execution.OperationID, execution.ID),
		Operation:   "operations.write_adapter_execution.create",
		Status:      execution.Status,
		Decision:    execution.Decision,
		Reasons:     execution.Reasons,
		Source:      "operations",
		Artifacts: []evidence.ArtifactRef{{
			Kind: "write_adapter_execution",
			ID:   execution.ID,
			Path: filepath.ToSlash(filepath.Join(".moyuan", "lifecycle", "deployments", "write-adapter-executions", execution.ID+".json")),
		}},
	}); err != nil {
		return WriteAdapterExecution{}, err
	}
	return execution, nil
}

func writeAdapterExecutionDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "write-adapter-executions")
}

func writeAdapterExecutionPath(rootDir string, id string) string {
	return filepath.Join(writeAdapterExecutionDir(rootDir), id+".json")
}
