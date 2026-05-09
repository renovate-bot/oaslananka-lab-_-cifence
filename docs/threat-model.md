# Threat Model

## Assets

- GitHub workflow definitions
- repository tokens and workflow permissions
- release workflow integrity
- SARIF, JSON, and Markdown reports
- release binaries, checksums, SBOM, and attestations

## Trust Boundaries

- The scanned repository is untrusted input.
- GitHub event payload fields such as pull request titles, issue bodies, comments, and branch names are untrusted.
- Third-party actions and reusable workflows are external code boundaries.
- Release assets are trusted only when built by the organization GitHub Actions workflow from the released tag.

## Assumptions

- CIFence parses workflow YAML locally and does not execute workflow steps.
- The CLI does not make network calls by default.
- The GitHub Action uses bundled binaries for normal usage and falls back to source build only for development or diagnostic recovery.
- SARIF upload occurs only when explicitly enabled by workflow configuration.

## Primary Risks

- unpinned or mutable actions and reusable workflows
- overly broad `GITHUB_TOKEN` permissions
- privileged `pull_request_target` workflows using untrusted PR data
- script injection through untrusted GitHub context interpolation
- mutable container and service images
- inherited reusable workflow secrets
- stale suppressions hiding real findings

## Controls

- static detection rules with stable IDs and remediation
- deterministic report output and stable fingerprints
- policy config validation with unknown-rule rejection
- required suppression reasons and expiry dates
- baseline mode that distinguishes existing and new findings
- workspace boundary checks in the GitHub Action wrapper
- SHA-pinned CI actions and explicit workflow permissions
- release-please managed versions and CI-built release assets

## Non-Goals

CIFence does not execute workflows, mutate repositories, run arbitrary repository scripts, start workflow runs, manage secrets, or publish packages. It reports findings so maintainers can make explicit review decisions.
