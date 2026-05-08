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
cifence version
cifence rules
cifence help
```

## Exit Codes

- `0`: success, including findings in `warn` mode
- `1`: scan or enforcement failure
- `2`: invalid arguments

## Discovery

When scanning a directory, CIFence discovers `.github/workflows/*.yml` and `.github/workflows/*.yaml`.

## Reports

Default output is a Markdown summary to stdout. `--json`, `--sarif`, and `--markdown` write deterministic reports to the provided paths. The GitHub Action uses `cifence-results/` with `cifence.json`, `cifence.sarif`, and `cifence.md`.
