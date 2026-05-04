package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"moyuan-code/internal/api"
	"moyuan-code/internal/auth"
	"moyuan-code/internal/comprehension"
	"moyuan-code/internal/controlplane"
	"moyuan-code/internal/git"
	"moyuan-code/internal/issues"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/memory"
	"moyuan-code/internal/orchestrator"
	"moyuan-code/internal/quality"
	"moyuan-code/internal/repair"
	runrecord "moyuan-code/internal/run"
	"moyuan-code/internal/runtime"
	"moyuan-code/internal/store"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

func Run(ctx context.Context, argv []string, stdout io.Writer, stderr io.Writer) int {
	cwd, err := filepath.Abs(".")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	rootFlag := flagValue(argv, "--root", "")
	if rootFlag != "" {
		cwd, _ = filepath.Abs(rootFlag)
		argv = removeFlag(argv, "--root")
	}
	if len(argv) == 0 || argv[0] == "--help" || argv[0] == "-h" {
		fmt.Fprint(stdout, usage())
		return 0
	}
	var result any
	var text string
	var exitCode int
	switch argv[0] {
	case "project":
		text, result, exitCode, err = handleProject(ctx, argv[1:], cwd)
	case "auth":
		text, result, exitCode, err = handleAuth(argv[1:], cwd)
	case "api":
		text, result, exitCode, err = handleAPI(ctx, argv[1:], cwd)
	case "init":
		text, result, exitCode, err = handleInit(argv[1:], cwd)
	case "workspace":
		text, result, exitCode, err = handleWorkspace(argv[1:], cwd)
	case "comprehend":
		text, result, exitCode, err = handleComprehend(ctx, argv[1:], cwd)
	case "status":
		result = git.StatusOf(ctx, mustRoot(cwd))
		exitCode = 0
	case "git":
		text, result, exitCode, err = handleGit(ctx, argv[1:], cwd)
	case "issue":
		text, result, exitCode, err = handleIssue(argv[1:], cwd)
	case "run":
		text, result, exitCode, err = handleRun(argv[1:], cwd)
	case "quality":
		text, result, exitCode, err = handleQuality(ctx, argv[1:], cwd)
	case "runtime":
		text, result, exitCode, err = handleRuntime(ctx, argv[1:], cwd)
	case "orchestrator":
		text, result, exitCode, err = handleOrchestrator(ctx, argv[1:], cwd)
	case "memory":
		text, result, exitCode, err = handleMemory(argv[1:], cwd)
	case "repair":
		text, result, exitCode, err = handleRepair(argv[1:], cwd)
	case "logs":
		text, result, exitCode, err = handleLogs(argv[1:], cwd)
	default:
		fmt.Fprintln(stderr, "unknown command")
		fmt.Fprint(stderr, usage())
		return 1
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if text != "" {
		fmt.Fprint(stdout, text)
		return exitCode
	}
	if result != nil {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintln(stdout, string(data))
	}
	return exitCode
}

func usage() string {
	return strings.Join([]string{
		"moyuan project add --local <path>",
		"moyuan project add --remote <git-url>",
		"moyuan project list",
		"moyuan api serve [--addr 127.0.0.1:8080]",
		"moyuan auth init-owner [--name <name>]",
		"moyuan auth whoami",
		"moyuan init <path>",
		"moyuan comprehend [--full] [--since <commit>]",
		"moyuan status",
		"moyuan workspace doctor",
		"moyuan git status",
		"moyuan git branch list",
		"moyuan git sync [--comprehend]",
		"moyuan issue graph <epic-id>",
		"moyuan issue schedule <epic-id>",
		"moyuan run <task-id>",
		"moyuan quality check <task-id>",
		"moyuan quality report <report-id>",
		"moyuan runtime health <runtime-id>",
		"moyuan runtime invoke <runtime-id> --prompt <command>",
		"moyuan orchestrator plan <epic-id>",
		"moyuan orchestrator run <issue-id> [--runtime local_shell] [--prompt <command>]",
		"moyuan orchestrator status <issue-id>",
		"moyuan orchestrator issue status <issue-id>",
		"moyuan orchestrator run status <run-id>",
		"moyuan memory add --summary <text> [--kind fact]",
		"moyuan memory search <query>",
		"moyuan memory compact",
		"moyuan repair signal --type <type> --summary <text>",
		"moyuan logs tail [--stream run] [--limit 20]",
		"",
	}, "\n")
}

func handleProject(ctx context.Context, args []string, cwd string) (string, any, int, error) {
	if len(args) == 0 {
		return "unknown project command\n", nil, 1, nil
	}
	switch args[0] {
	case "add":
		local := flagValue(args, "--local", "")
		remote := flagValue(args, "--remote", "")
		dest := flagValue(args, "--dest", "")
		if local != "" {
			rootDir, _ := filepath.Abs(local)
			if _, err := workspace.Ensure(rootDir); err != nil {
				return "", nil, 1, err
			}
			owner, err := auth.InitOwner(rootDir, filepath.Base(rootDir))
			if err != nil {
				return "", nil, 1, err
			}
			if err := git.BindLocal(rootDir); err != nil {
				return "", nil, 1, err
			}
			if _, err := comprehension.Full(ctx, rootDir, nil); err != nil {
				return "", nil, 1, err
			}
			if _, _, err := issues.GeneratePhase1(rootDir); err != nil {
				return "", nil, 1, err
			}
			_, _ = workspace.Ensure(cwd)
			project := controlplane.Project{
				ID:      textutil.Slugify(filepath.Base(rootDir)),
				Name:    filepath.Base(rootDir),
				Root:    rootDir,
				Source:  map[string]any{"type": "local_path", "provider": "local", "path": rootDir},
				OwnerID: owner.ActorID,
				Status:  "active",
			}
			registeredProject, err := controlplane.Register(cwd, project)
			if err != nil {
				return "", nil, 1, err
			}
			if err := syncProjectToStore(cwd, registeredProject); err != nil {
				return "", nil, 1, err
			}
			return "project added: " + rootDir + "\n", nil, 0, nil
		}
		if remote != "" {
			destDir := dest
			if destDir == "" {
				destDir = git.DefaultRemoteProjectDir(cwd, remote)
			}
			destDir, _ = filepath.Abs(destDir)
			if err := git.Clone(ctx, remote, destDir); err != nil {
				return "", nil, 1, err
			}
			if _, err := workspace.Ensure(destDir); err != nil {
				return "", nil, 1, err
			}
			owner, err := auth.InitOwner(destDir, filepath.Base(destDir))
			if err != nil {
				return "", nil, 1, err
			}
			if err := git.BindRemote(destDir, remote, "generic_git"); err != nil {
				return "", nil, 1, err
			}
			if _, err := comprehension.Full(ctx, destDir, nil); err != nil {
				return "", nil, 1, err
			}
			if _, _, err := issues.GeneratePhase1(destDir); err != nil {
				return "", nil, 1, err
			}
			_, _ = workspace.Ensure(cwd)
			project := controlplane.Project{
				ID:      textutil.Slugify(filepath.Base(destDir)),
				Name:    filepath.Base(destDir),
				Root:    destDir,
				Source:  map[string]any{"type": "remote_git", "provider": "generic_git", "url": remote, "clone_path": destDir},
				OwnerID: owner.ActorID,
				Status:  "active",
			}
			registeredProject, err := controlplane.Register(cwd, project)
			if err != nil {
				return "", nil, 1, err
			}
			if err := syncProjectToStore(cwd, registeredProject); err != nil {
				return "", nil, 1, err
			}
			return "project added: " + destDir + "\n", nil, 0, nil
		}
		return "missing --local or --remote\n", nil, 1, nil
	case "list":
		projects, err := controlplane.List(cwd)
		if err != nil {
			return "", nil, 1, err
		}
		lines := []string{}
		for _, project := range projects {
			lines = append(lines, "- "+project.ID+" "+project.Root+" "+project.Status)
		}
		if len(lines) == 0 {
			return "", nil, 0, nil
		}
		return strings.Join(lines, "\n") + "\n", nil, 0, nil
	}
	return "unknown project command\n", nil, 1, nil
}

func handleAPI(ctx context.Context, args []string, cwd string) (string, any, int, error) {
	rootDir := mustRoot(cwd)
	if len(args) == 0 {
		return "unknown api command\n", nil, 1, nil
	}
	switch args[0] {
	case "serve":
		addr := flagValue(args, "--addr", "127.0.0.1:8080")
		db, err := store.Open(rootDir)
		if err != nil {
			return "", nil, 1, err
		}
		defer db.Close()
		server := &http.Server{Addr: addr, Handler: api.NewRouter(api.Options{RootDir: rootDir, Store: &db})}
		err = server.ListenAndServe()
		if err == http.ErrServerClosed || ctx.Err() != nil {
			return "", nil, 0, nil
		}
		return "", nil, 1, err
	}
	return "unknown api command\n", nil, 1, nil
}

func syncProjectToStore(rootDir string, project controlplane.Project) error {
	db, err := store.Open(rootDir)
	if err != nil {
		return err
	}
	defer db.Close()
	return db.UpsertProject(project)
}

func handleAuth(args []string, cwd string) (string, any, int, error) {
	rootDir := mustRoot(cwd)
	if len(args) == 0 {
		return "unknown auth command\n", nil, 1, nil
	}
	switch args[0] {
	case "init-owner":
		name := flagValue(args, "--name", filepath.Base(rootDir))
		owner, err := auth.InitOwner(rootDir, name)
		if err != nil {
			return "", nil, 1, err
		}
		return owner.ActorID + "\n", nil, 0, nil
	case "whoami":
		owner, err := auth.Whoami(rootDir)
		return "", owner, 0, err
	}
	return "unknown auth command\n", nil, 1, nil
}

func handleInit(args []string, cwd string) (string, any, int, error) {
	target := cwd
	if len(args) > 0 {
		target, _ = filepath.Abs(args[0])
	}
	_, err := workspace.Ensure(target)
	if err != nil {
		return "", nil, 1, err
	}
	return "workspace initialized: " + target + "\n", nil, 0, nil
}

func handleWorkspace(args []string, cwd string) (string, any, int, error) {
	rootDir := mustRoot(cwd)
	if len(args) > 0 && args[0] == "doctor" {
		ws, err := workspace.Load(rootDir)
		if err != nil {
			return "", nil, 1, err
		}
		return "", map[string]any{"root": rootDir, "project": ws.Project, "repository": ws.Repository, "access": ws.Access}, 0, nil
	}
	return "unknown workspace command\n", nil, 1, nil
}

func handleComprehend(ctx context.Context, args []string, cwd string) (string, any, int, error) {
	rootDir := mustRoot(cwd)
	since := flagValue(args, "--since", "")
	var sincePtr *string
	if since != "" && !hasFlag(args, "--full") {
		sincePtr = &since
	}
	profile, err := comprehension.Full(ctx, rootDir, sincePtr)
	return "", profile, 0, err
}

func handleGit(ctx context.Context, args []string, cwd string) (string, any, int, error) {
	rootDir := mustRoot(cwd)
	if len(args) == 0 {
		return "unknown git command\n", nil, 1, nil
	}
	switch args[0] {
	case "status":
		return "", git.StatusOf(ctx, rootDir), 0, nil
	case "branch":
		if len(args) > 1 && args[1] == "list" {
			branches := git.Branches(ctx, rootDir)
			lines := []string{}
			for _, branch := range branches {
				lines = append(lines, "- "+branch)
			}
			if len(lines) == 0 {
				return "", nil, 0, nil
			}
			return strings.Join(lines, "\n") + "\n", nil, 0, nil
		}
	case "sync":
		result, err := git.Sync(ctx, rootDir)
		if err != nil {
			return "", nil, 1, err
		}
		if hasFlag(args, "--comprehend") {
			_, err = comprehension.Incremental(ctx, rootDir, "git-sync")
			if err != nil {
				return "", nil, 1, err
			}
		}
		return "", result, 0, nil
	}
	return "unknown git command\n", nil, 1, nil
}

func handleIssue(args []string, cwd string) (string, any, int, error) {
	rootDir := mustRoot(cwd)
	if len(args) == 0 {
		return "unknown issue command\n", nil, 1, nil
	}
	epicID := "phase1-epic"
	if len(args) > 1 {
		epicID = args[1]
	}
	switch args[0] {
	case "graph":
		graph, ok, err := issues.LoadGraph(rootDir, epicID)
		if err != nil {
			return "", nil, 1, err
		}
		if !ok {
			return "", map[string]any{}, 1, nil
		}
		return "", graph, 0, nil
	case "schedule":
		schedule, ok, err := issues.LoadSchedule(rootDir, epicID)
		if err != nil {
			return "", nil, 1, err
		}
		if !ok {
			return "", map[string]any{}, 1, nil
		}
		return "", schedule, 0, nil
	}
	return "unknown issue command\n", nil, 1, nil
}

func handleRun(args []string, cwd string) (string, any, int, error) {
	rootDir := mustRoot(cwd)
	taskID := "task-unknown"
	if len(args) > 0 {
		taskID = args[0]
	}
	record, err := runrecord.Create(rootDir, taskID, map[string]any{"issue_id": taskID, "mode": "queued"})
	return "", record, 0, err
}

func handleQuality(ctx context.Context, args []string, cwd string) (string, any, int, error) {
	rootDir := mustRoot(cwd)
	if len(args) == 0 {
		return "unknown quality command\n", nil, 1, nil
	}
	switch args[0] {
	case "check":
		taskID := "task-unknown"
		if len(args) > 1 {
			taskID = args[1]
		}
		report, err := quality.Run(ctx, rootDir, taskID)
		if err != nil {
			return "", nil, 1, err
		}
		code := 0
		if report.Status == "failed" {
			code = 1
		}
		return "", report, code, nil
	case "report":
		if len(args) < 2 {
			return "missing report id\n", nil, 1, nil
		}
		report, ok, err := quality.Read(rootDir, args[1])
		if err != nil {
			return "", nil, 1, err
		}
		if !ok {
			return "", map[string]any{}, 1, nil
		}
		return "", report, 0, nil
	}
	return "unknown quality command\n", nil, 1, nil
}

func handleRuntime(ctx context.Context, args []string, cwd string) (string, any, int, error) {
	rootDir := mustRoot(cwd)
	if len(args) == 0 {
		return "unknown runtime command\n", nil, 1, nil
	}
	switch args[0] {
	case "health":
		runtimeID := "local_shell"
		if len(args) > 1 {
			runtimeID = args[1]
		}
		health := runtime.HealthCheck(rootDir, runtimeID)
		code := 0
		if !health.OK {
			code = 1
		}
		return "", health, code, nil
	case "invoke":
		runtimeID := "local_shell"
		if len(args) > 1 {
			runtimeID = args[1]
		}
		prompt := flagValue(args, "--prompt", "")
		run, err := runrecord.Create(rootDir, "runtime-invoke", map[string]any{"runtime_id": runtimeID, "mode": "manual"})
		if err != nil {
			return "", nil, 1, err
		}
		result, err := runtime.Invoke(ctx, rootDir, runtime.Invocation{RunID: run.ID, RuntimeID: runtimeID, IssueID: "runtime-invoke", Prompt: prompt, WorktreePath: rootDir})
		if err != nil {
			return "", nil, 1, err
		}
		code := 0
		if result.Status != "completed" {
			code = 1
		}
		return "", result, code, nil
	}
	return "unknown runtime command\n", nil, 1, nil
}

func handleOrchestrator(ctx context.Context, args []string, cwd string) (string, any, int, error) {
	rootDir := mustRoot(cwd)
	if len(args) == 0 {
		return "unknown orchestrator command\n", nil, 1, nil
	}
	switch args[0] {
	case "plan":
		epicID := "phase1-epic"
		if len(args) > 1 {
			epicID = args[1]
		}
		plan, err := orchestrator.Plan(rootDir, epicID)
		return "", plan, 0, err
	case "run":
		if len(args) >= 3 && args[1] == "status" {
			state, ok, err := orchestrator.LoadRunState(rootDir, args[2])
			if err != nil {
				return "", nil, 1, err
			}
			if !ok {
				return "", map[string]any{}, 1, nil
			}
			return "", state, 0, nil
		}
		issueID := "task-unknown"
		if len(args) > 1 {
			issueID = args[1]
		}
		runtimeID := flagValue(args, "--runtime", "local_shell")
		prompt := flagValue(args, "--prompt", "")
		result, err := orchestrator.RunIssue(ctx, rootDir, issueID, runtimeID, prompt)
		code := 0
		if result.Status != "" && result.Status != "accepted" {
			code = 1
		}
		return "", result, code, err
	case "status":
		if len(args) < 2 {
			return "missing issue id\n", nil, 1, nil
		}
		state, ok, err := orchestrator.LoadIssueState(rootDir, args[1])
		if err != nil {
			return "", nil, 1, err
		}
		if !ok {
			return "", map[string]any{}, 1, nil
		}
		return "", state, 0, nil
	case "issue":
		if len(args) >= 3 && args[1] == "status" {
			state, ok, err := orchestrator.LoadIssueState(rootDir, args[2])
			if err != nil {
				return "", nil, 1, err
			}
			if !ok {
				return "", map[string]any{}, 1, nil
			}
			return "", state, 0, nil
		}
	}
	return "unknown orchestrator command\n", nil, 1, nil
}

func handleMemory(args []string, cwd string) (string, any, int, error) {
	rootDir := mustRoot(cwd)
	if len(args) == 0 {
		return "unknown memory command\n", nil, 1, nil
	}
	switch args[0] {
	case "add":
		summary := flagValue(args, "--summary", "")
		if summary == "" {
			return "missing --summary\n", nil, 1, nil
		}
		kind := flagValue(args, "--kind", "fact")
		record, err := memory.Add(rootDir, kind, summary, []string{}, "cli")
		return "", record, 0, err
	case "search":
		query := ""
		if len(args) > 1 {
			query = args[1]
		}
		records, err := memory.Search(rootDir, query, 10)
		if err != nil {
			return "", nil, 1, err
		}
		if len(records) == 0 {
			return "", nil, 0, nil
		}
		return strings.Join(records, "\n") + "\n", nil, 0, nil
	case "compact":
		summary, err := memory.Compact(rootDir)
		return "", summary, 0, err
	}
	return "unknown memory command\n", nil, 1, nil
}

func handleRepair(args []string, cwd string) (string, any, int, error) {
	rootDir := mustRoot(cwd)
	if len(args) == 0 {
		return "unknown repair command\n", nil, 1, nil
	}
	switch args[0] {
	case "signal":
		signalType := flagValue(args, "--type", "runtime_error")
		summary := flagValue(args, "--summary", "")
		if summary == "" {
			return "missing --summary\n", nil, 1, nil
		}
		sourceID := flagValue(args, "--source", "")
		signal, err := repair.CaptureSignal(rootDir, signalType, summary, sourceID)
		if err != nil {
			return "", nil, 1, err
		}
		candidate, err := repair.Classify(rootDir, signal)
		if err != nil {
			return "", nil, 1, err
		}
		plan, err := repair.PlanRepair(rootDir, candidate)
		if err != nil {
			return "", nil, 1, err
		}
		return "", map[string]any{"signal": signal, "candidate": candidate, "repair_plan": plan}, 0, nil
	}
	return "unknown repair command\n", nil, 1, nil
}

func handleLogs(args []string, cwd string) (string, any, int, error) {
	rootDir := mustRoot(cwd)
	if len(args) == 0 || args[0] != "tail" {
		return "unknown logs command\n", nil, 1, nil
	}
	stream := flagValue(args, "--stream", "run")
	limit, _ := strconv.Atoi(flagValue(args, "--limit", "20"))
	if limit <= 0 {
		limit = 20
	}
	lines, err := logging.Tail(rootDir, stream, limit)
	if err != nil {
		return "", nil, 1, err
	}
	if len(lines) == 0 {
		return "", nil, 0, nil
	}
	return strings.Join(lines, "\n") + "\n", nil, 0, nil
}

func mustRoot(cwd string) string {
	if root, ok := workspace.ResolveRoot(cwd); ok {
		return root
	}
	return cwd
}

func flagValue(args []string, name string, fallback string) string {
	for i, arg := range args {
		if arg == name && i+1 < len(args) {
			return args[i+1]
		}
	}
	return fallback
}

func hasFlag(args []string, name string) bool {
	for _, arg := range args {
		if arg == name {
			return true
		}
	}
	return false
}

func removeFlag(args []string, name string) []string {
	out := []string{}
	for i := 0; i < len(args); i++ {
		if args[i] == name {
			i++
			continue
		}
		out = append(out, args[i])
	}
	return out
}
