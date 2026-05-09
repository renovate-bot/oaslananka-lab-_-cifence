# Contributing

CIFence is a security-sensitive automation project. Changes should be small, reviewable, and backed by deterministic tests.

## Development Flow

1. Create a branch from `main`.
2. Run `pnpm install --frozen-lockfile`.
3. Run `task ci` before opening a pull request.
4. Use Conventional Commits for every commit.
5. Keep workflow changes pinned to full 40-character commit SHAs.

Recommended branch names use a short type and topic, such as `feat/policy-config`, `test/add-analyzer-fixtures`, `security/expand-workflow-rules`, or `ci/harden-workflows`.

## Pull Requests

Pull requests should explain the behavior change, security impact, tests run, and any release impact. Do not include credentials, local notes, private operational text, or generated machine logs.

Release pull requests are managed by release-please from Conventional Commit history. Do not hand-edit release versions or changelog entries outside that flow.

## Adding Rules

Add a rule definition, analyzer logic, fixture workflow, expected JSON, unit tests, SARIF coverage when needed, and documentation in `docs/rules.md`.

## Local Checks

```bash
pnpm install --frozen-lockfile
pnpm run format:check
pnpm run lint
pnpm run typecheck
go test ./...
pnpm run fixtures
pnpm run smoke:action
pnpm run security
task ci
```
