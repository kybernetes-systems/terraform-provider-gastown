# Test fixture for mock_provider unit tests
# This file provides the resource configurations used by tests/unit_mock_test.tftest.hcl

terraform {
  required_version = ">= 1.5.0"

  required_providers {
    gastown = {
      source = "kybernetes-systems/gastown"
    }
  }
}

variable "hq_path" {
  type    = string
  default = "/home/user/gt-test"
}

variable "owner_email" {
  type    = string
  default = null
}

variable "rig_name" {
  type    = string
  default = "test-rig"
}

variable "rig_repo" {
  type    = string
  default = "git@github.com:example/test.git"
}

variable "runtime" {
  type    = string
  default = "claude"
}

variable "max_polecats" {
  type    = number
  default = 3
}

variable "crew_name" {
  type    = string
  default = "test-crew"
}

variable "role" {
  type    = string
  default = "coder"
}

provider "gastown" {
  hq_path = var.hq_path
}

# HQ resource for testing
resource "gastown_hq" "test" {
  path        = var.hq_path
  owner_email = var.owner_email
}

# Rig resource for testing
resource "gastown_rig" "test" {
  hq_path      = var.hq_path
  name         = var.rig_name
  repo         = var.rig_repo
  runtime      = var.runtime
  max_polecats = var.max_polecats
}

# Crew resource for testing
resource "gastown_crew" "test" {
  hq_path = var.hq_path
  rig     = var.rig_name
  name    = var.crew_name
  role    = var.role
}
