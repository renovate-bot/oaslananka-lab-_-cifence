# Repository Topology

CIFence uses two repositories with identical Git content and separate authority roles.

- Source and original repository: `oaslananka/cifence`
- CI/CD mirror repository: `oaslananka-lab/cifence`

The personal repository is the source of the project’s Git content. The organization repository mirrors that content and is the only repository where CI/CD, validation, security scanning, and release automation are intended to execute.

## Git Content

Branches, tags, and commits are Git objects. They can be mirrored with normal Git fetch and push operations. The required invariant is that `main`, active branches, and `v*` tags point to the same commits in both repositories.

## GitHub Metadata

Issues, pull requests, releases, labels, milestones, and comments are GitHub API objects. They are not mirrored by Git. `scripts/sync-repositories.mjs` reports read-only metadata differences first. Safe write sync requires explicit token setup, duplicate protection, and operator approval.

## Authority Boundary

CI/CD workflows are guarded with `github.repository == 'oaslananka-lab/cifence'`. If workflows exist in the personal repository, they should remain inert for CI/CD authority.
