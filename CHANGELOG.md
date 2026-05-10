# Changelog

All notable changes after the initial 0.1.0 bootstrap are managed by release-please.

## [0.2.2](https://github.com/oaslananka-lab/cifence/compare/v0.2.1...v0.2.2) (2026-05-10)


### Bug Fixes

* **build:** use linux canonical binary freshness ([b9d2bf0](https://github.com/oaslananka-lab/cifence/commit/b9d2bf01aa3e7796cefcd96608e46a6fd24d2339))
* **sarif:** exclude suppressed findings from uploads ([4e60e59](https://github.com/oaslananka-lab/cifence/commit/4e60e5991e2fa62b36a8c0d84942dc243f1e652f))

## [0.2.1](https://github.com/oaslananka-lab/cifence/compare/v0.2.0...v0.2.1) (2026-05-10)


### Bug Fixes

* **action:** align fallback CLI build with release flags ([25924f2](https://github.com/oaslananka-lab/cifence/commit/25924f23933847e5124fa66d67b3cccb52293586))
* **ci:** stabilize fuzz and review-thread gates ([09682fd](https://github.com/oaslananka-lab/cifence/commit/09682fd2153c68d4455f47e2ced8386243828ea1))
* **config:** normalize suppression evidence ([f53cb9a](https://github.com/oaslananka-lab/cifence/commit/f53cb9abe2594abb34a6e6524ecf30ebf39018a3))
* **config:** support explicit file-level suppressions ([8bdd112](https://github.com/oaslananka-lab/cifence/commit/8bdd1129562ae9b4c18eab932ba25dd98f7466b9))
* **rules:** handle sequence workflow sinks ([a09371a](https://github.com/oaslananka-lab/cifence/commit/a09371a79fa9a814de24f230864a259a70e1c294))
* **rules:** harden workflow analyzer coverage ([358a17d](https://github.com/oaslananka-lab/cifence/commit/358a17d27a3e488c94730c1a919f9f0f6456a6ff))

## [0.2.0](https://github.com/oaslananka-lab/cifence/compare/v0.1.1...v0.2.0) (2026-05-09)


### Features

* **analyzer:** add policy config and baseline support ([6846327](https://github.com/oaslananka-lab/cifence/commit/684632720563cf3cd7707e70b5d88c3f2f87eee4))
* **rules:** expand workflow security detections ([f6fd8b7](https://github.com/oaslananka-lab/cifence/commit/f6fd8b7ea46b8f88bed9a7f0afa5436f0e7703ac))


### Bug Fixes

* **action:** enforce workspace boundary and versioned packaging ([c727c7c](https://github.com/oaslananka-lab/cifence/commit/c727c7c6c189cdaf24570a95115fb8a20a5cb1d4))
* **build:** make packaged binaries deterministic ([e1e593a](https://github.com/oaslananka-lab/cifence/commit/e1e593a8efff6a2a4bd09169edb866a49552da9a))
* **ci:** disable release job caches ([93884f7](https://github.com/oaslananka-lab/cifence/commit/93884f7d562dd61b25bc13d8e82cd6495dde9d45))
* **format:** exclude release-managed changelog ([21a3c8c](https://github.com/oaslananka-lab/cifence/commit/21a3c8c3ce8a88714fa7e0cfa3ad91a06e115e79))
* **release:** refresh generated files on release PRs ([45bee02](https://github.com/oaslananka-lab/cifence/commit/45bee02ee8caa652b42a3e6b68ee504aad02f908))
* **release:** select release PR without pipefail hazards ([96e2e4b](https://github.com/oaslananka-lab/cifence/commit/96e2e4b85a99d41f0d69b4f27486f353dda3c3a5))
* **rules:** address review gate findings ([a594632](https://github.com/oaslananka-lab/cifence/commit/a5946323ba21dfe05a5d42c0cff27d11cbc09f15))
* **sarif:** use custom partial fingerprint key ([05525af](https://github.com/oaslananka-lab/cifence/commit/05525af9fb04ee22a00635d26d5346d35a02cd46))


### Performance Improvements

* **analyzer:** reuse workflow snippets during enrichment ([08ba172](https://github.com/oaslananka-lab/cifence/commit/08ba17260936846c3e18d190da2b661bd974daed))

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
