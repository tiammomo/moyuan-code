package operations

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/deployment"
	"moyuan-code/internal/evidence"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/release"
	"moyuan-code/internal/serverresources"
	"moyuan-code/internal/workspace"
)

type TimelineOptions struct {
	Type        string `json:"type,omitempty"`
	Status      string `json:"status,omitempty"`
	Decision    string `json:"decision,omitempty"`
	Environment string `json:"environment,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

type TimelineItem struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Operation    string         `json:"operation"`
	Status       string         `json:"status"`
	Decision     string         `json:"decision"`
	Reasons      []string       `json:"reasons,omitempty"`
	PrimaryRef   string         `json:"primary_ref,omitempty"`
	SecondaryRef string         `json:"secondary_ref,omitempty"`
	Environment  string         `json:"environment,omitempty"`
	EvidenceRefs []string       `json:"evidence_refs,omitempty"`
	Timestamp    string         `json:"timestamp,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type Detail struct {
	ID            string                 `json:"id"`
	OperationType string                 `json:"operation_type"`
	Operation     string                 `json:"operation"`
	Status        string                 `json:"status"`
	Decision      string                 `json:"decision"`
	Reasons       []string               `json:"reasons"`
	PrimaryRef    string                 `json:"primary_ref,omitempty"`
	SecondaryRef  string                 `json:"secondary_ref,omitempty"`
	StartedAt     string                 `json:"started_at,omitempty"`
	FinishedAt    string                 `json:"finished_at,omitempty"`
	CreatedAt     string                 `json:"created_at,omitempty"`
	Summary       Summary                `json:"summary"`
	Evidence      []evidence.Record      `json:"evidence"`
	Artifacts     []evidence.ArtifactRef `json:"artifacts,omitempty"`
}

type Summary struct {
	Mode              string `json:"mode,omitempty"`
	ReleaseID         string `json:"release_id,omitempty"`
	Version           string `json:"version,omitempty"`
	Provider          string `json:"provider,omitempty"`
	DeploymentID      string `json:"deployment_id,omitempty"`
	Environment       string `json:"environment,omitempty"`
	ActionCount       int    `json:"action_count,omitempty"`
	StepCount         int    `json:"step_count,omitempty"`
	ResourceCount     int    `json:"resource_count,omitempty"`
	EvidenceCount     int    `json:"evidence_count,omitempty"`
	ArtifactCount     int    `json:"artifact_count,omitempty"`
	RemoteStatus      string `json:"remote_status,omitempty"`
	SmokeDecision     string `json:"smoke_decision,omitempty"`
	MonitorDecision   string `json:"monitor_decision,omitempty"`
	RollbackDecision  string `json:"rollback_decision,omitempty"`
	ApprovalID        string `json:"approval_id,omitempty"`
	ApprovalConsumed  bool   `json:"approval_consumed,omitempty"`
	WriteEnabled      bool   `json:"write_enabled,omitempty"`
	RemoteExecEnabled bool   `json:"remote_exec_enabled,omitempty"`
}

type deploymentRiskHandoffTimeline struct {
	ID             string   `json:"id"`
	SourceType     string   `json:"source_type"`
	SourceID       string   `json:"source_id"`
	Status         string   `json:"status"`
	Decision       string   `json:"decision"`
	FailureClass   string   `json:"failure_class"`
	SignalID       string   `json:"signal_id,omitempty"`
	BugCandidateID string   `json:"bug_candidate_id,omitempty"`
	RepairPlanID   string   `json:"repair_plan_id,omitempty"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
	Reasons        []string `json:"reasons"`
	ReviewRequired bool     `json:"review_required"`
	ReviewID       string   `json:"review_id,omitempty"`
	ReviewedAt     string   `json:"reviewed_at,omitempty"`
	ReviewDecision string   `json:"review_decision,omitempty"`
	CreatedAt      string   `json:"created_at"`
}

type deploymentRiskReviewTimeline struct {
	ID             string   `json:"id"`
	HandoffID      string   `json:"handoff_id"`
	SourceType     string   `json:"source_type"`
	SourceID       string   `json:"source_id"`
	Decision       string   `json:"decision"`
	Status         string   `json:"status"`
	ReviewerID     string   `json:"reviewer_id,omitempty"`
	Reason         string   `json:"reason,omitempty"`
	NextStep       string   `json:"next_step,omitempty"`
	FailureClass   string   `json:"failure_class,omitempty"`
	BugCandidateID string   `json:"bug_candidate_id,omitempty"`
	RepairPlanID   string   `json:"repair_plan_id,omitempty"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
	CreatedAt      string   `json:"created_at"`
}

func Timeline(rootDir string, options TimelineOptions) ([]TimelineItem, error) {
	options = normalizeTimelineOptions(options)
	sourceLimit := options.Limit * 4
	if sourceLimit < 50 {
		sourceLimit = 50
	}
	if sourceLimit > 200 {
		sourceLimit = 200
	}
	items := []TimelineItem{}
	add := func(item TimelineItem) {
		if timelineMatches(item, options) {
			items = append(items, item)
		}
	}

	providerExecutions, err := release.ListProviderExecutions(rootDir, sourceLimit)
	if err != nil {
		return nil, err
	}
	for _, execution := range providerExecutions {
		refs, err := evidenceRefs(rootDir, "release_provider_execution", execution.ID, nil)
		if err != nil {
			return nil, err
		}
		add(TimelineItem{
			ID:           execution.ID,
			Type:         "release_provider_execution",
			Operation:    "release.provider." + execution.Mode,
			Status:       execution.Status,
			Decision:     execution.Decision,
			Reasons:      append([]string{}, execution.Reasons...),
			PrimaryRef:   execution.ReleaseID,
			SecondaryRef: firstNonEmpty(execution.Provider, execution.Version),
			EvidenceRefs: refs,
			Timestamp:    firstNonEmpty(execution.FinishedAt, execution.StartedAt),
			Metadata: map[string]any{
				"candidate_id":        execution.CandidateID,
				"mode":                execution.Mode,
				"provider":            execution.Provider,
				"version":             execution.Version,
				"write_enabled":       execution.WriteEnabled,
				"approval_consumed":   execution.ApprovalConsumed,
				"remote_action_count": len(execution.RemotePlan.Actions),
			},
		})
	}

	executions, err := deployment.ListExecutions(rootDir, sourceLimit)
	if err != nil {
		return nil, err
	}
	for _, execution := range executions {
		refs, err := evidenceRefs(rootDir, "deployment_execution", execution.ID, nil)
		if err != nil {
			return nil, err
		}
		add(TimelineItem{
			ID:           execution.ID,
			Type:         "deployment_execution",
			Operation:    "deployment.execute." + execution.Mode,
			Status:       execution.Status,
			Decision:     execution.Decision,
			Reasons:      append([]string{}, execution.Reasons...),
			PrimaryRef:   execution.DeploymentID,
			SecondaryRef: execution.ReleaseID,
			Environment:  execution.Environment,
			EvidenceRefs: refs,
			Timestamp:    firstNonEmpty(execution.FinishedAt, execution.StartedAt),
			Metadata: map[string]any{
				"mode":                execution.Mode,
				"resource_count":      len(execution.Resources),
				"step_count":          len(execution.Steps),
				"smoke_decision":      execution.SmokeReport.Decision,
				"monitor_decision":    execution.MonitorReport.Decision,
				"rollback_decision":   execution.RollbackSuggestion.Decision,
				"remote_exec_enabled": execution.RemoteExecEnabled,
				"approval_consumed":   execution.ApprovalConsumed,
				"rollback_required":   execution.RollbackSuggestion.Required,
			},
		})
	}

	rollbacks, err := deployment.ListRollbackExecutions(rootDir, sourceLimit)
	if err != nil {
		return nil, err
	}
	for _, rollback := range rollbacks {
		refs, err := evidenceRefs(rootDir, "deployment_rollback_execution", rollback.ID, nil)
		if err != nil {
			return nil, err
		}
		add(TimelineItem{
			ID:           rollback.ID,
			Type:         "rollback_execution",
			Operation:    "deployment.rollback.execute." + rollback.Mode,
			Status:       rollback.Status,
			Decision:     rollback.Decision,
			Reasons:      append([]string{}, rollback.Reasons...),
			PrimaryRef:   rollback.ExecutionID,
			SecondaryRef: rollback.DeploymentID,
			Environment:  rollback.Environment,
			EvidenceRefs: refs,
			Timestamp:    firstNonEmpty(rollback.FinishedAt, rollback.StartedAt),
			Metadata: map[string]any{
				"mode":              rollback.Mode,
				"release_id":        rollback.ReleaseID,
				"execution_enabled": rollback.ExecutionEnabled,
				"approval_consumed": rollback.ApprovalConsumed,
			},
		})
	}

	monitorSummaries, err := deployment.ListMonitorSummaries(rootDir, sourceLimit)
	if err != nil {
		return nil, err
	}
	for _, summary := range monitorSummaries {
		add(TimelineItem{
			ID:           summary.ID,
			Type:         "deployment_monitor_summary",
			Operation:    "deployment.monitor.summary",
			Status:       summary.Status,
			Decision:     summary.Decision,
			Reasons:      append([]string{}, summary.Reasons...),
			PrimaryRef:   environmentOrAll(summary.Environment),
			Environment:  summary.Environment,
			EvidenceRefs: append([]string{}, summary.EvidenceIDs...),
			Timestamp:    summary.CreatedAt,
			Metadata: map[string]any{
				"history_count":  summary.HistoryCount,
				"failed_count":   summary.FailedCount,
				"blocked_count":  summary.BlockedCount,
				"manual_count":   summary.ManualCount,
				"rollback_count": summary.RollbackCount,
			},
		})
	}

	rehearsals, err := deployment.ListRehearsals(rootDir, sourceLimit)
	if err != nil {
		return nil, err
	}
	for _, rehearsal := range rehearsals {
		refs, err := evidenceRefs(rootDir, "deployment_rehearsal", rehearsal.ID, rehearsal.EvidenceIDs)
		if err != nil {
			return nil, err
		}
		add(TimelineItem{
			ID:           rehearsal.ID,
			Type:         "deployment_rehearsal",
			Operation:    "deployment.rehearsal",
			Status:       rehearsal.Status,
			Decision:     rehearsal.Decision,
			Reasons:      append([]string{}, rehearsal.Reasons...),
			PrimaryRef:   firstNonEmpty(rehearsal.ExecutionID, rehearsal.DeploymentID, rehearsal.CandidateID),
			SecondaryRef: rehearsal.ReleaseID,
			Environment:  rehearsal.Environment,
			EvidenceRefs: refs,
			Timestamp:    firstNonEmpty(rehearsal.FinishedAt, rehearsal.CreatedAt),
			Metadata: map[string]any{
				"monitor_summary_id":    rehearsal.MonitorSummaryID,
				"monitor_decision":      rehearsal.MonitorDecision,
				"rollback_execution_id": rehearsal.RollbackExecutionID,
				"rollback_decision":     rehearsal.RollbackDecision,
				"timeline_count":        len(rehearsal.Timeline),
			},
		})
	}

	admissions, err := deployment.ListReleaseAdmissions(rootDir, sourceLimit)
	if err != nil {
		return nil, err
	}
	for _, admission := range admissions {
		refs, err := evidenceRefs(rootDir, "release_admission", admission.ID, admission.EvidenceIDs)
		if err != nil {
			return nil, err
		}
		add(TimelineItem{
			ID:           admission.ID,
			Type:         "release_admission",
			Operation:    "release.admission",
			Status:       admission.Status,
			Decision:     admission.Decision,
			Reasons:      append([]string{}, admission.Reasons...),
			PrimaryRef:   firstNonEmpty(admission.RehearsalID, admission.ExecutionID, admission.DeploymentID, admission.CandidateID),
			SecondaryRef: admission.PolicyID,
			Environment:  admission.Environment,
			EvidenceRefs: refs,
			Timestamp:    admission.CreatedAt,
			Metadata: map[string]any{
				"policy_id":          admission.PolicyID,
				"policy_version":     admission.PolicyVersion,
				"matched_rule_count": len(admission.MatchedRules),
				"signal_count":       len(admission.Signals),
			},
		})
	}

	schedulerRuns, err := deployment.ListRehearsalSchedulerRuns(rootDir, sourceLimit)
	if err != nil {
		return nil, err
	}
	for _, run := range schedulerRuns {
		refs, err := evidenceRefs(rootDir, "rehearsal_scheduler_run", run.ID, run.EvidenceIDs)
		if err != nil {
			return nil, err
		}
		add(TimelineItem{
			ID:           run.ID,
			Type:         "rehearsal_scheduler_run",
			Operation:    "deployment.rehearsal.scheduler",
			Status:       run.Status,
			Decision:     run.Decision,
			Reasons:      append([]string{}, run.Reasons...),
			PrimaryRef:   firstNonEmpty(run.ExecutionID, run.DeploymentID, run.CandidateID),
			SecondaryRef: run.Trigger,
			Environment:  run.Environment,
			EvidenceRefs: refs,
			Timestamp:    firstNonEmpty(run.FinishedAt, run.StartedAt),
			Metadata: map[string]any{
				"target_count":  len(run.Targets),
				"created_count": run.CreatedCount,
				"skipped_count": run.SkippedCount,
				"blocked_count": run.BlockedCount,
				"manual_count":  run.ManualCount,
			},
		})
	}

	handoffs, err := listDeploymentRiskHandoffs(rootDir, sourceLimit)
	if err != nil {
		return nil, err
	}
	for _, handoff := range handoffs {
		add(TimelineItem{
			ID:           handoff.ID,
			Type:         "deployment_risk_handoff",
			Operation:    "repair.deployment_risk.handoff",
			Status:       handoff.Status,
			Decision:     handoff.Decision,
			Reasons:      append([]string{}, handoff.Reasons...),
			PrimaryRef:   handoff.SourceID,
			SecondaryRef: handoff.FailureClass,
			EvidenceRefs: append([]string{}, handoff.EvidenceRefs...),
			Timestamp:    firstNonEmpty(handoff.ReviewedAt, handoff.CreatedAt),
			Metadata: map[string]any{
				"source_type":      handoff.SourceType,
				"review_required":  handoff.ReviewRequired,
				"review_id":        handoff.ReviewID,
				"review_decision":  handoff.ReviewDecision,
				"repair_plan_id":   handoff.RepairPlanID,
				"bug_candidate_id": handoff.BugCandidateID,
			},
		})
	}

	riskReviews, err := listDeploymentRiskReviews(rootDir, sourceLimit)
	if err != nil {
		return nil, err
	}
	for _, review := range riskReviews {
		add(TimelineItem{
			ID:           review.ID,
			Type:         "deployment_risk_review",
			Operation:    "repair.deployment_risk.review",
			Status:       review.Status,
			Decision:     review.Decision,
			Reasons:      compactStrings([]string{review.Reason}),
			PrimaryRef:   review.HandoffID,
			SecondaryRef: review.NextStep,
			EvidenceRefs: append([]string{}, review.EvidenceRefs...),
			Timestamp:    review.CreatedAt,
			Metadata: map[string]any{
				"source_type":      review.SourceType,
				"source_id":        review.SourceID,
				"failure_class":    review.FailureClass,
				"reviewer_id":      review.ReviewerID,
				"repair_plan_id":   review.RepairPlanID,
				"bug_candidate_id": review.BugCandidateID,
			},
		})
	}

	healthScans, err := serverresources.ListHealthScans(rootDir, sourceLimit)
	if err != nil {
		return nil, err
	}
	for _, scan := range healthScans {
		add(TimelineItem{
			ID:          scan.ID,
			Type:        "resource_health_scan",
			Operation:   "server_resource.health_scan",
			Status:      scan.Status,
			Decision:    scan.Decision,
			Reasons:     append([]string{}, scan.Reasons...),
			PrimaryRef:  environmentOrAll(scan.Environment),
			Environment: scan.Environment,
			Timestamp:   scan.CreatedAt,
			Metadata: map[string]any{
				"result_count": len(scan.Results),
			},
		})
	}

	maintenanceRecords, err := serverresources.ListMaintenance(rootDir, sourceLimit)
	if err != nil {
		return nil, err
	}
	for _, record := range maintenanceRecords {
		add(TimelineItem{
			ID:          record.ID,
			Type:        "resource_maintenance",
			Operation:   "server_resource.maintenance." + record.Type,
			Status:      record.Status,
			Decision:    record.Decision,
			Reasons:     compactStrings([]string{record.Reason}),
			PrimaryRef:  record.ResourceID,
			Environment: record.Environment,
			Timestamp:   record.CreatedAt,
			Metadata: map[string]any{
				"expires_at":       record.ExpiresAt,
				"new_expires_at":   record.NewExpiresAt,
				"expiration_state": record.ExpirationState,
				"health_status":    record.HealthStatus,
			},
		})
	}

	lifecycleAlerts, err := serverresources.ListLifecycleAlerts(rootDir, sourceLimit)
	if err != nil {
		return nil, err
	}
	for _, alert := range lifecycleAlerts {
		add(TimelineItem{
			ID:          alert.ID,
			Type:        "resource_lifecycle_alert",
			Operation:   "server_resource.lifecycle_alert." + alert.Type,
			Status:      alert.Status,
			Decision:    alert.Decision,
			Reasons:     compactStrings([]string{alert.Reason}),
			PrimaryRef:  alert.ResourceID,
			Environment: alert.Environment,
			Timestamp:   alert.CreatedAt,
			Metadata: map[string]any{
				"severity":           alert.Severity,
				"expires_at":         alert.ExpiresAt,
				"expiration_state":   alert.ExpirationState,
				"health_status":      alert.HealthStatus,
				"maintenance_window": alert.MaintenanceWindow,
			},
		})
	}

	resources, err := serverresources.List(rootDir)
	if err != nil {
		return nil, err
	}
	for _, resource := range resources {
		add(TimelineItem{
			ID:          resource.ID,
			Type:        "server_resource",
			Operation:   "server_resource.inventory",
			Status:      resource.Status,
			Decision:    "SERVER_RESOURCE_RECORDED",
			PrimaryRef:  resource.Host,
			Environment: resource.Environment,
			Timestamp:   firstNonEmpty(resource.UpdatedAt, resource.CreatedAt),
			Metadata: map[string]any{
				"provider":         resource.Provider,
				"owner":            resource.Owner,
				"expires_at":       resource.ExpiresAt,
				"expiration_state": resource.ExpirationState,
				"health_status":    resource.Healthcheck.LastStatus,
			},
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		left := parseTimelineTime(items[i].Timestamp)
		right := parseTimelineTime(items[j].Timestamp)
		if !left.Equal(right) {
			return left.After(right)
		}
		return items[i].Type+"|"+items[i].ID > items[j].Type+"|"+items[j].ID
	})
	if len(items) > options.Limit {
		return items[:options.Limit], nil
	}
	return items, nil
}

func Load(rootDir string, operationType string, operationID string) (Detail, bool, error) {
	switch normalizeType(operationType) {
	case "release_provider", "release_provider_execution":
		return loadReleaseProvider(rootDir, operationID)
	case "deployment", "deployment_execution":
		return loadDeployment(rootDir, operationID)
	case "evidence":
		return loadEvidence(rootDir, operationID)
	default:
		return Detail{}, false, nil
	}
}

func loadReleaseProvider(rootDir string, id string) (Detail, bool, error) {
	execution, found, err := release.LoadProviderExecution(rootDir, id)
	if err != nil || !found {
		return Detail{}, found, err
	}
	records, err := evidence.List(rootDir, evidence.ListOptions{ParentType: "release_provider_execution", ParentID: execution.ID, Limit: 100})
	if err != nil {
		return Detail{}, false, err
	}
	artifacts := collectArtifacts(records)
	detail := Detail{
		ID:            execution.ID,
		OperationType: "release_provider",
		Operation:     "release.provider." + execution.Mode,
		Status:        execution.Status,
		Decision:      execution.Decision,
		Reasons:       append([]string{}, execution.Reasons...),
		PrimaryRef:    execution.ReleaseID,
		SecondaryRef:  firstNonEmpty(execution.Provider, execution.Version),
		StartedAt:     execution.StartedAt,
		FinishedAt:    execution.FinishedAt,
		Summary: Summary{
			Mode:             execution.Mode,
			ReleaseID:        execution.ReleaseID,
			Version:          execution.Version,
			Provider:         execution.Provider,
			ActionCount:      len(execution.RemotePlan.Actions),
			EvidenceCount:    len(records),
			ArtifactCount:    len(artifacts),
			RemoteStatus:     execution.RemotePlan.Status,
			ApprovalID:       execution.ApprovalID,
			ApprovalConsumed: execution.ApprovalConsumed,
			WriteEnabled:     execution.WriteEnabled,
		},
		Evidence:  records,
		Artifacts: artifacts,
	}
	return detail, true, nil
}

func loadDeployment(rootDir string, id string) (Detail, bool, error) {
	execution, found, err := deployment.LoadExecution(rootDir, id)
	if err != nil || !found {
		return Detail{}, found, err
	}
	records, err := evidence.List(rootDir, evidence.ListOptions{ParentType: "deployment_execution", ParentID: execution.ID, Limit: 100})
	if err != nil {
		return Detail{}, false, err
	}
	artifacts := collectArtifacts(records)
	detail := Detail{
		ID:            execution.ID,
		OperationType: "deployment",
		Operation:     "deployment.execute." + execution.Mode,
		Status:        execution.Status,
		Decision:      execution.Decision,
		Reasons:       append([]string{}, execution.Reasons...),
		PrimaryRef:    execution.DeploymentID,
		SecondaryRef:  execution.Environment,
		StartedAt:     execution.StartedAt,
		FinishedAt:    execution.FinishedAt,
		Summary: Summary{
			Mode:              execution.Mode,
			ReleaseID:         execution.ReleaseID,
			DeploymentID:      execution.DeploymentID,
			Environment:       execution.Environment,
			StepCount:         len(execution.Steps),
			ResourceCount:     len(execution.Resources),
			EvidenceCount:     len(records),
			ArtifactCount:     len(artifacts),
			SmokeDecision:     execution.SmokeReport.Decision,
			MonitorDecision:   execution.MonitorReport.Decision,
			RollbackDecision:  execution.RollbackSuggestion.Decision,
			ApprovalID:        execution.ApprovalID,
			RemoteExecEnabled: execution.RemoteExecEnabled,
		},
		Evidence:  records,
		Artifacts: artifacts,
	}
	return detail, true, nil
}

func loadEvidence(rootDir string, id string) (Detail, bool, error) {
	record, found, err := evidence.Load(rootDir, id)
	if err != nil || !found {
		return Detail{}, found, err
	}
	detail := Detail{
		ID:            record.ID,
		OperationType: "evidence",
		Operation:     record.Operation,
		Status:        record.Status,
		Decision:      record.Decision,
		Reasons:       append([]string{}, record.Reasons...),
		PrimaryRef:    record.ParentID,
		SecondaryRef:  firstNonEmpty(record.SubjectID, record.SubjectType),
		CreatedAt:     record.CreatedAt,
		Summary: Summary{
			EvidenceCount: 1,
			ArtifactCount: len(record.Artifacts),
		},
		Evidence:  []evidence.Record{record},
		Artifacts: append([]evidence.ArtifactRef{}, record.Artifacts...),
	}
	return detail, true, nil
}

func collectArtifacts(records []evidence.Record) []evidence.ArtifactRef {
	seen := map[string]bool{}
	artifacts := []evidence.ArtifactRef{}
	for _, record := range records {
		for _, artifact := range record.Artifacts {
			key := artifact.Kind + "|" + artifact.ID + "|" + artifact.Path
			if seen[key] {
				continue
			}
			seen[key] = true
			artifacts = append(artifacts, artifact)
		}
	}
	return artifacts
}

func normalizeType(value string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "-", "_")
}

func normalizeTimelineOptions(options TimelineOptions) TimelineOptions {
	options.Type = normalizeType(options.Type)
	options.Status = normalizeType(options.Status)
	options.Decision = normalizeType(options.Decision)
	options.Environment = normalizeType(options.Environment)
	if options.Limit <= 0 {
		options.Limit = 20
	}
	if options.Limit > 100 {
		options.Limit = 100
	}
	return options
}

func timelineMatches(item TimelineItem, options TimelineOptions) bool {
	if options.Type != "" && normalizeType(item.Type) != options.Type {
		return false
	}
	if options.Status != "" && normalizeType(item.Status) != options.Status {
		return false
	}
	if options.Decision != "" && normalizeType(item.Decision) != options.Decision {
		return false
	}
	if options.Environment != "" && normalizeType(item.Environment) != options.Environment {
		return false
	}
	return true
}

func evidenceRefs(rootDir string, parentType string, parentID string, fallback []string) ([]string, error) {
	refs := append([]string{}, fallback...)
	records, err := evidence.List(rootDir, evidence.ListOptions{ParentType: parentType, ParentID: parentID, Limit: 100})
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		if record.ID != "" {
			refs = appendUnique(refs, record.ID)
		}
	}
	return compactStrings(refs), nil
}

func listDeploymentRiskHandoffs(rootDir string, limit int) ([]deploymentRiskHandoffTimeline, error) {
	if err := fsutil.EnsureDir(deploymentRiskHandoffTimelineDir(rootDir)); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(deploymentRiskHandoffTimelineDir(rootDir))
	if err != nil {
		return nil, err
	}
	handoffs := []deploymentRiskHandoffTimeline{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var handoff deploymentRiskHandoffTimeline
		found, err := fsutil.ReadJSON(filepath.Join(deploymentRiskHandoffTimelineDir(rootDir), entry.Name()), &handoff)
		if err != nil {
			return nil, err
		}
		if found && handoff.ID != "" {
			handoffs = append(handoffs, handoff)
		}
	}
	sort.SliceStable(handoffs, func(i, j int) bool {
		return handoffs[i].CreatedAt > handoffs[j].CreatedAt
	})
	if limit <= 0 {
		limit = 20
	}
	if len(handoffs) > limit {
		return handoffs[:limit], nil
	}
	return handoffs, nil
}

func listDeploymentRiskReviews(rootDir string, limit int) ([]deploymentRiskReviewTimeline, error) {
	if err := fsutil.EnsureDir(deploymentRiskReviewTimelineDir(rootDir)); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(deploymentRiskReviewTimelineDir(rootDir))
	if err != nil {
		return nil, err
	}
	reviews := []deploymentRiskReviewTimeline{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var review deploymentRiskReviewTimeline
		found, err := fsutil.ReadJSON(filepath.Join(deploymentRiskReviewTimelineDir(rootDir), entry.Name()), &review)
		if err != nil {
			return nil, err
		}
		if found && review.ID != "" {
			reviews = append(reviews, review)
		}
	}
	sort.SliceStable(reviews, func(i, j int) bool {
		return reviews[i].CreatedAt > reviews[j].CreatedAt
	})
	if limit <= 0 {
		limit = 20
	}
	if len(reviews) > limit {
		return reviews[:limit], nil
	}
	return reviews, nil
}

func deploymentRiskHandoffTimelineDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).RepairDir, "deployment-risk-handoffs")
}

func deploymentRiskReviewTimelineDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).RepairDir, "deployment-risk-reviews")
}

func appendUnique(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func compactStrings(values []string) []string {
	out := []string{}
	for _, value := range values {
		out = appendUnique(out, value)
	}
	return out
}

func environmentOrAll(environment string) string {
	if strings.TrimSpace(environment) == "" {
		return "all"
	}
	return environment
}

func parseTimelineTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed
	}
	return time.Time{}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
