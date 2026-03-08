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
	t.Cleanup(cancel)

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
// It checks both CWD and command-line environment variables to find daemons
// even if their working directory has been deleted.
func findTestDaemons(hqPath string) []daemonProcess {
	var daemons []daemonProcess

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

		cwd := ""
		cwdValid := true

		// Try to read the CWD symlink (may fail if directory was deleted)
		cwd, err = os.Readlink(filepath.Join("/proc", pid, "cwd"))
		if err != nil {
			cwdValid = false
		}

		// Read the command line to determine role and environment
		cmdlineBytes, err := os.ReadFile(filepath.Join("/proc", pid, "cmdline"))
		if err != nil {
			continue
		}
		cmdline := string(cmdlineBytes)

		// Check if it's a claude process with Gas Town role
		if !strings.Contains(cmdline, "GAS TOWN") {
			continue
		}

		// Check if this process is related to our test HQ:
		// 1. CWD is under hqPath (normal case)
		// 2. GT_HQ environment variable points to hqPath
		// 3. GT_WORKSPACE environment variable points under hqPath
		isTestDaemon := false
		if cwdValid && strings.HasPrefix(cwd, hqPath) {
			isTestDaemon = true
		} else {
			// Check environment variables
			environBytes, err := os.ReadFile(filepath.Join("/proc", pid, "environ"))
			if err == nil {
				environ := string(environBytes)
				for _, env := range strings.Split(environ, "\x00") {
					if strings.HasPrefix(env, "GT_HQ=") {
						if strings.TrimPrefix(env, "GT_HQ=") == hqPath {
							isTestDaemon = true
						}
					}
					if strings.HasPrefix(env, "GT_WORKSPACE=") {
						if strings.HasPrefix(strings.TrimPrefix(env, "GT_WORKSPACE="), hqPath) {
							isTestDaemon = true
						}
					}
				}
			}
		}

		if !isTestDaemon {
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

		// If cwd is invalid (deleted), mark it as such
		if !cwdValid {
			cwd = "(deleted working directory)"
		}

		daemons = append(daemons, daemonProcess{pid: pid, role: role, cwd: cwd})
	}

	return daemons
}

// killProcess forcefully kills a process by PID.
// It attempts to kill the entire process group first, then falls back to the process itself.
func killProcess(pidStr string) error {
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return err
	}

	// First try: kill process group (negative PID) - this kills all child processes
	// This works if the process was started with setpgid:true (ProcessGroupID set)
	err = syscall.Kill(-pid, syscall.SIGKILL)
	if err == nil {
		return nil
	}

	// Check if error is "no such process" vs other error
	if err != syscall.ESRCH {
		// Process exists but couldn't kill group - try killing process directly
		// (the group may not be valid anymore)
	}

	// Second try: kill just the process
	err = syscall.Kill(pid, syscall.SIGKILL)
	if err == nil {
		return nil
	}

	// If process doesn't exist, that's fine - it was already gone
	if err == syscall.ESRCH {
		return nil
	}

	// Third try: SIGTERM then SIGKILL for stubborn processes
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		if err == syscall.ESRCH {
			return nil
		}
		// Give it a moment
		time.Sleep(100 * time.Millisecond)
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}

	return nil
}

// CleanupTestHQ terminates all Gas Town daemon processes associated with a test HQ.
// This should be called in t.Cleanup after creating an HQ in tests.
// It first attempts graceful shutdown via gt down, then force kills any remaining processes.
func CleanupTestHQ(t testing.TB, hqPath string) {
	t.Helper()

	// First, try graceful shutdown with gt down
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	// Try gt down to gracefully stop all services
	if _, err := os.Stat(hqPath); err == nil {
		cmd := exec.CommandContext(ctx, "gt", "down")
		cmd.Dir = hqPath
		cmd.Env = append(os.Environ(), "GT_HQ="+hqPath)
		_ = cmd.Run()
	}

	// Try to stop deacon gracefully if possible
	deaconDir := filepath.Join(hqPath, "deacon")
	if _, err := os.Stat(deaconDir); err == nil {
		cmd := exec.CommandContext(ctx, "gt", "deacon", "stop")
		cmd.Dir = hqPath
		cmd.Env = append(os.Environ(), "GT_HQ="+hqPath)
		_ = cmd.Run()
	}

	// Give processes a moment to shut down gracefully
	time.Sleep(1 * time.Second)

	// Find all test daemons
	daemons := findTestDaemons(hqPath)
	if len(daemons) == 0 {
		// Even if no daemons found, still try to kill by scanning for processes with matching env
		daemons = findTestDaemonsByEnv(hqPath)
	}

	if len(daemons) == 0 {
		// Still nothing - try by PID files in the HQ directory
		daemons = findTestDaemonsByPidFiles(hqPath)
	}

	if len(daemons) == 0 {
		// No daemons found - just clean up broken symlinks and return
		cleanupBrokenSymlinks(t, hqPath)
		return
	}

	t.Logf("CleanupTestHQ: found %d daemon processes to terminate", len(daemons))

	// Build a set of PIDs to kill (to avoid duplicates)
	pidSet := make(map[string]daemonProcess)
	for _, d := range daemons {
		pidSet[d.pid] = d
	}

	// First pass: SIGTERM to allow graceful shutdown
	for _, d := range pidSet {
		t.Logf("CleanupTestHQ: sending SIGTERM to %s (PID %s) in %s", d.role, d.pid, d.cwd)
		_ = signalProcess(d.pid, syscall.SIGTERM)
	}

	// Give processes time to terminate gracefully
	time.Sleep(500 * time.Millisecond)

	// Second pass: SIGKILL for remaining processes
	for _, d := range pidSet {
		if err := killProcess(d.pid); err != nil {
			t.Logf("CleanupTestHQ: failed to kill %s (PID %s): %v", d.role, d.pid, err)
		} else {
			t.Logf("CleanupTestHQ: killed %s (PID %s)", d.role, d.pid)
		}
	}

	// Wait a moment for processes to die
	time.Sleep(500 * time.Millisecond)

	// Verify cleanup
	remaining := findTestDaemons(hqPath)
	if len(remaining) == 0 {
		remaining = findTestDaemonsByEnv(hqPath)
	}
	if len(remaining) == 0 {
		remaining = findTestDaemonsByPidFiles(hqPath)
	}
	if len(remaining) > 0 {
		// Force kill any remaining
		for _, d := range remaining {
			t.Logf("CleanupTestHQ: force killing remaining %s (PID %s)", d.role, d.pid)
			_ = killProcess(d.pid)
		}
		time.Sleep(250 * time.Millisecond)

		// Final check
		final := findTestDaemons(hqPath)
		if len(final) > 0 {
			roles := make([]string, len(final))
			for i, d := range final {
				roles[i] = fmt.Sprintf("%s:%s", d.role, d.pid)
			}
			t.Errorf("CleanupTestHQ: %d daemon processes still running after cleanup: %v", len(final), roles)
		}
	}

	// Clean up broken symlinks
	cleanupBrokenSymlinks(t, hqPath)

	// Also clean up any tmux sessions associated with this HQ
	cleanupTmuxSessions(t, hqPath)
}

// findTestDaemonsByEnv finds daemon processes by checking environment variables
// when the working directory has been deleted.
func findTestDaemonsByEnv(hqPath string) []daemonProcess {
	var daemons []daemonProcess

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

		// Read the command line
		cmdlineBytes, err := os.ReadFile(filepath.Join("/proc", pid, "cmdline"))
		if err != nil {
			continue
		}
		cmdline := string(cmdlineBytes)

		// Check if it's a claude process with Gas Town
		if !strings.Contains(cmdline, "GAS TOWN") {
			continue
		}

		// Check environment for GT_HQ or GT_WORKSPACE
		environBytes, err := os.ReadFile(filepath.Join("/proc", pid, "environ"))
		if err != nil {
			continue
		}

		environ := string(environBytes)
		isTestDaemon := false
		for _, env := range strings.Split(environ, "\x00") {
			if strings.HasPrefix(env, "GT_HQ=") {
				if strings.TrimPrefix(env, "GT_HQ=") == hqPath {
					isTestDaemon = true
				}
			}
			if strings.HasPrefix(env, "GT_WORKSPACE=") {
				if strings.HasPrefix(strings.TrimPrefix(env, "GT_WORKSPACE="), hqPath) {
					isTestDaemon = true
				}
			}
		}

		if !isTestDaemon {
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

		daemons = append(daemons, daemonProcess{pid: pid, role: role, cwd: "(env match)"})
	}

	return daemons
}

// findTestDaemonsByPidFiles finds daemon processes by looking for PID files in the HQ directory.
func findTestDaemonsByPidFiles(hqPath string) []daemonProcess {
	var daemons []daemonProcess

	// Look for PID files in common locations
	pidPatterns := []string{
		"**/mayor.pid",
		"**/deacon.pid",
		"**/witness.pid",
		"**/*.pid",
	}

	for _, pattern := range pidPatterns {
		matches, err := filepath.Glob(filepath.Join(hqPath, pattern))
		if err != nil {
			continue
		}
		for _, pidFile := range matches {
			pidBytes, err := os.ReadFile(pidFile)
			if err != nil {
				continue
			}
			pidStr := strings.TrimSpace(string(pidBytes))
			if pidStr == "" {
				continue
			}

			// Determine role from filename
			role := "unknown"
			name := filepath.Base(pidFile)
			if strings.Contains(name, "mayor") {
				role = "mayor"
			} else if strings.Contains(name, "deacon") {
				role = "deacon"
			} else if strings.Contains(name, "witness") {
				role = "witness"
			}

			daemons = append(daemons, daemonProcess{pid: pidStr, role: role, cwd: pidFile})
		}
	}

	return daemons
}

// signalProcess sends a signal to a process.
func signalProcess(pidStr string, sig syscall.Signal) error {
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return err
	}
	return syscall.Kill(pid, sig)
}

// cleanupBrokenSymlinks removes broken symlinks in the HQ directory.
func cleanupBrokenSymlinks(t testing.TB, hqPath string) {
	if _, err := os.Stat(hqPath); err != nil {
		return // HQ directory doesn't exist
	}

	// Walk the directory and remove broken symlinks
	filepath.WalkDir(hqPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.Type()&os.ModeSymlink != 0 {
			// Check if symlink is broken
			if _, err := os.Lstat(path); err != nil {
				if err == syscall.ENOENT {
					t.Logf("CleanupTestHQ: removing broken symlink: %s", path)
					_ = os.Remove(path)
				}
			}
		}
		return nil
	})

	// Also clean up stale heartbeat files
	heartbeatPatterns := []string{
		"**/heartbeat/*",
		"**/.heartbeat",
	}

	for _, pattern := range heartbeatPatterns {
		matches, err := filepath.Glob(filepath.Join(hqPath, pattern))
		if err != nil {
			continue
		}
		for _, heartbeatFile := range matches {
			info, err := os.Stat(heartbeatFile)
			if err != nil {
				continue
			}
			// Remove heartbeat files older than 10 minutes
			if time.Since(info.ModTime()) > 10*time.Minute {
				t.Logf("CleanupTestHQ: removing stale heartbeat file: %s", heartbeatFile)
				_ = os.Remove(heartbeatFile)
			}
		}
	}
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
