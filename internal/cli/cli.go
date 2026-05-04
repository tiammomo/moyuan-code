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
	"moyuan-code/internal/deployment"
	"moyuan-code/internal/git"
	"moyuan-code/internal/gitprovider"
	"moyuan-code/internal/issues"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/memory"
	"moyuan-code/internal/orchestrator"
	"moyuan-code/internal/providers"
	"moyuan-code/internal/quality"
	"moyuan-code/internal/release"
	"moyuan-code/internal/repair"
	"moyuan-code/internal/requirement"
	"moyuan-code/internal/review"
	runrecord "moyuan-code/internal/run"
	"moyuan-code/internal/runtime"
	"moyuan-code/internal/serverresources"
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
	case "requirement":
		text, result, exitCode, err = handleRequirement(argv[1:], cwd)
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
	case "review":
		text, result, exitCode, err = handleReview(argv[1:], cwd)
	case "model":
		text, result, exitCode, err = handleModel(argv[1:], cwd)
	case "release":
		text, result, exitCode, err = handleRelease(ctx, argv[1:], cwd)
	case "resources":
		text, result, exitCode, err = handleResources(ctx, argv[1:], cwd)
	case "deploy":
		text, result, exitCode, err = handleDeploy(ctx, argv[1:], cwd)
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
		"moyuan git provider plan <issue-id>",
		"moyuan git provider show <plan-id>",
		"moyuan requirement plan --text <text>",
		"moyuan issue graph <epic-id>",
		"moyuan issue schedule <epic-id>",
		"moyuan run <task-id>",
		"moyuan quality check <task-id>",
		"moyuan quality report <report-id>",
		"moyuan runtime health <runtime-id>",
		"moyuan runtime invoke <runtime-id> --prompt <command> [--provider <provider-id>] [--model <model-id>]",
		"moyuan orchestrator plan <epic-id>",
		"moyuan orchestrator run <issue-id> [--role backend] [--runtime local_shell] [--provider <provider-id>] [--prompt <command>]",
		"moyuan orchestrator run list [--limit 20]",
		"moyuan orchestrator status <issue-id>",
		"moyuan orchestrator issue status <issue-id>",
		"moyuan orchestrator run status <run-id>",
		"moyuan memory add --summary <text> [--kind fact]",
		"moyuan memory search <query>",
		"moyuan memory candidates",
		"moyuan memory compact",
		"moyuan repair signal --type <type> --summary <text>",
		"moyuan repair run <plan-id> [--runtime local_shell] [--prompt <command>]",
		"moyuan repair status <attempt-id>",
		"moyuan review merge-decision <issue-id>",
		"moyuan model provider add --id <id> --vendor <vendor> --api-type <type> [--auth-ref env:KEY]",
		"moyuan model provider list",
		"moyuan model provider show <provider>",
		"moyuan model provider disable <provider>",
		"moyuan model route --role <role> [--task-type <type>] [--output-type <type>] [--repo-edit]",
		"moyuan release suggest [--version v0.1.0] [--min-issues 3]",
		"moyuan release show <release-id>",
		"moyuan resources add --id <id> --environment test_dev --host <host>",
		"moyuan resources list",
		"moyuan resources show <resource-id>",
		"moyuan resources disable <resource-id>",
		"moyuan resources expiration scan",
		"moyuan resources health scan [--environment test_dev] [--resource <resource-id>] [--approved]",
		"moyuan deploy plan <release-id> --environment test_dev [--resource <resource-id>]",
		"moyuan deploy execute <deployment-id> [--mode dry_run] [--approved] [--command <safe-command>]",
		"moyuan deploy show <deployment-id>",
		"moyuan deploy execution <execution-id>",
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
	case "provider":
		if len(args) < 2 {
			return "unknown git provider command\n", nil, 1, nil
		}
		switch args[1] {
		case "plan":
			if len(args) < 3 {
				return "missing issue id\n", nil, 1, nil
			}
			plan, err := gitprovider.CreatePlan(ctx, rootDir, args[2])
			code := 0
			if plan.Status == "blocked" {
				code = 1
			}
			return "", plan, code, err
		case "show":
			if len(args) < 3 {
				return "missing plan id\n", nil, 1, nil
			}
			plan, ok, err := gitprovider.Load(rootDir, args[2])
			if err != nil {
				return "", nil, 1, err
			}
			if !ok {
				return "", map[string]any{}, 1, nil
			}
			return "", plan, 0, nil
		}
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

func handleRequirement(args []string, cwd string) (string, any, int, error) {
	rootDir := mustRoot(cwd)
	if len(args) == 0 {
		return "unknown requirement command\n", nil, 1, nil
	}
	switch args[0] {
	case "plan":
		text := flagValue(args, "--text", "")
		if text == "" {
			return "missing --text\n", nil, 1, nil
		}
		plan, err := requirement.PlanFromText(rootDir, text)
		if err != nil {
			return "", nil, 1, err
		}
		code := 0
		if plan.ClarificationDecision.Required {
			code = 1
		}
		return "", plan, code, nil
	}
	return "unknown requirement command\n", nil, 1, nil
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
		result, err := runtime.Invoke(ctx, rootDir, runtime.Invocation{
			RunID:        run.ID,
			RuntimeID:    runtimeID,
			ProviderID:   flagValue(args, "--provider", ""),
			ModelID:      flagValue(args, "--model", ""),
			IssueID:      "runtime-invoke",
			Prompt:       prompt,
			WorktreePath: rootDir,
		})
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
		if len(args) >= 2 && args[1] == "list" {
			states, err := orchestrator.ListRunStates(rootDir, flagInt(args, "--limit", 20))
			return "", states, 0, err
		}
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
		result, err := orchestrator.RunIssueWithOptions(ctx, rootDir, issueID, orchestrator.RunOptions{
			RuntimeID:  runtimeID,
			ProviderID: flagValue(args, "--provider", ""),
			ModelID:    flagValue(args, "--model", ""),
			Role:       flagValue(args, "--role", "backend"),
			Prompt:     prompt,
		})
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
		decision, err := memory.Submit(rootDir, kind, summary, []string{}, "cli")
		return "", decision, 0, err
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
	case "candidates":
		decisions, err := memory.ListCandidates(rootDir, 20)
		return "", decisions, 0, err
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
	case "run":
		if len(args) < 2 {
			return "missing repair plan id\n", nil, 1, nil
		}
		runtimeID := flagValue(args, "--runtime", "local_shell")
		prompt := flagValue(args, "--prompt", "")
		attempt, err := repair.RunAttempt(context.Background(), rootDir, args[1], runtimeID, prompt)
		code := 0
		if attempt.Status != "" && attempt.Status != "repaired" {
			code = 1
		}
		return "", attempt, code, err
	case "status":
		if len(args) < 2 {
			return "missing repair attempt id\n", nil, 1, nil
		}
		attempt, ok, err := repair.LoadAttempt(rootDir, args[1])
		if err != nil {
			return "", nil, 1, err
		}
		if !ok {
			return "", map[string]any{}, 1, nil
		}
		return "", attempt, 0, nil
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

func handleReview(args []string, cwd string) (string, any, int, error) {
	rootDir := mustRoot(cwd)
	if len(args) == 0 {
		return "unknown review command\n", nil, 1, nil
	}
	switch args[0] {
	case "merge-decision":
		if len(args) < 2 {
			return "missing issue id\n", nil, 1, nil
		}
		decision, err := review.DecideMerge(rootDir, args[1])
		if err != nil {
			return "", nil, 1, err
		}
		code := 0
		if decision.Status != "ready_to_merge" {
			code = 1
		}
		return "", decision, code, nil
	}
	return "unknown review command\n", nil, 1, nil
}

func handleModel(args []string, cwd string) (string, any, int, error) {
	rootDir := mustRoot(cwd)
	if len(args) == 0 {
		return "unknown model command\n", nil, 1, nil
	}
	switch args[0] {
	case "provider":
		return handleModelProvider(args[1:], rootDir)
	case "route":
		role := flagValue(args, "--role", "backend")
		decision, err := providers.Route(rootDir, providers.RouteRequest{
			Role:                  role,
			TaskType:              flagValue(args, "--task-type", ""),
			OutputType:            flagValue(args, "--output-type", ""),
			RequiresRepoEdit:      hasFlag(args, "--repo-edit"),
			IncludesSecrets:       hasFlag(args, "--includes-secrets"),
			IncludesSensitiveCode: hasFlag(args, "--includes-sensitive-code"),
			IncludesProjectMemory: hasFlag(args, "--includes-project-memory"),
		})
		code := 0
		if decision.Blocked {
			code = 1
		}
		return "", decision, code, err
	}
	return "unknown model command\n", nil, 1, nil
}

func handleModelProvider(args []string, rootDir string) (string, any, int, error) {
	if len(args) == 0 {
		return "unknown model provider command\n", nil, 1, nil
	}
	switch args[0] {
	case "add":
		provider := providers.Provider{
			ID:                   flagValue(args, "--id", ""),
			Name:                 flagValue(args, "--name", ""),
			Vendor:               flagValue(args, "--vendor", ""),
			APIType:              flagValue(args, "--api-type", ""),
			BaseURL:              flagValue(args, "--base-url", ""),
			AuthRef:              flagValue(args, "--auth-ref", ""),
			RuntimeID:            flagValue(args, "--runtime", ""),
			NativeRuntime:        hasFlag(args, "--native-runtime"),
			RequireProviderLabel: hasFlag(args, "--require-provider-label"),
			DataPolicy: providers.DataPolicy{
				AllowSensitiveCode:     hasFlag(args, "--allow-sensitive-code"),
				AllowProjectMemory:     hasFlag(args, "--allow-project-memory"),
				AllowProductionContext: hasFlag(args, "--allow-production-context"),
			},
			Models:          modelsFromCLI(args),
			AllowedUseCases: flagValues(args, "--use-case"),
		}
		if provider.ID == "" {
			return "missing --id\n", nil, 1, nil
		}
		if provider.Vendor == "" {
			return "missing --vendor\n", nil, 1, nil
		}
		if provider.APIType == "" {
			return "missing --api-type\n", nil, 1, nil
		}
		provider.Enabled = !hasFlag(args, "--disabled")
		saved, err := providers.Upsert(rootDir, provider)
		return "", saved, 0, err
	case "list":
		list, err := providers.List(rootDir)
		return "", list, 0, err
	case "show":
		if len(args) < 2 {
			return "missing provider id\n", nil, 1, nil
		}
		provider, ok, err := providers.Show(rootDir, args[1])
		if err != nil {
			return "", nil, 1, err
		}
		if !ok {
			return "", map[string]any{}, 1, nil
		}
		return "", provider, 0, nil
	case "disable":
		if len(args) < 2 {
			return "missing provider id\n", nil, 1, nil
		}
		provider, ok, err := providers.Disable(rootDir, args[1])
		if err != nil {
			return "", nil, 1, err
		}
		if !ok {
			return "", map[string]any{}, 1, nil
		}
		return "", provider, 0, nil
	}
	return "unknown model provider command\n", nil, 1, nil
}

func handleRelease(ctx context.Context, args []string, cwd string) (string, any, int, error) {
	rootDir := mustRoot(cwd)
	if len(args) == 0 {
		return "unknown release command\n", nil, 1, nil
	}
	switch args[0] {
	case "suggest":
		minIssues, _ := strconv.Atoi(flagValue(args, "--min-issues", "3"))
		plan, err := release.Suggest(ctx, rootDir, release.SuggestOptions{
			Version:   flagValue(args, "--version", ""),
			MinIssues: minIssues,
		})
		code := 0
		if plan.Status == "blocked" {
			code = 1
		}
		return "", plan, code, err
	case "show":
		if len(args) < 2 {
			return "missing release id\n", nil, 1, nil
		}
		plan, ok, err := release.Load(rootDir, args[1])
		if err != nil {
			return "", nil, 1, err
		}
		if !ok {
			return "", map[string]any{}, 1, nil
		}
		return "", plan, 0, nil
	}
	return "unknown release command\n", nil, 1, nil
}

func handleResources(ctx context.Context, args []string, cwd string) (string, any, int, error) {
	rootDir := mustRoot(cwd)
	if len(args) == 0 {
		return "unknown resources command\n", nil, 1, nil
	}
	switch args[0] {
	case "add":
		cpu, _ := strconv.Atoi(flagValue(args, "--cpu", "0"))
		memoryGB, _ := strconv.Atoi(flagValue(args, "--memory-gb", "0"))
		diskGB, _ := strconv.Atoi(flagValue(args, "--disk-gb", "0"))
		resource, err := serverresources.Add(rootDir, serverresources.Resource{
			ID:                flagValue(args, "--id", ""),
			Environment:       flagValue(args, "--environment", ""),
			Host:              flagValue(args, "--host", ""),
			Provider:          flagValue(args, "--provider", ""),
			Region:            flagValue(args, "--region", ""),
			InstanceID:        flagValue(args, "--instance-id", ""),
			Owner:             flagValue(args, "--owner", ""),
			Purpose:           flagValue(args, "--purpose", ""),
			AuthRef:           flagValue(args, "--auth-ref", ""),
			ExpiresAt:         flagValue(args, "--expires-at", ""),
			MaintenanceWindow: flagValue(args, "--maintenance-window", ""),
			Spec: serverresources.ServerSpec{
				CPU:      cpu,
				MemoryGB: memoryGB,
				DiskGB:   diskGB,
				OS:       flagValue(args, "--os", ""),
			},
			Healthcheck: serverresources.Healthcheck{
				Type:       flagValue(args, "--health-type", ""),
				Target:     flagValue(args, "--health-target", ""),
				LastStatus: flagValue(args, "--health-status", ""),
			},
		})
		return "", resource, 0, err
	case "list":
		resources, err := serverresources.List(rootDir)
		return "", resources, 0, err
	case "show":
		if len(args) < 2 {
			return "missing resource id\n", nil, 1, nil
		}
		resource, ok, err := serverresources.Show(rootDir, args[1])
		if err != nil {
			return "", nil, 1, err
		}
		if !ok {
			return "", map[string]any{}, 1, nil
		}
		return "", resource, 0, nil
	case "disable":
		if len(args) < 2 {
			return "missing resource id\n", nil, 1, nil
		}
		resource, ok, err := serverresources.Disable(rootDir, args[1])
		if err != nil {
			return "", nil, 1, err
		}
		if !ok {
			return "", map[string]any{}, 1, nil
		}
		return "", resource, 0, nil
	case "expiration":
		if len(args) >= 2 && args[1] == "scan" {
			resources, err := serverresources.ExpirationScan(rootDir)
			return "", resources, 0, err
		}
	case "health":
		if len(args) >= 2 && args[1] == "scan" {
			report, err := serverresources.HealthScan(ctx, rootDir, serverresources.HealthScanOptions{
				Environment: flagValue(args, "--environment", ""),
				ResourceIDs: flagValues(args, "--resource"),
				Approved:    hasFlag(args, "--approved"),
			})
			code := 0
			if report.Status == "blocked" || report.Status == "attention_required" {
				code = 1
			}
			return "", report, code, err
		}
	}
	return "unknown resources command\n", nil, 1, nil
}

func handleDeploy(ctx context.Context, args []string, cwd string) (string, any, int, error) {
	rootDir := mustRoot(cwd)
	if len(args) == 0 {
		return "unknown deploy command\n", nil, 1, nil
	}
	switch args[0] {
	case "plan":
		if len(args) < 2 {
			return "missing release id\n", nil, 1, nil
		}
		plan, err := deployment.CreatePlan(rootDir, deployment.PlanOptions{
			ReleaseID:   args[1],
			Environment: flagValue(args, "--environment", ""),
			ResourceIDs: flagValues(args, "--resource"),
			Approved:    hasFlag(args, "--approved"),
		})
		code := 0
		if plan.Status == "blocked" {
			code = 1
		}
		return "", plan, code, err
	case "show", "status":
		if len(args) < 2 {
			return "missing deployment id\n", nil, 1, nil
		}
		plan, ok, err := deployment.Load(rootDir, args[1])
		if err != nil {
			return "", nil, 1, err
		}
		if !ok {
			return "", map[string]any{}, 1, nil
		}
		return "", plan, 0, nil
	case "execute":
		if len(args) < 2 {
			return "missing deployment id\n", nil, 1, nil
		}
		execution, err := deployment.Execute(ctx, rootDir, deployment.ExecuteOptions{
			DeploymentID: args[1],
			Mode:         flagValue(args, "--mode", "dry_run"),
			Approved:     hasFlag(args, "--approved"),
			Commands:     flagValues(args, "--command"),
		})
		code := 0
		if execution.Status == "blocked" || execution.Status == "failed" {
			code = 1
		}
		return "", execution, code, err
	case "execution":
		if len(args) < 2 {
			return "missing execution id\n", nil, 1, nil
		}
		execution, ok, err := deployment.LoadExecution(rootDir, args[1])
		if err != nil {
			return "", nil, 1, err
		}
		if !ok {
			return "", map[string]any{}, 1, nil
		}
		return "", execution, 0, nil
	}
	return "unknown deploy command\n", nil, 1, nil
}

func modelsFromCLI(args []string) []providers.Model {
	values := flagValues(args, "--model")
	models := []providers.Model{}
	for _, value := range values {
		models = append(models, providers.Model{ID: value})
	}
	return models
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

func flagInt(args []string, name string, fallback int) int {
	value := flagValue(args, name, "")
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func hasFlag(args []string, name string) bool {
	for _, arg := range args {
		if arg == name {
			return true
		}
	}
	return false
}

func flagValues(args []string, name string) []string {
	values := []string{}
	for i, arg := range args {
		if arg == name && i+1 < len(args) {
			values = append(values, args[i+1])
		}
	}
	return values
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
