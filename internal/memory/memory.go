package memory

import (
	"path/filepath"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/workspace"
)

type Record struct {
	ID        string   `json:"id"`
	Kind      string   `json:"kind"`
	Summary   string   `json:"summary"`
	Tags      []string `json:"tags"`
	Source    string   `json:"source"`
	CreatedAt string   `json:"created_at"`
	Compact   bool     `json:"compact"`
}

func Add(rootDir string, kind string, summary string, tags []string, source string) (Record, error) {
	if kind == "" {
		kind = "fact"
	}
	record := Record{
		ID:        "mem-" + time.Now().UTC().Format("20060102150405.000000000"),
		Kind:      kind,
		Summary:   summary,
		Tags:      tags,
		Source:    source,
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Compact:   false,
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).MemoryDir, "records.jsonl"), record); err != nil {
		return Record{}, err
	}
	_ = logging.Log(rootDir, "memory", "memory.record.added", map[string]any{"memory_id": record.ID, "kind": kind})
	return record, nil
}

func Search(rootDir string, query string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}
	lines, err := fsutil.TailLines(filepath.Join(workspace.ForRoot(rootDir).MemoryDir, "records.jsonl"), 500)
	if err != nil {
		return nil, err
	}
	result := []string{}
	query = strings.ToLower(query)
	for _, line := range lines {
		if query == "" || strings.Contains(strings.ToLower(line), query) {
			result = append(result, line)
			if len(result) >= limit {
				break
			}
		}
	}
	_ = logging.Log(rootDir, "memory", "memory.retrieve.completed", map[string]any{"query": query, "count": len(result)})
	return result, nil
}

func Compact(rootDir string) (map[string]any, error) {
	lines, err := fsutil.TailLines(filepath.Join(workspace.ForRoot(rootDir).MemoryDir, "records.jsonl"), 2000)
	if err != nil {
		return nil, err
	}
	summary := map[string]any{
		"created_at":    time.Now().UTC().Format(time.RFC3339Nano),
		"records_seen":  len(lines),
		"strategy":      "phase1-tail-summary",
		"output_status": "candidate",
	}
	if err := fsutil.WriteJSON(filepath.Join(workspace.ForRoot(rootDir).MemoryDir, "compact-latest.json"), summary); err != nil {
		return nil, err
	}
	_ = logging.Log(rootDir, "memory", "memory.compact.completed", summary)
	return summary, nil
}
