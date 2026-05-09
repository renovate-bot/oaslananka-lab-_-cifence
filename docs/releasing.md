# Releasing

CIFence uses release-please manifest mode. Version numbers come from Conventional Commit history, SemVer, `release-please-config.json`, `.release-please-manifest.json`, and release-please workflow outputs.

## Release Flow

1. Merge code changes with Conventional Commits.
2. The organization workflow in `oaslananka-lab/cifence` runs `googleapis/release-please-action` in manifest mode.
3. Release-please opens or updates a release pull request with version and changelog changes.
4. The release workflow refreshes committed Action distribution files on that release pull request.
5. After the release pull request is merged, release-please creates the GitHub Release.
6. The release-assets job builds binaries from a clean checkout of the released tag and uploads assets to the GitHub Release.

## Generated Assets

Each release includes:

- `cifence-linux-amd64`
- `cifence-linux-arm64`
- `cifence-darwin-amd64`
- `cifence-darwin-arm64`
- `cifence-windows-amd64.exe`
- `checksums.txt`
- `cifence-sbom.spdx.json`
- `provenance.json`
- GitHub artifact attestations for checksums and SBOM

Release assets are built in GitHub Actions only. Local machine artifacts are never uploaded.

## Tag Policy

Published `v*` tags are immutable. Do not move, delete, or recreate public release tags. Do not rewrite `v0.1.0` or any later published tag. If a published release has a defect, fix forward with a new SemVer release.

## Publishing Boundary

The release workflow does not publish to npm, PyPI, GHCR, DockerHub, Homebrew, Scoop, VS Marketplace, Open VSX, or any package registry. CIFence is distributed as a GitHub Action and as GitHub Release assets.

## Repository Boundary

The personal repository `oaslananka/cifence` is the source repository. The organization repository `oaslananka-lab/cifence` carries the same Git content and runs CI/CD and release automation. Git refs must remain aligned; metadata sync is documented separately in `docs/automation/repository-sync.md`.
