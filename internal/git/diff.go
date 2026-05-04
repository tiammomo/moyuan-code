package git

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/process"
	"moyuan-code/internal/workspace"
)

type Snapshot struct {
	IsRepo       bool     `json:"is_repo"`
	Dirty        bool     `json:"dirty"`
	UserDirty    bool     `json:"user_dirty"`
	Branch       *string  `json:"branch,omitempty"`
	Remote       *string  `json:"remote,omitempty"`
	Head         *string  `json:"head,omitempty"`
	Files        []string `json:"files"`
	UserFiles    []string `json:"user_files"`
	CapturedAt   string   `json:"captured_at"`
	ControlScope string   `json:"control_scope"`
}

type DiffCapture struct {
	Status            string   `json:"status"`
	ChangedFiles      []string `json:"changed_files"`
	DiffSummaryPath   string   `json:"diff_summary_path,omitempty"`
	PreExistingDirty  bool     `json:"pre_existing_dirty"`
	NewDirty          bool     `json:"new_dirty"`
	ProtectedFiles    []string `json:"protected_files,omitempty"`
	UnavailableReason *string  `json:"unavailable_reason,omitempty"`
	Before            Snapshot `json:"before"`
	After             Snapshot `json:"after"`
}

func CaptureSnapshot(ctx context.Context, rootDir string) Snapshot {
	status := StatusOf(ctx, rootDir)
	head := headOf(ctx, rootDir)
	userFiles := filterControlFiles(status.Files)
	return Snapshot{
		IsRepo:       status.IsRepo,
		Dirty:        status.Dirty,
		UserDirty:    len(userFiles) > 0,
		Branch:       status.Branch,
		Remote:       status.Remote,
		Head:         head,
		Files:        status.Files,
		UserFiles:    userFiles,
		CapturedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		ControlScope: workspace.DirName,
	}
}

func CaptureDiff(ctx context.Context, rootDir string, before Snapshot, after Snapshot, summaryPath string, protectedPatterns []string) (DiffCapture, error) {
	capture := DiffCapture{
		Status:           "available",
		ChangedFiles:     diffFileSet(before.UserFiles, after.UserFiles),
		DiffSummaryPath:  summaryPath,
		PreExistingDirty: before.UserDirty,
		NewDirty:         !before.UserDirty && after.UserDirty,
		ProtectedFiles:   FilterProtectedFiles(after.UserFiles, protectedPatterns),
		Before:           before,
		After:            after,
	}
	if !before.IsRepo || !after.IsRepo {
		reason := "not a git repository"
		capture.Status = "diff_unavailable"
		capture.UnavailableReason = &reason
	}
	if summaryPath != "" {
		if err := fsutil.WriteText(summaryPath, renderDiffSummary(ctx, rootDir, capture)); err != nil {
			return capture, err
		}
	}
	return capture, nil
}

func FilterProtectedFiles(files []string, patterns []string) []string {
	protected := []string{}
	for _, file := range files {
		for _, pattern := range patterns {
			if matchPathPattern(file, pattern) {
				protected = append(protected, file)
				break
			}
		}
	}
	sort.Strings(protected)
	return protected
}

func headOf(ctx context.Context, rootDir string) *string {
	if !IsRepo(ctx, rootDir) {
		return nil
	}
	res := process.RunCommand(ctx, rootDir, "git", "rev-parse", "HEAD")
	if res.Code != 0 {
		return nil
	}
	head := strings.TrimSpace(res.Stdout)
	if head == "" {
		return nil
	}
	return &head
}

func filterControlFiles(statusFiles []string) []string {
	files := []string{}
	for _, line := range statusFiles {
		path := statusFilePath(line)
		if path == "" || isControlFile(path) {
			continue
		}
		files = append(files, path)
	}
	sort.Strings(files)
	return files
}

func statusFilePath(line string) string {
	line = strings.TrimSpace(line)
	if len(line) >= 3 && line[2] == ' ' {
		line = strings.TrimSpace(line[3:])
	} else if len(line) >= 2 && line[1] == ' ' {
		line = strings.TrimSpace(line[2:])
	}
	if strings.Contains(line, " -> ") {
		parts := strings.Split(line, " -> ")
		line = parts[len(parts)-1]
	}
	line = strings.Trim(line, `"`)
	line = filepath.ToSlash(line)
	line = strings.TrimPrefix(line, "./")
	return line
}

func isControlFile(path string) bool {
	path = filepath.ToSlash(path)
	path = strings.TrimPrefix(path, "./")
	return path == workspace.DirName || strings.HasPrefix(path, workspace.DirName+"/")
}

func diffFileSet(before []string, after []string) []string {
	seenBefore := map[string]bool{}
	for _, file := range before {
		seenBefore[file] = true
	}
	changed := []string{}
	for _, file := range after {
		if !seenBefore[file] {
			changed = append(changed, file)
		}
	}
	sort.Strings(changed)
	return changed
}

func matchPathPattern(path string, pattern string) bool {
	path = filepath.ToSlash(strings.TrimPrefix(path, "./"))
	pattern = filepath.ToSlash(strings.TrimSpace(pattern))
	if pattern == "" {
		return false
	}
	if pattern == path || strings.TrimSuffix(pattern, "/") == path {
		return true
	}
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	}
	if ok, _ := filepath.Match(pattern, path); ok {
		return true
	}
	if !strings.Contains(pattern, "/") {
		base := filepath.Base(path)
		if ok, _ := filepath.Match(pattern, base); ok {
			return true
		}
	}
	return false
}

func renderDiffSummary(ctx context.Context, rootDir string, capture DiffCapture) string {
	lines := []string{
		"# Runtime Diff Summary",
		"",
		"- Status: `" + capture.Status + "`",
		"- Pre Existing Dirty: `" + boolText(capture.PreExistingDirty) + "`",
		"- New Dirty: `" + boolText(capture.NewDirty) + "`",
		"- Before Head: `" + stringPtr(capture.Before.Head) + "`",
		"- After Head: `" + stringPtr(capture.After.Head) + "`",
		"- Branch: `" + stringPtr(capture.After.Branch) + "`",
		"",
		"## Changed Files",
	}
	if len(capture.ChangedFiles) == 0 {
		lines = append(lines, "- none")
	} else {
		for _, file := range capture.ChangedFiles {
			lines = append(lines, "- `"+file+"`")
		}
	}
	lines = append(lines, "", "## Protected Files")
	if len(capture.ProtectedFiles) == 0 {
		lines = append(lines, "- none")
	} else {
		for _, file := range capture.ProtectedFiles {
			lines = append(lines, "- `"+file+"`")
		}
	}
	lines = append(lines, "", "## Git Status Before")
	lines = appendStatus(lines, capture.Before.Files)
	lines = append(lines, "", "## Git Status After")
	lines = appendStatus(lines, capture.After.Files)
	if capture.Status == "available" {
		stat := strings.TrimSpace(process.RunCommand(ctx, rootDir, "git", "diff", "--stat").Stdout)
		cachedStat := strings.TrimSpace(process.RunCommand(ctx, rootDir, "git", "diff", "--cached", "--stat").Stdout)
		lines = append(lines, "", "## Diff Stat")
		if stat == "" && cachedStat == "" {
			lines = append(lines, "- none")
		}
		if stat != "" {
			lines = append(lines, "```text", stat, "```")
		}
		if cachedStat != "" {
			lines = append(lines, "```text", cachedStat, "```")
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func appendStatus(lines []string, files []string) []string {
	if len(files) == 0 {
		return append(lines, "- clean")
	}
	for _, file := range files {
		lines = append(lines, "- `"+file+"`")
	}
	return lines
}

func boolText(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func stringPtr(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
