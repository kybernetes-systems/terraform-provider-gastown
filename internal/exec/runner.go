package exec

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
)

// Runner executes gt and bd CLI commands.
type Runner interface {
	GT(ctx context.Context, args ...string) (string, error)
	BD(ctx context.Context, args ...string) (string, error)
}

type runner struct {
	hqPath string
}

// NewRunner returns a Runner that executes gt and bd with the given HQ path
// set as GT_TOWN_ROOT in the environment.
func NewRunner(hqPath string) Runner {
	return &runner{hqPath: hqPath}
}

func (r *runner) GT(ctx context.Context, args ...string) (string, error) {
	return run(ctx, "gt", r.hqPath, args)
}

func (r *runner) BD(ctx context.Context, args ...string) (string, error) {
	return run(ctx, "bd", r.hqPath, args)
}

func run(ctx context.Context, bin, hqPath string, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	if hqPath != "" {
		if _, err := os.Stat(hqPath); err == nil {
			cmd.Dir = hqPath
		}
		cmd.Env = append(cmd.Environ(), "GT_TOWN_ROOT="+hqPath)
	}
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s %v: %w\n%s", bin, args, err, combined.String())
	}
	return combined.String(), nil
}
