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

// findTestDaemons finds all processes running from or associated with test HQ directories.
// It checks CWD, command-line arguments, and environment variables.
func findTestDaemons(hqPath string) []daemonProcess {
	var daemons []daemonProcess

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return daemons
	}

	selfPid := strconv.Itoa(os.Getpid())
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid := entry.Name()
		if _, err := strconv.Atoi(pid); err != nil {
			continue
		}

		if pid == selfPid {
			continue
		}

		// Read the command line
		cmdlineBytes, err := os.ReadFile(filepath.Join("/proc", pid, "cmdline"))
		if err != nil {
			continue
		}
		cmdline := string(cmdlineBytes)

		cwd := ""
		cwdValid := true
		// Try to read the CWD symlink
		cwd, err = os.Readlink(filepath.Join("/proc", pid, "cwd"))
		if err != nil {
			cwdValid = false
		}

		isTestDaemon := false
		reason := ""

		// 1. Check if CWD is under hqPath
		if cwdValid && strings.HasPrefix(cwd, hqPath) {
			isTestDaemon = true
			reason = "cwd"
		}

		// 2. Check if hqPath appears in the command line
		if !isTestDaemon && strings.Contains(cmdline, hqPath) {
			isTestDaemon = true
			reason = "cmdline"
		}

		// 3. Check environment variables
		if !isTestDaemon {
			environBytes, err := os.ReadFile(filepath.Join("/proc", pid, "environ"))
			if err == nil {
				environ := string(environBytes)
				for _, env := range strings.Split(environ, "\x00") {
					if strings.HasPrefix(env, "GT_HQ=") || strings.HasPrefix(env, "GT_TOWN_ROOT=") || strings.HasPrefix(env, "GT_WORKSPACE=") {
						val := ""
						if strings.HasPrefix(env, "GT_HQ=") {
							val = strings.TrimPrefix(env, "GT_HQ=")
						} else if strings.HasPrefix(env, "GT_TOWN_ROOT=") {
							val = strings.TrimPrefix(env, "GT_TOWN_ROOT=")
						} else {
							val = strings.TrimPrefix(env, "GT_WORKSPACE=")
						}

						if val == hqPath || strings.HasPrefix(val, hqPath+string(os.PathSeparator)) {
							isTestDaemon = true
							reason = "env"
							break
						}
					}
				}
			}
		}

		// 4. Check open file descriptors (ultimate fallback for lingering Dolt/database processes)
		if !isTestDaemon {
			fdDir := filepath.Join("/proc", pid, "fd")
			fds, err := os.ReadDir(fdDir)
			if err == nil {
				for _, fd := range fds {
					target, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
					if err == nil && strings.HasPrefix(target, hqPath) {
						isTestDaemon = true
						reason = "fd"
						break
					}
				}
			}
		}

		if !isTestDaemon {
			continue
		}

		// Determine role for logging
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
		} else if strings.Contains(cmdline, "beads") {
			role = "beads"
		} else if strings.Contains(cmdline, "tmux") {
			role = "tmux"
		} else if strings.Contains(cmdline, "claude") {
			role = "claude"
		}

		if !cwdValid {
			cwd = "(deleted working directory)"
		}

		daemons = append(daemons, daemonProcess{pid: pid, role: fmt.Sprintf("%s(%s)", role, reason), cwd: cwd})
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

	selfPid := os.Getpid()
	selfPgid, _ := syscall.Getpgid(selfPid)

	// Try to get the process group ID
	pgid, err := syscall.Getpgid(pid)
	if err == nil {
		if pgid == selfPgid {
			fmt.Printf("killProcess: skipping group kill for PID %d because it is in our own process group %d\n", pid, pgid)
		} else {
			// Kill the entire process group
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
		}
	}

	if pid == selfPid {
		fmt.Printf("killProcess: skipping self-kill for PID %d\n", pid)
		return nil
	}

	// Also kill the process itself just in case
	err = syscall.Kill(pid, syscall.SIGKILL)
	if err == nil || err == syscall.ESRCH {
		return nil
	}

	// Final attempt with SIGTERM then SIGKILL
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		if err == syscall.ESRCH {
			return nil
		}
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

	// Clean up any tmux sessions first, as they are often the parents
	cleanupTmuxSessions(t, hqPath)

	// Try graceful shutdown with gt down
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := os.Stat(hqPath); err == nil {
		// Use a local exec to avoid runner dependencies
		cmd := exec.CommandContext(ctx, "gt", "down")
		cmd.Dir = hqPath
		cmd.Env = append(os.Environ(), "GT_HQ="+hqPath, "GT_TOWN_ROOT="+hqPath)
		_ = cmd.Run()
	}

	// Find and kill all associated processes
	for i := 0; i < 3; i++ {
		daemons := findTestDaemons(hqPath)
		if len(daemons) == 0 {
			break
		}

		if i == 0 {
			t.Logf("CleanupTestHQ: found %d associated processes to terminate", len(daemons))
		} else {
			t.Logf("CleanupTestHQ: retry %d, %d processes still lingering", i, len(daemons))
		}

		for _, d := range daemons {
			if err := killProcess(d.pid); err != nil && err != syscall.ESRCH {
				t.Logf("CleanupTestHQ: failed to kill %s (PID %s): %v", d.role, d.pid, err)
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Final check
	if final := findTestDaemons(hqPath); len(final) > 0 {
		var summary []string
		for _, d := range final {
			summary = append(summary, fmt.Sprintf("%s:%s", d.role, d.pid))
		}
		t.Errorf("CleanupTestHQ: %d processes still running after cleanup: %v", len(final), summary)
	}

	// Clean up broken symlinks and heartbeats
	cleanupBrokenSymlinks(t, hqPath)
}

// findTestDaemonsByEnv is preserved for compatibility but mostly redundant now.
func findTestDaemonsByEnv(hqPath string) []daemonProcess {
	return findTestDaemons(hqPath)
}

// findTestDaemonsByPidFiles is preserved for compatibility.
func findTestDaemonsByPidFiles(hqPath string) []daemonProcess {
	var daemons []daemonProcess
	pidPatterns := []string{"**/mayor.pid", "**/deacon.pid", "**/witness.pid", "**/*.pid"}
	for _, pattern := range pidPatterns {
		matches, _ := filepath.Glob(filepath.Join(hqPath, pattern))
		for _, pidFile := range matches {
			if data, err := os.ReadFile(pidFile); err == nil {
				pidStr := strings.TrimSpace(string(data))
				if pidStr != "" {
					daemons = append(daemons, daemonProcess{pid: pidStr, role: "pidfile", cwd: pidFile})
				}
			}
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
		return
	}
	_ = filepath.WalkDir(hqPath, func(path string, d os.DirEntry, err error) error {
		if err == nil && d.Type()&os.ModeSymlink != 0 {
			if _, err := os.Lstat(path); err != nil && err == syscall.ENOENT {
				_ = os.Remove(path)
			}
		}
		return nil
	})
	// Clean up stale heartbeats
	matches, _ := filepath.Glob(filepath.Join(hqPath, "**/heartbeat/*"))
	for _, f := range matches {
		if info, err := os.Stat(f); err == nil && time.Since(info.ModTime()) > 5*time.Minute {
			_ = os.Remove(f)
		}
	}
}

// cleanupTmuxSessions removes tmux sessions and servers associated with a test HQ.
func cleanupTmuxSessions(t testing.TB, hqPath string) {
	// 1. Kill sessions on the default server
	killTmuxSessionsOnSocket(t, "", hqPath)

	// 2. Find other tmux sockets in /tmp/tmux-UID/
	uid := os.Getuid()
	tmuxDir := fmt.Sprintf("/tmp/tmux-%d", uid)
	entries, err := os.ReadDir(tmuxDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		socketPath := filepath.Join(tmuxDir, entry.Name())
		// Skip non-socket files
		if info, err := os.Lstat(socketPath); err == nil && info.Mode()&os.ModeSocket != 0 {
			killTmuxSessionsOnSocket(t, socketPath, hqPath)
		}
	}
}

func killTmuxSessionsOnSocket(t testing.TB, socketPath, hqPath string) {
	args := []string{}
	if socketPath != "" {
		args = append(args, "-S", socketPath)
	}
	args = append(args, "ls", "-F", "#{session_name} #{pane_current_path}")

	cmd := exec.Command("tmux", args...)
	output, err := cmd.Output()
	if err != nil {
		return
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		sessionName := fields[0]
		sessionPath := fields[1]

		if strings.HasPrefix(sessionPath, hqPath) {
			t.Logf("CleanupTestHQ: killing tmux session %s on socket %s", sessionName, socketPath)
			killArgs := []string{}
			if socketPath != "" {
				killArgs = append(killArgs, "-S", socketPath)
			}
			killArgs = append(killArgs, "kill-session", "-t", sessionName)
			_ = exec.Command("tmux", killArgs...).Run()
		}
	}

	// If no sessions left on a custom socket, we should try to kill the server
	if socketPath != "" {
		checkCmd := exec.Command("tmux", "-S", socketPath, "ls")
		if err := checkCmd.Run(); err != nil {
			// No sessions left, kill the server
			_ = exec.Command("tmux", "-S", socketPath, "kill-server").Run()
		}
	}
}

