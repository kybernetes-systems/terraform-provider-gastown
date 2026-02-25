# ADR 0003: Delete on gastown_rig Means Dock and Stop, Not File Removal

**Status:** Accepted  
**Date:** 2026-02  
**Deciders:** Phil Mocek / kybernetes.systems

---

## Context

When a `gastown_rig` resource is removed from Terraform state (either via
`terraform destroy` or by removing the resource block), the provider must
decide what to do with the live rig.

Gas Town has a two-level operational control system:

- **Park** (`gt rig park`): local, ephemeral stop. Stored in the wisp
  layer. Does not sync to other clones. Daemon will not auto-restart.
- **Dock** (`gt rig dock`): global, persistent shutdown. Stored as a
  label on the rig identity bead, which syncs via git to all clones.
  All clones see the rig as offline.

A third option exists: `gt rig remove`, which removes the rig from the
HQ registry. A fourth option: manually delete the rig directory from the
filesystem.

## Decision

**Delete on `gastown_rig` calls `gt rig stop <rig>` followed by
`gt rig dock <rig>`. It does not call `gt rig remove` and does not
delete the rig directory.**

## Rationale

Gas Town rigs accumulate irreplaceable operational history: `.beads/`
JSONL ledgers, git worktree state, completed convoys, forensic findings,
policy patches in various states of draft. For the Policy Change Factory
use case, this history is civic record. Deleting it on `terraform destroy`
would be catastrophic and irreversible.

The dock operation signals to all clones of the workspace that the rig
is intentionally offline. This is the correct semantic for "this rig is
no longer being managed by this Terraform configuration" — it does not
mean the rig never existed or that its history should be erased.

If a rig is truly to be decommissioned and its data purged, that is a
deliberate human act performed outside Terraform. Terraform's destroy
should be safe to run without data loss anxiety.

This decision aligns with the Manifesto's immutability directive:

> The past is etched in lead. We do not edit history; we plant new logic
> to overwrite the failures of the old.

## Consequences

- **Positive:** `terraform destroy` is safe. Rig history is preserved.
- **Positive:** A docked rig can be undocked and re-adopted into Terraform
  management simply by re-adding the resource block.
- **Negative:** Repeated create/destroy cycles will accumulate docked rigs
  on disk. Operators must manually clean up with `gt rig remove` if needed.
- **Negative:** After `terraform destroy`, the rig still exists on disk
  in a docked state. Operators who expect destroy to clean up fully will
  be surprised. The provider must document this clearly.
- **Testing note:** Tests must assert that the rig directory exists after
  delete and that `gt rig status` shows `docked`, not that the directory
  is absent.
