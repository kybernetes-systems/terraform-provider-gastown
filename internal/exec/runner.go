package exec

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Runner executes gt and bd CLI commands.
type Runner interface {
	GT(ctx context.Context, args ...string) (string, error)
	BD(ctx context.Context, args ...string) (string, error)
	HQPath() string
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

func (r *runner) HQPath() string {
	return r.hqPath
}

func run(ctx context.Context, bin, hqPath string, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	if hqPath != "" {
		cleanedHqPath := filepath.Clean(hqPath)
		if _, err := os.Stat(cleanedHqPath); err == nil {
			cmd.Dir = cleanedHqPath
		}
		cmd.Env = append(cmd.Environ(), "GT_TOWN_ROOT="+cleanedHqPath)
	}
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	if err := cmd.Run(); err != nil {
		output := combined.String()
		// Check for "not found" errors and return typed error for robust handling
		if isNotFoundError(output) {
			return "", &NotFoundError{
				Resource: extractResourceType(args),
				Name:     extractResourceName(args),
			}
		}
		return "", fmt.Errorf("%s %v: %w\n%s", bin, args, err, output)
	}
	return combined.String(), nil
}

// isNotFoundError checks if the error output indicates a "not found" condition.
// This centralizes the brittle string matching in one place.
func isNotFoundError(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "not found") ||
		strings.Contains(lower, "no such") ||
		strings.Contains(lower, "does not exist")
}

// extractResourceType attempts to determine the resource type from command arguments.
func extractResourceType(args []string) string {
	if len(args) >= 1 {
		return args[0]
	}
	return "resource"
}

// extractResourceName attempts to extract the resource name from command arguments.
func extractResourceName(args []string) string {
	// Common patterns: "rig status <name>", "crew list <rig>"
	if len(args) >= 3 {
		return args[2]
	}
	if len(args) >= 2 {
		return args[1]
	}
	return ""
}
