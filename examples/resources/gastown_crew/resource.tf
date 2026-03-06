resource "gastown_hq" "main" {
  path = "/home/user/gt"
}

resource "gastown_rig" "coder" {
  hq_path = gastown_hq.main.path
  name    = "coder"
  repo    = "https://github.com/user/coder-rig.git"
}

resource "gastown_crew" "helper" {
  hq_path = gastown_hq.main.path
  rig     = gastown_rig.coder.name
  name    = "helper"
  role    = "coder"
}
