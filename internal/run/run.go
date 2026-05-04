package run

import (
	"crypto/rand"
	"encoding/hex"
	"path/filepath"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/workspace"
)

type Record struct {
	ID        string         `json:"id"`
	TaskID    string         `json:"task_id"`
	Status    string         `json:"status"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
	Payload   map[string]any `json:"payload"`
}

func makeID(taskID string) string {
	buf := make([]byte, 3)
	_, _ = rand.Read(buf)
	return "run-" + taskID + "-" + time.Now().UTC().Format("20060102150405") + "-" + hex.EncodeToString(buf)
}

func Create(rootDir string, taskID string, payload map[string]any) (Record, error) {
	if taskID == "" {
		taskID = "task-unknown"
	}
	if payload == nil {
		payload = map[string]any{}
	}
	record := Record{
		ID:        makeID(taskID),
		TaskID:    taskID,
		Status:    "queued",
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Payload:   payload,
	}
	paths := workspace.ForRoot(rootDir)
	if err := fsutil.WriteJSON(filepath.Join(paths.RunsDir, record.ID+".json"), record); err != nil {
		return Record{}, err
	}
	_ = fsutil.AppendJSONL(filepath.Join(paths.RunsDir, "events.jsonl"), map[string]any{
		"ts":      time.Now().UTC().Format(time.RFC3339Nano),
		"event":   "run.created",
		"run_id":  record.ID,
		"task_id": taskID,
	})
	_ = logging.Log(rootDir, "run", "run.created", map[string]any{"run_id": record.ID, "task_id": taskID})
	return record, nil
}
