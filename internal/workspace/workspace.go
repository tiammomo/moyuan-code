package workspace

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/textutil"
)

type ProjectConfig struct {
	SchemaVersion int `json:"schema_version"`
	Project       struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		Root        string  `json:"root"`
		Type        string  `json:"type"`
		Description *string `json:"description"`
	} `json:"project"`
	Stack struct {
		Languages       []string `json:"languages"`
		Frameworks      []string `json:"frameworks"`
		PackageManagers []string `json:"package_managers"`
		BuildCommands   []string `json:"build_commands"`
		TestCommands    []string `json:"test_commands"`
		LintCommands    []string `json:"lint_commands"`
	} `json:"stack"`
	Workspace struct {
		ProtectedPaths []string `json:"protected_paths"`
		WritablePaths  []string `json:"writable_paths"`
	} `json:"workspace"`
}

type RepositoryConfig struct {
	SchemaVersion int `json:"schema_version"`
	Repository    struct {
		Source struct {
			Type      string  `json:"type"`
			Provider  string  `json:"provider"`
			LocalPath string  `json:"local_path,omitempty"`
			URL       *string `json:"url"`
			ClonePath *string `json:"clone_path"`
		} `json:"source"`
		DefaultRemote string  `json:"default_remote"`
		DefaultBranch *string `json:"default_branch"`
	} `json:"repository"`
	Git struct {
		BranchPolicy map[string]any `json:"branch_policy"`
		CommitPolicy map[string]any `json:"commit_policy"`
	} `json:"git"`
}

type AccessConfig struct {
	SchemaVersion int `json:"schema_version"`
	Access        struct {
		Mode           string              `json:"mode"`
		LocalOwnerID   *string             `json:"local_owner_id"`
		OrganizationID *string             `json:"organization_id"`
		ProjectRoles   map[string][]string `json:"project_roles"`
		ApprovalPolicy map[string]any      `json:"approval_policy"`
		Audit          map[string]any      `json:"audit"`
	} `json:"access"`
}

type Workspace struct {
	Paths      Paths
	Project    ProjectConfig
	Repository RepositoryConfig
	Access     AccessConfig
}

type StateFile struct {
	RootDir    string           `json:"rootDir"`
	CreatedAt  string           `json:"createdAt"`
	Phase      string           `json:"phase"`
	Runtime    string           `json:"runtime"`
	SchemaVer  int              `json:"schema_ver"`
	Project    ProjectConfig    `json:"project"`
	Repository RepositoryConfig `json:"repository"`
	Access     AccessConfig     `json:"access"`
}

type ValidationReport struct {
	Root      string            `json:"root"`
	Status    string            `json:"status"`
	Issues    []ValidationIssue `json:"issues"`
	CheckedAt string            `json:"checked_at"`
}

type ValidationIssue struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Path     string `json:"path,omitempty"`
}

func DefaultProjectConfig(rootDir string) ProjectConfig {
	name := filepath.Base(rootDir)
	cfg := ProjectConfig{SchemaVersion: 1}
	cfg.Project.ID = textutil.Slugify(name)
	cfg.Project.Name = name
	cfg.Project.Root = "."
	cfg.Project.Type = "single-repo"
	cfg.Stack.Languages = []string{}
	cfg.Stack.Frameworks = []string{}
	cfg.Stack.PackageManagers = []string{}
	cfg.Stack.BuildCommands = []string{}
	cfg.Stack.TestCommands = []string{}
	cfg.Stack.LintCommands = []string{}
	cfg.Workspace.ProtectedPaths = []string{".env", ".env.*"}
	cfg.Workspace.WritablePaths = []string{"docs", "scripts", "src", "cmd", "internal", "workers"}
	return cfg
}

func DefaultRepositoryConfig(rootDir string) RepositoryConfig {
	cfg := RepositoryConfig{SchemaVersion: 1}
	cfg.Repository.Source.Type = "local_path"
	cfg.Repository.Source.Provider = "local"
	cfg.Repository.Source.LocalPath = rootDir
	cfg.Repository.Source.URL = nil
	cfg.Repository.Source.ClonePath = nil
	cfg.Repository.DefaultRemote = "origin"
	cfg.Git.BranchPolicy = map[string]any{
		"mode":                   "task_branch",
		"naming":                 "moyuan/{issue_id}-{slug}",
		"base":                   "default_branch",
		"sync_before_run":        true,
		"require_clean_worktree": true,
		"allow_auto_commit":      false,
		"allow_auto_push":        false,
		"allow_auto_pr":          false,
	}
	cfg.Git.CommitPolicy = map[string]any{
		"enabled":             true,
		"format":              "conventional_commits",
		"require_issue_ref":   true,
		"require_quality_ref": true,
	}
	return cfg
}

func DefaultAccessConfig() AccessConfig {
	cfg := AccessConfig{SchemaVersion: 1}
	cfg.Access.Mode = "local_single_user"
	cfg.Access.ProjectRoles = map[string][]string{"owner": {"*"}}
	cfg.Access.ApprovalPolicy = map[string]any{}
	cfg.Access.Audit = map[string]any{"enabled": true}
	return cfg
}

func Ensure(rootDir string) (Workspace, error) {
	paths := ForRoot(rootDir)
	if err := EnsureDirs(paths); err != nil {
		return Workspace{}, err
	}
	statePath := filepath.Join(paths.MoyuanDir, "workspace.json")
	project := DefaultProjectConfig(paths.RootDir)
	repository := DefaultRepositoryConfig(paths.RootDir)
	access := DefaultAccessConfig()
	if fsutil.Exists(statePath) {
		var state StateFile
		if _, err := fsutil.ReadJSON(statePath, &state); err != nil {
			return Workspace{}, err
		}
		if state.Project.SchemaVersion != 0 {
			project = state.Project
		}
		if state.Repository.SchemaVersion != 0 {
			repository = state.Repository
		}
		if state.Access.SchemaVersion != 0 {
			access = state.Access
		}
	}
	if !fsutil.Exists(paths.ProjectYAML) {
		if err := fsutil.WriteText(paths.ProjectYAML, renderProjectYAML(project)); err != nil {
			return Workspace{}, err
		}
	}
	if !fsutil.Exists(paths.RepositoryYAML) {
		if err := fsutil.WriteText(paths.RepositoryYAML, renderRepositoryYAML(repository)); err != nil {
			return Workspace{}, err
		}
	}
	if !fsutil.Exists(paths.AccessYAML) {
		if err := fsutil.WriteText(paths.AccessYAML, renderAccessYAML(access)); err != nil {
			return Workspace{}, err
		}
	}
	if !fsutil.Exists(statePath) {
		if err := fsutil.WriteJSON(statePath, map[string]any{
			"rootDir":    paths.RootDir,
			"createdAt":  time.Now().UTC().Format(time.RFC3339Nano),
			"phase":      "phase1",
			"runtime":    "go-control-plane",
			"schema_ver": 1,
			"project":    project,
			"repository": repository,
			"access":     access,
		}); err != nil {
			return Workspace{}, err
		}
	}
	return Load(paths.RootDir)
}

func Load(rootDir string) (Workspace, error) {
	paths := ForRoot(rootDir)
	ws := Workspace{Paths: paths}
	state := StateFile{}
	found, err := fsutil.ReadJSON(filepath.Join(paths.MoyuanDir, "workspace.json"), &state)
	if err != nil {
		return ws, err
	}
	if found {
		ws.Project = state.Project
		ws.Repository = state.Repository
		ws.Access = state.Access
		return ws, nil
	}
	ws.Project = DefaultProjectConfig(rootDir)
	ws.Repository = DefaultRepositoryConfig(rootDir)
	ws.Access = DefaultAccessConfig()
	return ws, nil
}

func Validate(rootDir string) (ValidationReport, error) {
	paths := ForRoot(rootDir)
	report := ValidationReport{
		Root:      paths.RootDir,
		Status:    "passed",
		Issues:    []ValidationIssue{},
		CheckedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	if !fsutil.Exists(paths.MoyuanDir) {
		report.add("error", "workspace_dir_missing", ".moyuan directory is missing", paths.MoyuanDir)
		return report.finish(), nil
	}
	for _, required := range []struct {
		code string
		path string
	}{
		{code: "workspace_state_missing", path: filepath.Join(paths.MoyuanDir, "workspace.json")},
		{code: "project_config_missing", path: paths.ProjectYAML},
		{code: "repository_config_missing", path: paths.RepositoryYAML},
		{code: "access_config_missing", path: paths.AccessYAML},
	} {
		if !fsutil.Exists(required.path) {
			report.add("error", required.code, "required workspace config file is missing", required.path)
		}
	}
	ws, err := Load(paths.RootDir)
	if err != nil {
		report.add("error", "workspace_state_unreadable", err.Error(), filepath.Join(paths.MoyuanDir, "workspace.json"))
		return report.finish(), nil
	}
	validateProject(&report, ws.Project)
	validateRepository(&report, ws.Repository)
	validateAccess(&report, ws.Access)
	return report.finish(), nil
}

func SaveProject(rootDir string, cfg ProjectConfig) error {
	paths := ForRoot(rootDir)
	if err := updateState(paths, func(state *StateFile) { state.Project = cfg }); err != nil {
		return err
	}
	return fsutil.WriteText(paths.ProjectYAML, renderProjectYAML(cfg))
}

func SaveRepository(rootDir string, cfg RepositoryConfig) error {
	paths := ForRoot(rootDir)
	if err := updateState(paths, func(state *StateFile) { state.Repository = cfg }); err != nil {
		return err
	}
	return fsutil.WriteText(paths.RepositoryYAML, renderRepositoryYAML(cfg))
}

func SaveAccess(rootDir string, cfg AccessConfig) error {
	paths := ForRoot(rootDir)
	if err := updateState(paths, func(state *StateFile) { state.Access = cfg }); err != nil {
		return err
	}
	return fsutil.WriteText(paths.AccessYAML, renderAccessYAML(cfg))
}

func updateState(paths Paths, mutate func(*StateFile)) error {
	state := StateFile{}
	_, _ = fsutil.ReadJSON(filepath.Join(paths.MoyuanDir, "workspace.json"), &state)
	if state.RootDir == "" {
		state.RootDir = paths.RootDir
		state.CreatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		state.Phase = "phase1"
		state.Runtime = "go-control-plane"
		state.SchemaVer = 1
	}
	mutate(&state)
	return fsutil.WriteJSON(filepath.Join(paths.MoyuanDir, "workspace.json"), state)
}

func validateProject(report *ValidationReport, cfg ProjectConfig) {
	if cfg.SchemaVersion != 1 {
		report.add("error", "project_schema_version_invalid", "project schema_version must be 1", ".moyuan/project.yaml")
	}
	if strings.TrimSpace(cfg.Project.ID) == "" {
		report.add("error", "project_id_required", "project.id is required", "project.id")
	}
	if strings.TrimSpace(cfg.Project.Name) == "" {
		report.add("error", "project_name_required", "project.name is required", "project.name")
	}
	if strings.TrimSpace(cfg.Project.Root) == "" {
		report.add("error", "project_root_required", "project.root is required", "project.root")
	}
	if strings.TrimSpace(cfg.Project.Type) == "" {
		report.add("error", "project_type_required", "project.type is required", "project.type")
	}
	if len(cfg.Workspace.ProtectedPaths) == 0 {
		report.add("warning", "protected_paths_empty", "workspace.protected_paths should protect secrets and policy files", "workspace.protected_paths")
	}
	if len(cfg.Workspace.WritablePaths) == 0 {
		report.add("warning", "writable_paths_empty", "workspace.writable_paths should constrain agent edits", "workspace.writable_paths")
	}
}

func validateRepository(report *ValidationReport, cfg RepositoryConfig) {
	if cfg.SchemaVersion != 1 {
		report.add("error", "repository_schema_version_invalid", "repository schema_version must be 1", ".moyuan/repository.yaml")
	}
	sourceType := strings.TrimSpace(cfg.Repository.Source.Type)
	if sourceType == "" {
		report.add("error", "repository_source_type_required", "repository.source.type is required", "repository.source.type")
	}
	if cfg.Repository.Source.Provider == "" {
		report.add("error", "repository_provider_required", "repository.source.provider is required", "repository.source.provider")
	}
	switch sourceType {
	case "local_path":
		if strings.TrimSpace(cfg.Repository.Source.LocalPath) == "" {
			report.add("error", "repository_local_path_required", "repository.source.local_path is required for local_path source", "repository.source.local_path")
		}
	case "remote_git":
		if cfg.Repository.Source.URL == nil || strings.TrimSpace(*cfg.Repository.Source.URL) == "" {
			report.add("error", "repository_url_required", "repository.source.url is required for remote_git source", "repository.source.url")
		}
	case "":
	default:
		report.add("warning", "repository_source_type_unknown", "repository.source.type is not a known source type", "repository.source.type")
	}
	if strings.TrimSpace(cfg.Repository.DefaultRemote) == "" {
		report.add("warning", "default_remote_empty", "repository.default_remote is empty; git sync may need explicit remote", "repository.default_remote")
	}
	if cfg.Git.BranchPolicy == nil {
		report.add("error", "branch_policy_required", "git.branch_policy is required", "git.branch_policy")
	}
	if cfg.Git.CommitPolicy == nil {
		report.add("error", "commit_policy_required", "git.commit_policy is required", "git.commit_policy")
	}
}

func validateAccess(report *ValidationReport, cfg AccessConfig) {
	if cfg.SchemaVersion != 1 {
		report.add("error", "access_schema_version_invalid", "access schema_version must be 1", ".moyuan/policies/access.yaml")
	}
	if strings.TrimSpace(cfg.Access.Mode) == "" {
		report.add("error", "access_mode_required", "access.mode is required", "access.mode")
	}
	if cfg.Access.ProjectRoles == nil || len(cfg.Access.ProjectRoles) == 0 {
		report.add("error", "project_roles_required", "access.project_roles must define at least one role", "access.project_roles")
	}
}

func (report *ValidationReport) add(severity string, code string, message string, path string) {
	report.Issues = append(report.Issues, ValidationIssue{Severity: severity, Code: code, Message: message, Path: path})
}

func (report ValidationReport) finish() ValidationReport {
	hasError := false
	hasWarning := false
	for _, issue := range report.Issues {
		if issue.Severity == "error" {
			hasError = true
		}
		if issue.Severity == "warning" {
			hasWarning = true
		}
	}
	switch {
	case hasError:
		report.Status = "failed"
	case hasWarning:
		report.Status = "warning"
	default:
		report.Status = "passed"
	}
	return report
}

func renderProjectYAML(cfg ProjectConfig) string {
	return strings.Join([]string{
		"schema_version: 1",
		"project:",
		"  id: " + quote(cfg.Project.ID),
		"  name: " + quote(cfg.Project.Name),
		"  root: " + quote(cfg.Project.Root),
		"  type: " + quote(cfg.Project.Type),
		"  description: null",
		"stack:",
		"  languages:",
		renderList(cfg.Stack.Languages, 4),
		"  frameworks:",
		renderList(cfg.Stack.Frameworks, 4),
		"  package_managers:",
		renderList(cfg.Stack.PackageManagers, 4),
		"  build_commands:",
		renderList(cfg.Stack.BuildCommands, 4),
		"  test_commands:",
		renderList(cfg.Stack.TestCommands, 4),
		"  lint_commands:",
		renderList(cfg.Stack.LintCommands, 4),
		"workspace:",
		"  protected_paths:",
		renderList(cfg.Workspace.ProtectedPaths, 4),
		"  writable_paths:",
		renderList(cfg.Workspace.WritablePaths, 4),
		"",
	}, "\n")
}

func renderRepositoryYAML(cfg RepositoryConfig) string {
	url := "null"
	if cfg.Repository.Source.URL != nil {
		url = quote(*cfg.Repository.Source.URL)
	}
	clonePath := "null"
	if cfg.Repository.Source.ClonePath != nil {
		clonePath = quote(*cfg.Repository.Source.ClonePath)
	}
	defaultBranch := "null"
	if cfg.Repository.DefaultBranch != nil {
		defaultBranch = quote(*cfg.Repository.DefaultBranch)
	}
	return strings.Join([]string{
		"schema_version: 1",
		"repository:",
		"  source:",
		"    type: " + quote(cfg.Repository.Source.Type),
		"    provider: " + quote(cfg.Repository.Source.Provider),
		"    local_path: " + quote(cfg.Repository.Source.LocalPath),
		"    url: " + url,
		"    clone_path: " + clonePath,
		"  default_remote: " + quote(cfg.Repository.DefaultRemote),
		"  default_branch: " + defaultBranch,
		"git:",
		"  branch_policy:",
		"    mode: " + quote(fmt.Sprint(cfg.Git.BranchPolicy["mode"])),
		"    naming: " + quote(fmt.Sprint(cfg.Git.BranchPolicy["naming"])),
		"    require_clean_worktree: true",
		"    allow_auto_commit: false",
		"    allow_auto_push: false",
		"  commit_policy:",
		"    enabled: true",
		"    format: conventional_commits",
		"    require_issue_ref: true",
		"    require_quality_ref: true",
		"",
	}, "\n")
}

func renderAccessYAML(cfg AccessConfig) string {
	owner := "null"
	if cfg.Access.LocalOwnerID != nil {
		owner = quote(*cfg.Access.LocalOwnerID)
	}
	return strings.Join([]string{
		"schema_version: 1",
		"access:",
		"  mode: " + quote(cfg.Access.Mode),
		"  local_owner_id: " + owner,
		"  organization_id: null",
		"  project_roles:",
		"    owner:",
		"      - \"*\"",
		"  approval_policy: {}",
		"  audit:",
		"    enabled: true",
		"",
	}, "\n")
}

func renderList(items []string, indent int) string {
	prefix := strings.Repeat(" ", indent)
	if len(items) == 0 {
		return prefix + "[]"
	}
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, prefix+"- "+quote(item))
	}
	return strings.Join(lines, "\n")
}

func quote(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	return "\"" + value + "\""
}
