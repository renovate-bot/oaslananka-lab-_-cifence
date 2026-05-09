# CIFence

CIFence is a static security analyzer and policy engine for GitHub Actions workflows. It finds dangerous workflow patterns before they reach `main`, produces deterministic CLI output, and can run as a GitHub Action.

## Why It Exists

GitHub Actions workflows often control releases, credentials, package publishing, and repository write access. CIFence focuses on the high-risk configuration mistakes that are easy to miss in review: broad token permissions, mutable action references, unpinned actions, and unsafe `pull_request_target` checkout patterns.

## Install

```bash
go install github.com/oaslananka/cifence/cmd/cifence@v0.1.1
```

Go is only required for local source development. Normal GitHub Action usage runs the bundled CLI binary for the runner platform.

For local development from this repository:

```bash
pnpm install --frozen-lockfile
pnpm run action:build
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
cifence scan --mode enforce --fail-on medium
cifence scan --baseline cifence.baseline.json --update-baseline
cifence scan --baseline cifence.baseline.json --mode enforce
cifence rules
cifence version
```

`warn` mode exits `0` even with findings. `enforce` mode exits non-zero when a non-suppressed, non-baselined finding meets the fail threshold. The default fail threshold is `high`; `--fail-on` or `cifence.yml` can set `low`, `medium`, `high`, or `critical`.

## Policy Configuration

CIFence loads `cifence.yml`, `cifence.yaml`, `.cifence.yml`, or `.cifence.yaml` from the scan root when present.

```yaml
version: 1

severity:
  fail_on: high

rules:
  CF-ACT-001:
    enabled: true
    severity: medium
    allow:
      - actions/checkout@0123456789abcdef0123456789abcdef01234567

paths:
  include:
    - .github/workflows/*.yml
    - .github/workflows/*.yaml
  exclude:
    - .github/workflows/generated-*.yml

suppressions:
  - rule: CF-ACT-001
    path: .github/workflows/legacy.yml
    reason: "Vendor action has no immutable release yet"
    expires: "2026-07-01"
```

Suppression `reason` and `expires` are required. Expired suppressions are reported as findings. See `docs/configuration.md` and `schemas/config.schema.json`.

## Baselines

Baselines help existing repositories adopt enforce mode incrementally.

```bash
cifence scan --baseline cifence.baseline.json --update-baseline
cifence scan --baseline cifence.baseline.json --mode enforce
```

Existing baseline findings remain visible in reports but do not fail enforce mode. New findings still fail when they meet the configured threshold. Baseline entries use stable fingerprints derived from rule ID, normalized path, evidence, and YAML path.

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
      - uses: actions/checkout@<FULL_40_CHARACTER_SHA_FOR_CHECKOUT_V6>
      - uses: oaslananka/cifence@v0.1.1
        env:
          GITHUB_TOKEN: ${{ github.token }}
        with:
          mode: warn
          sarif: "true"
          upload-sarif: "true"
```

## Rules

CIFence detects broad token permissions, unpinned actions and reusable workflows, mutable refs, script injection from untrusted GitHub context, unsafe `pull_request_target` behavior, unpinned container images, and inherited reusable-workflow secrets. See `docs/rules.md` for the full rule catalog and remediation guidance.

## Output Formats

CIFence writes Markdown, JSON, and SARIF 2.1.0. JSON, Markdown, and SARIF are deterministic and contain no timestamps or machine-specific absolute paths in golden fixture tests.

## SARIF And Code Scanning

SARIF upload is off by default. The action uploads SARIF only when `upload-sarif: "true"` is set and a token with `security-events: write` is provided. SARIF includes rule metadata, remediation, semantic version, snippets, rule indexes, and partial fingerprints for GitHub Code Scanning.

## Security And Privacy

CIFence does not execute workflow steps, run arbitrary scripts from the scanned repository, require cloud credentials, or upload repository data by default. It parses YAML as untrusted input and reports only the evidence needed to explain each finding.

The GitHub Action resolves its own downloaded action directory before looking for packaged binaries, so a consumer repository does not need to install Go or contain CIFence source code. The action rejects absolute scan paths outside `GITHUB_WORKSPACE` unless `allow-outside-workspace: "true"` is explicitly set.

## Current Scope

- Repository scans discover workflow files directly under `.github/workflows`.
- Nested files such as `.github/workflows/sub/example.yml` are not treated as executable GitHub workflows during repository discovery.
- Explicit file paths can still be scanned for diagnostics.
- CIFence reports findings only; it does not rewrite workflows or apply automatic remediation.

## Development

```bash
pnpm install --frozen-lockfile
task ci
```

See `docs/development.md` and `docs/testing.md` for the full local workflow.

## Repository And Release Model

The personal repository `oaslananka/cifence` is the source and original repository. The organization repository `oaslananka-lab/cifence` mirrors the same Git content and runs CI/CD workflows because organization Actions are the reliable execution environment. Branches, tags, and commits sync through Git. Issues, pull requests, releases, labels, milestones, and comments require GitHub API based sync and are reported by `scripts/sync-repositories.mjs` before any future write sync is enabled.

Release-please manifest mode manages version state from Conventional Commit history. The organization release workflow creates GitHub Releases only when release-please reports `release_created == true`, then attaches CI-built binaries, checksums, SBOM, and attestations. Package and container publishing are not enabled.

## Support

Use GitHub issues for non-sensitive bugs and feature requests. Use private vulnerability reporting for security issues.
