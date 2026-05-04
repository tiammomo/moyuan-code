package deployment

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
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

func planPath(rootDir string, id string) string {
	return filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, id+".json")
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

func normalizeToken(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	return strings.ReplaceAll(value, "-", "_")
}
