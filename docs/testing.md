# Testing

## Unit Tests

Go tests cover workflow discovery, YAML parsing, permissions detection, action reference parsing, mutable ref detection, unsafe `pull_request_target` checkout detection, invalid YAML diagnostics, SARIF generation, JSON stability, Markdown stability, and CLI exit codes.

```bash
go test ./...
```

## Fixtures

Fixture validation compares exact JSON output with golden files:

```bash
pnpm run fixtures
```

## SARIF Validation

SARIF output is generated as version 2.1.0 and includes rule metadata, artifact locations, and source regions.

```bash
cifence scan --format sarif
```

## Action Smoke Tests

The GitHub Action wrapper is bundled with esbuild, validates `action.yml`, and runs through repository CI.

```bash
pnpm run build
node scripts/validate-action-metadata.mjs
```

## Local CI

```bash
task ci
```
