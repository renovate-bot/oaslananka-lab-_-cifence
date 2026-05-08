# Development

## Prerequisites

- Go 1.26.x
- Node.js 24.x
- pnpm 11.x
- Task 3.x

Enable pnpm through Corepack when needed:

```bash
corepack enable
corepack prepare pnpm@11.0.8 --activate
```

## Setup

```bash
pnpm install --frozen-lockfile
go mod download
```

## Commands

```bash
task format
task lint
task typecheck
task test
task action:build
task fixtures
task release:dry-run
task sync:check
task build
task ci
```

## Fixture Testing

Each workflow fixture in `tests/fixtures/workflows` has a matching deterministic JSON report in `tests/fixtures/expected`.

## Adding A Rule

1. Add the rule metadata in `internal/rules/definitions.go`.
2. Add detection logic under `internal/rules`.
3. Add unit tests and a workflow fixture.
4. Add or update the expected JSON report.
5. Document the rule in `docs/rules.md`.
