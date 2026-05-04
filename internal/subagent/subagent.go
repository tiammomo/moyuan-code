package subagent

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

type CreateOptions struct {
	ParentType     string   `json:"parent_type"`
	ParentID       string   `json:"parent_id"`
	IssueID        string   `json:"issue_id"`
	RunID          string   `json:"run_id"`
	Role           string   `json:"role"`
	RuntimeID      string   `json:"runtime_id"`
	ProviderID     string   `json:"provider_id,omitempty"`
	ModelID        string   `json:"model_id,omitempty"`
	Skills         []string `json:"skills"`
	MemoryScope    []string `json:"memory_scope"`
	ReadScope      []string `json:"read_scope"`
	WriteScope     []string `json:"write_scope"`
	OutputContract []string `json:"output_contract"`
}

type Instance struct {
	ID             string   `json:"id"`
	ParentType     string   `json:"parent_type"`
	ParentID       string   `json:"parent_id"`
	IssueID        string   `json:"issue_id"`
	RunID          string   `json:"run_id"`
	Role           string   `json:"role"`
	RuntimeID      string   `json:"runtime_id"`
	ProviderID     string   `json:"provider_id,omitempty"`
	ModelID        string   `json:"model_id,omitempty"`
	Status         string   `json:"status"`
	Skills         []string `json:"skills"`
	MemoryScope    []string `json:"memory_scope"`
	ReadScope      []string `json:"read_scope"`
	WriteScope     []string `json:"write_scope"`
	OutputContract []string `json:"output_contract"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
}

func Create(rootDir string, options CreateOptions) (Instance, error) {
	if err := fsutil.EnsureDir(instancesDir(rootDir)); err != nil {
		return Instance{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	issueID := strings.TrimSpace(options.IssueID)
	if issueID == "" {
		issueID = "issue-unknown"
	}
	runSlug := textutil.Slugify(options.RunID)
	if len(runSlug) > 16 {
		runSlug = runSlug[len(runSlug)-16:]
	}
	instance := Instance{
		ID:             strings.TrimSuffix("subagent-"+textutil.Slugify(issueID)+"-"+time.Now().UTC().Format("20060102150405")+"-"+runSlug, "-"),
		ParentType:     defaultString(options.ParentType, "issue"),
		ParentID:       defaultString(options.ParentID, issueID),
		IssueID:        issueID,
		RunID:          strings.TrimSpace(options.RunID),
		Role:           defaultString(options.Role, "backend"),
		RuntimeID:      defaultString(options.RuntimeID, "local_shell"),
		ProviderID:     strings.TrimSpace(options.ProviderID),
		ModelID:        strings.TrimSpace(options.ModelID),
		Status:         "dispatched",
		Skills:         normalizeList(options.Skills),
		MemoryScope:    normalizeList(options.MemoryScope),
		ReadScope:      normalizeList(options.ReadScope),
		WriteScope:     normalizeList(options.WriteScope),
		OutputContract: normalizeList(options.OutputContract),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := fsutil.WriteJSON(instancePath(rootDir, instance.ID), instance); err != nil {
		return Instance{}, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(instancesDir(rootDir), "instances.jsonl"), instance); err != nil {
		return Instance{}, err
	}
	_ = logging.Log(rootDir, "run", "subagent.created", map[string]any{"subagent_id": instance.ID, "issue_id": instance.IssueID, "run_id": instance.RunID, "role": instance.Role, "runtime_id": instance.RuntimeID})
	return instance, nil
}

func Finish(rootDir string, id string, status string) (Instance, bool, error) {
	instance, found, err := Load(rootDir, id)
	if err != nil || !found {
		return instance, found, err
	}
	instance.Status = defaultString(status, "completed")
	instance.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := fsutil.WriteJSON(instancePath(rootDir, instance.ID), instance); err != nil {
		return Instance{}, false, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(instancesDir(rootDir), "instances.jsonl"), instance); err != nil {
		return Instance{}, false, err
	}
	_ = logging.Log(rootDir, "run", "subagent.finished", map[string]any{"subagent_id": instance.ID, "status": instance.Status, "run_id": instance.RunID})
	return instance, true, nil
}

func Load(rootDir string, id string) (Instance, bool, error) {
	var instance Instance
	found, err := fsutil.ReadJSON(instancePath(rootDir, strings.TrimSpace(id)), &instance)
	return instance, found, err
}

func List(rootDir string, limit int) ([]Instance, error) {
	if err := fsutil.EnsureDir(instancesDir(rootDir)); err != nil {
		return nil, err
	}
	lines, err := fsutil.TailLines(filepath.Join(instancesDir(rootDir), "instances.jsonl"), limit*2)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	instances := []Instance{}
	for idx := len(lines) - 1; idx >= 0; idx-- {
		var instance Instance
		if err := json.Unmarshal([]byte(lines[idx]), &instance); err != nil {
			continue
		}
		if instance.ID == "" || seen[instance.ID] {
			continue
		}
		seen[instance.ID] = true
		instances = append(instances, instance)
		if limit > 0 && len(instances) >= limit {
			break
		}
	}
	sort.SliceStable(instances, func(i, j int) bool {
		return instances[i].UpdatedAt > instances[j].UpdatedAt
	})
	return instances, nil
}

func instancesDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).AgentsDir, "subagents")
}

func instancePath(rootDir string, id string) string {
	return filepath.Join(instancesDir(rootDir), id+".json")
}

func normalizeList(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func defaultString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
