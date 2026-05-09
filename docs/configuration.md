# Configuration

CIFence loads a policy file named `cifence.yml`, `cifence.yaml`, `.cifence.yml`, or `.cifence.yaml` from the scan root. Use `--config` to point to a different file.

## Schema

The JSON schema is `schemas/config.schema.json`.

```yaml
version: 1

severity:
  fail_on: high

rules:
  CF-PERM-002:
    enabled: true
    severity: medium
  CF-ACT-001:
    enabled: true
    allow:
      - actions/checkout@0123456789abcdef0123456789abcdef01234567

paths:
  include:
    - .github/workflows/*.yml
    - .github/workflows/*.yaml
  exclude:
    - .github/workflows/generated-*.yml

suppressions:
  - rule: CF-ACT-001
    path: .github/workflows/legacy.yml
    yaml_path: jobs.scan.steps[0].uses
    evidence: vendor/action@v1
    reason: "Vendor action has no immutable release yet"
    expires: "2026-07-01"
```

## Severity

`severity.fail_on` sets the default fail threshold for enforce mode. The CLI flag `--fail-on low|medium|high|critical` overrides the config value.

## Rules

Each rule can be disabled, assigned a different severity, or given an exact evidence allow-list. Unknown rule IDs fail config validation so typos do not silently weaken policy.

## Paths

Repository discovery scans files directly under `.github/workflows` by default. Nested paths are not treated as executable GitHub workflows during repository discovery because GitHub only loads workflows from the top-level workflows directory. Explicit file paths can still be scanned.

## Suppressions

Suppressions require:

- `rule`: known CIFence rule ID
- `path`: normalized report path
- `fingerprint`, or both `yaml_path` and `evidence`
- `reason`: human-readable justification
- `expires`: `YYYY-MM-DD`

Expired suppressions are reported as `CF-SUP-001` findings. Active suppressions remain visible in JSON with suppression metadata and do not fail enforce mode.

Rule and path alone are intentionally insufficient. A suppression must match the exact finding fingerprint, or the exact YAML path and evidence, so a later finding in the same file with the same rule is still reported as new.
Use `yaml_path: ""` for file-level findings that do not have a YAML node path.

## Baselines

Baselines are separate from suppressions:

```bash
cifence scan --baseline cifence.baseline.json --update-baseline
cifence scan --baseline cifence.baseline.json --mode enforce
```

Existing baseline findings are reported with `baseline_state: existing` and do not fail enforce mode. New findings are reported with `baseline_state: new` and fail when they meet the threshold.
