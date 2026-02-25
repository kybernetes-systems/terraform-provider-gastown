# ADR 0011: Tests Must Not Spawn Polecats

**Status:** Accepted  
**Date:** 2026-02  
**Deciders:** Phil Mocek / kybernetes.systems

---

## Context

Polecats are Gas Town's ephemeral worker agents — long-lived daemon
processes spawned by `gt rig start` and managed by the Gas Town daemon.
Unlike rigs and crew workspaces, polecats are not infrastructure; they
are live AI coding agents actively consuming API quota, writing to git
worktrees, and potentially modifying bead state.

Tests run in isolated HQs (ADR 0004), but HQ isolation does not
constrain the Gas Town daemon. A polecat spawned in a test HQ is a real
process running under the real daemon. If the test exits, crashes, or if
`t.TempDir()` is cleaned up while a polecat is running, the result is:

- An orphaned agent process with a dangling HQ path
- Potential writes to a deleted or reused directory
- API quota consumption with no operator oversight
- In the worst case: a polecat that cannot find its HQ and enters an
  undefined error loop

On a developer workstation where production polecats are also running,
orphaned test polecats are indistinguishable from legitimate agents.
Debugging this is expensive and the failure mode is silent.

## Decision

**Tests must never start polecats. The following commands are
prohibited in test code:**

- `gt rig start`
- `gt crew run`
- `gt convoy`
- Any `gt` subcommand whose help text includes "spawn", "start agent",
  or "run polecat"

**The exec.Runner used in tests must enforce this at the call site.**
A `SafeRunner` wrapper in `internal/testutil` wraps the standard Runner
and panics with a descriptive message if any prohibited command is
invoked. This makes violations fail loudly and immediately rather than
spawning a live agent.

```go
// SafeRunner wraps exec.Runner for use in tests.
// It panics if any command that could spawn polecats is called.
// See ADR 0011.
type SafeRunner struct {
    inner Runner
}

var prohibitedCommands = []string{"start", "crew run", "convoy"}

func (r *SafeRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
    for _, prohibited := range prohibitedCommands {
        if len(args) >= 2 && strings.Join(args[:2], " ") == prohibited {
            panic(fmt.Sprintf(
                "testutil.SafeRunner: prohibited command %q — tests must not spawn polecats (ADR 0011)",
                strings.Join(args, " "),
            ))
        }
    }
    return r.inner.Run(ctx, args...)
}
```

## Testing Running-State Behavior

Some provider behavior may differ based on rig operational state
(running vs parked vs docked). Tests that need to verify running-state
behavior use a pre-started rig provided via environment variable:

```
GASTOWN_TEST_RUNNING_RIG=<rig_name>
```

If this variable is not set, tests that require a running rig are
skipped with:

```
--- SKIP: TestRigRunningState (GASTOWN_TEST_RUNNING_RIG not set — start a rig manually and set this variable)
```

The operator is responsible for starting the rig before running these
tests and for stopping it afterward. This is an explicit, supervised
act, not an automated one.

## Teardown Guard

Every test that creates a rig (even in an isolated HQ) must call
`testutil.AssertNoPolecat(t, runner, rigName)` in a `t.Cleanup`
function registered immediately after rig creation:

```go
gt.RigAdd(ctx, name, repo)
t.Cleanup(func() {
    testutil.AssertNoPolecat(t, runner, name)
})
```

`AssertNoPolecat` calls `gt rig status --json` and checks that the
polecat count is zero. If any polecats are found, it calls
`gt rig park <name>` to request a stop, waits up to 10 seconds,
then fails the test with a message identifying the orphaned processes.
It does not silently swallow the condition.

## Rationale

The production HQ is where real polecats do real civic policy work.
An escaped test polecat in that environment is not merely an
inconvenience — it is an unaccountable agent acting without operator
intent. The Policy Change Factory's legitimacy depends on every agent
action being traceable to a deliberate human directive. Automated test
processes do not meet that bar.

Panicking on prohibited commands rather than returning an error is
intentional. A returned error could be ignored or mishandled. A panic
in a test is immediately visible, immediately attributable, and leaves
no ambiguity about what went wrong.

## Consequences

- **Positive:** No test can accidentally spawn a live agent.
- **Positive:** Violations are caught at the call site, not post-hoc.
- **Positive:** Running-state tests remain possible via the opt-in
  environment variable, under operator supervision.
- **Negative:** Running-state acceptance tests cannot run in unattended
  CI without a pre-running rig. This is acceptable — unattended CI
  should not be spawning AI agents.
- **Maintenance note:** The `prohibitedCommands` list in `SafeRunner`
  must be reviewed against the `gt` changelog on every version bump.
  New subcommands that spawn processes must be added to the list.
