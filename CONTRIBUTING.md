# Contributing

CIFence is a security-sensitive automation project. Changes should be small, reviewable, and backed by deterministic tests.

## Development Flow

1. Create a branch from `main`.
2. Run `pnpm install --frozen-lockfile`.
3. Run `task ci` before opening a pull request.
4. Use Conventional Commits for every commit.
5. Keep workflow changes pinned to full 40-character commit SHAs.

## Pull Requests

Pull requests should explain the behavior change, security impact, tests run, and any release impact. Do not include credentials, local notes, private operational text, or generated machine logs.

## Adding Rules

Add a rule definition, analyzer logic, fixture workflow, expected JSON, unit tests, SARIF coverage when needed, and documentation in `docs/rules.md`.
