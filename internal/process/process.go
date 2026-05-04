package process

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
)

type Result struct {
	Code   int
	Stdout string
	Stderr string
}

func RunCommand(ctx context.Context, cwd string, command string, args ...string) Result {
	return RunCommandInput(ctx, cwd, "", nil, command, args...)
}

func RunCommandInput(ctx context.Context, cwd string, stdin string, env []string, command string, args ...string) Result {
	cmd := exec.CommandContext(ctx, command, args...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	if env != nil {
		cmd.Env = env
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		code = 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		}
	}
	return Result{Code: code, Stdout: stdout.String(), Stderr: stderr.String()}
}

func RunShell(ctx context.Context, cwd string, command string) Result {
	return RunCommand(ctx, cwd, "sh", "-c", command)
}
