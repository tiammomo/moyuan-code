package release

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"sort"
	"time"

	"moyuan-code/internal/fsutil"
	gitadapter "moyuan-code/internal/git"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type CandidateOptions struct {
	ReleaseBatchID    string   `json:"release_batch_id"`
	DeploymentTargets []string `json:"deployment_targets,omitempty"`
	RequestedBy       string   `json:"requested_by,omitempty"`
}

type Candidate struct {
	ID                   string   `json:"id"`
	ReleaseBatchID       string   `json:"release_batch_id"`
	IntegrationApplyID   string   `json:"integration_apply_id,omitempty"`
	IntegrationPreviewID string   `json:"integration_preview_id,omitempty"`
	MergeQueueID         string   `json:"merge_queue_id,omitempty"`
	BatchID              string   `json:"batch_id,omitempty"`
	EpicID               string   `json:"epic_id,omitempty"`
	Status               string   `json:"status"`
	Decision             string   `json:"decision"`
	Version              string   `json:"version"`
	ReleaseBranch        string   `json:"release_branch"`
	SourceBranch         string   `json:"source_branch,omitempty"`
	RemoteName           string   `json:"remote_name,omitempty"`
	RemoteURL            string   `json:"remote_url,omitempty"`
	Provider             string   `json:"provider,omitempty"`
	ReadyItemCount       int      `json:"ready_item_count"`
	MinItems             int      `json:"min_items"`
	DeploymentTargets    []string `json:"deployment_targets"`
	Reasons              []string `json:"reasons"`
	Commands             []string `json:"commands"`
	RequestedBy          string   `json:"requested_by,omitempty"`
	CreatedAt            string   `json:"created_at"`
}

func PlanCandidate(ctx context.Context, rootDir string, options CandidateOptions) (Candidate, error) {
	if options.ReleaseBatchID == "" {
		return Candidate{}, errors.New("release_batch_id_required")
	}
	if options.RequestedBy == "" {
		options.RequestedBy = "system"
	}
	if len(options.DeploymentTargets) == 0 {
		options.DeploymentTargets = []string{"test_dev"}
	}
	now := time.Now().UTC()
	candidate := Candidate{
		ID:                "release-candidate-" + textutil.Slugify(options.ReleaseBatchID) + "-" + now.Format("20060102150405"),
		ReleaseBatchID:    options.ReleaseBatchID,
		Status:            "blocked",
		Decision:          "RELEASE_CANDIDATE_BLOCKED",
		DeploymentTargets: options.DeploymentTargets,
		Reasons:           []string{},
		Commands:          []string{},
		RequestedBy:       options.RequestedBy,
		CreatedAt:         now.Format(time.RFC3339Nano),
	}
	batch, found, err := LoadBatchPlan(rootDir, options.ReleaseBatchID)
	if err != nil {
		return Candidate{}, err
	}
	if !found {
		candidate.Reasons = append(candidate.Reasons, "release_batch_missing")
		return finishCandidate(rootDir, candidate)
	}
	candidate.IntegrationApplyID = batch.IntegrationApplyID
	candidate.IntegrationPreviewID = batch.IntegrationPreviewID
	candidate.MergeQueueID = batch.MergeQueueID
	candidate.BatchID = batch.BatchID
	candidate.EpicID = batch.EpicID
	candidate.Version = batch.Version
	candidate.ReleaseBranch = batch.ReleaseBranch
	candidate.SourceBranch = batch.SourceBranch
	candidate.ReadyItemCount = batch.ReadyItemCount
	candidate.MinItems = batch.MinItems
	candidate.Commands = append([]string{}, batch.Commands...)
	if batch.Status != "suggested" || batch.Decision != "RELEASE_BATCH_SUGGESTED" {
		candidate.Reasons = append(candidate.Reasons, "release_batch_not_suggested:"+batch.Decision)
		return finishCandidate(rootDir, candidate)
	}
	status := gitadapter.StatusOf(ctx, rootDir)
	if !status.IsRepo {
		candidate.Reasons = append(candidate.Reasons, "not_git_repository")
		return finishCandidate(rootDir, candidate)
	}
	ws, err := workspace.Load(rootDir)
	if err != nil {
		return Candidate{}, err
	}
	remoteName := ws.Repository.Repository.DefaultRemote
	if remoteName == "" {
		remoteName = "origin"
	}
	candidate.RemoteName = remoteName
	remoteURL, ok := remoteURL(ctx, rootDir, remoteName)
	if !ok {
		candidate.Reasons = append(candidate.Reasons, "remote_missing:"+remoteName)
		return finishCandidate(rootDir, candidate)
	}
	candidate.RemoteURL = remoteURL
	candidate.Provider = detectProvider(ws.Repository.Repository.Source.Provider, remoteURL)
	candidate.Status = "ready"
	candidate.Decision = "RELEASE_CANDIDATE_READY"
	candidate.Reasons = append(candidate.Reasons, "release_batch_ready")
	return finishCandidate(rootDir, candidate)
}

func LoadCandidate(rootDir string, id string) (Candidate, bool, error) {
	var candidate Candidate
	found, err := fsutil.ReadJSON(candidatePath(rootDir, id), &candidate)
	return candidate, found, err
}

func ListCandidates(rootDir string, releaseBatchID string, limit int) ([]Candidate, error) {
	if limit <= 0 {
		limit = 20
	}
	lines, err := fsutil.TailLines(candidatesJSONLPath(rootDir), limit*4)
	if err != nil {
		return nil, err
	}
	candidates := []Candidate{}
	for _, line := range lines {
		var candidate Candidate
		if err := json.Unmarshal([]byte(line), &candidate); err != nil {
			return nil, err
		}
		if candidate.ID == "" {
			continue
		}
		if releaseBatchID != "" && candidate.ReleaseBatchID != releaseBatchID {
			continue
		}
		candidates = append(candidates, candidate)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].CreatedAt > candidates[j].CreatedAt
	})
	if len(candidates) > limit {
		return candidates[:limit], nil
	}
	return candidates, nil
}

func finishCandidate(rootDir string, candidate Candidate) (Candidate, error) {
	if err := fsutil.WriteJSON(candidatePath(rootDir, candidate.ID), candidate); err != nil {
		return Candidate{}, err
	}
	if err := fsutil.AppendJSONL(candidatesJSONLPath(rootDir), candidate); err != nil {
		return Candidate{}, err
	}
	_ = logging.Log(rootDir, "release", "release.candidate.created", map[string]any{
		"release_candidate_id": candidate.ID,
		"release_batch_id":     candidate.ReleaseBatchID,
		"decision":             candidate.Decision,
		"status":               candidate.Status,
		"version":              candidate.Version,
	})
	return candidate, nil
}

func candidatePath(rootDir string, id string) string {
	return filepath.Join(workspace.ForRoot(rootDir).ReleasesDir, "candidates", id+".json")
}

func candidatesJSONLPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).ReleasesDir, "release-candidates.jsonl")
}
