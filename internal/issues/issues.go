package issues

import (
	"path/filepath"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type Epic struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

type Node struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Status    string   `json:"status"`
	DependsOn []string `json:"depends_on"`
}

type Graph struct {
	Epic  Epic   `json:"epic"`
	Nodes []Node `json:"nodes"`
}

type Schedule struct {
	Epic         Epic     `json:"epic"`
	Nodes        []Node   `json:"nodes"`
	ReadyQueue   []string `json:"ready_queue"`
	BlockedQueue []string `json:"blocked_queue"`
}

func Phase1Template() Graph {
	return Graph{
		Epic: Epic{ID: "phase1-epic", Title: "local-cli-mvp", Status: "planned"},
		Nodes: []Node{
			{ID: "phase1-001", Title: "workspace-core", Status: "ready", DependsOn: []string{}},
			{ID: "phase1-002", Title: "auth-context", Status: "blocked", DependsOn: []string{"phase1-001"}},
			{ID: "phase1-003", Title: "logging-audit", Status: "blocked", DependsOn: []string{"phase1-001"}},
			{ID: "phase1-004", Title: "cli-bootstrap", Status: "blocked", DependsOn: []string{"phase1-001", "phase1-002", "phase1-003"}},
			{ID: "phase1-005", Title: "git-adapter-basics", Status: "blocked", DependsOn: []string{"phase1-001", "phase1-002", "phase1-003"}},
			{ID: "phase1-006", Title: "runtime-adapters-core", Status: "blocked", DependsOn: []string{"phase1-001", "phase1-003"}},
			{ID: "phase1-007", Title: "project-comprehension", Status: "blocked", DependsOn: []string{"phase1-005"}},
			{ID: "phase1-008", Title: "orchestrator-core", Status: "blocked", DependsOn: []string{"phase1-004", "phase1-005", "phase1-006", "phase1-007"}},
			{ID: "phase1-009", Title: "scheduler-core", Status: "blocked", DependsOn: []string{"phase1-008"}},
			{ID: "phase1-010", Title: "quality-gates-core", Status: "blocked", DependsOn: []string{"phase1-003", "phase1-005", "phase1-006"}},
			{ID: "phase1-011", Title: "memory-basics", Status: "blocked", DependsOn: []string{"phase1-007", "phase1-008"}},
			{ID: "phase1-012", Title: "repair-basics", Status: "blocked", DependsOn: []string{"phase1-010", "phase1-011"}},
			{ID: "phase1-013", Title: "e2e-smoke", Status: "blocked", DependsOn: []string{"phase1-004", "phase1-005", "phase1-006", "phase1-007", "phase1-008", "phase1-009", "phase1-010", "phase1-011", "phase1-012"}},
		},
	}
}

func graphPath(rootDir string, epicID string) string {
	return filepath.Join(workspace.ForRoot(rootDir).IssueGraphsDir, textutil.Slugify(epicID)+".json")
}

func schedulePath(rootDir string, epicID string) string {
	return filepath.Join(workspace.ForRoot(rootDir).SchedulesDir, textutil.Slugify(epicID)+".json")
}

func LoadGraph(rootDir string, epicID string) (Graph, bool, error) {
	file := graphPath(rootDir, epicID)
	var graph Graph
	found, err := fsutil.ReadJSON(file, &graph)
	if err != nil {
		return Graph{}, false, err
	}
	if found {
		return graph, true, nil
	}
	if textutil.Slugify(epicID) == "phase1-epic" {
		return Phase1Template(), true, nil
	}
	return Graph{}, false, nil
}

func SaveGraph(rootDir string, graph Graph) error {
	if err := fsutil.WriteJSON(graphPath(rootDir, graph.Epic.ID), graph); err != nil {
		return err
	}
	return logging.Log(rootDir, "run", "issue.graph.saved", map[string]any{"epic_id": graph.Epic.ID})
}

func Summarize(graph Graph) Schedule {
	schedule := Schedule{Epic: graph.Epic, Nodes: graph.Nodes, ReadyQueue: []string{}, BlockedQueue: []string{}}
	for _, node := range graph.Nodes {
		switch node.Status {
		case "ready":
			schedule.ReadyQueue = append(schedule.ReadyQueue, node.ID)
		case "blocked":
			schedule.BlockedQueue = append(schedule.BlockedQueue, node.ID)
		}
	}
	return schedule
}

func LoadSchedule(rootDir string, epicID string) (Schedule, bool, error) {
	file := schedulePath(rootDir, epicID)
	var schedule Schedule
	found, err := fsutil.ReadJSON(file, &schedule)
	if err != nil {
		return Schedule{}, false, err
	}
	if found {
		return schedule, true, nil
	}
	graph, ok, err := LoadGraph(rootDir, epicID)
	if err != nil || !ok {
		return Schedule{}, ok, err
	}
	return Summarize(graph), true, nil
}

func SaveSchedule(rootDir string, schedule Schedule) error {
	if err := fsutil.WriteJSON(schedulePath(rootDir, schedule.Epic.ID), schedule); err != nil {
		return err
	}
	return logging.Log(rootDir, "run", "issue.schedule.saved", map[string]any{"epic_id": schedule.Epic.ID})
}

func GeneratePhase1(rootDir string) (Graph, Schedule, error) {
	graph := Phase1Template()
	if err := SaveGraph(rootDir, graph); err != nil {
		return Graph{}, Schedule{}, err
	}
	schedule := Summarize(graph)
	if err := SaveSchedule(rootDir, schedule); err != nil {
		return Graph{}, Schedule{}, err
	}
	return graph, schedule, nil
}
