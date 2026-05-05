package deployment

import (
	"errors"
	"strings"
	"time"

	"moyuan-code/internal/approvals"
	"moyuan-code/internal/release"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type CandidatePlanOptions struct {
	CandidateID string   `json:"candidate_id"`
	Environment string   `json:"environment"`
	ResourceIDs []string `json:"resource_ids"`
	Approved    bool     `json:"approved"`
}

func CreatePlanFromCandidate(rootDir string, options CandidatePlanOptions) (Plan, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return Plan{}, err
	}
	options.CandidateID = strings.TrimSpace(options.CandidateID)
	options.Environment = normalizeToken(options.Environment)
	options.ResourceIDs = normalizeIDs(options.ResourceIDs)
	if options.CandidateID == "" {
		return Plan{}, errors.New("candidate_id_required")
	}
	if options.Environment == "" {
		return Plan{}, errors.New("environment_required")
	}
	now := time.Now().UTC()
	plan := Plan{
		ID:          "deployment-" + textutil.Slugify(options.CandidateID+"-"+options.Environment) + "-" + now.Format("20060102150405"),
		ReleaseID:   options.CandidateID,
		Environment: options.Environment,
		Status:      "blocked",
		Decision:    "DEPLOY_BLOCKED",
		Reasons:     []string{},
		Resources:   []ResourceSummary{},
		CreatedAt:   now.Format(time.RFC3339Nano),
	}
	candidate, found, err := release.LoadCandidate(rootDir, options.CandidateID)
	if err != nil {
		return Plan{}, err
	}
	if !found {
		plan.Reasons = append(plan.Reasons, "release_candidate_not_found")
		return finish(rootDir, plan)
	}
	if candidate.Status != "ready" || candidate.Decision != "RELEASE_CANDIDATE_READY" {
		plan.Reasons = append(plan.Reasons, "release_candidate_not_ready:"+candidate.Decision)
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
	appendPlanResourceReadiness(&plan, resources)
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
				"release_candidate_id": plan.ReleaseID,
				"environment":          plan.Environment,
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
	plan.Reasons = append(plan.Reasons, "release_candidate_and_resources_ready")
	plan.SmokePlan = stepPlanFromCheckTemplate(defaultCheckTemplate("smoke", plan.Environment))
	plan.MonitorPlan = stepPlanFromCheckTemplate(defaultCheckTemplate("monitor", plan.Environment))
	plan.RollbackPlan = StepPlan{Status: "planned", Required: true, Actions: []string{"rollback to previous release if smoke or monitor fails"}}
	return finish(rootDir, plan)
}
