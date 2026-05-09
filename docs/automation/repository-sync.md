# Repository Sync

CIFence uses two repositories with the same Git content:

- Personal source repository: `oaslananka/cifence`
- Organization CI/CD repository: `oaslananka-lab/cifence`

The personal repository is the source repository. The organization repository exists so CI/CD, code scanning, release automation, and security checks can run in the organization environment.

## Required Invariants

- `main` must point to the same commit in both repositories.
- Active branches must point to the same commits in both repositories.
- `v*` tags must point to the same commits in both repositories.
- Release metadata, issues, pull requests, labels, milestones, and comments require GitHub API sync where safe.
- Workflow authority should run only in `oaslananka-lab/cifence`.
- Personal repository Actions may be unavailable; workflows must remain safe when present there.

## Script Behavior

`scripts/sync-repositories.mjs --check` prints a sync plan. It compares Git refs through `git ls-remote` and reports read-only metadata differences through the GitHub CLI when available.

`scripts/sync-repositories.mjs --apply` can apply safe missing Git ref updates from the personal source repository to the organization mirror. It refuses divergent refs because release tags and public branches must not be rewritten.

The workflow `.github/workflows/sync-from-personal.yml` runs only in `oaslananka-lab/cifence`. It never publishes, creates releases, auto-merges, auto-approves, rewrites divergent refs, or deletes unknown target refs.

## Metadata Sync Blocker

GitHub metadata write sync is intentionally not enabled in the first implementation. A future conservative writer needs `PERSONAL_REPO_SYNC_TOKEN`, exact scope review, and duplicate-prevention logic for issues, pull requests, releases, labels, milestones, and comments.

## Validation

Use:

```bash
git ls-remote --heads https://github.com/oaslananka/cifence.git
git ls-remote --heads https://github.com/oaslananka-lab/cifence.git
git ls-remote --tags https://github.com/oaslananka/cifence.git
git ls-remote --tags https://github.com/oaslananka-lab/cifence.git
node scripts/sync-repositories.mjs --check
node scripts/release-state.mjs
```
