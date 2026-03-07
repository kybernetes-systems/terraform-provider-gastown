// Package testutil provides test helpers for the Terraform provider.
// See ADR 0011: Tests Must Not Spawn Polecats.
package testutil

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
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

// HQPath returns the HQ path from the inner runner.
func (r *SafeRunner) HQPath() string {
	return r.inner.HQPath()
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
		if tfexec.IsNotFound(err) {
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

// daemonProcess represents a Gas Town daemon process found during cleanup.
type daemonProcess struct {
	pid  string
	role string // mayor, deacon, boot, witness
	cwd  string
}

// findTestDaemons finds Claude processes running from test directories.
func findTestDaemons(hqPath string) []daemonProcess {
	var daemons []daemonProcess

	// Look for claude processes with CWD under this HQ path
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return daemons
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid := entry.Name()
		if _, err := strconv.Atoi(pid); err != nil {
			continue
		}

		// Read the CWD symlink
		cwd, err := os.Readlink(filepath.Join("/proc", pid, "cwd"))
		if err != nil {
			continue
		}

		// Check if this process is running from our test HQ
		if !strings.HasPrefix(cwd, hqPath) {
			continue
		}

		// Read the command line to determine role
		cmdlineBytes, err := os.ReadFile(filepath.Join("/proc", pid, "cmdline"))
		if err != nil {
			continue
		}
		cmdline := string(cmdlineBytes)

		// Check if it's a claude process with Gas Town role
		if !strings.Contains(cmdline, "GAS TOWN") {
			continue
		}

		// Determine role
		role := "unknown"
		if strings.Contains(cmdline, "mayor") {
			role = "mayor"
		} else if strings.Contains(cmdline, "deacon") {
			role = "deacon"
		} else if strings.Contains(cmdline, "boot") {
			role = "boot"
		} else if strings.Contains(cmdline, "witness") {
			role = "witness"
		} else if strings.Contains(cmdline, "polecat") {
			role = "polecat"
		}

		daemons = append(daemons, daemonProcess{pid: pid, role: role, cwd: cwd})
	}

	return daemons
}

// killProcess forcefully kills a process by PID.
// It also attempts to kill the entire process group if the process is a group leader.
func killProcess(pidStr string) error {
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return err
	}

	// Try to kill the process group first (indicated by negative PID)
	// This only works if the process was started with Setpgid: true.
	err = syscall.Kill(-pid, syscall.SIGKILL)
	if err == nil {
		return nil
	}

	// Fallback to killing just the process
	return syscall.Kill(pid, syscall.SIGKILL)
}

// CleanupTestHQ terminates all Gas Town daemon processes associated with a test HQ.
// This should be called in t.Cleanup after creating an HQ in tests.
// It first attempts graceful shutdown, then force kills any remaining processes.
func CleanupTestHQ(t testing.TB, hqPath string) {
	t.Helper()

	// First, try to find and kill any deacon patrol processes
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to stop deacon gracefully if possible
	deaconDir := filepath.Join(hqPath, "deacon")
	if _, err := os.Stat(deaconDir); err == nil {
		// Try to stop deacon via gt command if available
		cmd := exec.CommandContext(ctx, "gt", "deacon", "stop")
		cmd.Dir = hqPath
		cmd.Env = append(os.Environ(), "GT_HQ="+hqPath)
		_ = cmd.Run() // Ignore errors - process may not exist
	}

	// Give processes a moment to shut down gracefully
	time.Sleep(500 * time.Millisecond)

	// Find all test daemons
	daemons := findTestDaemons(hqPath)
	if len(daemons) == 0 {
		return // Nothing to clean up
	}

	t.Logf("CleanupTestHQ: found %d daemon processes to terminate", len(daemons))

	// Terminate each daemon
	for _, d := range daemons {
		t.Logf("CleanupTestHQ: terminating %s (PID %s) in %s", d.role, d.pid, d.cwd)
		if err := killProcess(d.pid); err != nil {
			t.Logf("CleanupTestHQ: failed to kill %s (PID %s): %v", d.role, d.pid, err)
		}
	}

	// Wait a moment for processes to die
	time.Sleep(500 * time.Millisecond)

	// Verify cleanup
	remaining := findTestDaemons(hqPath)
	if len(remaining) > 0 {
		roles := make([]string, len(remaining))
		for i, d := range remaining {
			roles[i] = fmt.Sprintf("%s:%s", d.role, d.pid)
		}
		t.Errorf("CleanupTestHQ: %d daemon processes still running after cleanup: %v", len(remaining), roles)
	}

	// Also clean up any tmux sessions associated with this HQ
	cleanupTmuxSessions(t, hqPath)
}

// cleanupTmuxSessions removes tmux sessions associated with a test HQ.
func cleanupTmuxSessions(t testing.TB, hqPath string) {
	// Extract the test name from the path (e.g., /tmp/TestAcc_FullLifecycle123/...)
	base := filepath.Base(hqPath)
	if base == "" || base == "." || base == "/" {
		return
	}

	// Try to find and kill tmux sessions with names matching this HQ
	// Session names are typically: hq-deacon, hq-mayor, etc.
	cmd := exec.Command("tmux", "ls")
	output, err := cmd.Output()
	if err != nil {
		return // No tmux server running or no sessions
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) < 1 {
			continue
		}
		sessionName := strings.TrimSpace(parts[0])

		// Check if this session might be related to our test by checking its CWD
		infoCmd := exec.Command("tmux", "display-message", "-t", sessionName, "-p", "#{pane_current_path}")
		infoOutput, err := infoCmd.Output()
		if err != nil {
			continue
		}

		sessionPath := strings.TrimSpace(string(infoOutput))
		if strings.HasPrefix(sessionPath, hqPath) {
			t.Logf("CleanupTestHQ: killing tmux session %s", sessionName)
			killCmd := exec.Command("tmux", "kill-session", "-t", sessionName)
			_ = killCmd.Run()
		}
	}
}
