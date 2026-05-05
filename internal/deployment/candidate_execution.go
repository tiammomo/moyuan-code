package deployment

import (
	"context"
	"errors"
	"strings"
	"time"

	"moyuan-code/internal/release"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type CandidateExecuteOptions struct {
	CandidateID  string   `json:"candidate_id"`
	DeploymentID string   `json:"deployment_id,omitempty"`
	Environment  string   `json:"environment,omitempty"`
	Mode         string   `json:"mode,omitempty"`
	Approved     bool     `json:"approved,omitempty"`
	ApprovalID   string   `json:"approval_id,omitempty"`
	Commands     []string `json:"commands,omitempty"`
}

func ExecuteFromCandidate(ctx context.Context, rootDir string, options CandidateExecuteOptions) (Execution, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return Execution{}, err
	}
	options.CandidateID = strings.TrimSpace(options.CandidateID)
	options.DeploymentID = strings.TrimSpace(options.DeploymentID)
	options.Environment = normalizeToken(options.Environment)
	options.Mode = normalizeToken(options.Mode)
	options.ApprovalID = strings.TrimSpace(options.ApprovalID)
	options.Commands = normalizeCommands(options.Commands)
	if options.CandidateID == "" {
		return Execution{}, errors.New("candidate_id_required")
	}
	if options.Mode == "" {
		options.Mode = "dry_run"
	}
	candidate, found, err := release.LoadCandidate(rootDir, options.CandidateID)
	if err != nil {
		return Execution{}, err
	}
	if !found {
		return finishCandidateExecutionBlocked(rootDir, options, "release_candidate_not_found")
	}
	if candidate.Status != "ready" || candidate.Decision != "RELEASE_CANDIDATE_READY" {
		return finishCandidateExecutionBlocked(rootDir, options, "release_candidate_not_ready:"+candidate.Decision)
	}
	deploymentID := options.DeploymentID
	if deploymentID == "" {
		plan, found, err := LatestPlanForCandidate(rootDir, options.CandidateID, options.Environment)
		if err != nil {
			return Execution{}, err
		}
		if !found {
			return finishCandidateExecutionBlocked(rootDir, options, "deployment_plan_missing")
		}
		deploymentID = plan.ID
	} else {
		plan, found, err := Load(rootDir, deploymentID)
		if err != nil {
			return Execution{}, err
		}
		if !found {
			return finishCandidateExecutionBlocked(rootDir, options, "deployment_plan_missing")
		}
		if plan.ReleaseID != options.CandidateID {
			return finishCandidateExecutionBlocked(rootDir, options, "deployment_candidate_mismatch")
		}
	}
	return Execute(ctx, rootDir, ExecuteOptions{
		DeploymentID: deploymentID,
		Mode:         options.Mode,
		Approved:     options.Approved,
		ApprovalID:   options.ApprovalID,
		Commands:     options.Commands,
	})
}

func LatestPlanForCandidate(rootDir string, candidateID string, environment string) (Plan, bool, error) {
	plans, err := ListPlans(rootDir, 100)
	if err != nil {
		return Plan{}, false, err
	}
	environment = normalizeToken(environment)
	for _, plan := range plans {
		if plan.ReleaseID != strings.TrimSpace(candidateID) {
			continue
		}
		if environment != "" && plan.Environment != environment {
			continue
		}
		return plan, true, nil
	}
	return Plan{}, false, nil
}

func finishCandidateExecutionBlocked(rootDir string, options CandidateExecuteOptions, reason string) (Execution, error) {
	now := time.Now().UTC()
	execution := Execution{
		ID:           "deploy-exec-candidate-" + textutil.Slugify(options.CandidateID+"-"+options.Mode) + "-" + now.Format("20060102150405"),
		DeploymentID: options.DeploymentID,
		ReleaseID:    options.CandidateID,
		Environment:  options.Environment,
		Mode:         options.Mode,
		Status:       "blocked",
		Decision:     "DEPLOY_EXECUTION_BLOCKED",
		Reasons:      []string{reason},
		Steps:        []ExecutionStep{},
		StartedAt:    now.Format(time.RFC3339Nano),
	}
	return finishExecution(rootDir, execution)
}
