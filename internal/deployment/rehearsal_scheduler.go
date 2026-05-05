package deployment

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"moyuan-code/internal/evidence"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type RehearsalSchedulerOptions struct {
	Trigger       string `json:"trigger,omitempty"`
	CandidateID   string `json:"candidate_id,omitempty"`
	DeploymentID  string `json:"deployment_id,omitempty"`
	ExecutionID   string `json:"execution_id,omitempty"`
	Environment   string `json:"environment,omitempty"`
	MonitorLimit  int    `json:"monitor_limit,omitempty"`
	MaxTargets    int    `json:"max_targets,omitempty"`
	SkipAdmission bool   `json:"skip_admission,omitempty"`
	RequestedBy   string `json:"requested_by,omitempty"`
}

type RehearsalSchedulerRun struct {
	ID            string                     `json:"id"`
	Trigger       string                     `json:"trigger"`
	RequestedBy   string                     `json:"requested_by,omitempty"`
	CandidateID   string                     `json:"candidate_id,omitempty"`
	DeploymentID  string                     `json:"deployment_id,omitempty"`
	ExecutionID   string                     `json:"execution_id,omitempty"`
	Environment   string                     `json:"environment,omitempty"`
	Status        string                     `json:"status"`
	Decision      string                     `json:"decision"`
	Reasons       []string                   `json:"reasons"`
	MaxTargets    int                        `json:"max_targets"`
	SkipAdmission bool                       `json:"skip_admission"`
	Targets       []RehearsalSchedulerTarget `json:"targets"`
	CreatedCount  int                        `json:"created_count"`
	SkippedCount  int                        `json:"skipped_count"`
	BlockedCount  int                        `json:"blocked_count"`
	ManualCount   int                        `json:"manual_count"`
	RehearsalIDs  []string                   `json:"rehearsal_ids,omitempty"`
	AdmissionIDs  []string                   `json:"admission_ids,omitempty"`
	EvidenceIDs   []string                   `json:"evidence_ids,omitempty"`
	StartedAt     string                     `json:"started_at"`
	FinishedAt    string                     `json:"finished_at,omitempty"`
}

type RehearsalSchedulerTarget struct {
	Type         string `json:"type"`
	CandidateID  string `json:"candidate_id,omitempty"`
	DeploymentID string `json:"deployment_id,omitempty"`
	ExecutionID  string `json:"execution_id,omitempty"`
	Environment  string `json:"environment,omitempty"`
	Status       string `json:"status"`
	Decision     string `json:"decision"`
	Reason       string `json:"reason,omitempty"`
	RehearsalID  string `json:"rehearsal_id,omitempty"`
	AdmissionID  string `json:"admission_id,omitempty"`
}

func RunRehearsalScheduler(ctx context.Context, rootDir string, options RehearsalSchedulerOptions) (RehearsalSchedulerRun, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return RehearsalSchedulerRun{}, err
	}
	options = normalizeRehearsalSchedulerOptions(options)
	now := time.Now().UTC()
	run := RehearsalSchedulerRun{
		ID:            "rehearsal-scheduler-" + textutil.Slugify(rehearsalSchedulerSeed(options)) + "-" + now.Format("20060102150405") + "-" + strconv.FormatInt(now.UnixNano()%1_000_000_000, 10),
		Trigger:       options.Trigger,
		RequestedBy:   options.RequestedBy,
		CandidateID:   options.CandidateID,
		DeploymentID:  options.DeploymentID,
		ExecutionID:   options.ExecutionID,
		Environment:   options.Environment,
		Status:        "running",
		Decision:      "REHEARSAL_SCHEDULER_RUNNING",
		Reasons:       []string{},
		MaxTargets:    options.MaxTargets,
		SkipAdmission: options.SkipAdmission,
		Targets:       []RehearsalSchedulerTarget{},
		StartedAt:     now.Format(time.RFC3339Nano),
	}
	targets, err := selectRehearsalSchedulerTargets(rootDir, options)
	if err != nil {
		return RehearsalSchedulerRun{}, err
	}
	if len(targets) == 0 {
		run.Status = "blocked"
		run.Decision = "REHEARSAL_SCHEDULER_NO_TARGETS"
		run.Reasons = append(run.Reasons, "scheduler_targets_missing")
		return finishRehearsalSchedulerRun(rootDir, run)
	}
	for _, target := range targets {
		executed, err := executeRehearsalSchedulerTarget(ctx, rootDir, options, target)
		if err != nil {
			return RehearsalSchedulerRun{}, err
		}
		run.Targets = append(run.Targets, executed)
		if executed.RehearsalID != "" {
			run.RehearsalIDs = appendUnique(run.RehearsalIDs, executed.RehearsalID)
		}
		if executed.AdmissionID != "" {
			run.AdmissionIDs = appendUnique(run.AdmissionIDs, executed.AdmissionID)
		}
		switch executed.Status {
		case "created", "completed":
			run.CreatedCount++
		case "skipped":
			run.SkippedCount++
		case "manual_required":
			run.ManualCount++
		case "blocked":
			run.BlockedCount++
		}
	}
	return finishRehearsalSchedulerRun(rootDir, finalizeRehearsalSchedulerRun(run))
}

func LoadRehearsalSchedulerRun(rootDir string, id string) (RehearsalSchedulerRun, bool, error) {
	var run RehearsalSchedulerRun
	found, err := fsutil.ReadJSON(rehearsalSchedulerRunPath(rootDir, id), &run)
	return run, found, err
}

func ListRehearsalSchedulerRuns(rootDir string, limit int) ([]RehearsalSchedulerRun, error) {
	if err := fsutil.EnsureDir(rehearsalSchedulerRunDir(rootDir)); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(rehearsalSchedulerRunDir(rootDir))
	if err != nil {
		return nil, err
	}
	runs := []RehearsalSchedulerRun{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var run RehearsalSchedulerRun
		found, err := fsutil.ReadJSON(filepath.Join(rehearsalSchedulerRunDir(rootDir), entry.Name()), &run)
		if err != nil {
			return nil, err
		}
		if found && run.ID != "" {
			runs = append(runs, run)
		}
	}
	sort.SliceStable(runs, func(i, j int) bool {
		return runs[i].StartedAt > runs[j].StartedAt
	})
	if limit <= 0 {
		limit = 20
	}
	if len(runs) > limit {
		return runs[:limit], nil
	}
	return runs, nil
}

func normalizeRehearsalSchedulerOptions(options RehearsalSchedulerOptions) RehearsalSchedulerOptions {
	options.Trigger = normalizeToken(options.Trigger)
	if options.Trigger == "" {
		options.Trigger = "manual"
	}
	options.CandidateID = strings.TrimSpace(options.CandidateID)
	options.DeploymentID = strings.TrimSpace(options.DeploymentID)
	options.ExecutionID = strings.TrimSpace(options.ExecutionID)
	options.Environment = normalizeToken(options.Environment)
	options.RequestedBy = strings.TrimSpace(options.RequestedBy)
	if options.MonitorLimit <= 0 {
		options.MonitorLimit = 10
	}
	if options.MaxTargets <= 0 {
		options.MaxTargets = 3
	}
	if options.MaxTargets > 10 {
		options.MaxTargets = 10
	}
	return options
}

func selectRehearsalSchedulerTargets(rootDir string, options RehearsalSchedulerOptions) ([]RehearsalSchedulerTarget, error) {
	if options.ExecutionID != "" {
		return []RehearsalSchedulerTarget{{
			Type:         "execution",
			CandidateID:  options.CandidateID,
			DeploymentID: options.DeploymentID,
			ExecutionID:  options.ExecutionID,
			Environment:  options.Environment,
		}}, nil
	}
	if options.DeploymentID != "" {
		return []RehearsalSchedulerTarget{{
			Type:         "deployment",
			CandidateID:  options.CandidateID,
			DeploymentID: options.DeploymentID,
			Environment:  options.Environment,
		}}, nil
	}
	if options.CandidateID != "" {
		return []RehearsalSchedulerTarget{{
			Type:        "candidate",
			CandidateID: options.CandidateID,
			Environment: options.Environment,
		}}, nil
	}
	executions, err := ListExecutions(rootDir, options.MaxTargets*4)
	if err != nil {
		return nil, err
	}
	targets := []RehearsalSchedulerTarget{}
	for _, execution := range executions {
		if options.Environment != "" && execution.Environment != options.Environment {
			continue
		}
		targets = append(targets, RehearsalSchedulerTarget{
			Type:         "execution",
			DeploymentID: execution.DeploymentID,
			ExecutionID:  execution.ID,
			Environment:  execution.Environment,
		})
		if len(targets) >= options.MaxTargets {
			break
		}
	}
	return targets, nil
}

func executeRehearsalSchedulerTarget(ctx context.Context, rootDir string, options RehearsalSchedulerOptions, target RehearsalSchedulerTarget) (RehearsalSchedulerTarget, error) {
	if existing, found, err := latestRehearsalForSchedulerTarget(rootDir, target); err != nil {
		return RehearsalSchedulerTarget{}, err
	} else if found {
		target.RehearsalID = existing.ID
		if options.SkipAdmission {
			target.Status = "skipped"
			target.Decision = "REHEARSAL_SCHEDULER_TARGET_SKIPPED"
			target.Reason = "rehearsal_already_exists"
			return target, nil
		}
		if admission, admissionFound, err := latestAdmissionForRehearsal(rootDir, existing.ID); err != nil {
			return RehearsalSchedulerTarget{}, err
		} else if admissionFound {
			target.AdmissionID = admission.ID
			target.Status = "skipped"
			target.Decision = "REHEARSAL_SCHEDULER_TARGET_SKIPPED"
			target.Reason = "admission_already_exists"
			return target, nil
		}
		admission, err := BuildReleaseAdmission(ctx, rootDir, ReleaseAdmissionOptions{RehearsalID: existing.ID, MonitorLimit: options.MonitorLimit})
		if err != nil {
			return RehearsalSchedulerTarget{}, err
		}
		target.AdmissionID = admission.ID
		return finalizeSchedulerTargetFromAdmission(target, admission), nil
	}

	rehearsal, err := BuildRehearsal(ctx, rootDir, RehearsalOptions{
		CandidateID:  target.CandidateID,
		DeploymentID: target.DeploymentID,
		ExecutionID:  target.ExecutionID,
		Environment:  target.Environment,
		MonitorLimit: options.MonitorLimit,
	})
	if err != nil {
		return RehearsalSchedulerTarget{}, err
	}
	target.CandidateID = firstNonEmpty(target.CandidateID, rehearsal.CandidateID)
	target.DeploymentID = firstNonEmpty(target.DeploymentID, rehearsal.DeploymentID)
	target.ExecutionID = firstNonEmpty(target.ExecutionID, rehearsal.ExecutionID)
	target.Environment = firstNonEmpty(target.Environment, rehearsal.Environment)
	target.RehearsalID = rehearsal.ID
	if rehearsal.Status == "blocked" {
		target.Status = "blocked"
		target.Decision = "REHEARSAL_SCHEDULER_REHEARSAL_BLOCKED"
		target.Reason = firstReason(rehearsal.Reasons)
		return target, nil
	}
	if options.SkipAdmission {
		target.Status = "created"
		target.Decision = "REHEARSAL_SCHEDULER_REHEARSAL_CREATED"
		target.Reason = "admission_skipped_by_option"
		return target, nil
	}
	admission, err := BuildReleaseAdmission(ctx, rootDir, ReleaseAdmissionOptions{RehearsalID: rehearsal.ID, MonitorLimit: options.MonitorLimit})
	if err != nil {
		return RehearsalSchedulerTarget{}, err
	}
	target.AdmissionID = admission.ID
	return finalizeSchedulerTargetFromAdmission(target, admission), nil
}

func finalizeSchedulerTargetFromAdmission(target RehearsalSchedulerTarget, admission ReleaseAdmission) RehearsalSchedulerTarget {
	target.Reason = firstReason(admission.Reasons)
	switch admission.Status {
	case "allowed":
		target.Status = "created"
		target.Decision = "REHEARSAL_SCHEDULER_ADMISSION_ALLOWED"
	case "manual_required":
		target.Status = "manual_required"
		target.Decision = "REHEARSAL_SCHEDULER_ADMISSION_MANUAL_REQUIRED"
	case "blocked":
		target.Status = "blocked"
		target.Decision = "REHEARSAL_SCHEDULER_ADMISSION_BLOCKED"
	default:
		target.Status = "manual_required"
		target.Decision = "REHEARSAL_SCHEDULER_ADMISSION_REVIEW_REQUIRED"
	}
	return target
}

func latestRehearsalForSchedulerTarget(rootDir string, target RehearsalSchedulerTarget) (DeploymentRehearsal, bool, error) {
	rehearsals, err := ListRehearsals(rootDir, 100)
	if err != nil {
		return DeploymentRehearsal{}, false, err
	}
	for _, rehearsal := range rehearsals {
		if target.ExecutionID != "" && rehearsal.ExecutionID == target.ExecutionID {
			return rehearsal, true, nil
		}
		if target.DeploymentID != "" && rehearsal.DeploymentID == target.DeploymentID {
			if target.CandidateID != "" && rehearsal.CandidateID != target.CandidateID {
				continue
			}
			if target.Environment != "" && rehearsal.Environment != target.Environment {
				continue
			}
			return rehearsal, true, nil
		}
		if target.CandidateID != "" && rehearsal.CandidateID == target.CandidateID {
			if target.Environment != "" && rehearsal.Environment != target.Environment {
				continue
			}
			return rehearsal, true, nil
		}
	}
	return DeploymentRehearsal{}, false, nil
}

func latestAdmissionForRehearsal(rootDir string, rehearsalID string) (ReleaseAdmission, bool, error) {
	admissions, err := ListReleaseAdmissions(rootDir, 100)
	if err != nil {
		return ReleaseAdmission{}, false, err
	}
	for _, admission := range admissions {
		if admission.RehearsalID == rehearsalID {
			return admission, true, nil
		}
	}
	return ReleaseAdmission{}, false, nil
}

func finalizeRehearsalSchedulerRun(run RehearsalSchedulerRun) RehearsalSchedulerRun {
	switch {
	case run.BlockedCount > 0:
		run.Status = "attention_required"
		run.Decision = "REHEARSAL_SCHEDULER_ATTENTION_REQUIRED"
		run.Reasons = appendUnique(run.Reasons, "scheduler_targets_blocked")
	case run.ManualCount > 0:
		run.Status = "attention_required"
		run.Decision = "REHEARSAL_SCHEDULER_ATTENTION_REQUIRED"
		run.Reasons = appendUnique(run.Reasons, "scheduler_targets_manual_required")
	case run.CreatedCount > 0:
		run.Status = "completed"
		run.Decision = "REHEARSAL_SCHEDULER_COMPLETED"
		run.Reasons = appendUnique(run.Reasons, "scheduler_targets_created")
	case run.SkippedCount > 0:
		run.Status = "completed"
		run.Decision = "REHEARSAL_SCHEDULER_NOOP"
		run.Reasons = appendUnique(run.Reasons, "scheduler_targets_already_linked")
	default:
		run.Status = "blocked"
		run.Decision = "REHEARSAL_SCHEDULER_NO_TARGETS"
		run.Reasons = appendUnique(run.Reasons, "scheduler_targets_missing")
	}
	return run
}

func finishRehearsalSchedulerRun(rootDir string, run RehearsalSchedulerRun) (RehearsalSchedulerRun, error) {
	run.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := fsutil.EnsureDir(rehearsalSchedulerRunDir(rootDir)); err != nil {
		return RehearsalSchedulerRun{}, err
	}
	if err := fsutil.WriteJSON(rehearsalSchedulerRunPath(rootDir, run.ID), run); err != nil {
		return RehearsalSchedulerRun{}, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "rehearsal-scheduler-runs.jsonl"), run); err != nil {
		return RehearsalSchedulerRun{}, err
	}
	record, err := evidence.Add(rootDir, evidence.AddOptions{
		ParentType:  "rehearsal_scheduler_run",
		ParentID:    run.ID,
		SubjectType: "deployment",
		SubjectID:   rehearsalSchedulerSubjectID(run),
		Operation:   "deployment.rehearsal.scheduler",
		Status:      run.Status,
		Decision:    run.Decision,
		Reasons:     run.Reasons,
		Source:      "deployment",
		Artifacts: []evidence.ArtifactRef{{
			Kind: "rehearsal_scheduler_run",
			ID:   run.ID,
			Path: filepath.ToSlash(filepath.Join(".moyuan", "lifecycle", "deployments", "rehearsal-scheduler-runs", run.ID+".json")),
		}},
	})
	if err != nil {
		return RehearsalSchedulerRun{}, err
	}
	run.EvidenceIDs = appendUnique(run.EvidenceIDs, record.ID)
	if err := fsutil.WriteJSON(rehearsalSchedulerRunPath(rootDir, run.ID), run); err != nil {
		return RehearsalSchedulerRun{}, err
	}
	_ = logging.Log(rootDir, "release", "deployment.rehearsal.scheduler.completed", map[string]any{
		"scheduler_run_id": run.ID,
		"trigger":          run.Trigger,
		"decision":         run.Decision,
		"status":           run.Status,
		"created_count":    run.CreatedCount,
		"skipped_count":    run.SkippedCount,
		"blocked_count":    run.BlockedCount,
		"manual_count":     run.ManualCount,
	})
	return run, nil
}

func rehearsalSchedulerSeed(options RehearsalSchedulerOptions) string {
	for _, value := range []string{options.ExecutionID, options.DeploymentID, options.CandidateID, options.Environment, options.Trigger} {
		if value != "" {
			return value
		}
	}
	return "latest"
}

func rehearsalSchedulerSubjectID(run RehearsalSchedulerRun) string {
	for _, value := range []string{run.ExecutionID, run.DeploymentID, run.CandidateID, run.Environment, run.Trigger} {
		if value != "" {
			return value
		}
	}
	return "unknown"
}

func rehearsalSchedulerRunDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "rehearsal-scheduler-runs")
}

func rehearsalSchedulerRunPath(rootDir string, id string) string {
	return filepath.Join(rehearsalSchedulerRunDir(rootDir), id+".json")
}
