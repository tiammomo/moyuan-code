package orchestrator

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/issues"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/workspace"
)

type IssueState struct {
	IssueID         string            `json:"issue_id"`
	EpicID          string            `json:"epic_id"`
	Status          string            `json:"status"`
	LastRunID       string            `json:"last_run_id,omitempty"`
	BlockedReason   string            `json:"blocked_reason,omitempty"`
	QualityReportID string            `json:"quality_report_id,omitempty"`
	UpdatedAt       string            `json:"updated_at"`
	History         []Transition      `json:"history"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

type RunState struct {
	RunID           string       `json:"run_id"`
	IssueID         string       `json:"issue_id"`
	Status          string       `json:"status"`
	SubagentID      string       `json:"subagent_id,omitempty"`
	RuntimeID       string       `json:"runtime_id,omitempty"`
	RuntimeStatus   string       `json:"runtime_status,omitempty"`
	QualityStatus   string       `json:"quality_status,omitempty"`
	QualityReportID string       `json:"quality_report_id,omitempty"`
	UpdatedAt       string       `json:"updated_at"`
	History         []Transition `json:"history"`
}

type Transition struct {
	From   string `json:"from,omitempty"`
	To     string `json:"to"`
	Reason string `json:"reason,omitempty"`
	RunID  string `json:"run_id,omitempty"`
	At     string `json:"at"`
}

func LoadIssueState(rootDir string, issueID string) (IssueState, bool, error) {
	var state IssueState
	found, err := fsutil.ReadJSON(issueStatePath(rootDir, issueID), &state)
	return state, found, err
}

func LoadRunState(rootDir string, runID string) (RunState, bool, error) {
	var state RunState
	found, err := fsutil.ReadJSON(runStatePath(rootDir, runID), &state)
	return state, found, err
}

func ListRunStates(rootDir string, limit int) ([]RunState, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return nil, err
	}
	dir := filepath.Join(workspace.ForRoot(rootDir).OrchestratorDir, "run-states")
	if err := fsutil.EnsureDir(dir); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	states := []RunState{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var state RunState
		found, err := fsutil.ReadJSON(filepath.Join(dir, entry.Name()), &state)
		if err != nil {
			return nil, err
		}
		if found && state.RunID != "" {
			states = append(states, state)
		}
	}
	sort.SliceStable(states, func(i, j int) bool {
		return states[i].UpdatedAt > states[j].UpdatedAt
	})
	if limit > 0 && len(states) > limit {
		return states[:limit], nil
	}
	return states, nil
}

func transitionIssue(rootDir string, epicID string, issueID string, status string, reason string, runID string, mutate func(*IssueState)) (IssueState, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	state, found, err := LoadIssueState(rootDir, issueID)
	if err != nil {
		return IssueState{}, err
	}
	if !found {
		state = IssueState{
			IssueID:   issueID,
			EpicID:    epicID,
			Status:    "created",
			UpdatedAt: now,
			History:   []Transition{},
			Metadata:  map[string]string{},
		}
	}
	previous := state.Status
	state.Status = status
	state.UpdatedAt = now
	if runID != "" {
		state.LastRunID = runID
	}
	if reason != "" {
		state.BlockedReason = reason
	}
	state.History = append(state.History, Transition{From: previous, To: status, Reason: reason, RunID: runID, At: now})
	if mutate != nil {
		mutate(&state)
	}
	if err := fsutil.WriteJSON(issueStatePath(rootDir, issueID), state); err != nil {
		return IssueState{}, err
	}
	if err := syncGraphIssueStatus(rootDir, epicID, issueID, status); err != nil {
		return IssueState{}, err
	}
	_ = logging.Log(rootDir, "run", "orchestrator.issue.transitioned", map[string]any{"issue_id": issueID, "from": previous, "to": status, "run_id": runID})
	return state, nil
}

func transitionRun(rootDir string, issueID string, runID string, status string, reason string, mutate func(*RunState)) (RunState, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	state, found, err := LoadRunState(rootDir, runID)
	if err != nil {
		return RunState{}, err
	}
	if !found {
		state = RunState{
			RunID:     runID,
			IssueID:   issueID,
			Status:    "created",
			UpdatedAt: now,
			History:   []Transition{},
		}
	}
	previous := state.Status
	state.Status = status
	state.UpdatedAt = now
	state.History = append(state.History, Transition{From: previous, To: status, Reason: reason, RunID: runID, At: now})
	if mutate != nil {
		mutate(&state)
	}
	if err := fsutil.WriteJSON(runStatePath(rootDir, runID), state); err != nil {
		return RunState{}, err
	}
	_ = logging.Log(rootDir, "run", "orchestrator.run.transitioned", map[string]any{"issue_id": issueID, "run_id": runID, "from": previous, "to": status})
	return state, nil
}

func issueStatePath(rootDir string, issueID string) string {
	return filepath.Join(workspace.ForRoot(rootDir).OrchestratorDir, "issue-states", issueID+".json")
}

func runStatePath(rootDir string, runID string) string {
	return filepath.Join(workspace.ForRoot(rootDir).OrchestratorDir, "run-states", runID+".json")
}

func syncGraphIssueStatus(rootDir string, epicID string, issueID string, status string) error {
	graph, ok, err := issues.LoadGraph(rootDir, epicID)
	if err != nil || !ok {
		return err
	}
	for idx := range graph.Nodes {
		if graph.Nodes[idx].ID == issueID {
			graph.Nodes[idx].Status = status
			break
		}
	}
	if err := issues.SaveGraph(rootDir, graph); err != nil {
		return err
	}
	return issues.SaveSchedule(rootDir, issues.Summarize(graph))
}
