# Claude Code Handoff: terraform-provider-gastown

## Purpose

A Terraform provider that manages Gas Town workspaces as declarative
infrastructure. The operator describes the desired state of a Gas Town town,
its rigs, crew workspaces, and rig configuration in HCL; the provider calls
the `gt` and `bd` CLIs to reconcile live state.

This is infrastructure-as-code for the Gas Town orchestration stack.

---

## Read Before Writing Any Code

### Prerequisites — Verify These First

Before touching Go code, confirm all of the following are present and working:

```bash
gt version          # must be >= 0.5.0
bd version          # must be >= 0.44.0
git version         # must be >= 2.25 (worktree support required)
go version          # must be >= 1.23
git config user.name   # must be non-empty (required for bd sync commits)
git config user.email  # must be non-empty
```

If any check fails, stop and fix it. Do not proceed. The test strategy
depends on real `gt` and `bd` binaries.

### Skills to Consult

Read these skill files before writing any code:

```
/mnt/skills/user/new-terraform-provider/SKILL.md
/mnt/skills/user/provider-resources/SKILL.md
/mnt/skills/user/terraform-test/SKILL.md
/mnt/skills/user/terraform-style-guide/SKILL.md
```

---

## Decisions Already Made — Do Not Revisit

| Decision | Value |
|---|---|
| Module path | `github.com/kybernetes-systems/terraform-provider-gastown` |
| License | Apache 2.0 |
| SDK | Terraform Plugin Framework (not SDKv2) |
| `gt` integration | Shell out via `os/exec` — do NOT parse internal files directly |
| Delete behavior on `gastown_rig` | `gt rig stop` + `gt rig dock` — operational shutdown, not file removal |
| Git | Managed by `gt`/`bd` — provider never calls `git` directly |
| Commit convention | `feat:` `test:` `fix:` `docs:` `chore:` — one logical change per commit |
| Test command | `go test ./...` |
| Test isolation | Every test gets its own `gt install` HQ in `t.TempDir()` |

---

## Gas Town Concepts You Must Understand

### Workspace Hierarchy

```
~/gt/                          ← HQ (created by: gt install ~/gt --git)
├── CLAUDE.md
├── AGENTS.md
├── mayor/
│   ├── town.json
│   ├── rigs.json
│   └── overseer.json
└── .beads/                    ← Town-level beads DB (prefix: hq-)

~/gt/<rig-name>/               ← Rig (created by: gt rig add <name> <repo>)
├── config.json                ← Runtime config (provider, command, args, env)
├── .repo.git/                 ← Shared bare repo
├── .beads/                    ← Rig-level beads DB (prefix: <2-char>-)
├── plugins/
├── mayor/rig/
├── refinery/rig/
├── crew/
├── witness/
└── polecats/

~/gt/<rig-name>/crew/<name>/   ← Crew workspace (created by: gt crew add <name> --rig <rig>)
```

### Runtime Config Schema (`config.json`)

```json
{
  "runtime": {
    "provider": "claude",
    "command": "claude",
    "args": [],
    "prompt_mode": "none",
    "env": {}
  }
}
```

Valid built-in `provider` values: `claude`, `gemini`, `codex`, `cursor`,
`auggie`, `amp`. Custom agents registered via `gt config agent set <name> <cmd>`.

### Rig Operational States

Gas Town has two-level operational control:

| State | Command | Scope | Persistence |
|---|---|---|---|
| `operational` | default | — | — |
| `parked` | `gt rig park <rig>` | local town only | ephemeral (wisp layer) |
| `docked` | `gt rig dock <rig>` | all clones via git | permanent until undocked |

**Delete on `gastown_rig` = `gt rig stop <rig>` + `gt rig dock <rig>`.**
This is operational shutdown. It does not remove files. The rig remains
registered; it is simply marked offline globally.

### Configuration Property Layers

Gas Town resolves config through four layers (highest priority first):
1. Wisp layer — `.beads-wisp/config/` — transient, local only
2. Rig bead labels — `.beads/` — synced via git globally
3. Town defaults — `~/gt/config.json`
4. System defaults — compiled into `gt`

The provider reads effective config via `gt rig config show <rig>` and
compares against HCL state. Discrepancies are drift.

### What the Provider Does NOT Manage

The provider manages structural containers. It does not manage:
- Individual beads or issues (`bd create`, etc.) — that is operational work
- Convoys — ephemeral operational work tracking
- Polecats — ephemeral workers spawned by Gas Town itself
- Formulas — TOML workflow templates (future resource, not in scope)

---

## Resource Model

### Three Resources

```hcl
# 1. The HQ — one per workspace
resource "gastown_hq" "main" {
  path        = "/home/user/gt"
  owner_email = "admin@example.com"
  git         = true
}

# 2. A Rig — one per project
resource "gastown_rig" "mirror" {
  hq_path      = gastown_hq.main.path
  name         = "mirror"
  repo         = "https://github.com/kybernetes-systems/rig-mirror.git"
  runtime      = "claude"
  max_polecats = 4
}

# 3. A Crew workspace — one per named human or persistent agent
resource "gastown_crew" "deacon" {
  hq_path  = gastown_hq.main.path
  rig_name = gastown_rig.mirror.name
  name     = "deacon"
}
```

### CRUD Mapping

#### `gastown_hq`

| Operation | `gt` command |
|---|---|
| Create | `gt install <path> --git --owner <email>` |
| Read | Check `mayor/town.json` exists; parse name field |
| Update | `path` and `git` are ForceNew — no in-place update |
| Delete | `gt uninstall --force` |

#### `gastown_rig`

| Operation | `gt` command |
|---|---|
| Create | `gt rig add <name> <repo>` then `gt rig config set <name> runtime <val>` |
| Read | `gt rig status <name> --json` + `gt rig config show <name>` |
| Update | `gt rig config set <name> <key> <val>` for mutable fields |
| Delete | `gt rig stop <name>` + `gt rig dock <name>` |

`name` and `repo` are `ForceNew`. `runtime` and `max_polecats` are mutable.

#### `gastown_crew`

| Operation | `gt` command |
|---|---|
| Create | `gt crew add <name> --rig <rig>` |
| Read | `gt crew list --rig <rig> --json` |
| Update | All fields are ForceNew |
| Delete | `gt crew remove <name> --rig <rig>` |

---

## TDD Workflow Protocol

Follow this cycle exactly, for every change:

1. Write a failing test
2. Run `go test ./...` — confirm it fails for the right reason
3. Write minimum code to make it pass
4. Run `go test ./...` — confirm green
5. Commit atomically with a scoped message
6. Refactor if needed, re-run, commit again

**Never commit red tests. Never skip the failing step.**

---

## Repository Layout

```
terraform-provider-gastown/
├── main.go
├── go.mod
├── go.sum
├── LICENSE
├── README.md
├── internal/
│   ├── provider/
│   │   ├── provider.go
│   │   └── provider_test.go
│   ├── exec/
│   │   ├── runner.go               # os/exec wrapper for gt and bd
│   │   └── runner_test.go
│   └── gastown/
│       ├── hq/
│       │   ├── resource.go
│       │   └── resource_test.go
│       ├── rig/
│       │   ├── resource.go
│       │   └── resource_test.go
│       └── crew/
│           ├── resource.go
│           └── resource_test.go
└── testdata/
    ├── hq_basic.tf
    ├── rig_basic.tf
    └── full_stack.tf
```

---

## Phase 0: Repository Bootstrap

```bash
mkdir terraform-provider-gastown && cd terraform-provider-gastown
git init
git config user.name "$(git config --global user.name)"
git config user.email "$(git config --global user.email)"
go mod init github.com/kybernetes-systems/terraform-provider-gastown
go get github.com/hashicorp/terraform-plugin-framework@latest
go mod tidy
go build -o /dev/null
```

Write `LICENSE` (Apache 2.0 full text) and `README.md` (see bottom of doc).

Commit: `chore: initialize go module and bootstrap repository`

**Exit criterion:** `go build -o /dev/null` succeeds.

---

## Phase 1: exec.Runner

Build a clean, testable wrapper around `os/exec` before any resource code.

### Interface

```go
// internal/exec/runner.go
type Runner interface {
    GT(ctx context.Context, args ...string) (string, error)
    BD(ctx context.Context, args ...string) (string, error)
}
```

### Implementation

- `GT` runs `gt <args>` with `GT_TOWN_ROOT` set to the HQ path
- `BD` runs `bd <args>` similarly
- Both capture stdout+stderr; on non-zero exit, return error containing full stderr
- Never use `/bin/sh` — explicit argument lists only

### TDD Sequence

**Test 1** — `GT` executes `gt version` and returns non-empty output:
```go
func TestRunner_GT_version(t *testing.T) { ... }
```

**Test 2** — Non-zero exit returns error containing stderr:
```go
func TestRunner_GT_nonzeroExitReturnsError(t *testing.T) { ... }
```

Commit: `feat: implement exec runner`

**Exit criterion:** All runner tests pass.

---

## Phase 2: Provider Contract

### Provider Schema

```hcl
provider "gastown" {
  hq_path = "~/gt"   # required
}
```

### TDD Sequence

**Test 1** — Provider validates with `hq_path` set → green → commit: `feat: provider schema`

**Test 2** — Provider rejects empty `hq_path` → green → commit: `test: provider rejects empty hq_path`

**Test 3** — Provider registers `gastown_hq`, `gastown_rig`, `gastown_crew` → green → commit: `feat: register resource types`

**Exit criterion:** All three resource types registered. All tests pass.

---

## Phase 3: `gastown_hq` Resource

### Schema

```hcl
resource "gastown_hq" "main" {
  path        = "/home/user/gt"     # required, ForceNew
  owner_email = "admin@example.com" # optional
  git         = true                # optional, default true
}
```

Computed: `id` (= `path`), `name` (from `mayor/town.json`)

### TDD Sequence

**Test 1** — Create calls `gt install` and `mayor/town.json` exists:
```go
func TestHQResource_Create_callsGtInstall(t *testing.T) { ... }
func TestHQResource_Create_townJsonExists(t *testing.T) { ... }
```

**Test 2** — Read after Create returns identical state (idempotent):
```go
func TestHQResource_Read_idempotent(t *testing.T) { ... }
```

**Test 3** — Changing `path` triggers ForceNew:
```go
func TestHQResource_ForceNew_onPathChange(t *testing.T) { ... }
```

**Test 4** — Delete calls `gt uninstall --force`:
```go
func TestHQResource_Delete_callsUninstall(t *testing.T) { ... }
```

Each test gets its own HQ in `t.TempDir()`. Never reuse paths across tests.

**Exit criterion:** Full CRUD round-trip passes. No temp dirs leak.

---

## Phase 4: `gastown_rig` Resource

### Schema

```hcl
resource "gastown_rig" "mirror" {
  hq_path      = gastown_hq.main.path   # required, ForceNew
  name         = "mirror"               # required, ForceNew
  repo         = "https://..."          # required, ForceNew
  runtime      = "claude"               # optional, default "claude"
  max_polecats = 4                      # optional, default 3
}
```

Computed: `id` (= `<hq_path>/<name>`), `status`, `prefix`

### TDD Sequence

**Test 1** — Create calls `gt rig add` and rig directory exists:
```go
func TestRigResource_Create_callsRigAdd(t *testing.T) { ... }
func TestRigResource_Create_rigDirExists(t *testing.T) { ... }
```

**Test 2** — Create sets runtime via `gt rig config set`:
```go
func TestRigResource_Create_setsRuntime(t *testing.T) { ... }
```

**Test 3** — Read reflects effective config from `gt rig config show`:
```go
func TestRigResource_Read_reflectsEffectiveConfig(t *testing.T) { ... }
```

**Test 4** — Update of `runtime` calls `gt rig config set`:
```go
func TestRigResource_Update_runtime(t *testing.T) { ... }
```

**Test 5** — `name` or `repo` change triggers ForceNew:
```go
func TestRigResource_ForceNew_onNameChange(t *testing.T) { ... }
func TestRigResource_ForceNew_onRepoChange(t *testing.T) { ... }
```

**Test 6** — Delete calls `gt rig stop` then `gt rig dock`; rig dir still exists:
```go
func TestRigResource_Delete_stopsAndDocks(t *testing.T) { ... }
func TestRigResource_Delete_doesNotRemoveFiles(t *testing.T) { ... }
```

**Test 7** — Drift: manually edit `config.json`; next plan shows diff:
```go
func TestRigResource_DriftDetection(t *testing.T) { ... }
```

**Exit criterion:** Full CRUD passes. Rig directory exists after delete.

---

## Phase 5: `gastown_crew` Resource

### Schema

```hcl
resource "gastown_crew" "deacon" {
  hq_path  = gastown_hq.main.path     # required, ForceNew
  rig_name = gastown_rig.mirror.name  # required, ForceNew
  name     = "deacon"                 # required, ForceNew
}
```

Computed: `id` (= `<hq_path>/<rig_name>/crew/<name>`), `path`

All attributes are ForceNew.

### TDD Sequence

**Test 1** — Create calls `gt crew add` and `crew/<name>/` exists:
```go
func TestCrewResource_Create_callsCrewAdd(t *testing.T) { ... }
func TestCrewResource_Create_dirExists(t *testing.T) { ... }
```

**Test 2** — Read detects workspace via `gt crew list --json`:
```go
func TestCrewResource_Read_detectsExistence(t *testing.T) { ... }
```

**Test 3** — Delete calls `gt crew remove`:
```go
func TestCrewResource_Delete_callsCrewRemove(t *testing.T) { ... }
```

**Test 4** — Create fails with descriptive error when rig does not exist:
```go
func TestCrewResource_Create_missingRig_returnsError(t *testing.T) { ... }
```

**Exit criterion:** Full CRUD passes. Dependency on rig enforced.

---

## Phase 6: Integration & Acceptance Suite

```go
func TestAcc_FullLifecycle(t *testing.T) {
    // apply: creates HQ + rig + crew
    // plan: no diff (idempotency)
    // update runtime: apply succeeds, plan clean
    // destroy: rig is docked (not deleted), crew is removed
}

func TestAcc_DriftScenario(t *testing.T) {
    // apply, then manually edit rig config.json
    // next plan must show non-empty diff
}

func TestAcc_Concurrency(t *testing.T) {
    t.Parallel()
    // two separate HQ paths, apply simultaneously
    // no cross-contamination
}
```

Run with: `go test ./... -run TestAcc -v`

---

## Phase 7: Documentation & Release

```bash
go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
tfplugindocs generate

go install github.com/golangci-lint/golangci-lint/cmd/golangci-lint@latest
golangci-lint run
```

Add `.github/workflows/release.yml` with GoReleaser targeting
`linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`.

### Pre-release checklist

- [ ] `go test ./...` zero failures
- [ ] `go vet ./...` clean
- [ ] `golangci-lint run` clean
- [ ] `tfplugindocs generate` complete
- [ ] All three resources have example HCL in `testdata/`
- [ ] No hardcoded paths outside test fixtures
- [ ] `LICENSE` and `README.md` present

---

## README.md Content

```markdown
# terraform-provider-gastown

Terraform provider for managing Gas Town workspaces as infrastructure-as-code.

## Resources

- `gastown_hq` — Gas Town HQ (workspace root)
- `gastown_rig` — Project container wrapping a git repository
- `gastown_crew` — Persistent crew workspace within a rig

## Requirements

- Terraform >= 1.7
- `gt` >= 0.5.0 on PATH
- `bd` >= 0.44.0 on PATH
- `git` >= 2.25 on PATH
- `git config user.name` and `user.email` must be set

## Usage

\```hcl
provider "gastown" {
  hq_path = "/home/user/gt"
}

resource "gastown_hq" "main" {
  path        = "/home/user/gt"
  owner_email = "user@example.com"
}

resource "gastown_rig" "mirror" {
  hq_path  = gastown_hq.main.path
  name     = "mirror"
  repo     = "https://github.com/kybernetes-systems/rig-mirror.git"
  runtime  = "claude"
}

resource "gastown_crew" "deacon" {
  hq_path  = gastown_hq.main.path
  rig_name = gastown_rig.mirror.name
  name     = "deacon"
}
\```

## License

Apache 2.0
```

---

## Error Handling Standards

All CRUD errors must use `resp.Diagnostics.AddError()` with:
- A capitalized summary (e.g., `"Error creating rig"`)
- A detail string including the resource name and full stderr from `gt`

Never swallow errors. Never panic. Never call `os.Exit`.

---

## Hard Constraints

- Call `gt` and `bd` via `os/exec` with explicit argument lists.
  Never parse internal Gas Town files directly except where no `gt` command
  exposes the needed data.
- All test HQ paths use `t.TempDir()`. No test writes to the real filesystem.
- No global state. Each resource operation is self-contained.
- No shell intermediaries. No `bash -c`. Direct exec only.
