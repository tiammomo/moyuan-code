package scheduler

import (
	"path/filepath"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/issues"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/workspace"
)

type Plan struct {
	EpicID        string            `json:"epic_id"`
	ReadyQueue    []string          `json:"ready_queue"`
	BlockedQueue  []string          `json:"blocked_queue"`
	RunningQueue  []string          `json:"running_queue"`
	ReviewQueue   []string          `json:"review_queue"`
	BlockedReason map[string]string `json:"blocked_reason"`
	Parallelism   int               `json:"parallelism"`
	CreatedAt     string            `json:"created_at"`
}

func Build(rootDir string, epicID string, maxParallel int) (Plan, error) {
	if maxParallel <= 0 {
		maxParallel = 1
	}
	graph, ok, err := issues.LoadGraph(rootDir, epicID)
	if err != nil {
		return Plan{}, err
	}
	if !ok {
		return Plan{EpicID: epicID, ReadyQueue: []string{}, BlockedQueue: []string{}, RunningQueue: []string{}, ReviewQueue: []string{}, BlockedReason: map[string]string{}, Parallelism: 0, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}, nil
	}
	summary := issues.Summarize(graph)
	plan := Plan{
		EpicID:        epicID,
		ReadyQueue:    summary.ReadyQueue,
		BlockedQueue:  summary.BlockedQueue,
		RunningQueue:  []string{},
		ReviewQueue:   []string{},
		BlockedReason: map[string]string{},
		Parallelism:   min(maxParallel, len(summary.ReadyQueue)),
		CreatedAt:     time.Now().UTC().Format(time.RFC3339Nano),
	}
	for _, node := range graph.Nodes {
		if node.Status == "blocked" {
			plan.BlockedReason[node.ID] = "waiting_dependencies"
		}
		switch node.Status {
		case "running", "quality_checking", "verifying":
			plan.RunningQueue = append(plan.RunningQueue, node.ID)
		case "reviewing":
			plan.ReviewQueue = append(plan.ReviewQueue, node.ID)
		}
	}
	if err := fsutil.WriteJSON(filepath.Join(workspace.ForRoot(rootDir).SchedulerDir, epicID+"-plan.json"), plan); err != nil {
		return Plan{}, err
	}
	_ = logging.Log(rootDir, "run", "scheduler.plan.created", map[string]any{"epic_id": epicID, "ready": len(plan.ReadyQueue), "parallelism": plan.Parallelism})
	return plan, nil
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
