# Release

CIFence uses release-please manifest mode. The personal repository `oaslananka/cifence` is the source repository, while `oaslananka-lab/cifence` runs CI/CD and release automation with the same git content.

## Version Source

Version state comes from:

- Conventional Commit history
- SemVer
- `release-please-config.json`
- `.release-please-manifest.json`
- release-please outputs

The bootstrap version was `0.1.0`; the current manifest version is controlled by release-please.

## Manual Version Inputs

Manual version inputs, manual tags, manual GitHub Releases, and manual changelog edits are not part of the release process.

## Artifacts

Release workflow scaffolding builds release artifacts in GitHub Actions after release-please creates a release in the organization repository. Tags and release metadata should be mirrored to the personal source repository so both repositories remain aligned.

Release jobs are guarded to run only in `oaslananka-lab/cifence`. The first bootstrap keeps release execution disabled unless the repository variable `CIFENCE_RELEASE_AUTOMATION_ENABLED` is explicitly set to `true`.

## Publishing

The release workflow does not publish to npm, PyPI, GHCR, DockerHub, VS Marketplace, Open VSX, MCP Registry, Homebrew, Scoop, or any other registry.
