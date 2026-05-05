package release

import (
	"errors"
	"strings"
)

type CandidateProviderPublishOptions struct {
	CandidateID string `json:"candidate_id"`
	Approved    bool   `json:"approved,omitempty"`
	ApprovalID  string `json:"approval_id,omitempty"`
}

func ProviderPublishForCandidate(rootDir string, options CandidateProviderPublishOptions) (ProviderExecution, bool, error) {
	options.CandidateID = strings.TrimSpace(options.CandidateID)
	options.ApprovalID = strings.TrimSpace(options.ApprovalID)
	if options.CandidateID == "" {
		return ProviderExecution{}, false, errors.New("candidate_id_required")
	}
	candidate, found, err := LoadCandidate(rootDir, options.CandidateID)
	if err != nil || !found {
		return ProviderExecution{}, found, err
	}
	plan := releasePlanFromCandidate(candidate)
	if candidate.Status != "ready" || candidate.Decision != "RELEASE_CANDIDATE_READY" {
		execution := newCandidateProviderPublishBlocked(plan, candidate.ID, "release_candidate_not_ready:"+candidate.Decision)
		return finishProviderExecution(rootDir, execution)
	}
	if !candidateReleaseBranchApplied(rootDir, candidate.ID) {
		execution := newCandidateProviderPublishBlocked(plan, candidate.ID, "release_branch_apply_missing")
		return finishProviderExecution(rootDir, execution)
	}
	if !candidateProviderPreviewCompleted(rootDir, candidate.ID) {
		execution := newCandidateProviderPublishBlocked(plan, candidate.ID, "provider_preview_missing")
		return finishProviderExecution(rootDir, execution)
	}
	return providerPublishPlan(rootDir, plan, ProviderOptions{
		ReleaseID:   candidate.ID,
		CandidateID: candidate.ID,
		Approved:    options.Approved,
		ApprovalID:  options.ApprovalID,
	})
}

func releasePlanFromCandidate(candidate Candidate) Plan {
	return Plan{
		ID:            candidate.ID,
		Status:        "suggested",
		Decision:      "RELEASE_SUGGESTED",
		Version:       candidate.Version,
		ReleaseBranch: candidate.ReleaseBranch,
		RemoteName:    candidate.RemoteName,
		RemoteURL:     candidate.RemoteURL,
		Provider:      candidate.Provider,
		Reasons:       append([]string{}, candidate.Reasons...),
		CreatedAt:     candidate.CreatedAt,
	}
}

func newCandidateProviderPublishBlocked(plan Plan, candidateID string, reason string) ProviderExecution {
	execution := newProviderExecution(plan, "publish")
	execution.CandidateID = candidateID
	execution.Status = "blocked"
	execution.Decision = "RELEASE_CANDIDATE_PROVIDER_PUBLISH_BLOCKED"
	execution.Reasons = append(execution.Reasons, reason)
	return execution
}

func candidateProviderPreviewCompleted(rootDir string, candidateID string) bool {
	previews, err := ListCandidateProviderPreviews(rootDir, candidateID, 20)
	if err != nil {
		return false
	}
	for _, preview := range previews {
		if preview.Status == "completed" && preview.Decision == "RELEASE_CANDIDATE_PROVIDER_PREVIEW_READY" {
			return true
		}
	}
	return false
}
