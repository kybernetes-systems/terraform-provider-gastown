# ADR 0006: JSON Output Parsing Strategy for gt and bd

**Status:** Accepted  
**Date:** 2026-02  
**Deciders:** Phil Mocek / kybernetes.systems

---

## Context

The Read implementations for all three resources need machine-readable
output from `gt` and `bd`. The known commands are:

| Command | JSON flag existence | Confidence |
|---|---|---|
| `gt rig status` | `--json` assumed | Unverified |
| `gt crew list` | `--json` assumed | Unverified |
| `gt rig config show` | `--json` assumed | Unverified |
| `gt rig list` | `--json` assumed | Unverified |
| `bd list` | `--json` likely | Unverified |

These flags are assumed from CLI patterns observed in the Gas Town
README but have not been verified in the source. There is a real risk
that some commands lack `--json` or that the flag exists but produces
a different schema than assumed.

## Decision

**Phase 1 of the implementation must verify every `--json` flag before
writing any Read code. Verification is done by calling each command with
`--help` and parsing the help output for the flag. If `--json` is absent,
use `--format json` or `--output json` (common alternatives). If none
exist, use a known workaround for that command documented here.**

Fallback strategy if no JSON flag exists for a command:

1. Parse the human-readable output with a narrow, comment-documented
   regex specific to that command's current output format.
2. Gate the regex behind a version check so it fails loudly on `gt`
   version upgrades rather than silently returning wrong data.
3. Open a GitHub issue on steveyegge/gastown requesting `--json` and
   document the issue number in the code comment.

## JSON Schema Validation

Each `--json` response must be validated against a typed Go struct
(not `map[string]interface{}`). The struct definition must include
a comment citing the `gt` version against which it was verified.
Example:

```go
// rigStatus is the parsed output of `gt rig status --json`.
// Verified against gt v0.5.2. See also: ADR 0006.
type rigStatus struct {
    Name  string `json:"name"`
    State string `json:"state"` // "running", "parked", "docked"
    ...
}
```

Unknown fields must be preserved (use `json.Decoder` with
`DisallowUnknownFields` disabled) to avoid breaking on new fields added
in future `gt` versions.

## Consequences

- **Positive:** Schema drift is caught at the typed struct boundary, not
  scattered through provider logic.
- **Positive:** Workarounds are version-gated and documented in code.
- **Negative:** Phase 1 requires an exploratory `gt` session to verify
  all flags before any Read code is written. This is non-negotiable.
- **Action item for Claude Code:** Before writing any Read implementation,
  run each command listed in the table above with `--json` and capture
  a sample output. Store samples in `testdata/fixtures/` and reference
  them in unit tests.
