# Terraform Provider Tests

This directory contains Terraform test files (`.tftest.hcl`) for validating the Gas Town provider.

## Test Files

| File | Purpose | Requirements |
|------|---------|--------------|
| `unit_mock_test.tftest.hcl` | Unit tests using mock_provider (fast, no gt CLI needed) | Terraform >= 1.7.0 |
| `mock_test_fixture.tf` | Test fixture resources for mock tests | - |

## Running Tests

### Unit Tests (Mock Provider)

These tests use Terraform 1.7+ `mock_provider` to validate resource configurations without requiring the Gas Town CLI:

```bash
terraform test tests/unit_mock_test.tftest.hcl
```

**Benefits:**
- Fast execution (no process spawning)
- No Gas Town installation required
- No credentials needed
- Safe for CI/CD pipelines

**Limitations:**
- Only validates configuration logic, not actual gt CLI behavior
- Computed attributes return mocked values
- Cannot test real resource lifecycle (create, read, update, delete)

### Requirements

- Terraform >= 1.7.0 (for mock_provider support)
- Provider built locally: `go build -o terraform-provider-gastown`

## Adding New Tests

To add new mock_provider tests:

1. Add mock resources to the `mock_provider "gastown"` block in `unit_mock_test.tftest.hcl`
2. Add variables to `mock_test_fixture.tf` if needed
3. Create `run` blocks with assertions

Example:

```hcl
run "test_new_scenario" {
  command = plan

  variables {
    hq_path = "/custom/path"
  }

  assert {
    condition     = gastown_hq.test.path == "/custom/path"
    error_message = "HQ path should match input variable"
  }
}
```
