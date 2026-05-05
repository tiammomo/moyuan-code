package release

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type CandidateProviderPreview struct {
	ID             string               `json:"id"`
	CandidateID    string               `json:"candidate_id"`
	ReleaseBatchID string               `json:"release_batch_id,omitempty"`
	Version        string               `json:"version,omitempty"`
	Provider       string               `json:"provider,omitempty"`
	RemoteName     string               `json:"remote_name,omitempty"`
	RemoteURL      string               `json:"remote_url,omitempty"`
	Status         string               `json:"status"`
	Decision       string               `json:"decision"`
	Reasons        []string             `json:"reasons"`
	RemotePlan     RemotePlan           `json:"remote_plan"`
	PRMR           CandidatePRMRPreview `json:"pr_mr"`
	CreatedAt      string               `json:"created_at"`
}

type CandidatePRMRPreview struct {
	Type            string `json:"type"`
	Title           string `json:"title,omitempty"`
	Body            string `json:"body,omitempty"`
	BaseBranch      string `json:"base_branch,omitempty"`
	HeadBranch      string `json:"head_branch,omitempty"`
	CreateMode      string `json:"create_mode"`
	RemoteStatus    string `json:"remote_status,omitempty"`
	PreviewDecision string `json:"preview_decision,omitempty"`
	PreviewReason   string `json:"preview_reason,omitempty"`
}

func ProviderPreviewForCandidate(rootDir string, candidateID string) (CandidateProviderPreview, bool, error) {
	candidate, found, err := LoadCandidate(rootDir, candidateID)
	if err != nil || !found {
		return CandidateProviderPreview{}, found, err
	}
	now := time.Now().UTC()
	preview := CandidateProviderPreview{
		ID:             "release-candidate-provider-preview-" + textutil.Slugify(candidateID) + "-" + now.Format("20060102150405"),
		CandidateID:    candidate.ID,
		ReleaseBatchID: candidate.ReleaseBatchID,
		Version:        candidate.Version,
		Provider:       candidate.Provider,
		RemoteName:     candidate.RemoteName,
		RemoteURL:      candidate.RemoteURL,
		Status:         "blocked",
		Decision:       "RELEASE_CANDIDATE_PROVIDER_PREVIEW_BLOCKED",
		Reasons:        []string{},
		CreatedAt:      now.Format(time.RFC3339Nano),
	}
	if candidate.Status != "ready" || candidate.Decision != "RELEASE_CANDIDATE_READY" {
		preview.Reasons = append(preview.Reasons, "release_candidate_not_ready:"+candidate.Decision)
		return finishCandidateProviderPreview(rootDir, preview)
	}
	if !candidateReleaseBranchApplied(rootDir, candidate.ID) {
		preview.Reasons = append(preview.Reasons, "release_branch_apply_missing")
		return finishCandidateProviderPreview(rootDir, preview)
	}
	plan := Plan{
		ID:            candidate.ID,
		Status:        "suggested",
		Decision:      "RELEASE_SUGGESTED",
		Version:       candidate.Version,
		ReleaseBranch: candidate.ReleaseBranch,
		RemoteName:    candidate.RemoteName,
		RemoteURL:     candidate.RemoteURL,
		Provider:      candidate.Provider,
	}
	preview.RemotePlan = buildProviderRemotePlan(rootDir, plan)
	preview.PRMR = buildCandidatePRMRPreview(rootDir, candidate)
	preview.Status = "completed"
	preview.Decision = "RELEASE_CANDIDATE_PROVIDER_PREVIEW_READY"
	preview.Reasons = append(preview.Reasons, "no_remote_release_actions_executed")
	return finishCandidateProviderPreview(rootDir, preview)
}

func LoadCandidateProviderPreview(rootDir string, id string) (CandidateProviderPreview, bool, error) {
	var preview CandidateProviderPreview
	found, err := fsutil.ReadJSON(candidateProviderPreviewPath(rootDir, id), &preview)
	return preview, found, err
}

func ListCandidateProviderPreviews(rootDir string, candidateID string, limit int) ([]CandidateProviderPreview, error) {
	if limit <= 0 {
		limit = 20
	}
	lines, err := fsutil.TailLines(candidateProviderPreviewsJSONLPath(rootDir), limit*4)
	if err != nil {
		return nil, err
	}
	previews := []CandidateProviderPreview{}
	for _, line := range lines {
		var preview CandidateProviderPreview
		if err := json.Unmarshal([]byte(line), &preview); err != nil {
			return nil, err
		}
		if preview.ID == "" {
			continue
		}
		if candidateID != "" && preview.CandidateID != candidateID {
			continue
		}
		previews = append(previews, preview)
	}
	sort.SliceStable(previews, func(i, j int) bool {
		return previews[i].CreatedAt > previews[j].CreatedAt
	})
	if len(previews) > limit {
		return previews[:limit], nil
	}
	return previews, nil
}

func candidateReleaseBranchApplied(rootDir string, candidateID string) bool {
	applies, err := ListCandidateApplies(rootDir, candidateID, 20)
	if err != nil {
		return false
	}
	for _, apply := range applies {
		if apply.Status == "applied" && apply.Decision == "RELEASE_BRANCH_APPLY_COMPLETED" {
			return true
		}
	}
	return false
}

func CandidateReleaseBranchApplied(rootDir string, candidateID string) bool {
	return candidateReleaseBranchApplied(rootDir, candidateID)
}

func buildCandidatePRMRPreview(rootDir string, candidate Candidate) CandidatePRMRPreview {
	ws, _ := workspace.Load(rootDir)
	base := "main"
	if ws.Repository.Repository.DefaultBranch != nil && strings.TrimSpace(*ws.Repository.Repository.DefaultBranch) != "" {
		base = strings.TrimSpace(*ws.Repository.Repository.DefaultBranch)
	}
	prmrType := "manual"
	decision := "PR_MR_PREVIEW_MANUAL_REQUIRED"
	reason := "provider_without_pr_mr_preview"
	remoteStatus := "manual_required"
	switch normalize(candidate.Provider) {
	case "github", "gitee":
		prmrType = "pull_request"
		decision = "PR_MR_PREVIEW_READY"
		reason = "no_remote_pr_mr_created"
		remoteStatus = "preview_ready"
	case "gitlab":
		prmrType = "merge_request"
		decision = "PR_MR_PREVIEW_READY"
		reason = "no_remote_pr_mr_created"
		remoteStatus = "preview_ready"
	}
	return CandidatePRMRPreview{
		Type:            prmrType,
		Title:           "Release " + candidate.Version,
		Body:            "Release candidate " + candidate.ID + " from " + candidate.SourceBranch + " to " + candidate.ReleaseBranch + ".",
		BaseBranch:      base,
		HeadBranch:      candidate.ReleaseBranch,
		CreateMode:      "preview",
		RemoteStatus:    remoteStatus,
		PreviewDecision: decision,
		PreviewReason:   reason,
	}
}

func finishCandidateProviderPreview(rootDir string, preview CandidateProviderPreview) (CandidateProviderPreview, bool, error) {
	if err := fsutil.WriteJSON(candidateProviderPreviewPath(rootDir, preview.ID), preview); err != nil {
		return CandidateProviderPreview{}, true, err
	}
	if err := fsutil.AppendJSONL(candidateProviderPreviewsJSONLPath(rootDir), preview); err != nil {
		return CandidateProviderPreview{}, true, err
	}
	_ = logging.Log(rootDir, "release", "release.candidate_provider_preview.created", map[string]any{
		"release_candidate_provider_preview_id": preview.ID,
		"release_candidate_id":                  preview.CandidateID,
		"decision":                              preview.Decision,
		"status":                                preview.Status,
		"provider":                              preview.Provider,
	})
	return preview, true, nil
}

func candidateProviderPreviewPath(rootDir string, id string) string {
	return filepath.Join(workspace.ForRoot(rootDir).ReleasesDir, "candidate-provider-previews", id+".json")
}

func candidateProviderPreviewsJSONLPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).ReleasesDir, "release-candidate-provider-previews.jsonl")
}
