# Security Policy

CIFence treats workflow files as untrusted input and does not execute target repository workflow code.

## Reporting

Do not report suspected vulnerabilities in a public issue. Use GitHub private vulnerability reporting when available. If private reporting is unavailable, contact the repository owner through GitHub before sharing details.

## Supported Version

The initial supported line is `0.1.x` once the first release is published.

## Security Model

- Default scanning is local-only and offline.
- SARIF upload is disabled unless explicitly enabled.
- Repository contents are not sent to external services by the CLI.
- Findings avoid printing secret values and focus on workflow structure, action references, and permissions.
