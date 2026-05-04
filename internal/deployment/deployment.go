package deployment

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/approvals"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/process"
	"moyuan-code/internal/release"
	"moyuan-code/internal/serverresources"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type PlanOptions struct {
	ReleaseID   string   `json:"release_id"`
	Environment string   `json:"environment"`
	ResourceIDs []string `json:"resource_ids"`
	Approved    bool     `json:"approved"`
}

type Plan struct {
	ID             string            `json:"id"`
	ReleaseID      string            `json:"release_id"`
	Environment    string            `json:"environment"`
	Status         string            `json:"status"`
	Decision       string            `json:"decision"`
	Reasons        []string          `json:"reasons"`
	Resources      []ResourceSummary `json:"resources"`
	SmokePlan      StepPlan          `json:"smoke_plan"`
	MonitorPlan    StepPlan          `json:"monitor_plan"`
	RollbackPlan   StepPlan          `json:"rollback_plan"`
	ManualRequired bool              `json:"manual_required"`
	ApprovalID     string            `json:"approval_id,omitempty"`
	CreatedAt      string            `json:"created_at"`
}

type ResourceSummary struct {
	ID          string `json:"id"`
	Environment string `json:"environment"`
	Host        string `json:"host"`
	Status      string `json:"status"`
}

type StepPlan struct {
	Status   string   `json:"status"`
	Actions  []string `json:"actions"`
	Window   string   `json:"window,omitempty"`
	Required bool     `json:"required"`
}

type ExecuteOptions struct {
	DeploymentID string   `json:"deployment_id"`
	Mode         string   `json:"mode"`
	Approved     bool     `json:"approved"`
	Commands     []string `json:"commands"`
}

type Execution struct {
	ID                 string             `json:"id"`
	DeploymentID       string             `json:"deployment_id"`
	ReleaseID          string             `json:"release_id"`
	Environment        string             `json:"environment"`
	Mode               string             `json:"mode"`
	Status             string             `json:"status"`
	Decision           string             `json:"decision"`
	Reasons            []string           `json:"reasons"`
	Resources          []ResourceSummary  `json:"resources"`
	Steps              []ExecutionStep    `json:"steps"`
	RemotePlan         *RemotePlan        `json:"remote_plan,omitempty"`
	SmokeReport        CheckReport        `json:"smoke_report,omitempty"`
	MonitorReport      CheckReport        `json:"monitor_report,omitempty"`
	RollbackSuggestion RollbackSuggestion `json:"rollback_suggestion,omitempty"`
	ApprovalID         string             `json:"approval_id,omitempty"`
	RemoteExecEnabled  bool               `json:"remote_exec_enabled"`
	StartedAt          string             `json:"started_at"`
	FinishedAt         string             `json:"finished_at,omitempty"`
}

type ExecutionStep struct {
	Name       string   `json:"name"`
	Status     string   `json:"status"`
	Command    string   `json:"command,omitempty"`
	Output     string   `json:"output,omitempty"`
	Error      string   `json:"error,omitempty"`
	Allowlist  []string `json:"allowlist,omitempty"`
	StartedAt  string   `json:"started_at,omitempty"`
	FinishedAt string   `json:"finished_at,omitempty"`
}

type RemotePlan struct {
	Status    string         `json:"status"`
	Decision  string         `json:"decision"`
	Targets   []RemoteTarget `json:"targets"`
	CreatedAt string         `json:"created_at"`
}

type RemoteTarget struct {
	ResourceID  string   `json:"resource_id"`
	Environment string   `json:"environment"`
	Host        string   `json:"host"`
	Provider    string   `json:"provider,omitempty"`
	AuthRef     string   `json:"auth_ref"`
	Status      string   `json:"status"`
	Reason      string   `json:"reason,omitempty"`
	Commands    []string `json:"commands,omitempty"`
}

type CheckReport struct {
	Status    string        `json:"status,omitempty"`
	Decision  string        `json:"decision,omitempty"`
	Results   []CheckResult `json:"results,omitempty"`
	CheckedAt string        `json:"checked_at,omitempty"`
}

type CheckResult struct {
	ResourceID string `json:"resource_id"`
	Target     string `json:"target,omitempty"`
	Status     string `json:"status"`
	Reason     string `json:"reason,omitempty"`
	HTTPStatus int    `json:"http_status,omitempty"`
}

type RollbackSuggestion struct {
	Required bool     `json:"required,omitempty"`
	Decision string   `json:"decision,omitempty"`
	Reason   string   `json:"reason,omitempty"`
	Actions  []string `json:"actions,omitempty"`
}

func CreatePlan(rootDir string, options PlanOptions) (Plan, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return Plan{}, err
	}
	options.Environment = normalizeToken(options.Environment)
	options.ResourceIDs = normalizeIDs(options.ResourceIDs)
	if options.ReleaseID == "" {
		return Plan{}, errors.New("release_id_required")
	}
	if options.Environment == "" {
		return Plan{}, errors.New("environment_required")
	}
	now := time.Now().UTC()
	plan := Plan{
		ID:          "deployment-" + textutil.Slugify(options.ReleaseID+"-"+options.Environment) + "-" + now.Format("20060102150405"),
		ReleaseID:   options.ReleaseID,
		Environment: options.Environment,
		Status:      "blocked",
		Decision:    "DEPLOY_BLOCKED",
		Reasons:     []string{},
		Resources:   []ResourceSummary{},
		CreatedAt:   now.Format(time.RFC3339Nano),
	}
	releasePlan, found, err := release.Load(rootDir, options.ReleaseID)
	if err != nil {
		return Plan{}, err
	}
	if !found {
		plan.Reasons = append(plan.Reasons, "release_not_found")
		return finish(rootDir, plan)
	}
	if releasePlan.Decision != "RELEASE_SUGGESTED" {
		plan.Reasons = append(plan.Reasons, "release_not_suggested:"+releasePlan.Decision)
		return finish(rootDir, plan)
	}
	resources, err := resolveResources(rootDir, options.Environment, options.ResourceIDs)
	if err != nil {
		return Plan{}, err
	}
	if len(resources) == 0 {
		plan.Reasons = append(plan.Reasons, "server_resource_missing:"+options.Environment)
		return finish(rootDir, plan)
	}
	for _, resource := range resources {
		plan.Resources = append(plan.Resources, ResourceSummary{ID: resource.ID, Environment: resource.Environment, Host: resource.Host, Status: resource.Status})
		if resource.Status != "active" {
			plan.Reasons = append(plan.Reasons, "server_resource_not_active:"+resource.ID)
		}
		if resource.Environment != options.Environment {
			plan.Reasons = append(plan.Reasons, "server_resource_environment_mismatch:"+resource.ID)
		}
		if resource.Healthcheck.LastStatus == "failed" || resource.Healthcheck.LastStatus == "unhealthy" {
			plan.Reasons = append(plan.Reasons, "server_resource_unhealthy:"+resource.ID)
		}
	}
	if len(plan.Reasons) > 0 {
		return finish(rootDir, plan)
	}
	if options.Environment == "production" && !options.Approved {
		plan.ManualRequired = true
		plan.Reasons = append(plan.Reasons, "production_approval_required")
		approval, err := approvals.Request(rootDir, approvals.RequestOptions{
			TargetType:  "deployment_plan",
			TargetID:    plan.ID,
			Action:      "deploy.production.plan",
			RiskLevel:   "critical",
			RequestedBy: "system",
			Reason:      "production deployment plan requires approval",
			Metadata: map[string]any{
				"release_id":  plan.ReleaseID,
				"environment": plan.Environment,
			},
		})
		if err != nil {
			return Plan{}, err
		}
		plan.ApprovalID = approval.ID
		return finish(rootDir, plan)
	}
	plan.Status = "planned"
	plan.Decision = "DEPLOY_PLAN_READY"
	plan.ManualRequired = true
	plan.Reasons = append(plan.Reasons, "release_and_resources_ready")
	plan.SmokePlan = StepPlan{Status: "planned", Required: true, Actions: []string{"run configured smoke tests", "record smoke result"}}
	plan.MonitorPlan = StepPlan{Status: "planned", Required: true, Window: "30m", Actions: []string{"watch configured monitor signals", "record monitor window result"}}
	plan.RollbackPlan = StepPlan{Status: "planned", Required: true, Actions: []string{"rollback to previous release if smoke or monitor fails"}}
	return finish(rootDir, plan)
}

func Load(rootDir string, id string) (Plan, bool, error) {
	var plan Plan
	found, err := fsutil.ReadJSON(planPath(rootDir, id), &plan)
	return plan, found, err
}

func ListPlans(rootDir string, limit int) ([]Plan, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(workspace.ForRoot(rootDir).DeploymentsDir)
	if err != nil {
		return nil, err
	}
	plans := []Plan{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var plan Plan
		found, err := fsutil.ReadJSON(filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, entry.Name()), &plan)
		if err != nil {
			return nil, err
		}
		if found && plan.ID != "" {
			plans = append(plans, plan)
		}
	}
	sort.SliceStable(plans, func(i, j int) bool {
		return plans[i].CreatedAt > plans[j].CreatedAt
	})
	if limit > 0 && len(plans) > limit {
		return plans[:limit], nil
	}
	return plans, nil
}

func Execute(ctx context.Context, rootDir string, options ExecuteOptions) (Execution, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return Execution{}, err
	}
	options.DeploymentID = strings.TrimSpace(options.DeploymentID)
	options.Mode = normalizeToken(options.Mode)
	options.Commands = normalizeCommands(options.Commands)
	if options.DeploymentID == "" {
		return Execution{}, errors.New("deployment_id_required")
	}
	if options.Mode == "" {
		options.Mode = "dry_run"
	}
	now := time.Now().UTC()
	execution := Execution{
		ID:           "deploy-exec-" + textutil.Slugify(options.DeploymentID+"-"+options.Mode) + "-" + now.Format("20060102150405"),
		DeploymentID: options.DeploymentID,
		Mode:         options.Mode,
		Status:       "blocked",
		Decision:     "DEPLOY_EXECUTION_BLOCKED",
		Reasons:      []string{},
		Steps:        []ExecutionStep{},
		StartedAt:    now.Format(time.RFC3339Nano),
	}
	plan, found, err := Load(rootDir, options.DeploymentID)
	if err != nil {
		return Execution{}, err
	}
	if !found {
		execution.Reasons = append(execution.Reasons, "deployment_not_found")
		return finishExecution(rootDir, execution)
	}
	execution.ReleaseID = plan.ReleaseID
	execution.Environment = plan.Environment
	execution.Resources = plan.Resources
	if plan.Status != "planned" || plan.Decision != "DEPLOY_PLAN_READY" {
		execution.Reasons = append(execution.Reasons, "deployment_plan_not_ready:"+plan.Decision)
		return finishExecution(rootDir, execution)
	}
	if len(plan.Resources) == 0 {
		execution.Reasons = append(execution.Reasons, "deployment_resources_missing")
		return finishExecution(rootDir, execution)
	}
	if requiresExecutionApproval(options.Mode) && !options.Approved {
		execution.Reasons = append(execution.Reasons, "execution_approval_required")
		approval, err := approvals.Request(rootDir, approvals.RequestOptions{
			TargetType:  "deployment_execution",
			TargetID:    execution.ID,
			Action:      "deploy.execute." + options.Mode,
			RiskLevel:   riskForExecution(plan.Environment),
			RequestedBy: "system",
			Reason:      "non dry-run deployment execution requires approval",
			Metadata: map[string]any{
				"deployment_id": execution.DeploymentID,
				"release_id":    execution.ReleaseID,
				"environment":   execution.Environment,
				"mode":          execution.Mode,
			},
		})
		if err != nil {
			return Execution{}, err
		}
		execution.ApprovalID = approval.ID
		return finishExecution(rootDir, execution)
	}
	if plan.Environment == "production" && isRealExecutionMode(options.Mode) {
		execution.ManualBlock("production_real_execution_not_enabled")
		return finishExecution(rootDir, execution)
	}
	switch options.Mode {
	case "dry_run":
		execution.Status = "completed"
		execution.Decision = "DEPLOY_EXECUTION_DRY_RUN"
		execution.Reasons = append(execution.Reasons, "no_remote_or_local_commands_executed")
		execution.Steps = dryRunSteps(plan, options.Commands)
	case "ssh_preview":
		remotePlan, steps, ok, reasons := buildSSHPreview(rootDir, plan, options.Commands)
		execution.RemotePlan = &remotePlan
		execution.Steps = steps
		execution.Reasons = append(execution.Reasons, reasons...)
		if ok {
			execution.Status = "completed"
			execution.Decision = "DEPLOY_SSH_PREVIEW_READY"
			_ = logging.Log(rootDir, "release", "deployment.ssh.previewed", map[string]any{
				"deployment_id": execution.DeploymentID,
				"release_id":    execution.ReleaseID,
				"environment":   execution.Environment,
				"targets":       len(remotePlan.Targets),
				"decision":      execution.Decision,
			})
		}
	case "local_shell":
		if len(options.Commands) == 0 {
			execution.Reasons = append(execution.Reasons, "commands_required")
			return finishExecution(rootDir, execution)
		}
		steps, ok, reasons := runLocalShell(ctx, rootDir, options.Commands)
		execution.Steps = steps
		execution.Reasons = append(execution.Reasons, reasons...)
		if ok {
			smoke, monitor, rollback, postSteps, postOK, postReasons := runPostDeploymentChecks(ctx, rootDir, plan)
			execution.SmokeReport = smoke
			execution.MonitorReport = monitor
			execution.RollbackSuggestion = rollback
			execution.Steps = append(execution.Steps, postSteps...)
			execution.Reasons = append(execution.Reasons, postReasons...)
			switch {
			case !postOK && smoke.Status == "failed":
				execution.Status = "failed"
				execution.Decision = "DEPLOY_SMOKE_FAILED"
			case !postOK && monitor.Status == "failed":
				execution.Status = "failed"
				execution.Decision = "DEPLOY_MONITOR_FAILED"
			default:
				execution.Status = "completed"
				execution.Decision = "DEPLOY_EXECUTION_COMPLETED"
			}
		} else {
			execution.Status = "failed"
			execution.Decision = "DEPLOY_EXECUTION_FAILED"
			execution.RollbackSuggestion = rollbackFor(plan, "deploy_command_failed")
		}
	case "ssh_execute":
		execution.RemoteExecEnabled = sshExecutionEnabled()
		remotePlan, steps, ok, reasons := buildSSHExecutionPlan(rootDir, plan, options.Commands, execution.RemoteExecEnabled)
		execution.RemotePlan = &remotePlan
		execution.Steps = steps
		execution.Reasons = append(execution.Reasons, reasons...)
		switch {
		case !execution.RemoteExecEnabled:
			execution.Decision = "DEPLOY_EXECUTION_BLOCKED"
		case ok:
			execution.Decision = "DEPLOY_SSH_EXECUTION_GUARDED_READY"
			_ = logging.Log(rootDir, "release", "deployment.ssh.execution.guarded", map[string]any{
				"deployment_id": execution.DeploymentID,
				"release_id":    execution.ReleaseID,
				"environment":   execution.Environment,
				"targets":       len(remotePlan.Targets),
				"decision":      execution.Decision,
			})
		default:
			execution.Decision = "DEPLOY_SSH_EXECUTION_BLOCKED"
		}
	default:
		execution.Reasons = append(execution.Reasons, "execution_mode_not_allowed:"+options.Mode)
	}
	return finishExecution(rootDir, execution)
}

func LoadExecution(rootDir string, id string) (Execution, bool, error) {
	var execution Execution
	found, err := fsutil.ReadJSON(executionPath(rootDir, id), &execution)
	return execution, found, err
}

func ListExecutions(rootDir string, limit int) ([]Execution, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return nil, err
	}
	dir := filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "executions")
	if err := fsutil.EnsureDir(dir); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	executions := []Execution{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var execution Execution
		found, err := fsutil.ReadJSON(filepath.Join(dir, entry.Name()), &execution)
		if err != nil {
			return nil, err
		}
		if found && execution.ID != "" {
			executions = append(executions, execution)
		}
	}
	sort.SliceStable(executions, func(i, j int) bool {
		return executions[i].StartedAt > executions[j].StartedAt
	})
	if limit > 0 && len(executions) > limit {
		return executions[:limit], nil
	}
	return executions, nil
}

func finish(rootDir string, plan Plan) (Plan, error) {
	if err := fsutil.WriteJSON(planPath(rootDir, plan.ID), plan); err != nil {
		return Plan{}, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "plans.jsonl"), plan); err != nil {
		return Plan{}, err
	}
	_ = logging.Log(rootDir, "release", "deployment.plan.created", map[string]any{"deployment_id": plan.ID, "release_id": plan.ReleaseID, "decision": plan.Decision, "status": plan.Status, "environment": plan.Environment})
	return plan, nil
}

func finishExecution(rootDir string, execution Execution) (Execution, error) {
	execution.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := fsutil.WriteJSON(executionPath(rootDir, execution.ID), execution); err != nil {
		return Execution{}, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "executions.jsonl"), execution); err != nil {
		return Execution{}, err
	}
	_ = logging.Log(rootDir, "release", "deployment.execution.created", map[string]any{
		"execution_id":  execution.ID,
		"deployment_id": execution.DeploymentID,
		"decision":      execution.Decision,
		"status":        execution.Status,
		"environment":   execution.Environment,
		"mode":          execution.Mode,
	})
	return execution, nil
}

func planPath(rootDir string, id string) string {
	return filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, id+".json")
}

func executionPath(rootDir string, id string) string {
	return filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "executions", id+".json")
}

func resolveResources(rootDir string, environment string, ids []string) ([]serverresources.Resource, error) {
	if len(ids) > 0 {
		resources := []serverresources.Resource{}
		for _, id := range ids {
			resource, ok, err := serverresources.Show(rootDir, id)
			if err != nil {
				return nil, err
			}
			if ok {
				resources = append(resources, resource)
			}
		}
		return resources, nil
	}
	all, err := serverresources.List(rootDir)
	if err != nil {
		return nil, err
	}
	resources := []serverresources.Resource{}
	for _, resource := range all {
		if resource.Environment == environment && resource.Status == "active" {
			resources = append(resources, resource)
		}
	}
	return resources, nil
}

func normalizeIDs(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(strings.ToLower(value))
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func normalizeCommands(values []string) []string {
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func normalizeToken(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	return strings.ReplaceAll(value, "-", "_")
}

func riskForExecution(environment string) string {
	if environment == "production" {
		return "critical"
	}
	return "high"
}

func requiresExecutionApproval(mode string) bool {
	return isRealExecutionMode(mode)
}

func isRealExecutionMode(mode string) bool {
	switch mode {
	case "dry_run", "ssh_preview":
		return false
	default:
		return true
	}
}

func dryRunSteps(plan Plan, commands []string) []ExecutionStep {
	steps := []ExecutionStep{
		{Name: "deploy", Status: "dry_run", Output: "deployment command preview only"},
		{Name: "smoke", Status: "dry_run", Output: strings.Join(plan.SmokePlan.Actions, "; ")},
		{Name: "monitor", Status: "dry_run", Output: strings.Join(plan.MonitorPlan.Actions, "; ")},
		{Name: "rollback", Status: "dry_run", Output: strings.Join(plan.RollbackPlan.Actions, "; ")},
	}
	for _, command := range commands {
		steps = append(steps, ExecutionStep{Name: "command_preview", Status: "dry_run", Command: command, Allowlist: safeShellPrefixes()})
	}
	return steps
}

func buildSSHPreview(rootDir string, plan Plan, commands []string) (RemotePlan, []ExecutionStep, bool, []string) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	remotePlan := RemotePlan{
		Status:    "planned",
		Decision:  "SSH_PREVIEW_READY",
		Targets:   []RemoteTarget{},
		CreatedAt: now,
	}
	steps := []ExecutionStep{}
	reasons := []string{"ssh_preview_no_remote_commands_executed"}
	ok := true
	previewCommands := remotePreviewCommands(plan, commands)
	for _, summary := range plan.Resources {
		resource, found, err := serverresources.Show(rootDir, summary.ID)
		target := RemoteTarget{
			ResourceID:  summary.ID,
			Environment: summary.Environment,
			Host:        summary.Host,
			Status:      "planned",
			Reason:      "ssh_preview_ready",
			Commands:    append([]string{}, previewCommands...),
		}
		if err != nil || !found {
			target.Status = "blocked"
			target.Reason = "server_resource_not_found"
			reasons = append(reasons, "server_resource_not_found:"+summary.ID)
			ok = false
		} else {
			target.Environment = resource.Environment
			target.Host = resource.Host
			target.Provider = resource.Provider
			target.AuthRef = resource.AuthRef
			target.Status, target.Reason = validateRemoteTarget(plan, resource)
			if target.Status == "blocked" {
				reasons = append(reasons, target.Reason+":"+resource.ID)
				ok = false
			}
		}
		remotePlan.Targets = append(remotePlan.Targets, target)
		steps = append(steps, ExecutionStep{
			Name:       "ssh_preview",
			Status:     target.Status,
			Output:     target.ResourceID + ":" + target.Host + ":" + target.Reason,
			Allowlist:  []string{"preview_only", "secret_ref_only", "no_remote_command_executed"},
			StartedAt:  now,
			FinishedAt: time.Now().UTC().Format(time.RFC3339Nano),
		})
	}
	if len(remotePlan.Targets) == 0 {
		remotePlan.Status = "blocked"
		remotePlan.Decision = "SSH_PREVIEW_BLOCKED"
		reasons = append(reasons, "deployment_resources_missing")
		return remotePlan, steps, false, reasons
	}
	if !ok {
		remotePlan.Status = "blocked"
		remotePlan.Decision = "SSH_PREVIEW_BLOCKED"
	}
	return remotePlan, steps, ok, reasons
}

func buildSSHExecutionPlan(rootDir string, plan Plan, commands []string, enabled bool) (RemotePlan, []ExecutionStep, bool, []string) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	remotePlan := RemotePlan{
		Status:    "blocked",
		Decision:  "SSH_EXECUTION_NOT_ENABLED",
		Targets:   []RemoteTarget{},
		CreatedAt: now,
	}
	steps := []ExecutionStep{}
	reasons := []string{}
	if !enabled {
		reasons = append(reasons, "ssh_real_execution_not_enabled")
	}
	if len(commands) == 0 {
		reasons = append(reasons, "commands_required")
	}
	ok := enabled && len(commands) > 0
	for _, summary := range plan.Resources {
		resource, found, err := serverresources.Show(rootDir, summary.ID)
		target := RemoteTarget{
			ResourceID:  summary.ID,
			Environment: summary.Environment,
			Host:        summary.Host,
			Status:      "blocked",
			Reason:      "ssh_real_execution_not_enabled",
			Commands:    append([]string{}, commands...),
		}
		if err != nil || !found {
			target.Reason = "server_resource_not_found"
			reasons = append(reasons, "server_resource_not_found:"+summary.ID)
			ok = false
		} else {
			target.Environment = resource.Environment
			target.Host = resource.Host
			target.Provider = resource.Provider
			target.AuthRef = resource.AuthRef
			if enabled {
				target.Status, target.Reason = validateRemoteTarget(plan, resource)
				if target.Status == "planned" {
					target.Reason = "ssh_guarded_execution_ready"
				}
			}
			if enabled && len(commands) == 0 {
				target.Status = "blocked"
				target.Reason = "commands_required"
			}
			if target.Status == "blocked" && target.Reason != "ssh_real_execution_not_enabled" {
				reasons = append(reasons, target.Reason+":"+resource.ID)
				ok = false
			}
		}
		for _, command := range commands {
			if !isSafeSSHCommand(command) {
				target.Status = "blocked"
				target.Reason = "command_not_allowed"
				reasons = append(reasons, "command_not_allowed")
				ok = false
				break
			}
		}
		remotePlan.Targets = append(remotePlan.Targets, target)
		steps = append(steps, ExecutionStep{
			Name:       "ssh_execute",
			Status:     target.Status,
			Command:    commandSummary(commands),
			Output:     target.ResourceID + ":" + target.Host + ":" + target.Reason,
			Allowlist:  safeSSHPrefixes(),
			StartedAt:  now,
			FinishedAt: time.Now().UTC().Format(time.RFC3339Nano),
		})
	}
	if len(remotePlan.Targets) == 0 {
		reasons = append(reasons, "deployment_resources_missing")
		return remotePlan, steps, false, reasons
	}
	if ok {
		remotePlan.Status = "planned"
		remotePlan.Decision = "SSH_EXECUTION_GUARDED_READY"
		reasons = append(reasons, "ssh_guarded_execution_ready", "remote_ssh_command_runner_not_enabled")
		return remotePlan, steps, true, reasons
	}
	if enabled {
		remotePlan.Decision = "SSH_EXECUTION_BLOCKED"
	}
	return remotePlan, steps, false, reasons
}

func commandSummary(commands []string) string {
	if len(commands) == 0 {
		return ""
	}
	if len(commands) == 1 {
		return commands[0]
	}
	return fmt.Sprintf("%d remote commands", len(commands))
}

func validateRemoteTarget(plan Plan, resource serverresources.Resource) (string, string) {
	if resource.Status != "active" {
		return "blocked", "server_resource_not_active"
	}
	if resource.Environment != plan.Environment {
		return "blocked", "server_resource_environment_mismatch"
	}
	if strings.TrimSpace(resource.Host) == "" {
		return "blocked", "server_resource_host_required"
	}
	if strings.TrimSpace(resource.AuthRef) == "" {
		return "blocked", "server_resource_auth_ref_required"
	}
	if !isReference(resource.AuthRef) {
		return "blocked", "server_resource_auth_ref_must_be_reference"
	}
	return "planned", "ssh_preview_ready"
}

func remotePreviewCommands(plan Plan, commands []string) []string {
	if len(commands) > 0 {
		return append([]string{}, commands...)
	}
	return []string{
		"deploy release " + plan.ReleaseID,
		"run configured smoke tests",
		"watch configured monitor window",
	}
}

func isReference(value string) bool {
	value = strings.TrimSpace(value)
	return (strings.HasPrefix(value, "env:") && len(value) > len("env:")) || (strings.HasPrefix(value, "secret:") && len(value) > len("secret:"))
}

func runPostDeploymentChecks(ctx context.Context, rootDir string, plan Plan) (CheckReport, CheckReport, RollbackSuggestion, []ExecutionStep, bool, []string) {
	smoke := runCheckReport(ctx, rootDir, plan, "smoke")
	steps := []ExecutionStep{stepFromReport("smoke", smoke)}
	reasons := []string{}
	ok := smoke.Status != "failed"
	if ok {
		reasons = append(reasons, "smoke_"+smoke.Status)
	} else {
		reasons = append(reasons, "smoke_failed")
		rollback := rollbackFor(plan, "smoke_failed")
		steps = append(steps, stepFromRollback(rollback))
		logPostDeploymentChecks(rootDir, plan, smoke, CheckReport{}, rollback)
		return smoke, CheckReport{}, rollback, steps, false, reasons
	}
	monitor := runCheckReport(ctx, rootDir, plan, "monitor")
	steps = append(steps, stepFromReport("monitor", monitor))
	if monitor.Status == "failed" {
		reasons = append(reasons, "monitor_failed")
		rollback := rollbackFor(plan, "monitor_failed")
		steps = append(steps, stepFromRollback(rollback))
		logPostDeploymentChecks(rootDir, plan, smoke, monitor, rollback)
		return smoke, monitor, rollback, steps, false, reasons
	}
	reasons = append(reasons, "monitor_"+monitor.Status)
	rollback := RollbackSuggestion{Required: false, Decision: "ROLLBACK_NOT_REQUIRED", Reason: "smoke_and_monitor_not_failed"}
	logPostDeploymentChecks(rootDir, plan, smoke, monitor, rollback)
	return smoke, monitor, rollback, steps, true, reasons
}

func runCheckReport(ctx context.Context, rootDir string, plan Plan, checkType string) CheckReport {
	report := CheckReport{
		Status:    "passed",
		Decision:  strings.ToUpper(checkType) + "_PASSED",
		Results:   []CheckResult{},
		CheckedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	for _, summary := range plan.Resources {
		resource, found, err := serverresources.Show(rootDir, summary.ID)
		if err != nil || !found {
			report.Results = append(report.Results, CheckResult{ResourceID: summary.ID, Status: "failed", Reason: "resource_not_found"})
			report.Status = "failed"
			report.Decision = strings.ToUpper(checkType) + "_FAILED"
			continue
		}
		result := runResourceCheck(ctx, resource)
		report.Results = append(report.Results, result)
		if result.Status == "failed" || result.Status == "blocked" {
			report.Status = "failed"
			report.Decision = strings.ToUpper(checkType) + "_FAILED"
		}
	}
	if len(report.Results) == 0 {
		report.Status = "failed"
		report.Decision = strings.ToUpper(checkType) + "_FAILED"
		report.Results = append(report.Results, CheckResult{Status: "failed", Reason: "deployment_resources_missing"})
	}
	if report.Status == "passed" && allManualOrSkipped(report.Results) {
		report.Status = "manual_required"
		report.Decision = strings.ToUpper(checkType) + "_MANUAL_REQUIRED"
	}
	return report
}

func runResourceCheck(ctx context.Context, resource serverresources.Resource) CheckResult {
	result := CheckResult{ResourceID: resource.ID, Target: resource.Healthcheck.Target, Status: "manual_required", Reason: "manual_healthcheck"}
	checkType := normalizeToken(resource.Healthcheck.Type)
	if checkType == "" || checkType == "manual" {
		return result
	}
	if checkType != "http" && checkType != "https" {
		result.Status = "blocked"
		result.Reason = "healthcheck_type_not_allowed:" + checkType
		return result
	}
	target := strings.TrimSpace(resource.Healthcheck.Target)
	if target == "" {
		result.Status = "blocked"
		result.Reason = "healthcheck_target_required"
		return result
	}
	parsed, err := url.Parse(target)
	if err != nil || parsed.Hostname() == "" {
		result.Status = "blocked"
		result.Reason = "healthcheck_target_invalid"
		return result
	}
	scheme := normalizeToken(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		result.Status = "blocked"
		result.Reason = "healthcheck_scheme_not_allowed"
		return result
	}
	if scheme != checkType {
		result.Status = "blocked"
		result.Reason = "healthcheck_scheme_mismatch"
		return result
	}
	if parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "localhost" {
		result.Status = "blocked"
		result.Reason = "healthcheck_target_not_allowed"
		return result
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		result.Status = "blocked"
		result.Reason = "healthcheck_request_invalid"
		return result
	}
	response, err := (&http.Client{Timeout: 3 * time.Second}).Do(request)
	if err != nil {
		result.Status = "failed"
		result.Reason = "healthcheck_request_failed"
		return result
	}
	defer response.Body.Close()
	result.HTTPStatus = response.StatusCode
	if response.StatusCode >= 200 && response.StatusCode < 400 {
		result.Status = "passed"
		result.Reason = "healthcheck_ok"
		return result
	}
	result.Status = "failed"
	result.Reason = fmt.Sprintf("healthcheck_http_status:%d", response.StatusCode)
	return result
}

func stepFromReport(name string, report CheckReport) ExecutionStep {
	outputs := []string{}
	for _, result := range report.Results {
		output := result.ResourceID + ":" + result.Status
		if result.Reason != "" {
			output += ":" + result.Reason
		}
		outputs = append(outputs, output)
	}
	return ExecutionStep{
		Name:       name,
		Status:     report.Status,
		Output:     strings.Join(outputs, "; "),
		StartedAt:  report.CheckedAt,
		FinishedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func stepFromRollback(rollback RollbackSuggestion) ExecutionStep {
	return ExecutionStep{
		Name:       "rollback",
		Status:     "suggested",
		Output:     strings.Join(rollback.Actions, "; "),
		StartedAt:  time.Now().UTC().Format(time.RFC3339Nano),
		FinishedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func rollbackFor(plan Plan, reason string) RollbackSuggestion {
	actions := append([]string{}, plan.RollbackPlan.Actions...)
	if len(actions) == 0 {
		actions = []string{"restore previous release", "rerun smoke checks", "keep incident record open"}
	}
	return RollbackSuggestion{Required: true, Decision: "ROLLBACK_RECOMMENDED", Reason: reason, Actions: actions}
}

func allManualOrSkipped(results []CheckResult) bool {
	if len(results) == 0 {
		return false
	}
	for _, result := range results {
		if result.Status != "manual_required" && result.Status != "skipped" {
			return false
		}
	}
	return true
}

func logPostDeploymentChecks(rootDir string, plan Plan, smoke CheckReport, monitor CheckReport, rollback RollbackSuggestion) {
	if smoke.Status != "" {
		_ = logging.Log(rootDir, "release", "deployment.smoke.completed", map[string]any{"deployment_id": plan.ID, "release_id": plan.ReleaseID, "status": smoke.Status, "decision": smoke.Decision})
	}
	if monitor.Status != "" {
		_ = logging.Log(rootDir, "release", "deployment.monitor.completed", map[string]any{"deployment_id": plan.ID, "release_id": plan.ReleaseID, "status": monitor.Status, "decision": monitor.Decision})
	}
	if rollback.Required {
		_ = logging.Log(rootDir, "release", "deployment.rollback.suggested", map[string]any{"deployment_id": plan.ID, "release_id": plan.ReleaseID, "reason": rollback.Reason, "decision": rollback.Decision})
	}
}

func runLocalShell(ctx context.Context, rootDir string, commands []string) ([]ExecutionStep, bool, []string) {
	steps := []ExecutionStep{}
	reasons := []string{}
	ok := true
	for _, command := range commands {
		step := ExecutionStep{Name: "local_shell", Command: command, Allowlist: safeShellPrefixes(), StartedAt: time.Now().UTC().Format(time.RFC3339Nano)}
		if !isSafeShellCommand(command) {
			step.Status = "blocked"
			step.Error = "command_not_allowed"
			reasons = append(reasons, "command_not_allowed")
			ok = false
			step.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
			steps = append(steps, step)
			continue
		}
		result := process.RunShell(ctx, rootDir, command)
		step.Output = strings.TrimSpace(result.Stdout)
		step.Error = strings.TrimSpace(result.Stderr)
		if result.Code == 0 {
			step.Status = "completed"
		} else {
			step.Status = "failed"
			reasons = append(reasons, "command_failed")
			ok = false
		}
		step.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		steps = append(steps, step)
	}
	if ok {
		reasons = append(reasons, "allowed_local_shell_commands_completed")
	}
	return steps, ok, reasons
}

func isSafeShellCommand(command string) bool {
	if strings.ContainsAny(command, "\n\r") {
		return false
	}
	for _, token := range []string{";", "&&", "||", "`", "$(", ">", "<", "|"} {
		if strings.Contains(command, token) {
			return false
		}
	}
	for _, prefix := range safeShellPrefixes() {
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

func isSafeSSHCommand(command string) bool {
	if strings.ContainsAny(command, "\n\r") {
		return false
	}
	for _, token := range []string{";", "&&", "||", "`", "$(", ">", "<", "|"} {
		if strings.Contains(command, token) {
			return false
		}
	}
	for _, prefix := range safeSSHPrefixes() {
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

func safeShellPrefixes() []string {
	return []string{
		"true",
		"echo ",
		"printf ",
		"curl -fsS http://127.0.0.1",
		"curl -fsS http://localhost",
	}
}

func safeSSHPrefixes() []string {
	return []string{
		"true",
		"echo ",
		"printf ",
		"curl -fsS http://127.0.0.1",
		"curl -fsS http://localhost",
		"systemctl status ",
		"docker ps",
		"docker compose ps",
	}
}

func sshExecutionEnabled() bool {
	return os.Getenv("MOYUAN_ALLOW_SSH_EXECUTE") == "1"
}

func (execution *Execution) ManualBlock(reason string) {
	execution.Reasons = append(execution.Reasons, reason)
	execution.Status = "blocked"
	execution.Decision = "DEPLOY_EXECUTION_BLOCKED"
}
