package deployment

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
	ID           string            `json:"id"`
	DeploymentID string            `json:"deployment_id"`
	ReleaseID    string            `json:"release_id"`
	Environment  string            `json:"environment"`
	Mode         string            `json:"mode"`
	Status       string            `json:"status"`
	Decision     string            `json:"decision"`
	Reasons      []string          `json:"reasons"`
	Resources    []ResourceSummary `json:"resources"`
	Steps        []ExecutionStep   `json:"steps"`
	StartedAt    string            `json:"started_at"`
	FinishedAt   string            `json:"finished_at,omitempty"`
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
	if options.Mode != "dry_run" && !options.Approved {
		execution.Reasons = append(execution.Reasons, "execution_approval_required")
		return finishExecution(rootDir, execution)
	}
	if plan.Environment == "production" && options.Mode != "dry_run" {
		execution.ManualBlock("production_real_execution_not_enabled")
		return finishExecution(rootDir, execution)
	}
	switch options.Mode {
	case "dry_run":
		execution.Status = "completed"
		execution.Decision = "DEPLOY_EXECUTION_DRY_RUN"
		execution.Reasons = append(execution.Reasons, "no_remote_or_local_commands_executed")
		execution.Steps = dryRunSteps(plan, options.Commands)
	case "local_shell":
		if len(options.Commands) == 0 {
			execution.Reasons = append(execution.Reasons, "commands_required")
			return finishExecution(rootDir, execution)
		}
		steps, ok, reasons := runLocalShell(ctx, rootDir, options.Commands)
		execution.Steps = steps
		execution.Reasons = append(execution.Reasons, reasons...)
		if ok {
			execution.Status = "completed"
			execution.Decision = "DEPLOY_EXECUTION_COMPLETED"
		} else {
			execution.Status = "failed"
			execution.Decision = "DEPLOY_EXECUTION_FAILED"
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

func safeShellPrefixes() []string {
	return []string{
		"true",
		"echo ",
		"printf ",
		"curl -fsS http://127.0.0.1",
		"curl -fsS http://localhost",
	}
}

func (execution *Execution) ManualBlock(reason string) {
	execution.Reasons = append(execution.Reasons, reason)
	execution.Status = "blocked"
	execution.Decision = "DEPLOY_EXECUTION_BLOCKED"
}
