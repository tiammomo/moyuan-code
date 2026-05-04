package git

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"moyuan-code/internal/logging"
	"moyuan-code/internal/process"
	"moyuan-code/internal/workspace"
)

type Status struct {
	IsRepo bool     `json:"isRepo"`
	Dirty  bool     `json:"dirty"`
	Branch *string  `json:"branch"`
	Remote *string  `json:"remote"`
	Ahead  *int     `json:"ahead"`
	Behind *int     `json:"behind"`
	Files  []string `json:"files"`
}

func IsRepo(ctx context.Context, rootDir string) bool {
	res := process.RunCommand(ctx, rootDir, "git", "rev-parse", "--is-inside-work-tree")
	return res.Code == 0 && strings.TrimSpace(res.Stdout) == "true"
}

func StatusOf(ctx context.Context, rootDir string) Status {
	if !IsRepo(ctx, rootDir) {
		return Status{IsRepo: false, Dirty: false, Files: []string{}}
	}
	branch := strings.TrimSpace(process.RunCommand(ctx, rootDir, "git", "branch", "--show-current").Stdout)
	remoteRes := process.RunCommand(ctx, rootDir, "git", "remote", "get-url", "origin")
	status := process.RunCommand(ctx, rootDir, "git", "status", "--short")
	files := []string{}
	for _, line := range strings.Split(status.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	var branchPtr *string
	if branch != "" {
		branchPtr = &branch
	}
	var remotePtr *string
	if remoteRes.Code == 0 {
		remote := strings.TrimSpace(remoteRes.Stdout)
		if remote != "" {
			remotePtr = &remote
		}
	}
	return Status{IsRepo: true, Dirty: len(files) > 0, Branch: branchPtr, Remote: remotePtr, Files: files}
}

func Branches(ctx context.Context, rootDir string) []string {
	if !IsRepo(ctx, rootDir) {
		return []string{}
	}
	res := process.RunCommand(ctx, rootDir, "git", "branch", "--format", "%(refname:short)")
	if res.Code != 0 {
		return []string{}
	}
	branches := []string{}
	for _, line := range strings.Split(res.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			branches = append(branches, line)
		}
	}
	return branches
}

func Clone(ctx context.Context, url string, destDir string) error {
	res := process.RunCommand(ctx, "", "git", "clone", url, destDir)
	if res.Code != 0 {
		if strings.TrimSpace(res.Stderr) != "" {
			return errors.New(strings.TrimSpace(res.Stderr))
		}
		return errors.New("git clone failed")
	}
	return nil
}

func BindLocal(rootDir string) error {
	ws, err := workspace.Load(rootDir)
	if err != nil {
		return err
	}
	ws.Repository.Repository.Source.Type = "local_path"
	ws.Repository.Repository.Source.Provider = "local"
	ws.Repository.Repository.Source.LocalPath = rootDir
	ws.Repository.Repository.Source.URL = nil
	ws.Repository.Repository.Source.ClonePath = nil
	if err := workspace.SaveRepository(rootDir, ws.Repository); err != nil {
		return err
	}
	return logging.Log(rootDir, "git", "repository.bound.local", map[string]any{"rootDir": rootDir})
}

func BindRemote(rootDir string, url string, provider string) error {
	ws, err := workspace.Load(rootDir)
	if err != nil {
		return err
	}
	if provider == "" {
		provider = "generic_git"
	}
	ws.Repository.Repository.Source.Type = "remote_git"
	ws.Repository.Repository.Source.Provider = provider
	ws.Repository.Repository.Source.URL = &url
	ws.Repository.Repository.Source.LocalPath = ""
	ws.Repository.Repository.Source.ClonePath = nil
	if err := workspace.SaveRepository(rootDir, ws.Repository); err != nil {
		return err
	}
	return logging.Log(rootDir, "git", "repository.bound.remote", map[string]any{"url": url, "provider": provider})
}

func Sync(ctx context.Context, rootDir string) (map[string]any, error) {
	if !IsRepo(ctx, rootDir) {
		return nil, errors.New("not a git repository: " + rootDir)
	}
	fetch := process.RunCommand(ctx, rootDir, "git", "fetch", "--all", "--prune")
	if fetch.Code != 0 {
		return nil, errors.New(strings.TrimSpace(fetch.Stderr))
	}
	status := StatusOf(ctx, rootDir)
	return map[string]any{
		"branch": status.Branch,
		"remote": status.Remote,
	}, nil
}

func DefaultRemoteProjectDir(rootDir string, remote string) string {
	name := strings.TrimSuffix(filepath.Base(remote), ".git")
	if name == "" || name == "." || name == "/" {
		name = "remote-project"
	}
	return filepath.Join(rootDir, ".moyuan", "projects", name)
}
