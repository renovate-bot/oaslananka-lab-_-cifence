# GitHub Action

The action wraps the Go CLI and writes report files under `cifence-results/`. It uses a bundled CLI binary when present, otherwise it builds the local CLI once on the runner and executes that binary.

## Inputs

- `path`: repository or workflow path, default `.`
- `mode`: `warn` or `enforce`, default `warn`
- `sarif`: write SARIF, default `"true"`
- `json`: write JSON, default `"true"`
- `markdown`: write Markdown, default `"true"`
- `upload-sarif`: upload SARIF, default `"false"`

## Outputs

- `findings`
- `critical`
- `high`
- `medium`
- `low`
- `sarif-path`
- `json-path`
- `markdown-path`

## Permissions

Scanning needs `contents: read`. SARIF upload needs `security-events: write` and `GITHUB_TOKEN` in the environment.
