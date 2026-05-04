package evidence

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type Record struct {
	ID          string        `json:"id"`
	ParentType  string        `json:"parent_type"`
	ParentID    string        `json:"parent_id"`
	SubjectType string        `json:"subject_type,omitempty"`
	SubjectID   string        `json:"subject_id,omitempty"`
	Operation   string        `json:"operation"`
	Status      string        `json:"status"`
	Decision    string        `json:"decision"`
	Reasons     []string      `json:"reasons"`
	Artifacts   []ArtifactRef `json:"artifacts,omitempty"`
	Source      string        `json:"source"`
	CreatedAt   string        `json:"created_at"`
}

type ArtifactRef struct {
	Kind string `json:"kind"`
	ID   string `json:"id,omitempty"`
	Path string `json:"path,omitempty"`
}

type AddOptions struct {
	ParentType  string
	ParentID    string
	SubjectType string
	SubjectID   string
	Operation   string
	Status      string
	Decision    string
	Reasons     []string
	Artifacts   []ArtifactRef
	Source      string
}

type ListOptions struct {
	ParentType  string
	ParentID    string
	SubjectType string
	SubjectID   string
	Limit       int
}

func Add(rootDir string, options AddOptions) (Record, error) {
	if err := fsutil.EnsureDir(dir(rootDir)); err != nil {
		return Record{}, err
	}
	now := time.Now().UTC()
	timestamp := strings.ReplaceAll(now.Format("20060102150405.000000000"), ".", "")
	record := Record{
		ID:          "evidence-" + textutil.Slugify(options.ParentType+"-"+options.ParentID+"-"+options.Operation) + "-" + timestamp,
		ParentType:  normalize(options.ParentType),
		ParentID:    strings.TrimSpace(options.ParentID),
		SubjectType: normalize(options.SubjectType),
		SubjectID:   strings.TrimSpace(options.SubjectID),
		Operation:   normalizeOperation(options.Operation),
		Status:      normalize(options.Status),
		Decision:    strings.TrimSpace(options.Decision),
		Reasons:     append([]string{}, options.Reasons...),
		Artifacts:   append([]ArtifactRef{}, options.Artifacts...),
		Source:      normalize(options.Source),
		CreatedAt:   now.Format(time.RFC3339Nano),
	}
	if record.ParentType == "" {
		record.ParentType = "operation"
	}
	if record.ParentID == "" {
		record.ParentID = "unknown"
	}
	if record.Operation == "" {
		record.Operation = "operation.recorded"
	}
	if record.Source == "" {
		record.Source = "moyuan"
	}
	if err := fsutil.WriteJSON(path(rootDir, record.ID), record); err != nil {
		return Record{}, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(dir(rootDir), "evidence.jsonl"), record); err != nil {
		return Record{}, err
	}
	_ = logging.Log(rootDir, "audit", "evidence.recorded", map[string]any{
		"evidence_id": record.ID,
		"parent_type": record.ParentType,
		"parent_id":   record.ParentID,
		"operation":   record.Operation,
		"status":      record.Status,
		"decision":    record.Decision,
	})
	return record, nil
}

func Load(rootDir string, id string) (Record, bool, error) {
	id, ok := cleanID(id)
	if !ok {
		return Record{}, false, nil
	}
	var record Record
	found, err := fsutil.ReadJSON(path(rootDir, id), &record)
	return record, found, err
}

func List(rootDir string, options ListOptions) ([]Record, error) {
	if err := fsutil.EnsureDir(dir(rootDir)); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir(rootDir))
	if err != nil {
		return nil, err
	}
	records := []Record{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var record Record
		found, err := fsutil.ReadJSON(filepath.Join(dir(rootDir), entry.Name()), &record)
		if err != nil {
			return nil, err
		}
		if found && matches(record, options) {
			records = append(records, record)
		}
	}
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].CreatedAt > records[j].CreatedAt
	})
	if options.Limit > 0 && len(records) > options.Limit {
		return records[:options.Limit], nil
	}
	return records, nil
}

func matches(record Record, options ListOptions) bool {
	if normalize(options.ParentType) != "" && record.ParentType != normalize(options.ParentType) {
		return false
	}
	if strings.TrimSpace(options.ParentID) != "" && record.ParentID != strings.TrimSpace(options.ParentID) {
		return false
	}
	if normalize(options.SubjectType) != "" && record.SubjectType != normalize(options.SubjectType) {
		return false
	}
	if strings.TrimSpace(options.SubjectID) != "" && record.SubjectID != strings.TrimSpace(options.SubjectID) {
		return false
	}
	return true
}

func dir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).LifecycleDir, "evidence")
}

func path(rootDir string, id string) string {
	return filepath.Join(dir(rootDir), id+".json")
}

func cleanID(id string) (string, bool) {
	id = strings.TrimSpace(id)
	if id == "" || strings.Contains(id, "/") || strings.Contains(id, "\\") || strings.Contains(id, "..") {
		return "", false
	}
	return id, true
}

func normalize(value string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "-", "_")
}

func normalizeOperation(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	return strings.ReplaceAll(value, " ", ".")
}
