# Mock Runner for Test Cost Control Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a FakeRunner that mocks all `gt` and `bd` CLI invocations so acceptance tests run without spawning real Claude daemon processes, eliminating unnecessary costs.

**Architecture:**
- Create `internal/testutil/fake_runner.go` implementing the `Runner` interface with canned responses
- Modify `internal/provider/provider.go` to support a test mode that injects the FakeRunner
- Update `internal/provider/acceptance_test.go` to use FakeRunner via environment variable
- Resources already support injected runners, so no resource changes needed
- Tests will set environment variable `GASTOWN_USE_FAKE_RUNNER=true` to enable mock mode

**Tech Stack:** Go, Terraform Plugin Framework, testify/mock assertions

---

## Task 1: Create FakeRunner implementation

**Files:**
- Create: `internal/testutil/fake_runner.go`
- Test: `internal/testutil/fake_runner_test.go`

**Step 1: Write failing test for FakeRunner**

Create `internal/testutil/fake_runner_test.go`:

```go
package testutil

import (
	"context"
	"testing"
)

func TestFakeRunner_GTInstallHQ(t *testing.T) {
	runner := NewFakeRunner("/test/hq")

	output, err := runner.GT(context.Background(), "install", "/test/hq", "--git")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output == "" {
		t.Error("expected non-empty output from install command")
	}
}

func TestFakeRunner_HQPath(t *testing.T) {
	expectedPath := "/test/hq"
	runner := NewFakeRunner(expectedPath)

	if runner.HQPath() != expectedPath {
		t.Errorf("expected %q, got %q", expectedPath, runner.HQPath())
	}
}

func TestFakeRunner_GTRigAdd(t *testing.T) {
	runner := NewFakeRunner("/test/hq")

	output, err := runner.GT(context.Background(), "rig", "add", "test-rig", "https://github.com/test/repo.git")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestFakeRunner_GTRigStatus(t *testing.T) {
	runner := NewFakeRunner("/test/hq")

	output, err := runner.GT(context.Background(), "rig", "status", "test-rig")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return valid JSON-like status
	if !contains(output, "polecats") {
		t.Error("expected output to contain polecats count")
	}
}

func TestFakeRunner_GTCrewAdd(t *testing.T) {
	runner := NewFakeRunner("/test/hq")

	output, err := runner.GT(context.Background(), "crew", "add", "--rig", "test-rig", "test-crew", "operator")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestFakeRunner_GTCrewList(t *testing.T) {
	runner := NewFakeRunner("/test/hq")

	output, err := runner.GT(context.Background(), "crew", "list", "--rig", "test-rig")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestFakeRunner_BDStatus(t *testing.T) {
	runner := NewFakeRunner("/test/hq")

	output, err := runner.BD(context.Background(), "status")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestFakeRunner_NotFoundError(t *testing.T) {
	runner := NewFakeRunner("/test/hq")

	_, err := runner.GT(context.Background(), "crew", "remove", "--rig", "nonexistent", "--force", "test-crew")

	if err == nil {
		t.Error("expected NotFoundError for nonexistent crew")
	}
}

func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

**Step 2: Run test to verify failure**

```bash
cd /home/pmocek/sandbox/kybernetes-systems/terraform-provider-gastown
go test ./internal/testutil -run TestFakeRunner -v
```

Expected: FAIL — `NewFakeRunner` not defined

**Step 3: Write FakeRunner implementation**

Create `internal/testutil/fake_runner.go`:

```go
package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	tfexec "github.com/kybernetes-systems/terraform-provider-gastown/internal/exec"
)

// FakeRunner implements Runner interface with canned responses for testing.
// It never invokes real CLI commands, preventing spawning of daemon processes.
type FakeRunner struct {
	hqPath string
	// Track state for consistency across calls
	rigs  map[string]*rigState
	crews map[string]*crewState
}

type rigState struct {
	name       string
	repo       string
	runtime    string
	polecats   int
	exists     bool
}

type crewState struct {
	name  string
	role  string
	rig   string
	exists bool
}

// NewFakeRunner creates a FakeRunner with the given HQ path.
func NewFakeRunner(hqPath string) *FakeRunner {
	return &FakeRunner{
		hqPath: hqPath,
		rigs:   make(map[string]*rigState),
		crews:  make(map[string]*crewState),
	}
}

// GT executes a mocked gt command.
func (r *FakeRunner) GT(ctx context.Context, args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("gt: missing command")
	}

	cmd := args[0]
	switch cmd {
	case "install":
		return r.handleInstall(args[1:])
	case "down":
		return r.handleDown(args[1:])
	case "uninstall":
		return r.handleUninstall(args[1:])
	case "rig":
		return r.handleRig(args[1:])
	case "crew":
		return r.handleCrew(args[1:])
	default:
		return "", fmt.Errorf("gt: unknown command %q", cmd)
	}
}

// BD executes a mocked bd command.
func (r *FakeRunner) BD(ctx context.Context, args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("bd: missing command")
	}

	cmd := args[0]
	switch cmd {
	case "status":
		return `Dolt server running`, nil
	default:
		return "", fmt.Errorf("bd: unknown command %q", cmd)
	}
}

// HQPath returns the HQ path.
func (r *FakeRunner) HQPath() string {
	return r.hqPath
}

// handleInstall mocks: gt install <path> [--git] [--no-beads] [--owner <email>]
func (r *FakeRunner) handleInstall(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("gt install: missing path")
	}
	// Just return success without actual filesystem operations
	return "HQ installed successfully", nil
}

// handleDown mocks: gt down
func (r *FakeRunner) handleDown(args []string) (string, error) {
	return "HQ shut down successfully", nil
}

// handleUninstall mocks: gt uninstall [--force]
func (r *FakeRunner) handleUninstall(args []string) (string, error) {
	return "HQ uninstalled successfully", nil
}

// handleRig mocks rig subcommands
func (r *FakeRunner) handleRig(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("gt rig: missing subcommand")
	}

	subcmd := args[0]
	switch subcmd {
	case "add":
		return r.rigAdd(args[1:])
	case "config":
		return r.rigConfig(args[1:])
	case "status":
		return r.rigStatus(args[1:])
	case "stop":
		return r.rigStop(args[1:])
	case "dock":
		return r.rigDock(args[1:])
	default:
		return "", fmt.Errorf("gt rig: unknown subcommand %q", subcmd)
	}
}

// rigAdd mocks: gt rig add <name> <repo>
func (r *FakeRunner) rigAdd(args []string) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("gt rig add: missing name or repo")
	}
	name := args[0]
	repo := args[1]

	r.rigs[name] = &rigState{
		name:    name,
		repo:    repo,
		runtime: "claude", // default
		polecats: 0,
		exists:  true,
	}

	return fmt.Sprintf("Rig %q created successfully", name), nil
}

// rigConfig mocks: gt rig config set <name> <key> <value>
func (r *FakeRunner) rigConfig(args []string) (string, error) {
	if len(args) < 4 {
		return "", fmt.Errorf("gt rig config: invalid arguments")
	}

	if args[0] != "set" {
		return "", fmt.Errorf("gt rig config: unknown subcommand %q", args[0])
	}

	name := args[1]
	key := args[2]
	value := args[3]

	rig, ok := r.rigs[name]
	if !ok {
		return "", fmt.Errorf("rig %q not found", name)
	}

	switch key {
	case "runtime":
		rig.runtime = value
	case "max_polecats":
		// Parse value as int (simplified)
		if value == "0" {
			rig.polecats = 0
		}
	default:
		return "", fmt.Errorf("gt rig config: unknown key %q", key)
	}

	return fmt.Sprintf("Config updated for rig %q", name), nil
}

// rigStatus mocks: gt rig status <name> [--json]
func (r *FakeRunner) rigStatus(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("gt rig status: missing name")
	}

	name := args[0]
	rig, ok := r.rigs[name]
	if !ok {
		return "", &tfexec.NotFoundError{Resource: "rig", Name: name}
	}

	// Return JSON status response
	status := map[string]interface{}{
		"name":        rig.name,
		"repo":        rig.repo,
		"runtime":     rig.runtime,
		"polecats":    rig.polecats,
		"max_polecats": 0,
	}

	data, _ := json.Marshal(status)
	return string(data), nil
}

// rigStop mocks: gt rig stop <name>
func (r *FakeRunner) rigStop(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("gt rig stop: missing name")
	}

	name := args[0]
	if _, ok := r.rigs[name]; !ok {
		return "", &tfexec.NotFoundError{Resource: "rig", Name: name}
	}

	return fmt.Sprintf("Rig %q stopped", name), nil
}

// rigDock mocks: gt rig dock <name>
func (r *FakeRunner) rigDock(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("gt rig dock: missing name")
	}

	name := args[0]
	if _, ok := r.rigs[name]; !ok {
		return "", &tfexec.NotFoundError{Resource: "rig", Name: name}
	}

	// Remove from state
	delete(r.rigs, name)

	return fmt.Sprintf("Rig %q docked", name), nil
}

// handleCrew mocks crew subcommands
func (r *FakeRunner) handleCrew(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("gt crew: missing subcommand")
	}

	subcmd := args[0]
	switch subcmd {
	case "add":
		return r.crewAdd(args[1:])
	case "list":
		return r.crewList(args[1:])
	case "remove":
		return r.crewRemove(args[1:])
	default:
		return "", fmt.Errorf("gt crew: unknown subcommand %q", subcmd)
	}
}

// crewAdd mocks: gt crew add --rig <rig> <name> <role>
func (r *FakeRunner) crewAdd(args []string) (string, error) {
	if len(args) < 4 {
		return "", fmt.Errorf("gt crew add: invalid arguments")
	}

	if args[0] != "--rig" {
		return "", fmt.Errorf("gt crew add: expected --rig flag")
	}

	rigName := args[1]
	crewName := args[2]
	role := args[3]

	if _, ok := r.rigs[rigName]; !ok {
		return "", fmt.Errorf("rig %q not found", rigName)
	}

	key := fmt.Sprintf("%s/%s", rigName, crewName)
	r.crews[key] = &crewState{
		name:   crewName,
		role:   role,
		rig:    rigName,
		exists: true,
	}

	return fmt.Sprintf("Crew %q added to rig %q", crewName, rigName), nil
}

// crewList mocks: gt crew list --rig <rig>
func (r *FakeRunner) crewList(args []string) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("gt crew list: invalid arguments")
	}

	if args[0] != "--rig" {
		return "", fmt.Errorf("gt crew list: expected --rig flag")
	}

	rigName := args[1]

	// Find all crews for this rig
	var crews []map[string]string
	for _, crew := range r.crews {
		if crew.rig == rigName && crew.exists {
			crews = append(crews, map[string]string{
				"name": crew.name,
				"role": crew.role,
			})
		}
	}

	data, _ := json.Marshal(crews)
	return string(data), nil
}

// crewRemove mocks: gt crew remove --rig <rig> --force <name>
func (r *FakeRunner) crewRemove(args []string) (string, error) {
	if len(args) < 4 {
		return "", fmt.Errorf("gt crew remove: invalid arguments")
	}

	if args[0] != "--rig" {
		return "", fmt.Errorf("gt crew remove: expected --rig flag")
	}

	rigName := args[1]

	if args[2] != "--force" {
		return "", fmt.Errorf("gt crew remove: expected --force flag")
	}

	crewName := args[3]
	key := fmt.Sprintf("%s/%s", rigName, crewName)

	if _, ok := r.crews[key]; !ok {
		return "", &tfexec.NotFoundError{Resource: "crew", Name: crewName}
	}

	delete(r.crews, key)

	return fmt.Sprintf("Crew %q removed from rig %q", crewName, rigName), nil
}
```

**Step 4: Run test to verify it passes**

```bash
cd /home/pmocek/sandbox/kybernetes-systems/terraform-provider-gastown
go test ./internal/testutil -run TestFakeRunner -v
```

Expected: PASS — All tests pass

**Step 5: Commit**

```bash
cd /home/pmocek/sandbox/kybernetes-systems/terraform-provider-gastown
git add internal/testutil/fake_runner.go internal/testutil/fake_runner_test.go
git commit -m "test: add FakeRunner for mocking gt/bd CLI commands"
```

---

## Task 2: Add runner factory to provider

**Files:**
- Modify: `internal/provider/provider.go`
- Modify: `internal/provider/provider_test.go`

**Step 1: Write test for runner factory**

Add to `internal/provider/provider_test.go`:

```go
func TestProvider_configure_uses_fake_runner_when_env_set(t *testing.T) {
	// Set environment variable to enable fake runner
	t.Setenv("GASTOWN_USE_FAKE_RUNNER", "true")

	p := newProvider(t)

	var schemaResp provider.SchemaResponse
	p.Schema(context.Background(), provider.SchemaRequest{}, &schemaResp)

	configVal := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{"hq_path": tftypes.String}},
		map[string]tftypes.Value{"hq_path": tftypes.NewValue(tftypes.String, "/test/hq")},
	)
	req := provider.ConfigureRequest{
		Config: tfsdk.Config{Raw: configVal, Schema: schemaResp.Schema},
	}
	var resp provider.ConfigureResponse
	p.Configure(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}

	// Verify that ResourceData contains a FakeRunner
	runner, ok := resp.ResourceData.(*testutil.FakeRunner)
	if !ok {
		t.Errorf("expected *testutil.FakeRunner, got %T", resp.ResourceData)
	}
	if runner == nil {
		t.Error("expected non-nil runner")
	}
}
```

**Step 2: Run test to verify failure**

```bash
cd /home/pmocek/sandbox/kybernetes-systems/terraform-provider-gastown
go test ./internal/provider -run TestProvider_configure_uses_fake_runner_when_env_set -v
```

Expected: FAIL — Environment variable not checked

**Step 3: Modify provider.go to support fake runner**

Update `internal/provider/provider.go` Configure method (around line 52-72):

```go
func (p *GastownProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config struct {
		HQPath types.String `tfsdk:"hq_path"`
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if config.HQPath.IsNull() || config.HQPath.ValueString() == "" {
		resp.Diagnostics.Append(diag.NewAttributeErrorDiagnostic(
			path.Root("hq_path"),
			"Missing HQ path",
			"hq_path must be set to the Gas Town HQ directory.",
		))
		return
	}

	hqPath := config.HQPath.ValueString()
	var runner tfexec.Runner

	// Check if running in test mode with fake runner
	if os.Getenv("GASTOWN_USE_FAKE_RUNNER") == "true" {
		runner = testutil.NewFakeRunner(hqPath)
	} else {
		runner = tfexec.NewRunner(hqPath)
	}

	resp.DataSourceData = runner
	resp.ResourceData = runner
}
```

Add import at top of provider.go:

```go
import (
	"os"
	// ... other imports ...
	"github.com/kybernetes-systems/terraform-provider-gastown/internal/testutil"
)
```

**Step 4: Run test to verify it passes**

```bash
cd /home/pmocek/sandbox/kybernetes-systems/terraform-provider-gastown
go test ./internal/provider -run TestProvider_configure_uses_fake_runner_when_env_set -v
```

Expected: PASS

**Step 5: Verify existing provider tests still pass**

```bash
cd /home/pmocek/sandbox/kybernetes-systems/terraform-provider-gastown
go test ./internal/provider -v
```

Expected: All existing tests still pass

**Step 6: Commit**

```bash
cd /home/pmocek/sandbox/kybernetes-systems/terraform-provider-gastown
git add internal/provider/provider.go internal/provider/provider_test.go
git commit -m "feat: add environment variable to switch provider to FakeRunner for tests"
```

---

## Task 3: Update acceptance tests to use FakeRunner

**Files:**
- Modify: `internal/provider/acceptance_test.go`

**Step 1: Add environment setup to acceptance tests**

Modify all three test functions to set the environment variable. Update `internal/provider/acceptance_test.go`:

Replace `func TestAcc_FullLifecycle(t *testing.T) {` with:

```go
func TestAcc_FullLifecycle(t *testing.T) {
	// Use fake runner to avoid spawning real Claude daemon processes
	t.Setenv("GASTOWN_USE_FAKE_RUNNER", "true")

	hqPath := filepath.Join(t.TempDir(), "gt-lifecycle")
	t.Cleanup(func() { testutil.CleanupTestHQ(t, hqPath) })
	// ... rest of test unchanged
```

Replace `func TestAcc_DriftScenario(t *testing.T) {` with:

```go
func TestAcc_DriftScenario(t *testing.T) {
	// Use fake runner to avoid spawning real Claude daemon processes
	t.Setenv("GASTOWN_USE_FAKE_RUNNER", "true")

	hqPath := filepath.Join(t.TempDir(), "gt-drift")
	t.Cleanup(func() { testutil.CleanupTestHQ(t, hqPath) })
	// ... rest of test unchanged
```

Replace `func TestAcc_Concurrency(t *testing.T) {` with:

```go
func TestAcc_Concurrency(t *testing.T) {
	// Use fake runner to avoid spawning real Claude daemon processes
	t.Setenv("GASTOWN_USE_FAKE_RUNNER", "true")

	t.Parallel()

	hqPath1 := filepath.Join(t.TempDir(), "gt-con-1")
	hqPath2 := filepath.Join(t.TempDir(), "gt-con-2")
	// ... rest of test unchanged
```

**Step 2: Run acceptance tests with fake runner**

```bash
cd /home/pmocek/sandbox/kybernetes-systems/terraform-provider-gastown
TF_ACC=1 go test ./internal/provider -run TestAcc -v
```

Expected: All acceptance tests pass without spawning any real Claude processes

**Step 3: Verify no daemon processes spawned**

After tests complete, verify no Claude or polecat processes running:

```bash
ps aux | grep -i claude | grep -v grep
ps aux | grep -i polecat | grep -v grep
```

Expected: No output (no processes found)

**Step 4: Commit**

```bash
cd /home/pmocek/sandbox/kybernetes-systems/terraform-provider-gastown
git add internal/provider/acceptance_test.go
git commit -m "test: enable FakeRunner in acceptance tests to prevent spawning daemon processes"
```

---

## Task 4: Run full test suite and verify success

**Files:**
- No changes, testing only

**Step 1: Run full unit test suite**

```bash
cd /home/pmocek/sandbox/kybernetes-systems/terraform-provider-gastown
go test ./...
```

Expected: All tests pass

**Step 2: Run acceptance tests**

```bash
cd /home/pmocek/sandbox/kybernetes-systems/terraform-provider-gastown
TF_ACC=1 go test ./internal/provider -run TestAcc -v
```

Expected: Tests complete, no daemon processes spawned

**Step 3: Verify no orphaned processes**

```bash
ps aux | grep -E 'claude|polecat|mayor|deacon' | grep -v grep
```

Expected: No output — all processes cleaned up

**Step 4: Document changes**

Update `docs/adr/0011-no-polecats-in-tests.md` to note that acceptance tests now use FakeRunner:

```markdown
## Implementation Status

### Phase 1: SafeRunner (Completed)
- SafeRunner prevents spawning of specific commands (rig start, crew run, convoy)
- Tests set max_polecats = 0 to prevent polecat workers

### Phase 2: FakeRunner (Completed - 2026-03-06)
- All gt/bd commands are now mocked via FakeRunner
- Acceptance tests use GASTOWN_USE_FAKE_RUNNER=true environment variable
- Tests execute without spawning any real daemon processes
- Zero external dependencies during test execution
```

**Step 5: Final commit**

```bash
cd /home/pmocek/sandbox/kybernetes-systems/terraform-provider-gastown
git add docs/adr/0011-no-polecats-in-tests.md
git commit -m "docs: update ADR 0011 with FakeRunner implementation status"
```

---

## Summary

**Key Outcome:** Acceptance tests no longer spawn real Claude daemon processes

**Files Changed:**
- `internal/testutil/fake_runner.go` (NEW)
- `internal/testutil/fake_runner_test.go` (NEW)
- `internal/provider/provider.go` (MODIFIED)
- `internal/provider/provider_test.go` (MODIFIED)
- `internal/provider/acceptance_test.go` (MODIFIED)
- `docs/adr/0011-no-polecats-in-tests.md` (MODIFIED)

**Total commits:** 5 (one per task)
