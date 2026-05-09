# Security Policy

CIFence treats workflow files as untrusted input and does not execute target repository workflow code.

## Reporting

Do not report suspected vulnerabilities in a public issue. Use GitHub private vulnerability reporting when available. If private reporting is unavailable, contact the repository owner through GitHub before sharing details.

## Supported Versions

The supported release line is the latest published SemVer release. Security fixes are released forward; published tags are not rewritten.

## Security Model

- Default scanning is local-only and offline.
- SARIF upload is disabled unless explicitly enabled.
- Repository contents are not sent to external services by the CLI.
- Findings avoid printing secret values and focus on workflow structure, action references, and permissions.

## Scope

In scope:

- analyzer crashes or bypasses that hide expected findings
- unsafe GitHub Action wrapper path handling
- release asset integrity issues
- secret exposure in diagnostics or reports

Out of scope:

- findings from intentionally vulnerable test fixtures
- third-party service outages
- requests to mutate published tags or release assets
