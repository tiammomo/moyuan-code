package quality

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	gitadapter "moyuan-code/internal/git"
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
	ID           string    `json:"id"`
	TaskID       string    `json:"task_id"`
	CreatedAt    string    `json:"created_at"`
	Status       string    `json:"status"`
	ReviewStatus string    `json:"review_status"`
	Checks       []Check   `json:"checks"`
	Findings     []Finding `json:"findings"`
	ChangedFiles []string  `json:"changed_files"`
	DiffSummary  string    `json:"diff_summary_path,omitempty"`
}

type Finding struct {
	ID        string `json:"id"`
	Severity  string `json:"severity"`
	Category  string `json:"category"`
	Message   string `json:"message"`
	Path      string `json:"path,omitempty"`
	Blocking  bool   `json:"blocking"`
	CreatedAt string `json:"created_at"`
}

type ReviewInput struct {
	ChangedFiles    []string
	DiffSummaryPath string
	ProtectedFiles  []string
	RuntimeRisks    []string
}

func Run(ctx context.Context, rootDir string, taskID string) (Report, error) {
	return RunWithReview(ctx, rootDir, taskID, ReviewInput{})
}

func RunWithReview(ctx context.Context, rootDir string, taskID string, input ReviewInput) (Report, error) {
	ws, err := workspace.Load(rootDir)
	if err != nil {
		return Report{}, err
	}
	reportID := "quality-" + taskID + "-" + time.Now().UTC().Format("20060102150405")
	checks := []Check{}
	checks = appendChecks(ctx, rootDir, checks, "build", ws.Project.Stack.BuildCommands)
	checks = appendChecks(ctx, rootDir, checks, "lint", ws.Project.Stack.LintCommands)
	checks = appendChecks(ctx, rootDir, checks, "test", ws.Project.Stack.TestCommands)
	findings := reviewFindings(rootDir, input, ws.Project.Workspace.ProtectedPaths)
	status := "passed"
	for _, check := range checks {
		if check.Status == "failed" {
			status = "failed"
			break
		}
	}
	for _, finding := range findings {
		if finding.Blocking {
			status = "failed"
			break
		}
	}
	reviewStatus := "accepted"
	if status != "passed" {
		reviewStatus = "rejected"
	} else if hasNonBlockingFindings(findings) {
		reviewStatus = "accepted_with_findings"
	}
	report := Report{
		ID:           reportID,
		TaskID:       taskID,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339Nano),
		Status:       status,
		ReviewStatus: reviewStatus,
		Checks:       checks,
		Findings:     findings,
		ChangedFiles: input.ChangedFiles,
		DiffSummary:  input.DiffSummaryPath,
	}
	paths := reportPaths(rootDir, reportID)
	if err := fsutil.WriteJSON(paths.json, report); err != nil {
		return Report{}, err
	}
	md := []string{"# Quality Report", "", "- Task ID: `" + taskID + "`", "- Report ID: `" + reportID + "`", "- Status: `" + status + "`", "- Review Status: `" + reviewStatus + "`", ""}
	for _, check := range checks {
		line := "- " + check.Type + ": " + check.Status
		if check.Command != nil {
			line += " (" + *check.Command + ")"
		}
		md = append(md, line)
	}
	md = append(md, "", "## Findings")
	if len(findings) == 0 {
		md = append(md, "- none")
	} else {
		for _, finding := range findings {
			line := "- " + finding.Severity + " " + finding.Category + ": " + finding.Message
			if finding.Path != "" {
				line += " (`" + finding.Path + "`)"
			}
			md = append(md, line)
		}
	}
	if err := fsutil.WriteText(paths.md, strings.Join(md, "\n")+"\n"); err != nil {
		return Report{}, err
	}
	_ = fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).QualityDir, "events.jsonl"), map[string]any{
		"ts":            time.Now().UTC().Format(time.RFC3339Nano),
		"task_id":       taskID,
		"report_id":     reportID,
		"status":        status,
		"review_status": reviewStatus,
	})
	_ = logging.Log(rootDir, "quality", "quality.check.completed", map[string]any{"task_id": taskID, "report_id": reportID, "status": status, "review_status": reviewStatus, "findings": len(findings)})
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

func reviewFindings(rootDir string, input ReviewInput, protectedPatterns []string) []Finding {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	findings := []Finding{}
	protected := input.ProtectedFiles
	if len(protected) == 0 {
		protected = gitadapter.FilterProtectedFiles(input.ChangedFiles, protectedPatterns)
	}
	for _, file := range protected {
		findings = append(findings, finding("protected_path", "blocker", "protected path changed", file, true, now))
	}
	for _, file := range input.ChangedFiles {
		if looksLikeSecretFile(file) {
			findings = append(findings, finding("secret_file", "blocker", "sensitive file changed", file, true, now))
		}
	}
	for _, risk := range input.RuntimeRisks {
		if risk == "protected_paths_changed" || strings.HasPrefix(risk, "runtime_unavailable") {
			findings = append(findings, finding("runtime_risk", "blocker", risk, "", true, now))
		}
	}
	if len(input.ChangedFiles) > 40 {
		findings = append(findings, finding("diff_size", "high", "large diff requires focused review", "", true, now))
	}
	if input.DiffSummaryPath != "" && !fsutil.Exists(input.DiffSummaryPath) {
		findings = append(findings, finding("diff_summary", "medium", "diff summary artifact missing", input.DiffSummaryPath, false, now))
	}
	return findings
}

func finding(category string, severity string, message string, path string, blocking bool, createdAt string) Finding {
	id := "finding-" + category
	if path != "" {
		id += "-" + strings.ReplaceAll(strings.ReplaceAll(path, "/", "-"), ".", "-")
	}
	return Finding{ID: id, Severity: severity, Category: category, Message: message, Path: path, Blocking: blocking, CreatedAt: createdAt}
}

func hasNonBlockingFindings(findings []Finding) bool {
	for _, finding := range findings {
		if !finding.Blocking {
			return true
		}
	}
	return false
}

func looksLikeSecretFile(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	return name == ".env" || strings.HasPrefix(name, ".env.") || strings.Contains(name, "secret") || strings.Contains(name, "token") || strings.Contains(name, "apikey") || strings.Contains(name, "api-key")
}
