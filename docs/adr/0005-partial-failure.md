# ADR 0005: Partial Failure Handling During Multi-Step Creates

**Status:** Accepted  
**Date:** 2026-02  
**Deciders:** Phil Mocek / kybernetes.systems

---

## Context

Several resources require multiple sequential CLI calls to reach the
desired state. `gastown_rig` create calls:

1. `gt rig add <name> <repo>`
2. `gt rig config set runtime.provider <provider>`
3. `gt rig config set runtime.command <command>`
4. `gt rig config set max_polecats <n>` (if specified)

If step 1 succeeds and step 2 fails, the rig exists in Gas Town but its
configuration is incomplete. On the next `terraform apply`, step 1 will
fail ("rig already exists") without ever attempting step 2.

Terraform requires the provider to handle this explicitly. The options are:

**Option A — Atomic rollback:** If any config step fails, call `gt rig remove`
to undo step 1, return an error, and set no ID. The resource is not in state.

**Option B — Partial state + read:** If any config step fails, set the resource
ID anyway (the rig exists), store what was successfully set, and return an
error. On the next apply, Read will detect the drift and plan will re-apply
the missing configuration.

**Option C — Retry-safe create with Read:** After all steps, call Read
unconditionally. Store whatever state Read returns. If config steps failed
silently (no error returned but wrong value set), the next plan detects drift.

## Decision

**Option C for silent failures. Option A for hard failures after step 1.
If `gt rig add` succeeds but a `gt rig config set` returns a non-zero exit
code, call `gt rig dock <name>` (not remove) and return the error without
setting state. Always conclude successful creates by calling Read and storing
the authoritative state.**

Dock rather than remove on rollback to preserve any partial state for
human inspection. See ADR 0003.

## Rationale

Option B is appealing but requires the provider to track which sub-steps
completed, which is complex state to serialize into Terraform's schema.
Option A (rollback via remove) is used by most Terraform providers for
atomic create semantics, but ADR 0003 prohibits remove on rigs. Docking
on rollback achieves the same safety guarantee (no orphaned running rigs)
without deleting history.

Option C's unconditional Read after create catches cases where `gt rig config set`
exits 0 but wrote a different value than expected — a real failure mode
if the `gt` flag syntax changes.

## Consequences

- **Positive:** Orphaned rigs always end up docked, never running.
- **Positive:** Silent config mismatches are caught on the next plan.
- **Negative:** A failed create leaves a docked rig on disk that Terraform
  has no record of. The operator must manually `gt rig remove` or re-adopt
  it. This must be documented.
- **Testing requirement:** Tests must exercise the "step 1 succeeds,
  step 2 fails" scenario explicitly, asserting the rig is docked and
  no state ID is set.
