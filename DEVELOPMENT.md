# Developer Guide: Terraform Provider Gas Town

This document describes the internal architecture, testing strategy, and operational safety for maintainers of the `terraform-provider-gastown`.

## Architecture Overview

The provider acts as a declarative wrapper around the Gas Town (`gt`) and Beads (`bd`) CLI tools. It follows a layered approach:

### 1. The Provider Layer (`internal/provider/`)
Initializes the provider and configures the `Runner`. It supports dependency injection for testing via `NewForTesting`.

### 2. The Resource Layer (`internal/gastown/`)
Maps Terraform lifecycle methods to Gas Town CLI commands:
- **HQ**: `Create` installs the town root. `Update` is blocked (HQs are immutable).
- **Rig**: `Create` adds a rig and sets initial configuration (`runtime`, `max_polecats`).
- **Crew**: `Create` adds individual agent workspaces within a rig.

### 3. The Execution Layer (`internal/exec/`)
The **Runner** interface handles all process execution.
- **Process Group Isolation**: Each command started via `NewRunnerWithSetpgid` runs in its own process group. This is critical for reliable cleanup of daemonized processes (like `dolt sql-server`).
- **Typed Errors**: Uses `NotFoundError` to distinguish between missing resources (state removal) and actual execution failures.

## Testing Strategy & Trust

We use a "Trust but Verify" approach to ensure the provider behaves safely without requiring manual source code audits for every change.

### SafeRunner & FakeRunner
Governed by **ADR 0011**, tests are strictly prohibited from spawning live AI agents (polecats).
- **`SafeRunner`**: A wrapper that **panics** if a test attempts to run a "start" or "run" command.
- **`FakeRunner`**: A pure mock that simulates CLI behavior and creates minimal filesystem artifacts (like `town.json`) to satisfy resource `Read` methods.

### Acceptance Tests
Acceptance tests use `FakeRunner` by default. They verify the Terraform lifecycle (Create → Plan → Update → Destroy) without requiring a real `gt` installation.

### Unit Tests
Unit tests use `NewRunnerWithSetpgid` when real CLI calls are needed (e.g., testing `gt install`). They rely on `testutil.CleanupTestHQ` to "nuke" the entire process tree on completion.

## Operational Safety

### 1. Process Group Termination
If a test-spawned daemon detaches, we kill the entire process group:
```go
syscall.Kill(-pid, syscall.SIGKILL) // Note the negative PID
```

### 2. Signal Trapping
The `Makefile` provides a `testacc` target that traps `SIGINT/SIGTERM`. If you interrupt a test run with `Ctrl+C`, the Makefile will attempt to clean up any orphaned `dolt` servers or `beads` monitors automatically.

### 3. Input Validation
All user input is passed through `internal/validators/`. We strictly validate paths, repository URLs, and names to prevent shell injection and path traversal.

## Documentation Reference
- **ADR 0003**: Why deleting a rig means `dock` (parking history in the ground).
- **ADR 0011**: Why tests must never spawn polecats.
- **AGENTS.md**: Context for AI assistants working on this repo.
