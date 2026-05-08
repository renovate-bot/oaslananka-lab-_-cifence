# Release State Machine

`scripts/release-state.mjs` inspects local and GitHub release state.

It checks:

- `package.json` version
- release-please manifest version
- changelog presence
- `v*` tags
- GitHub Releases where accessible
- open release PRs where accessible
- blockers
- next safe command

`safe_to_publish` is `false` by default for the first release candidate.
