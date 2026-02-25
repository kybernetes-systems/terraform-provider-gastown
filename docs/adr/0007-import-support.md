# ADR 0007: Terraform Import Support

**Status:** Accepted  
**Date:** 2026-02  
**Deciders:** Phil Mocek / kybernetes.systems

---

## Context

Gas Town HQ, rigs, and crew workspaces may be created outside of
Terraform — by running `gt install` or `gt rig add` manually, or by
receiving a `gt pull` that syncs another operator's workspace. These
pre-existing resources must be adoptable into Terraform management
without destroying and recreating them.

Terraform supports this via `terraform import` and, for plugin-framework
providers, via the `ImportState` method on each resource.

## Decision

**All three resources implement `ImportState`. Import IDs are:**

| Resource | Import ID format | Example |
|---|---|---|
| `gastown_hq` | Absolute path to HQ root | `/home/phil/gt` |
| `gastown_rig` | `<rig_name>` | `kybernetes` |
| `gastown_crew` | `<rig_name>/<crew_name>` | `kybernetes/mayoragent` |

`ImportState` calls the same Read logic used in normal refresh. No
separate import path.

## Rationale

The Policy Change Factory use case almost guarantees that operators
will hand-initialize HQs and rigs before the Terraform configuration
exists (standing up the workspace to test it, then formalizing it as
code). Without import support, adoption requires destroy-and-recreate,
which ADR 0003 establishes is unacceptable for rigs with operational
history.

The import ID formats are chosen to be human-memorable and stable
across `gt` versions. Rig names and crew names are unique within a
HQ and do not change (ForceNew on both — see HANDOFF.md).

## Consequences

- **Positive:** Pre-existing workspaces can be brought under Terraform
  management without data loss.
- **Positive:** Import IDs are obvious to anyone familiar with Gas Town
  conventions.
- **Negative:** `gastown_hq` import requires the operator to know the
  absolute path, which may differ across machines if HQ root is not
  standardized. Recommend documenting `~/gt` as the conventional path.
- **Testing requirement:** Each resource needs an import test:
  create the resource outside Terraform, run `terraform import`, verify
  state matches the live resource, and verify `terraform plan` shows
  no diff.
