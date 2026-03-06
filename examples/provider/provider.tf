terraform {
  required_version = ">= 1.7"

  required_providers {
    gastown = {
      source  = "kybernetes-systems/gastown"
      version = ">= 0.1.0"
    }
  }
}

provider "gastown" {
  hq_path = "/path/to/gt/hq"
}
