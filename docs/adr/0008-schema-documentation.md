# ADR 0008: Schema Attribute Documentation Requirements

**Status:** Accepted  
**Date:** 2026-02  
**Deciders:** Phil Mocek / kybernetes.systems

---

## Context

`tfplugindocs generate` produces `docs/resources/*.md` from the
`Description` fields on resource schemas and example configurations
in `examples/`. If `Description` fields are empty, the generated docs
contain only attribute names and types — useless to operators.

The provider's target users include civic policy practitioners who are not
necessarily fluent in Terraform or Gas Town internals. Documentation quality
directly affects adoption.

## Decision

**Every schema attribute must have a `Description` string. All
descriptions must be written in the Architect voice (see ADR 0009)
and must meet these standards:**

- State what the attribute controls, not what type it is.
- For `Required` attributes: note when it triggers ForceNew and why.
- For `Optional` attributes: state the default value and its source
  (e.g., "Defaults to the Gas Town system default of 4.").
- For `Computed` attributes: explain how the value is determined
  and when it may change.
- Avoid "This attribute..." phrasing. Lead with the concept.

Additionally, each resource must have:
- A `MarkdownDescription` on the resource block itself, 2–4 sentences.
- At least one example `.tf` file in `examples/resources/<resource_name>/`
  used by `tfplugindocs` to populate the example block.
- A `# Import` section in the example directory's `import.sh` file.

## Attribute Description Examples

**Weak (rejected):**
```go
Description: "The name of the rig.",
```

**Strong (required):**
```go
Description: "Unique name for the rig within this HQ. " +
    "Set once at creation; changing this value forces replacement of " +
    "the resource (the existing rig is docked, a new one is created). " +
    "Corresponds to the <name> argument of `gt rig add`.",
```

## Consequences

- **Positive:** `tfplugindocs generate` produces docs worth reading.
- **Positive:** Attribute descriptions double as inline architectural
  documentation — a developer reading the schema understands the design
  intent without consulting the ADRs.
- **Negative:** Writing useful descriptions takes time. Claude Code
  must treat empty or single-word descriptions as a failing test
  condition.
- **Implementation note:** Add a linter check (custom `go vet` pass or
  test assertion) that fails if any public schema attribute has a
  `Description` shorter than 40 characters. This prevents the "This is
  the name." antipattern from shipping.
