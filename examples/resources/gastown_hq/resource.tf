resource "gastown_hq" "main" {
  path        = "/home/user/gt"
  owner_email = "user@example.com"
  git         = true
  no_beads    = false
}
