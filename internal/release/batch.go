package release

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/review"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type BatchOptions struct {
	IntegrationApplyID string `json:"integration_apply_id"`
	Version            string `json:"version,omitempty"`
	MinItems           int    `json:"min_items,omitempty"`
	RequestedBy        string `json:"requested_by,omitempty"`
}

type BatchPlan struct {
	ID                   string   `json:"id"`
	IntegrationApplyID   string   `json:"integration_apply_id"`
	IntegrationPreviewID string   `json:"integration_preview_id,omitempty"`
	MergeQueueID         string   `json:"merge_queue_id,omitempty"`
	BatchID              string   `json:"batch_id,omitempty"`
	EpicID               string   `json:"epic_id,omitempty"`
	Status               string   `json:"status"`
	Decision             string   `json:"decision"`
	Version              string   `json:"version"`
	ReleaseBranch        string   `json:"release_branch"`
	SourceBranch         string   `json:"source_branch,omitempty"`
	ReadyItemCount       int      `json:"ready_item_count"`
	MinItems             int      `json:"min_items"`
	Reasons              []string `json:"reasons"`
	Commands             []string `json:"commands"`
	RequestedBy          string   `json:"requested_by,omitempty"`
	CreatedAt            string   `json:"created_at"`
}

func PlanBatch(rootDir string, options BatchOptions) (BatchPlan, error) {
	if options.MinItems <= 0 {
		options.MinItems = 3
	}
	if options.RequestedBy == "" {
		options.RequestedBy = "system"
	}
	now := time.Now().UTC()
	version := normalizeVersion(options.Version, now)
	plan := BatchPlan{
		ID:                 "release-batch-" + textutil.Slugify(version) + "-" + now.Format("20060102150405"),
		IntegrationApplyID: options.IntegrationApplyID,
		Status:             "blocked",
		Decision:           "RELEASE_BATCH_BLOCKED",
		Version:            version,
		ReleaseBranch:      "release/" + version,
		MinItems:           options.MinItems,
		Reasons:            []string{},
		Commands:           []string{},
		RequestedBy:        options.RequestedBy,
		CreatedAt:          now.Format(time.RFC3339Nano),
	}
	apply, found, err := review.LoadIntegrationApply(rootDir, options.IntegrationApplyID)
	if err != nil {
		return BatchPlan{}, err
	}
	if !found {
		plan.Reasons = append(plan.Reasons, "integration_apply_missing")
		return finishBatchPlan(rootDir, plan)
	}
	plan.IntegrationPreviewID = apply.PreviewID
	plan.MergeQueueID = apply.MergeQueueID
	plan.BatchID = apply.BatchID
	plan.EpicID = apply.EpicID
	plan.SourceBranch = apply.TargetBranch
	if apply.Status != "applied" || apply.Decision != "INTEGRATION_APPLY_COMPLETED" {
		plan.Reasons = append(plan.Reasons, "integration_apply_not_completed:"+apply.Decision)
		return finishBatchPlan(rootDir, plan)
	}
	preview, found, err := review.LoadIntegrationPreview(rootDir, apply.PreviewID)
	if err != nil {
		return BatchPlan{}, err
	}
	if !found {
		plan.Reasons = append(plan.Reasons, "integration_preview_missing")
		return finishBatchPlan(rootDir, plan)
	}
	for _, item := range preview.Items {
		if item.Status == "ready" && item.Decision == "INTEGRATION_ITEM_READY" {
			plan.ReadyItemCount++
		}
	}
	plan.Commands = []string{
		"git checkout -b " + plan.ReleaseBranch + " " + plan.SourceBranch,
		"git push origin " + plan.ReleaseBranch,
		"git tag " + plan.Version,
	}
	if plan.ReadyItemCount < plan.MinItems {
		plan.Status = "not_ready"
		plan.Decision = "RELEASE_BATCH_NOT_READY"
		plan.Reasons = append(plan.Reasons, "ready_item_count_below_threshold:"+strconv.Itoa(plan.MinItems))
		return finishBatchPlan(rootDir, plan)
	}
	plan.Status = "suggested"
	plan.Decision = "RELEASE_BATCH_SUGGESTED"
	plan.Reasons = append(plan.Reasons, "ready_item_threshold_met")
	return finishBatchPlan(rootDir, plan)
}

func LoadBatchPlan(rootDir string, id string) (BatchPlan, bool, error) {
	var plan BatchPlan
	found, err := fsutil.ReadJSON(batchPlanPath(rootDir, id), &plan)
	return plan, found, err
}

func ListBatchPlans(rootDir string, applyID string, limit int) ([]BatchPlan, error) {
	if limit <= 0 {
		limit = 20
	}
	lines, err := fsutil.TailLines(batchPlansJSONLPath(rootDir), limit*4)
	if err != nil {
		return nil, err
	}
	plans := []BatchPlan{}
	for _, line := range lines {
		var plan BatchPlan
		if err := json.Unmarshal([]byte(line), &plan); err != nil {
			return nil, err
		}
		if plan.ID == "" {
			continue
		}
		if applyID != "" && plan.IntegrationApplyID != applyID {
			continue
		}
		plans = append(plans, plan)
	}
	sort.SliceStable(plans, func(i, j int) bool {
		return plans[i].CreatedAt > plans[j].CreatedAt
	})
	if len(plans) > limit {
		return plans[:limit], nil
	}
	return plans, nil
}

func finishBatchPlan(rootDir string, plan BatchPlan) (BatchPlan, error) {
	if err := fsutil.WriteJSON(batchPlanPath(rootDir, plan.ID), plan); err != nil {
		return BatchPlan{}, err
	}
	if err := fsutil.AppendJSONL(batchPlansJSONLPath(rootDir), plan); err != nil {
		return BatchPlan{}, err
	}
	return plan, nil
}

func batchPlanPath(rootDir string, id string) string {
	return filepath.Join(workspace.ForRoot(rootDir).ReleasesDir, "batches", id+".json")
}

func batchPlansJSONLPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).ReleasesDir, "release-batches.jsonl")
}
