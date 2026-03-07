# Standard Gas Town Workspace Example
# This example demonstrates a complete Gas Town setup including an HQ, 
# multiple Rigs, and several Crew members with different roles.

terraform {
  required_providers {
    gastown = {
      source = "kybernetes-systems/gastown"
    }
  }
}

provider "gastown" {
  # Base directory for the Gas Town installation
  hq_path = "/home/user/gt"
}

# 1. Initialize the Town HQ
resource "gastown_hq" "main" {
  path        = "/home/user/gt"
  owner_email = "user@example.com"
  git         = true
  no_beads    = false
}

# 2. Add an Engineering Rig
resource "gastown_rig" "engineering" {
  hq_path      = gastown_hq.main.path
  name         = "engineering"
  repo         = "git@github.com:kybernetes-systems/gastown-engine.git"
  runtime      = "claude"
  max_polecats = 5
}

# 3. Add an Ops Rig
resource "gastown_rig" "ops" {
  hq_path      = gastown_hq.main.path
  name         = "ops"
  repo         = "git@github.com:kybernetes-systems/gastown-ops.git"
  runtime      = "gemini"
  max_polecats = 3
}

# 4. Staff the Engineering Rig
resource "gastown_crew" "lead_dev" {
  hq_path = gastown_hq.main.path
  rig     = gastown_rig.engineering.name
  name    = "lead-dev"
  role    = "architect"
}

resource "gastown_crew" "qa_engineer" {
  hq_path = gastown_hq.main.path
  rig     = gastown_rig.engineering.name
  name    = "qa-tester"
  role    = "reviewer"
}

# 5. Staff the Ops Rig
resource "gastown_crew" "ops_lead" {
  hq_path = gastown_hq.main.path
  rig     = gastown_rig.ops.name
  name    = "ops-lead"
  role    = "operator"
}
