package issues

import (
	"fmt"
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
	Role      string   `json:"role,omitempty"`
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

func ProjectKickoffTemplate(rootDir string) Graph {
	projectName := filepath.Base(rootDir)
	nodes := []Node{
		{ID: "phase1-001", Title: "项目画像与技术栈确认", Status: "ready", Role: "architect", DependsOn: []string{}},
	}
	qualityDeps := []string{"phase1-001"}
	nextID := 2
	addNode := func(title string, role string) {
		id := phase1NodeID(nextID)
		nodes = append(nodes, Node{ID: id, Title: title, Status: "blocked", Role: role, DependsOn: []string{"phase1-001"}})
		qualityDeps = append(qualityDeps, id)
		nextID++
	}

	if hasAny(rootDir, []string{"backend", filepath.Join("backend", "pyproject.toml"), filepath.Join("backend", "requirements.txt"), "pyproject.toml", "requirements.txt"}) {
		title := "后端服务与数据基线"
		if hasAny(rootDir, []string{filepath.Join("backend", "pyproject.toml"), filepath.Join("backend", "requirements.txt"), "pyproject.toml", "requirements.txt"}) {
			title = "Python 后端服务与数据基线"
		}
		addNode(title, "backend")
	}
	if hasAny(rootDir, []string{"frontend", filepath.Join("frontend", "package.json"), filepath.Join("frontend", "next.config.mjs"), "package.json"}) {
		title := "前端应用运行与交互验证"
		if hasAny(rootDir, []string{filepath.Join("frontend", "next.config.mjs"), filepath.Join("frontend", "next.config.ts"), "next.config.mjs", "next.config.ts"}) {
			title = "Next.js 前端应用运行与交互验证"
		}
		addNode(title, "frontend")
	}
	if hasAny(rootDir, []string{"skills"}) {
		addNode("项目 Skills 与工具契约梳理", "backend")
	}
	if hasAny(rootDir, []string{"docs", "README.md"}) {
		addNode("需求文档与验收边界整理", "architect")
	}
	if len(nodes) == 1 {
		addNode("项目实现规划补齐", "backend")
	}
	nodes = append(nodes, Node{
		ID:        phase1NodeID(nextID),
		Title:     "当前项目测试与质量基线",
		Status:    "blocked",
		Role:      "quality_owner",
		DependsOn: qualityDeps,
	})

	return Graph{
		Epic:  Epic{ID: "phase1-epic", Title: projectName + " 项目接入基线", Status: "planned"},
		Nodes: nodes,
	}
}

func IsPhase1Template(graph Graph) bool {
	template := Phase1Template()
	if graph.Epic.ID != template.Epic.ID || graph.Epic.Title != template.Epic.Title || len(graph.Nodes) != len(template.Nodes) {
		return false
	}
	for index, node := range graph.Nodes {
		templateNode := template.Nodes[index]
		if node.ID != templateNode.ID || node.Title != templateNode.Title || node.Status != templateNode.Status {
			return false
		}
	}
	return true
}

func phase1NodeID(index int) string {
	return fmt.Sprintf("phase1-%03d", index)
}

func hasAny(rootDir string, relPaths []string) bool {
	for _, relPath := range relPaths {
		if fsutil.Exists(filepath.Join(rootDir, relPath)) {
			return true
		}
	}
	return false
}

func shouldUseProjectKickoff(rootDir string) bool {
	return hasAny(rootDir, []string{
		"backend",
		"frontend",
		"skills",
		"package.json",
		"pyproject.toml",
		"requirements.txt",
		filepath.Join("backend", "pyproject.toml"),
		filepath.Join("backend", "requirements.txt"),
		filepath.Join("frontend", "package.json"),
	})
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
		if IsPhase1Template(graph) && shouldUseProjectKickoff(rootDir) {
			return ProjectKickoffTemplate(rootDir), true, nil
		}
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

func GenerateProjectKickoff(rootDir string) (Graph, Schedule, error) {
	graph := ProjectKickoffTemplate(rootDir)
	if err := SaveGraph(rootDir, graph); err != nil {
		return Graph{}, Schedule{}, err
	}
	schedule := Summarize(graph)
	if err := SaveSchedule(rootDir, schedule); err != nil {
		return Graph{}, Schedule{}, err
	}
	return graph, schedule, nil
}
