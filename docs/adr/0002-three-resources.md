# ADR 0002: Three Terraform Resources — gastown_hq, gastown_rig, gastown_crew

**Status:** Accepted  
**Date:** 2026-02  
**Deciders:** Phil Mocek / kybernetes.systems

---

## Context

The Kybernetes Ordnance Manifesto and Strategic Handoff defined three
semantic primitives for the provider:

> The Rig: The land we hold.  
> The Policy: The seeds we plant.  
> The Role: The farmhands who defend the yield.

These were written before the Gas Town architecture was fully understood.
The actual Gas Town workspace hierarchy has three distinct structural
layers that must be provisioned: the HQ, the rig, and the crew workspace.

The question is how to map the Terraform resource model to the real
Gas Town object model.

## Decision

**The provider exposes three resources: `gastown_hq`, `gastown_rig`,
and `gastown_crew`. These map directly to the `gt install`, `gt rig add`,
and `gt crew add` commands.**

## Mapping from Lore to Technical

The Manifesto's three primitives survive — they describe the *operational*
layer, not the *structural* layer. Their technical correspondence is:

| Manifesto Primitive | Operational Meaning | Terraform Layer |
|---|---|---|
| The Rig | The land we hold | `gastown_rig` + `gastown_hq` |
| The Policy | The seeds we plant | Beads (`bd`) — NOT Terraform |
| The Role | The farmhands | `gastown_crew` (persistent) + polecats (ephemeral, not Terraform) |

**Policies (Beads) are explicitly out of scope for this provider.**
Beads are operational work units — forensic findings, civic patches,
PRR filings — created and updated during live agent runs. They are not
infrastructure; they are the *output* of infrastructure. Managing them
via Terraform `apply` would be a category error equivalent to managing
S3 object contents via a Terraform bucket resource.

## Consequences

- **Positive:** Provider resources have a clean 1:1 mapping to `gt`
  commands with no ambiguity.
- **Positive:** The lore mapping is explicit and documented, preventing
  future confusion between the Architect and Lead Farmer vocabularies.
- **Deferred:** A future `gastown_formula` resource could manage
  `.beads/formulas/` TOML workflow templates declaratively. This is
  appropriate for Terraform because formulas are templates (infrastructure),
  not instances (operational state). Not in scope for v1.
- **Deferred:** A future `gastown_plugin` resource could manage
  `plugins/` markdown files with TOML frontmatter.
- **Out of scope permanently:** Individual beads, convoys, polecats.
  These are runtime artifacts, not infrastructure.
