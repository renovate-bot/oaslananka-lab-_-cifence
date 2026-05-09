# GitHub Action

The action wraps the Go CLI and writes report files under `cifence-results/`. Normal marketplace usage uses a bundled CLI binary for the runner platform and does not require `actions/setup-go`.

Bundled binaries are packaged under:

- `dist/bin/linux-x64/cifence`
- `dist/bin/linux-arm64/cifence`
- `dist/bin/darwin-x64/cifence`
- `dist/bin/darwin-arm64/cifence`
- `dist/bin/win32-x64/cifence.exe`

The wrapper resolves the downloaded action directory with `GITHUB_ACTION_PATH` when GitHub provides it. When executed from the bundled JavaScript file directly, it derives the repository root from `dist/index.js`. The Go build fallback is reserved for local development and diagnostic recovery.

## Inputs

- `path`: repository or workflow path, default `.`
- `mode`: `warn` or `enforce`, default `warn`
- `fail-on`: fail threshold for enforce mode, default `high`
- `allow-outside-workspace`: allow absolute paths outside `GITHUB_WORKSPACE`, default `"false"`
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

## Workspace Boundary

By default, the Action resolves `path` inside `GITHUB_WORKSPACE` and rejects absolute paths outside that workspace. Self-hosted runner workflows that intentionally scan outside the checked-out repository must set `allow-outside-workspace: "true"`.
