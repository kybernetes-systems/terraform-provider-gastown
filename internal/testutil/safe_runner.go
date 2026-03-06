// Package testutil provides test helpers for the Terraform provider.
// See ADR 0011: Tests Must Not Spawn Polecats.
package testutil

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	tfexec "github.com/kybernetes-systems/terraform-provider-gastown/internal/exec"
)

// SafeRunner wraps exec.Runner for use in tests.
// It panics if any command that could spawn polecats is called.
// See ADR 0011.
type SafeRunner struct {
	inner tfexec.Runner
}

// prohibitedCommands lists command prefixes that are not allowed in tests.
// These commands can spawn live AI agent processes (polecats).
var prohibitedCommands = []string{
	"rig start",
	"crew run",
	"convoy",
}

// NewSafeRunner wraps a runner with polecat-spawn protection.
func NewSafeRunner(inner tfexec.Runner) *SafeRunner {
	return &SafeRunner{inner: inner}
}

// GT executes a gt command, panicking if the command is prohibited.
func (r *SafeRunner) GT(ctx context.Context, args ...string) (string, error) {
	cmd := strings.Join(args, " ")
	for _, prohibited := range prohibitedCommands {
		if strings.HasPrefix(cmd, prohibited) {
			panic(fmt.Sprintf(
				"testutil.SafeRunner: prohibited command %q — tests must not spawn polecats (ADR 0011)",
				cmd,
			))
		}
	}
	return r.inner.GT(ctx, args...)
}

// BD executes a bd command.
func (r *SafeRunner) BD(ctx context.Context, args ...string) (string, error) {
	return r.inner.BD(ctx, args...)
}

// AssertNoPolecat verifies that a rig has no running polecats.
// If polecats are found, it attempts to park the rig and fails the test.
// This should be called in a t.Cleanup immediately after rig creation.
// See ADR 0011.
func AssertNoPolecat(t testing.TB, runner tfexec.Runner, rigName string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check polecat count via rig status
	output, err := runner.GT(ctx, "rig", "status", rigName, "--json")
	if err != nil {
		// Rig may already be removed, which is fine
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no such") {
			return
		}
		t.Logf("AssertNoPolecat: could not get rig status: %v", err)
		return
	}

	// Parse polecat count from JSON output
	// Expected format: {"polecats": 0, ...} or similar
	count := parsePolecatCount(output)
	if count == 0 {
		return // No polecats, all good
	}

	// Found polecats - attempt to park the rig
	t.Logf("AssertNoPolecat: found %d polecats for rig %q, attempting to park", count, rigName)
	_, _ = runner.GT(ctx, "rig", "park", rigName)

	// Wait a moment for polecats to stop
	time.Sleep(2 * time.Second)

	// Check again
	output, err = runner.GT(ctx, "rig", "status", rigName, "--json")
	if err == nil {
		count = parsePolecatCount(output)
		if count > 0 {
			t.Errorf("AssertNoPolecat: rig %q has %d running polecats after test cleanup - possible orphaned processes (ADR 0011)", rigName, count)
		}
	}
}

// parsePolecatCount extracts the polecat count from gt rig status --json output.
// This is a simple parser that looks for "polecats": N in the JSON.
func parsePolecatCount(output string) int {
	// Look for "polecats": <number> pattern
	const prefix = `"polecats":`
	idx := strings.Index(output, prefix)
	if idx == -1 {
		// Try alternative format: polecats:
		idx = strings.Index(output, `"polecats" :`)
		if idx == -1 {
			return 0
		}
		idx = idx + len(`"polecats" :`)
	} else {
		idx = idx + len(prefix)
	}

	// Find the number after the colon
	rest := strings.TrimSpace(output[idx:])
	// Handle both "polecats": 0 and "polecats":0 formats
	rest = strings.TrimPrefix(rest, ":")
	rest = strings.TrimSpace(rest)

	// Find the end of the number
	end := 0
	for i, c := range rest {
		if c < '0' || c > '9' {
			end = i
			break
		}
	}
	if end == 0 && len(rest) > 0 {
		end = len(rest)
	}

	if end > 0 {
		count, err := strconv.Atoi(rest[:end])
		if err == nil {
			return count
		}
	}

	return 0
}
