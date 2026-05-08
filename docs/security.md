# Security

CIFence is designed for local analysis of workflow configuration.

## No External Upload By Default

The CLI does not make network calls and does not upload repository data. The action uploads SARIF only when `upload-sarif` is set to `"true"` and a token is provided.

## Secret Handling

The scanner treats the scanned repository as untrusted input. It does not mutate repository files, execute workflow code, start arbitrary workflow runs, or run arbitrary repository scripts. Findings report structural workflow evidence such as action references, permissions, and checkout refs. Runtime diagnostics redact token-like values before reporting fallback build failures.

## GitHub Token Permissions

Use the smallest permissions possible. For SARIF upload, GitHub code scanning requires `security-events: write`. Normal scanning requires only `contents: read`.

## Fork Pull Request Safety

Do not use `pull_request_target` to checkout and run untrusted pull request code. CIFence reports known unsafe checkout patterns for that event.

## SARIF Behavior

SARIF files are written locally unless upload is explicitly enabled. SARIF upload should be used only in trusted repository workflows with explicit permissions.

## Secure Release Model

Release automation is prepared with release-please and Conventional Commits. Public tags are immutable after publication. The `v0.1.0` tag and release must not be rewritten. Package publishing and container publishing are not part of the release process.

## Repository Sync Token Boundary

Git branch, tag, and commit sync is handled with Git. GitHub metadata sync for issues, pull requests, releases, labels, milestones, and comments requires API access. The first implementation reports metadata drift only; conservative write sync requires an explicitly approved `PERSONAL_REPO_SYNC_TOKEN` with the smallest practical scopes and duplicate-protection logic.
