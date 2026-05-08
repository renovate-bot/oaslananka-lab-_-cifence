# Rules

## CF-PERM-001: permissions: write-all is used

- Severity: critical
- Detection: workflow-level or job-level `permissions: write-all`
- Example:

```yaml
permissions: write-all
```

- Remediation: replace `write-all` with explicit least-privilege permissions.
- False-positive notes: intended emergency workflows should still use explicit scoped permissions.

## CF-PERM-002: missing explicit permissions block

- Severity: medium
- Detection: workflows without workflow-level permissions and jobs without job-level permissions.
- Example:

```yaml
jobs:
  test:
    runs-on: ubuntu-24.04
```

- Remediation: add explicit permissions, starting from `contents: read`.
- False-positive notes: inherited default token permissions are intentionally reported.

## CF-ACT-001: action reference is not pinned to a full commit SHA

- Severity: medium
- Detection: remote GitHub Actions references using tags or branches instead of a 40-character commit SHA, and Docker action references without an immutable digest.
- Example:

```yaml
- uses: actions/checkout@v6
- uses: docker://alpine:3.20
```

- Remediation: pin GitHub actions to a full commit SHA and Docker actions to an immutable digest.
- False-positive notes: local actions such as `./path/to/action` are ignored.

## CF-ACT-002: mutable action reference is used

- Severity: high
- Detection: mutable refs such as `main`, `master`, `dev`, `develop`, `trunk`, `latest`, `HEAD`, and Docker `latest` tags.
- Example:

```yaml
- uses: actions/checkout@main
- uses: docker://alpine:latest
```

- Remediation: pin GitHub actions to full SHAs and Docker images to immutable digests.
- False-positive notes: this rule supersedes CF-ACT-001 for known mutable refs.

## CF-TRG-001: pull_request_target checks out untrusted pull request code

- Severity: critical
- Detection: `pull_request_target` workflows where checkout is configured with untrusted pull request head refs, head SHAs, or `refs/pull/` values.
- Example:

```yaml
on:
  pull_request_target:

steps:
  - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd
    with:
      ref: ${{ github.event.pull_request.head.sha }}
```

- Remediation: use `pull_request` with read-only permissions, or avoid checking out and executing the untrusted head while write token or secrets are available.
- False-positive notes: safe `pull_request_target` workflows that do not checkout untrusted head code are not reported.
