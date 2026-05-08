# Changelog

All notable changes after the initial 0.1.0 bootstrap are managed by release-please.

## 0.1.1

Hotfix release.

Fixes:

- Fixes CIFence Action execution in clean consumer repositories.
- Resolves action root path detection.
- Bundles prebuilt CIFence CLI binaries so normal Action users do not need Go.
- Improves fallback build diagnostics.

Usage:

```yaml
- uses: oaslananka/cifence@v0.1.1
  with:
    mode: warn
```

No package registry publish was performed.

## 0.1.0

- Initial release candidate baseline for the CIFence CLI, GitHub Action wrapper, static analysis rules, reports, fixtures, and repository automation.
