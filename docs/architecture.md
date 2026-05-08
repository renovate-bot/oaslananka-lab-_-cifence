# Architecture

CIFence has two runtime surfaces with one scanner core.

## Go Scanner

The Go scanner owns workflow discovery, YAML parsing, rule execution, exit-code behavior, and report generation. It uses `gopkg.in/yaml.v3` so findings can point at source line and column locations.

## TypeScript Wrapper

The GitHub Action wrapper is a Node 24 JavaScript action compiled from TypeScript. It runs the local Go CLI from the checked-out action path, writes deterministic report files under `cifence-results/`, sets action outputs, writes the job summary, and uploads SARIF only when explicitly enabled.

## Output Pipeline

The analyzer returns a normalized report containing summary counts and ordered findings. JSON, Markdown, and SARIF renderers consume that report without adding timestamps or host-specific paths.

## Rule Engine

Rules are independent functions over the YAML AST. Parser code does not classify security behavior, and report code does not inspect YAML.

## Parser Strategy

CIFence treats repository files as untrusted input. Invalid YAML becomes a diagnostic finding rather than a crash.

## Future GitHub App Path

The scanner and report formats are stable enough to support a future GitHub App, but no server, billing, webhook processor, or hosted storage is included in the first release candidate.
