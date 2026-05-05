package operations

import (
	"strings"

	"moyuan-code/internal/deployment"
	"moyuan-code/internal/evidence"
	"moyuan-code/internal/release"
)

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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
