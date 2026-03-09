# ============================================================================
# Mock Provider Unit Tests
# ============================================================================
# These tests use Terraform 1.7+ mock_provider to validate resource
# configurations and schema logic without requiring the gt CLI.
#
# mock_provider Requirements: Terraform >= 1.7.0
# Run with: terraform test
# ============================================================================

# Mock the gastown provider to simulate resource operations without
# actually calling the gt CLI or creating real Gas Town processes.
mock_provider "gastown" {
  # Mock gastown_hq resource - returns predictable values for computed fields
  mock_resource "gastown_hq" {
    defaults = {
      id   = "/home/user/gt-mock"
      name = "mock-town"
    }
  }

  # Mock gastown_rig resource - returns predictable values for computed fields
  mock_resource "gastown_rig" {
    defaults = {
      id     = "/home/user/gt-mock/test-rig"
      status = "operational"
      prefix = "test"
    }
  }

  # Mock gastown_crew resource - returns predictable values for computed fields
  mock_resource "gastown_crew" {
    defaults = {
      id = "/home/user/gt-mock/test-rig/test-crew"
    }
  }
}

# ============================================================================
# HQ Resource Tests
# ============================================================================

run "test_hq_defaults" {
  command = plan

  variables {
    hq_path = "/home/user/gt-mock"
  }

  # Test default values for optional attributes
  assert {
    condition     = gastown_hq.test.git == true
    error_message = "HQ git attribute should default to true"
  }

  assert {
    condition     = gastown_hq.test.no_beads == false
    error_message = "HQ no_beads attribute should default to false"
  }

  # Test computed ID matches path
  assert {
    condition     = gastown_hq.test.id == "/home/user/gt-mock"
    error_message = "HQ id should match the configured path"
  }

  assert {
    condition     = gastown_hq.test.name == "mock-town"
    error_message = "HQ name should be computed from mock"
  }
}

run "test_hq_owner_email" {
  command = plan

  variables {
    hq_path     = "/home/user/gt-mock"
    owner_email = "admin@example.com"
  }

  assert {
    condition     = gastown_hq.test.owner_email == "admin@example.com"
    error_message = "HQ owner_email should match input"
  }
}

# ============================================================================
# Rig Resource Tests
# ============================================================================

run "test_rig_defaults" {
  command = plan

  variables {
    hq_path = "/home/user/gt-mock"
    rig_name = "test-rig"
    rig_repo = "git@github.com:example/test.git"
  }

  # Test default runtime
  assert {
    condition     = gastown_rig.test.runtime == "claude"
    error_message = "Rig runtime should default to 'claude'"
  }

  # Test default max_polecats
  assert {
    condition     = gastown_rig.test.max_polecats == 3
    error_message = "Rig max_polecats should default to 3"
  }

  # Test computed values from mock
  assert {
    condition     = gastown_rig.test.status == "operational"
    error_message = "Rig status should be computed from mock"
  }

  assert {
    condition     = gastown_rig.test.prefix == "test"
    error_message = "Rig prefix should be computed from mock"
  }
}

run "test_rig_custom_values" {
  command = plan

  variables {
    hq_path      = "/home/user/gt-mock"
    rig_name     = "custom-rig"
    rig_repo     = "git@github.com:example/custom.git"
    runtime      = "gemini"
    max_polecats = 5
  }

  assert {
    condition     = gastown_rig.test.runtime == "gemini"
    error_message = "Rig runtime should be configurable"
  }

  assert {
    condition     = gastown_rig.test.max_polecats == 5
    error_message = "Rig max_polecats should be configurable"
  }
}

# ============================================================================
# Crew Resource Tests
# ============================================================================

run "test_crew_creation" {
  command = plan

  variables {
    hq_path   = "/home/user/gt-mock"
    rig_name  = "test-rig"
    crew_name = "test-crew"
    role      = "coder"
  }

  assert {
    condition     = gastown_crew.test.name == "test-crew"
    error_message = "Crew name should match input"
  }

  assert {
    condition     = gastown_crew.test.rig == "test-rig"
    error_message = "Crew rig should match input"
  }

  assert {
    condition     = gastown_crew.test.role == "coder"
    error_message = "Crew role should match input"
  }

  # Test computed ID
  assert {
    condition     = gastown_crew.test.id == "/home/user/gt-mock/test-rig/test-crew"
    error_message = "Crew id should be computed from mock"
  }
}

run "test_crew_roles" {
  command = plan

  variables {
    hq_path   = "/home/user/gt-mock"
    rig_name  = "test-rig"
    crew_name = "reviewer-crew"
    role      = "reviewer"
  }

  assert {
    condition     = gastown_crew.test.role == "reviewer"
    error_message = "Crew should support 'reviewer' role"
  }
}
