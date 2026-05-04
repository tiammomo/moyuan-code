package quality

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/process"
	"moyuan-code/internal/workspace"
)

type Check struct {
	Type     string  `json:"type"`
	Command  *string `json:"command"`
	Started  string  `json:"started_at,omitempty"`
	Finished string  `json:"finished_at,omitempty"`
	Status   string  `json:"status"`
	ExitCode *int    `json:"exit_code,omitempty"`
	Stdout   string  `json:"stdout,omitempty"`
	Stderr   string  `json:"stderr,omitempty"`
	Reason   string  `json:"reason,omitempty"`
}

type Report struct {
	ID        string  `json:"id"`
	TaskID    string  `json:"task_id"`
	CreatedAt string  `json:"created_at"`
	Status    string  `json:"status"`
	Checks    []Check `json:"checks"`
}

func Run(ctx context.Context, rootDir string, taskID string) (Report, error) {
	ws, err := workspace.Load(rootDir)
	if err != nil {
		return Report{}, err
	}
	reportID := "quality-" + taskID + "-" + time.Now().UTC().Format("20060102150405")
	checks := []Check{}
	checks = appendChecks(ctx, rootDir, checks, "build", ws.Project.Stack.BuildCommands)
	checks = appendChecks(ctx, rootDir, checks, "lint", ws.Project.Stack.LintCommands)
	checks = appendChecks(ctx, rootDir, checks, "test", ws.Project.Stack.TestCommands)
	status := "passed"
	for _, check := range checks {
		if check.Status == "failed" {
			status = "failed"
			break
		}
	}
	report := Report{
		ID:        reportID,
		TaskID:    taskID,
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Status:    status,
		Checks:    checks,
	}
	paths := reportPaths(rootDir, reportID)
	if err := fsutil.WriteJSON(paths.json, report); err != nil {
		return Report{}, err
	}
	md := []string{"# Quality Report", "", "- Task ID: `" + taskID + "`", "- Report ID: `" + reportID + "`", "- Status: `" + status + "`", ""}
	for _, check := range checks {
		line := "- " + check.Type + ": " + check.Status
		if check.Command != nil {
			line += " (" + *check.Command + ")"
		}
		md = append(md, line)
	}
	if err := fsutil.WriteText(paths.md, strings.Join(md, "\n")+"\n"); err != nil {
		return Report{}, err
	}
	_ = fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).QualityDir, "events.jsonl"), map[string]any{
		"ts":        time.Now().UTC().Format(time.RFC3339Nano),
		"task_id":   taskID,
		"report_id": reportID,
		"status":    status,
	})
	_ = logging.Log(rootDir, "quality", "quality.check.completed", map[string]any{"task_id": taskID, "report_id": reportID, "status": status})
	return report, nil
}

func Read(rootDir string, reportID string) (Report, bool, error) {
	var report Report
	found, err := fsutil.ReadJSON(reportPaths(rootDir, reportID).json, &report)
	return report, found, err
}

func appendChecks(ctx context.Context, rootDir string, checks []Check, typ string, commands []string) []Check {
	if len(commands) == 0 {
		return append(checks, Check{Type: typ, Status: "skipped", Reason: "no " + typ + " command configured"})
	}
	for _, command := range commands {
		started := time.Now().UTC().Format(time.RFC3339Nano)
		result := process.RunShell(ctx, rootDir, command)
		finished := time.Now().UTC().Format(time.RFC3339Nano)
		status := "passed"
		if result.Code != 0 {
			status = "failed"
		}
		code := result.Code
		cmd := command
		checks = append(checks, Check{
			Type:     typ,
			Command:  &cmd,
			Started:  started,
			Finished: finished,
			Status:   status,
			ExitCode: &code,
			Stdout:   result.Stdout,
			Stderr:   result.Stderr,
		})
	}
	return checks
}

func reportPaths(rootDir string, id string) struct{ json, md string } {
	base := filepath.Join(workspace.ForRoot(rootDir).QualityDir, "reports")
	return struct{ json, md string }{
		json: filepath.Join(base, id+".json"),
		md:   filepath.Join(base, id+".md"),
	}
}
