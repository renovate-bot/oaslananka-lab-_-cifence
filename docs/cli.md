# CLI

## Commands

```bash
cifence scan [path]
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
cifence scan --config cifence.yml
cifence scan --baseline cifence.baseline.json --update-baseline
cifence scan --baseline cifence.baseline.json --mode enforce
cifence version
cifence rules
cifence help
```

## Exit Codes

- `0`: success, including findings in `warn` mode
- `1`: scan or enforcement failure
- `2`: invalid arguments

## Discovery

When scanning a directory, CIFence discovers workflow files directly under `.github/workflows` with `.yml` or `.yaml` extensions. Nested files under subdirectories are not treated as executable GitHub workflows during repository discovery. Explicit file paths can still be scanned.

## Reports

Default output is a Markdown summary to stdout. `--json`, `--sarif`, and `--markdown` write deterministic reports to the provided paths. The GitHub Action uses `cifence-results/` with `cifence.json`, `cifence.sarif`, and `cifence.md`.

## Fail Thresholds

`--mode enforce` fails on high and critical findings by default. Use `--fail-on low|medium|high|critical` or `severity.fail_on` in `cifence.yml` to change that threshold. Active suppressions and existing baseline findings do not fail enforce mode.
