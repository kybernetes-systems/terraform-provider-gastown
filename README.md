# terraform-provider-gastown

The Wasteland doesn't run itself.

Someone has to hold the land, staff the rigs, and keep the polecats working. Someone has to write it down. If it isn't written, it isn't law. If it isn't law, it isn't land.

This provider is how Gas Town gets declared—in HCL, under version control, with no room for argument about what was supposed to be there.

---

## Getting Started

To manage your Gas Town infrastructure, configure the provider with the path to your HQ and define your resources.

```hcl
terraform {
  required_providers {
    gastown = {
      source = "kybernetes-systems/gastown"
    }
  }
}

provider "gastown" {
  # The root directory for your Gas Town installation
  hq_path = "/home/user/my-town"
}

# 1. Initialize the Town HQ
resource "gastown_hq" "main" {
  path        = "/home/user/my-town"
  owner_email = "mayor@example.com"
  git         = true
}

# 2. Add a Rig (a repository where work happens)
resource "gastown_rig" "research" {
  hq_path      = gastown_hq.main.path
  name         = "research-lab"
  repo         = "git@github.com:your-org/research-policy.git"
  runtime      = "claude"
  max_polecats = 3
}

# 3. Staff the Rig with Crew members (AI agents)
resource "gastown_crew" "lead_scientist" {
  hq_path = gastown_hq.main.path
  rig     = gastown_rig.research.name
  name    = "archimedes"
  role    = "researcher"
}
```

---

## Resources

### `gastown_hq`
The foundational workspace. Installing an HQ sets up the `mayor` and the underlying `dolt` database that tracks all town history.
- **`path`**: Absolute path to the HQ directory.
- **`owner_email`**: The primary maintainer's email address.
- **`git`**: (Optional) Initialize a Git repository in the HQ (default: `true`).
- **`no_beads`**: (Optional) Skip beads initialization (default: `false`).

### `gastown_rig`
A Rig represents a specific project or repository under management.
- **`hq_path`**: Path to the parent HQ.
- **`name`**: Unique name for the rig.
- **`repo`**: Git URL or local path to the rig's policy/logic repository.
- **`runtime`**: (Optional) The execution environment (default: `claude`).
- **`max_polecats`**: (Optional) Maximum number of concurrent workers (default: `3`).

### `gastown_crew`
Crew members are the AI agents assigned to a specific rig.
- **`hq_path`**: Path to the parent HQ.
- **`rig`**: Name of the rig the crew member belongs to.
- **`name`**: Name of the agent.
- **`role`**: The specialized role for this agent (e.g., `researcher`, `coder`, `reviewer`).

---

## Operational Safety & Isolation

This provider is designed for **strict surgical isolation**. It will not inadvertently modify or delete other Gas Town installations on your machine.

- **Explicit Scoping**: Every `gt` command is executed with `GT_TOWN_ROOT` set to the specific `hq_path` defined in your HCL. 
- **Path Validation**: All paths are strictly validated to be absolute and free of parent directory traversals (`..`).
- **Process Isolation**: In test environments, every command runs in a unique process group, ensuring that all spawned daemons are reliably terminated during cleanup.
- **Resource Locking**: Terraform only manages resources it has explicitly created or imported, using the specific paths recorded in its state file.

---

## Persistence & Deletion

When you destroy a `gastown_rig`, it is "docked." This means the active services stop, but the history remains in the ground (the `dolt` database). History is never truly deleted unless you `uninstall` the entire HQ.

See `docs/adr/0003-delete-means-dock.md` for the architectural rationale.

---

For development details, see [DEVELOPMENT.md](./DEVELOPMENT.md).
