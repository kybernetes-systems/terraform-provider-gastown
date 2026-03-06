# Terraform Provider Critical Issues Review

**Review Date:** 2026-03-06  
**Reviewer:** Code Review Agent (Skeptical Mode)  
**Status:** 11 new issues created

---

## Executive Summary

A skeptical review of the terraform-provider-gastown codebase revealed **11 critical issues**, including:
- **2 silent data consistency bugs** (empty Update functions)
- **1 documentation inaccuracy** (import support claims)
- **1 security concern** (unvalidated input to shell)
- **Multiple reliability issues** (race conditions, brittle error handling)

---

## P0 (Critical) Issues - Fix Immediately

### 1. tfgt-m9e: Import Support Documentation Inaccurate
**File:** `docs/adr/0007-import-support.md`, `internal/gastown/rig/resource.go`, `internal/gastown/crew/resource.go`

**Problem:** ADR claims all 3 resources support import. Only HQ actually implements `ResourceWithImportState`.

**Code Evidence:**
```go
// Only HQ has this:
var _ resource.ResourceWithImportState = &HQResource{}

// Rig and crew are missing the interface and ImportState method
```

**Fix Options:**
1. Implement ImportState for rig and crew
2. Update ADR to reflect actual support

---

### 2. tfgt-9rg: HQ Update Silently Does Nothing
**File:** `internal/gastown/hq/resource.go:197-198`

**Problem:** Empty Update function causes permanent drift.

**Code:**
```go
func (r *HQResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
    // EMPTY - silently does nothing!
}
```

**Impact:** Users change `owner_email`, `git`, or `no_beads` → Terraform reports success → State updated → Actual HQ unchanged → Drift is permanent.

**Fix:** Return error or implement actual updates.

---

### 3. tfgt-0vt: Rig Update Partial Failure
**File:** `internal/gastown/rig/resource.go:198-213`

**Problem:** Update only handles `runtime`, ignores `max_polecats`.

**Code:**
```go
func (r *RigResource) Update(...) {
    // Only updates runtime!
    runner.GT(ctx, "rig", "config", "set", ..., "runtime", ...)
    
    // Claims all fields updated:
    resp.State.Set(ctx, &plan)
}
```

**Impact:** User changes `max_polecats` → Terraform reports success → State shows new value → Actual rig unchanged.

**Fix:** Update max_polecats or mark it ForceNew.

---

### 4. tfgt-ttm: No Input Validation (Security)
**Files:** All resource files

**Problem:** User input passed directly to shell commands.

**Vulnerable Code:**
```go
runner.GT(ctx, "rig", "add", plan.Name.ValueString(), plan.Repo.ValueString())
```

**Affected Attributes:**
- `path` - path traversal risk (`../../../etc/passwd`)
- `name` - shell metacharacters
- `repo` - unvalidated URLs
- `owner_email`, `role`

**Fix:** Add validators to all string attributes.

---

### 5. tfgt-thg: Orphaned Daemon Processes (Existing Issue)
**Status:** Already tracked, related to test cleanup

---

## P1 (High) Issues - Fix Soon

### 6. tfgt-o7k: Silent Failures in getPrefixFromGT
**File:** `internal/gastown/rig/resource.go:156-159`

**Problem:** Errors ignored, prefix silently empty.

```go
prefix, err := getPrefixFromGT(...)
if err == nil {
    plan.Prefix = types.StringValue(prefix)
}
// err != nil is silently ignored!
```

**Fix:** Add warning diagnostic on error.

---

### 7. tfgt-if3: Port Allocation Race Condition
**File:** `internal/gastown/hq/resource.go:136-150`

**Problem:** TOCTOU race in getFreePort.

```go
func getFreePort() (int, error) {
    l, err := net.Listen("tcp", "127.0.0.1:0")
    defer l.Close()  // Released here
    return l.Addr().(*net.TCPAddr).Port, nil
}
// Another process could take the port before Dolt starts!
```

**Fix:** Keep listener bound or use file-based port reservation.

---

### 8. tfgt-p23: Brittle Error String Matching
**File:** `internal/gastown/rig/resource.go:180-181`

**Problem:** Relies on specific English error messages.

```go
if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no such") {
```

**Impact:** If gt CLI changes error format, resources won't be removed from state.

**Fix:** Use structured error codes.

---

## P2 (Medium) Issues - Fix When Convenient

### 9. tfgt-czq: Test Flag in Production Code
**File:** `internal/gastown/hq/resource.go:135`

```go
if os.Getenv("TF_ACC") == "1" {
    // Test-only logic
}
```

**Problem:** Production code shouldn't know about test flags.

**Fix:** Make port config a proper provider option.

---

### 10. tfgt-vfr: No CheckDestroy in Tests
**File:** `internal/provider/acceptance_test.go`

**Problem:** Tests don't verify cleanup.

```go
resource.Test(t, resource.TestCase{
    CheckDestroy: nil,  // Missing!
})
```

**Fix:** Add CheckDestroy functions.

---

### 11. tfgt-xma: Crew Read No Diagnostic
**File:** `internal/gastown/crew/resource.go:147-151`

```go
if !found {
    resp.State.RemoveResource(ctx)
    return  // No diagnostic!
}
```

**Fix:** Add warning: "Crew member not found, removing from state"

---

## Recommendations by Priority

### Immediate (This Sprint)
1. Fix HQ Update (tfgt-9rg) - Return error or implement
2. Fix Rig Update (tfgt-0vt) - Handle max_polecats or mark ForceNew
3. Add input validation (tfgt-ttm) - Security

### This Month
4. Fix import documentation (tfgt-m9e) - Align docs with reality
5. Fix silent failures (tfgt-o7k) - Add diagnostics
6. Fix race condition (tfgt-if3) - Stabilize tests

### Backlog
7. Fix brittle error matching (tfgt-p23)
8. Add CheckDestroy (tfgt-vfr)
9. Remove TF_ACC check (tfgt-czq)
10. Add diagnostics (tfgt-xma)

---

## Files Requiring Changes

| File | Issues |
|------|--------|
| `internal/gastown/hq/resource.go` | tfgt-9rg, tfgt-if3, tfgt-czq |
| `internal/gastown/rig/resource.go` | tfgt-0vt, tfgt-o7k, tfgt-p23, tfgt-ttm |
| `internal/gastown/crew/resource.go` | tfgt-m9e, tfgt-xma, tfgt-ttm |
| `docs/adr/0007-import-support.md` | tfgt-m9e |
| `internal/provider/acceptance_test.go` | tfgt-vfr |

---

## Testing Recommendations

For each fix, add tests that:
1. **Flip tests:** Verify changes fail before fix, pass after
2. **Import tests:** Actually test import functionality
3. **Update tests:** Verify partial update failures are caught
4. **Security tests:** Verify invalid input is rejected
5. **Destroy tests:** Verify CheckDestroy works
