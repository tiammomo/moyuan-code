package workspace

import (
	"os"
	"path/filepath"

	"moyuan-code/internal/fsutil"
)

const DirName = ".moyuan"

type Paths struct {
	RootDir          string
	MoyuanDir        string
	ProjectYAML      string
	RepositoryYAML   string
	AccessYAML       string
	PermissionsJSON  string
	LoggingJSON      string
	AuthDir          string
	LogsDir          string
	LifecycleDir     string
	EpicsDir         string
	IssuesDir        string
	IssueGraphsDir   string
	SchedulesDir     string
	RunsDir          string
	QualityDir       string
	ReviewsDir       string
	BranchesDir      string
	MergeReportsDir  string
	PullRequestsDir  string
	ReleasesDir      string
	DeploymentsDir   string
	ComprehensionDir string
	MemoryDir        string
	AgentsDir        string
	RuntimesDir      string
	SkillsDir        string
	RuntimeDir       string
	OrchestratorDir  string
	SchedulerDir     string
	RepairDir        string
	ResourcesDir     string
	TmpDir           string
	LocksDir         string
}

func ForRoot(rootDir string) Paths {
	rootDir, _ = filepath.Abs(rootDir)
	moyuanDir := filepath.Join(rootDir, DirName)
	return Paths{
		RootDir:          rootDir,
		MoyuanDir:        moyuanDir,
		ProjectYAML:      filepath.Join(moyuanDir, "project.yaml"),
		RepositoryYAML:   filepath.Join(moyuanDir, "repository.yaml"),
		AccessYAML:       filepath.Join(moyuanDir, "policies", "access.yaml"),
		PermissionsJSON:  filepath.Join(moyuanDir, "policies", "permissions.json"),
		LoggingJSON:      filepath.Join(moyuanDir, "policies", "logging.json"),
		AuthDir:          filepath.Join(moyuanDir, "auth"),
		LogsDir:          filepath.Join(moyuanDir, "logs"),
		LifecycleDir:     filepath.Join(moyuanDir, "lifecycle"),
		EpicsDir:         filepath.Join(moyuanDir, "lifecycle", "epics"),
		IssuesDir:        filepath.Join(moyuanDir, "lifecycle", "issues"),
		IssueGraphsDir:   filepath.Join(moyuanDir, "lifecycle", "issue-graphs"),
		SchedulesDir:     filepath.Join(moyuanDir, "lifecycle", "schedules"),
		RunsDir:          filepath.Join(moyuanDir, "lifecycle", "runs"),
		QualityDir:       filepath.Join(moyuanDir, "lifecycle", "quality"),
		ReviewsDir:       filepath.Join(moyuanDir, "lifecycle", "reviews"),
		BranchesDir:      filepath.Join(moyuanDir, "lifecycle", "branches"),
		MergeReportsDir:  filepath.Join(moyuanDir, "lifecycle", "merge-reports"),
		PullRequestsDir:  filepath.Join(moyuanDir, "lifecycle", "pull-requests"),
		ReleasesDir:      filepath.Join(moyuanDir, "lifecycle", "releases"),
		DeploymentsDir:   filepath.Join(moyuanDir, "lifecycle", "deployments"),
		ComprehensionDir: filepath.Join(moyuanDir, "comprehension"),
		MemoryDir:        filepath.Join(moyuanDir, "memory"),
		AgentsDir:        filepath.Join(moyuanDir, "agents"),
		RuntimesDir:      filepath.Join(moyuanDir, "runtimes"),
		SkillsDir:        filepath.Join(moyuanDir, "skills"),
		RuntimeDir:       filepath.Join(moyuanDir, "runtime"),
		OrchestratorDir:  filepath.Join(moyuanDir, "orchestrator"),
		SchedulerDir:     filepath.Join(moyuanDir, "scheduler"),
		RepairDir:        filepath.Join(moyuanDir, "repair"),
		ResourcesDir:     filepath.Join(moyuanDir, "resources"),
		TmpDir:           filepath.Join(moyuanDir, "tmp"),
		LocksDir:         filepath.Join(moyuanDir, ".locks"),
	}
}

func ResolveRoot(startDir string) (string, bool) {
	current, _ := filepath.Abs(startDir)
	for {
		if fsutil.Exists(filepath.Join(current, DirName)) {
			return current, true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		current = parent
	}
}

func EnsureDirs(paths Paths) error {
	dirs := []string{
		paths.MoyuanDir,
		paths.AuthDir,
		paths.AgentsDir,
		paths.ComprehensionDir,
		paths.EpicsDir,
		paths.IssuesDir,
		paths.IssueGraphsDir,
		paths.SchedulesDir,
		paths.RunsDir,
		paths.QualityDir,
		filepath.Join(paths.QualityDir, "reports"),
		paths.ReviewsDir,
		paths.BranchesDir,
		paths.MergeReportsDir,
		paths.PullRequestsDir,
		paths.ReleasesDir,
		paths.DeploymentsDir,
		paths.LogsDir,
		paths.MemoryDir,
		filepath.Join(paths.MoyuanDir, "models"),
		filepath.Join(paths.MoyuanDir, "policies"),
		paths.RuntimesDir,
		paths.SkillsDir,
		paths.RuntimeDir,
		paths.OrchestratorDir,
		paths.SchedulerDir,
		paths.RepairDir,
		paths.ResourcesDir,
		paths.TmpDir,
		paths.LocksDir,
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}
