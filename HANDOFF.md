# Handoff

## Current Status
We have worked on Phase 6: Integration & acceptance suite (`tfgt-39c`).

The unit tests for resources and provider configuration have been completed and are passing.
However, the Terraform acceptance tests (run with `TF_ACC=1 go test ./internal/provider/... -run TestAcc -v`) are currently failing.

## Blockers / Next Steps
The main issue blocking the acceptance tests is that the `gastown_rig` resource under test calls `gt rig add <name> <repo-url>`, but the real `gt` CLI validation rejects local `file://` repository schemas (it only accepts remote URLs like `https://`, `git@`, etc.).

To resolve this, the next agent needs to:
1. Figure out a way to bypass the `gt rig add` git URL validation during acceptance tests. This could involve:
    - Setting up a more robust `gt` mock script in the test setup that perfectly intercepts the `gt rig add` command.
    - Hosting a local `git` daemon (HTTP/SSH) during the tests to provide a valid remote URL structure.
    - Modifying the underlying `gt` validation logic (if possible/applicable) to support local testing repositories.
2. Complete the full lifecycle, drift, and concurrency acceptance tests.

## Available Work
- Issue `tfgt-39c`: Phase 6: Integration & acceptance suite (IN PROGRESS)
- Continue working on the acceptance test suite and resolving the Git URL blocker.