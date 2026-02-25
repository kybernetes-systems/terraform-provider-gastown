# ADR 0001: Call gt and bd via os/exec, Do Not Parse Internal Files

**Status:** Accepted  
**Date:** 2026-02  
**Deciders:** Phil Mocek / kybernetes.systems

---

## Context

Gas Town stores its state in a mix of formats: `mayor/town.json`,
`mayor/rigs.json`, `config.json` per rig, `.beads/` JSONL databases,
wisp configs, and bead labels. These formats are internal implementation
details of `gt` and are not part of its public contract. The CHANGELOG
shows schema changes in nearly every release (v0.1 through v0.5 changed
rig identity beads, operational state labels, config resolution, and the
messaging system multiple times).

Two integration strategies are available:

**Option A — File parsing:** Read and write Gas Town's internal files
directly from Go. This would allow fully offline tests (no `gt` binary
required) and would give the provider fine-grained control.

**Option B — CLI delegation:** Call `gt` and `bd` with `os/exec` for all
operations. The provider treats the CLIs as the public API surface.

## Decision

**Option B. The provider calls `gt` and `bd` via `os/exec` with explicit
argument lists. It never reads or writes Gas Town internal files directly
except as a last resort when no CLI command exposes the needed data.**

## Rationale

Gas Town's internal file formats are unstable. The `gt` CLI is the stable
contract. Parsing internals would couple the provider to Gas Town's
implementation rather than its interface, causing silent breakage on every
`gt` upgrade.

Additionally, `gt` and `bd` perform side effects beyond file writes —
they update rig identity beads, trigger daemon notifications, and manage
git worktrees. Replicating this logic in the provider would be fragile and
incomplete.

The cost is that tests require real `gt` and `bd` binaries on PATH.
This is acceptable: both are installable in a single command, and the
test isolation strategy (see ADR 0005) keeps tests hermetic.

## Consequences

- **Positive:** Provider stays correct across `gt` upgrades as long as
  CLI flag signatures are stable.
- **Positive:** No reimplementation of Gas Town's internal logic.
- **Negative:** Tests require `gt` and `bd` installed. CI must install
  both before running `go test`.
- **Negative:** JSON output flag consistency (`--json`) must be verified
  per command. If a `gt` command lacks `--json`, the provider must parse
  human-readable output or use a workaround.
- **Risk:** If `gt` changes a flag signature without a major version bump,
  the provider breaks silently. Mitigation: pin `gt` version in CI and
  document the minimum version requirement explicitly.
