# Minimal Gas Town HQ (Mayor-only)
# This example demonstrates the simplest possible Gas Town setup:
# Just a Town HQ with no Rigs or Crew members.

terraform {
  required_providers {
    gastown = {
      source = "kybernetes-systems/gastown"
    }
  }
}

provider "gastown" {
  hq_path = "/home/user/gt-minimal"
}

# The Town HQ is the only required resource.
# Once created, the Mayor is active and the workspace is ready for manual use.
resource "gastown_hq" "main" {
  path        = "/home/user/gt-minimal"
  owner_email = "user@example.com"
}
