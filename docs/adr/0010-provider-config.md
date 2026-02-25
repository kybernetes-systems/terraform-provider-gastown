# ADR 0010: Provider Configuration — Binary Paths and Environment

**Status:** Accepted  
**Date:** 2026-02  
**Deciders:** Phil Mocek / kybernetes.systems

---

## Context

The provider shells out to `gt` and `bd`. In production deployments
(e.g., kybernetes.systems CI), these binaries may not be on the default
`$PATH`, or the operator may want to pin specific build artifacts rather
than system-installed versions.

Additionally, `GT_TOWN_ROOT` may need to be set explicitly if the
operator runs Terraform from a directory that is not the HQ root.

The provider needs a configuration block. The question is what it
should expose and what it should require.

## Decision

**The provider block exposes three optional attributes:**

```hcl
provider "gastown" {
  gt_path      = "/usr/local/bin/gt"   # default: resolved from PATH
  bd_path      = "/usr/local/bin/bd"   # default: resolved from PATH
  town_root    = "/home/phil/gt"       # default: $GT_TOWN_ROOT, then $HOME/gt
}
```

All three are optional. Resolution order for binary paths:
1. Provider block attribute
2. `GT_PATH` / `BD_PATH` environment variables
3. `exec.LookPath("gt")` / `exec.LookPath("bd")`
4. Error: binary not found, message includes install instructions

Resolution order for `town_root`:
1. Provider block `town_root` attribute
2. `GT_TOWN_ROOT` environment variable
3. `$HOME/gt` (Gas Town default)

The provider validates at configuration time that both binaries are
executable and meet the minimum version requirements (ADR 0004).

## Rationale

Explicit binary paths are necessary for CI environments and for
operators who maintain multiple `gt` versions. The `$HOME/gt` default
matches Gas Town's own convention and requires no configuration in the
common case.

Validating at provider configuration time (not at resource create time)
means a bad binary path fails fast with a clear error, rather than
failing silently during `terraform plan`.

## Consequences

- **Positive:** Zero-config for standard installations.
- **Positive:** CI pipelines can pin binaries explicitly without modifying
  system PATH.
- **Positive:** Version validation at startup prevents obscure failures
  mid-apply.
- **Negative:** If an operator has multiple HQs (unusual but possible),
  they need multiple provider configurations with aliases.
- **Testing note:** Tests must set `gt_path` and `bd_path` explicitly to
  the binaries found in the test environment, rather than relying on PATH.
  This makes test behavior reproducible regardless of system configuration.
