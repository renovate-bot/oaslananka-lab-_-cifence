# CIFence

CIFence is a static security analyzer and policy engine for GitHub Actions workflows. It finds dangerous workflow patterns before they reach `main`, produces deterministic CLI output, and can run as a GitHub Action.

## Why It Exists

GitHub Actions workflows often control releases, credentials, package publishing, and repository write access. CIFence focuses on the high-risk configuration mistakes that are easy to miss in review: broad token permissions, mutable action references, unpinned actions, and unsafe `pull_request_target` checkout patterns.

## Install

```bash
go install github.com/oaslananka/cifence/cmd/cifence@latest
```

For local development from this repository:

```bash
go build ./cmd/cifence
```

## CLI Quickstart

```bash
cifence scan
cifence scan --path .
cifence scan --format json
cifence scan --format sarif
cifence scan --format markdown
cifence scan --sarif cifence.sarif
cifence scan --json cifence.json
cifence scan --markdown cifence.md
cifence scan --mode warn
cifence scan --mode enforce
cifence rules
cifence version
```

`warn` mode exits `0` even with findings. `enforce` mode exits non-zero when high or critical findings are present.

## GitHub Action Quickstart

```yaml
name: CIFence

on:
  pull_request:
  push:
    branches: [main]

permissions:
  contents: read
  security-events: write

concurrency:
  group: ${{ github.workflow }}-${{ github.event.pull_request.number || github.ref }}
  cancel-in-progress: ${{ github.event_name == 'pull_request' }}

jobs:
  cifence:
    runs-on: ubuntu-24.04
    timeout-minutes: 10
    steps:
      - uses: actions/checkout@<FULL_SHA>
      - uses: oaslananka/cifence@<FULL_SHA_OR_VERSION_AFTER_RELEASE>
        env:
          GITHUB_TOKEN: ${{ github.token }}
        with:
          mode: warn
          sarif: "true"
          upload-sarif: "true"
```

## Rules

| Rule        | Severity | Purpose                                                                       |
| ----------- | -------- | ----------------------------------------------------------------------------- |
| CF-PERM-001 | critical | Detects `permissions: write-all`.                                             |
| CF-PERM-002 | medium   | Detects missing explicit workflow or job permissions.                         |
| CF-ACT-001  | medium   | Detects remote actions not pinned to full commit SHAs.                        |
| CF-ACT-002  | high     | Detects mutable refs and Docker `latest` tags.                                |
| CF-TRG-001  | critical | Detects unsafe `pull_request_target` checkout of untrusted pull request code. |

## Output Formats

CIFence writes Markdown, JSON, and SARIF 2.1.0. JSON and SARIF are deterministic and contain no timestamps or machine-specific paths in fixture tests.

## SARIF And Code Scanning

SARIF upload is off by default. The action uploads SARIF only when `upload-sarif: "true"` is set and a token with `security-events: write` is provided.

## Security And Privacy

CIFence does not execute workflow steps, run arbitrary scripts from the scanned repository, require cloud credentials, or upload repository data by default. It parses YAML as untrusted input and reports only the evidence needed to explain each finding.

## Current Limitations

- The first release focuses on GitHub Actions workflow files under `.github/workflows`.
- Policy configuration is reserved for `cifence.yml` in a future release; the first release uses the built-in rule set.
- It does not rewrite workflows or apply automatic remediation.
- It does not implement a GitHub App server or hosted dashboard.

## Development

```bash
pnpm install --frozen-lockfile
task ci
```

See `docs/development.md` and `docs/testing.md` for the full local workflow.

## Repository And Release Model

The personal repository `oaslananka/cifence` is the source and original repository. The organization repository `oaslananka-lab/cifence` mirrors the same Git content and runs CI/CD workflows because organization Actions are the reliable execution environment. Branches, tags, and commits sync through Git. Issues, pull requests, releases, labels, milestones, and comments require GitHub API based sync and are reported by `scripts/sync-repositories.mjs` before any future write sync is enabled.

Release-please manages version state from Conventional Commit history. Package and container publishing are not enabled.

## Support

Use GitHub issues for non-sensitive bugs and feature requests. Use private vulnerability reporting for security issues.
