terraform {
  required_providers {
    gastown = {
      source = "kybernetes-systems/gastown"
    }
  }
}

provider "gastown" {
  hq_path = "/path/to/gt/hq"
}
