---

# terraform-provider-gastown

The Wasteland doesn't run itself.

Someone has to hold the land, staff the rigs, and keep the polecats working. Someone has to write it down. If it isn't written, it isn't law. If it isn't law, it isn't land.

This provider is how Gas Town gets declared--in HCL, under version control, with no room for argument about what was supposed to be there.

```hcl
resource "gastown_rig" "kybernetes" {
  name = "kybernetes"
  repo = "git@github.com:kybernetes/policy.git"
}

resource "gastown_crew" "mayor" {
  name = "mayor"
  rig  = gastown_rig.kybernetes.name
}
```

When you destroy a rig, it goes dark. It does not disappear. The history stays in the ground. That was a decision. See `docs/adr/0003-delete-means-dock.md`.

The polecats are not your problem. Don't touch them from the tests.

---

## Operational Safety & Isolation

This provider is designed for **strict surgical isolation**. It will not inadvertently modify or delete other Gas Town installations on your machine.

- **Explicit Scoping**: Every `gt` command is executed with `GT_TOWN_ROOT` set to the specific `hq_path` defined in your HCL. 
- **Path Validation**: All paths are strictly validated to be absolute and free of parent directory traversals (`..`).
- **Process Isolation**: In test environments, every command runs in a unique process group, ensuring that all spawned daemons are reliably terminated during cleanup.
- **Resource Locking**: Terraform only manages resources it has explicitly created or imported, using the specific paths recorded in its state file.

For more details on the architecture and safety measures, see [DEVELOPMENT.md](./DEVELOPMENT.md).
