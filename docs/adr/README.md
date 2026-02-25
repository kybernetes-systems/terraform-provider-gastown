# Architectural Decision Records

This directory contains the architectural decision records (ADRs) for the
`terraform-provider-gastown` project.

ADRs document significant design choices, the context in which they were made,
the alternatives considered, and the consequences of the decision. They are
written in the Architect voice (see ADR 0009) and are the normative reference
for implementation decisions.

## Index

| # | Title | Status |
|---|---|---|
| [0001](0001-exec-over-file-parsing.md) | Call gt and bd via os/exec, Do Not Parse Internal Files | Accepted |
| [0002](0002-three-resources.md) | Three Terraform Resources — gastown_hq, gastown_rig, gastown_crew | Accepted |
| [0003](0003-delete-means-dock.md) | Delete on gastown_rig Means Dock and Stop, Not File Removal | Accepted |
| [0004](0004-test-isolation.md) | Test Isolation via t.TempDir and Prerequisite Guards | Accepted |
| [0005](0005-partial-failure.md) | Partial Failure Handling During Multi-Step Creates | Accepted |
| [0006](0006-json-output-parsing.md) | JSON Output Parsing Strategy for gt and bd | Accepted |
| [0007](0007-import-support.md) | Terraform Import Support | Accepted |
| [0008](0008-schema-documentation.md) | Schema Attribute Documentation Requirements | Accepted |
| [0009](0009-dual-register-docs.md) | Dual-Register Documentation — Architect Voice and Lead Farmer Voice | Accepted |
| [0010](0010-provider-config.md) | Provider Configuration — Binary Paths and Environment | Accepted |
| [0011](0011-no-polecats-in-tests.md) | Tests Must Not Spawn Polecats | Accepted |

## Statuses

- **Accepted:** Current normative decision.
- **Superseded:** Replaced by a later ADR (noted in both records).
- **Deprecated:** No longer applicable; documented for historical context.
- **Proposed:** Under discussion; not yet binding.

## Adding an ADR

1. Copy the template below.
2. Number sequentially.
3. Link from this index.
4. If the new ADR supersedes an existing one, update the existing ADR's
   Status field and add a `Superseded by: ADR XXXX` line.

```markdown
# ADR XXXX: Title

**Status:** Proposed  
**Date:** YYYY-MM  
**Deciders:** 

---

## Context

## Decision

## Rationale

## Consequences
```
