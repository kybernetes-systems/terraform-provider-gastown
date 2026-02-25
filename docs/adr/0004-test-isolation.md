# ADR 0004: Test Isolation via t.TempDir and Prerequisite Guards

**Status:** Accepted  
**Date:** 2026-02  
**Deciders:** Phil Mocek / kybernetes.systems

---

## Context

Gas Town is workspace-scoped: the `GT_TOWN_ROOT` environment variable
controls which HQ a given `gt` invocation operates against. Tests that
share a single HQ will corrupt each other's rig and crew state,
especially when run in parallel. A strategy is needed to make tests
hermetic and parallelizable.

Additionally, tests must run cleanly on machines where `gt`, `bd`, or
`git` are not installed, rather than failing with unhelpful panics or
misleading error messages.

## Decision

**Each test that exercises `gt` or `bd` creates an isolated HQ in
`t.TempDir()` and sets `GT_TOWN_ROOT` to that path in the exec.Runner
for that test. A `testutil.RequireGT`, `testutil.RequireBD`, and
`testutil.RequireGit` helper skips the test with a clear message if the
binary is not on PATH or does not meet the minimum version.**

Minimum versions enforced:
- `gt` >= 0.5.0
- `bd` >= 0.44.0
- `git` >= 2.25.0
- `git config user.name` and `git config user.email` must be set (required
  by `gt install --git`)

## Rationale

`t.TempDir()` directories are removed automatically after each test, even
on failure, preventing accumulation of stale state. Setting `GT_TOWN_ROOT`
per-runner rather than globally means parallel tests never interfere.

Prerequisite guards at the top of each relevant test produce human-
readable skip messages like:

```
--- SKIP: TestRigCreate (gt not found — install with: go install github.com/steveyegge/gastown/cmd/gt@latest)
```

This is strictly better than the alternative (tests that fail with
obscure exec errors or panic on nil runners).

## Consequences

- **Positive:** Tests are safe to run in parallel.
- **Positive:** CI failures due to missing tools are immediately diagnosable.
- **Positive:** Developer workstations without Gas Town installed can still
  run unit tests for `exec.Runner` (which mock the binary).
- **Negative:** Each test that calls `gt install` incurs real I/O and
  git initialization cost. This is unavoidable without mocking the CLI
  entirely (rejected in ADR 0001).
- **Implementation note:** The `exec.Runner` must accept `GT_TOWN_ROOT`
  as a constructor parameter, not read it from the process environment.
  Reading from the process environment would make tests non-hermetic.
