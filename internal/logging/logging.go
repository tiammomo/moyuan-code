package logging

import (
	"path/filepath"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/workspace"
)

type Event map[string]any

func Log(rootDir string, stream string, event string, data map[string]any) error {
	record := Event{
		"ts":     time.Now().UTC().Format(time.RFC3339Nano),
		"stream": stream,
		"event":  event,
	}
	for k, v := range data {
		record[k] = v
	}
	path := filepath.Join(workspace.ForRoot(rootDir).LogsDir, stream+".jsonl")
	return fsutil.AppendJSONL(path, record)
}

func Tail(rootDir string, stream string, limit int) ([]string, error) {
	path := filepath.Join(workspace.ForRoot(rootDir).LogsDir, stream+".jsonl")
	return fsutil.TailLines(path, limit)
}
