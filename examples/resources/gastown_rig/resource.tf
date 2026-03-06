resource "gastown_hq" "main" {
  path        = "/home/user/gt"
  owner_email = "user@example.com"
}

resource "gastown_rig" "coder" {
  hq_path      = gastown_hq.main.path
  name         = "coder"
  repo         = "https://github.com/user/coder-rig.git"
  runtime      = "claude"
  max_polecats = 3
}
