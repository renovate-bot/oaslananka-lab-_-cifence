# Rules

CIFence rule IDs are stable. Severity defaults can be overridden in `cifence.yml`; remediation text remains attached to JSON, Markdown, and SARIF output.

## Permissions

### CF-PERM-001: permissions: write-all is used

- Severity: critical
- Rationale: `write-all` grants broad repository mutation capability to every step in scope.
- Example: `permissions: write-all`
- Remediation: replace `write-all` with explicit least-privilege scopes.

### CF-PERM-002: missing explicit permissions block

- Severity: medium
- Rationale: implicit token defaults are harder to review and may drift with repository settings.
- Example: a workflow or job without `permissions`.
- Remediation: add explicit workflow and job permissions, starting with `contents: read`.

### CF-PERM-003: dangerous write permission on pull request event

- Severity: high
- Rationale: PR-like events can be influenced by untrusted contributors.
- Example: `on: pull_request` with `contents: write`.
- Remediation: keep PR workflows read-only and move writes into trusted follow-up workflows.

### CF-PERM-004: id-token write without trust restriction

- Severity: high
- Rationale: OIDC tokens can authorize cloud or deployment access.
- Example: `id-token: write` on a job without an environment or trusted branch restriction.
- Remediation: grant OIDC only in trusted jobs protected by environment or branch policy.

### CF-PERM-005: job-level privilege escalation

- Severity: medium
- Rationale: job permissions that exceed the workflow baseline are easy to miss in review.
- Example: workflow `contents: read`, job `contents: write`.
- Remediation: isolate elevated jobs and document why the scope is required.

### CF-PERM-006: write token shared with third-party action

- Severity: high
- Rationale: a compromised third-party action can use the job token.
- Example: a job with `contents: write` and a non-`actions/*` remote action.
- Remediation: split third-party actions into read-only jobs or remove write scopes.

### CF-PERM-007: unknown GitHub token permission scope

- Severity: medium
- Rationale: unknown scopes are often typos, and future scopes should be reviewed before being treated as safe.
- Example: `permissions: typo-scope: write`
- Remediation: verify the scope against GitHub Actions documentation, remove typos, or update CIFence for newly introduced scopes.

## Actions And Reusable Workflows

### CF-ACT-001: action reference is not pinned to a full commit SHA

- Severity: medium
- Rationale: tags can be moved and are not immutable supply-chain references.
- Example: `uses: actions/checkout@v6`
- Remediation: pin remote actions to a full 40-character commit SHA. Local actions such as `./actions/foo` are ignored.

### CF-ACT-002: mutable action reference is used

- Severity: high
- Rationale: branch-like refs and Docker `latest` can change without review.
- Example: `uses: actions/checkout@main` or `uses: docker://alpine:latest`
- Remediation: pin actions to full SHAs and Docker actions to immutable digests.

### CF-ACT-003: reusable workflow is not pinned to a full commit SHA

- Severity: medium
- Rationale: remote reusable workflows execute code controlled by another ref.
- Example: `uses: org/repo/.github/workflows/ci.yml@v1`
- Remediation: pin remote reusable workflows to full commit SHAs.

### CF-ACT-004: reusable workflow uses a mutable ref

- Severity: high
- Rationale: branch-like reusable workflow refs can change between runs.
- Example: `uses: org/repo/.github/workflows/ci.yml@main`
- Remediation: pin reusable workflows to full commit SHAs.

## Script Injection

### CF-INJ-001: untrusted GitHub context interpolated into run step

- Severity: high
- Rationale: PR, issue, and comment fields can contain shell metacharacters.
- Example: `run: echo "${{ github.event.pull_request.title }}"`
- Remediation: avoid direct shell interpolation; pass values through environment variables and quote safely.

### CF-INJ-002: untrusted context passed into github-script

- Severity: high
- Rationale: direct expression interpolation can turn event text into JavaScript source.
- Example: `script: core.info("${{ github.event.pull_request.body }}")`
- Remediation: read event payload fields from `context.payload` and validate or escape them.

### CF-INJ-003: untrusted context used in workflow data field

- Severity: medium
- Rationale: cache keys, artifact names, and action command arguments can affect workflow behavior.
- Example: `key: ${{ github.head_ref }}`
- Remediation: use trusted stable identifiers or sanitize untrusted values.

### CF-CACHE-001: cache key uses attacker-controlled context

- Severity: high
- Rationale: attacker-controlled cache keys can poison state reused by later jobs or privileged workflows.
- Example: `key: ${{ github.event.pull_request.title }}`
- Remediation: use trusted cache keys and avoid crossing trust boundaries between untrusted and privileged workflows.

### CF-ENV-001: untrusted context written to GitHub environment file

- Severity: high
- Rationale: GitHub environment files affect later step environment, path, and outputs.
- Example: `run: echo "NAME=${{ github.event.pull_request.title }}" >> "$GITHUB_ENV"`
- Remediation: never append untrusted event data directly to `GITHUB_ENV`, `GITHUB_PATH`, or `GITHUB_OUTPUT`; validate and encode values first.

## pull_request_target

### CF-TRG-001: pull_request_target checks out untrusted pull request code

- Severity: critical
- Rationale: `pull_request_target` runs with base repository trust while PR code is attacker controlled.
- Example: checkout `ref: ${{ github.event.pull_request.head.sha }}`
- Remediation: use `pull_request` for untrusted code, or never checkout the untrusted head when secrets or write token are available.

### CF-TRG-002: pull_request_target executes shell commands

- Severity: critical
- Rationale: shell steps in privileged PR-target workflows are high-risk when they touch PR data.
- Example: `on: pull_request_target` with `run: gh pr checkout`.
- Remediation: avoid shell execution in `pull_request_target` unless all inputs are trusted and tightly scoped.

### CF-TRG-003: pull_request_target uses third-party action

- Severity: high
- Rationale: third-party action code runs with the privileged workflow token.
- Example: a `pull_request_target` job using `owner/action@...`.
- Remediation: use first-party pinned actions or move third-party execution to read-only `pull_request` workflows.

### CF-TRG-004: pull_request_target has write token

- Severity: critical
- Rationale: a write-capable token on PR-target events increases blast radius.
- Example: `pull-requests: write` on a `pull_request_target` workflow.
- Remediation: set PR-target workflows to read-only unless a narrow trusted write is unavoidable.

### CF-TRG-005: pull_request_target uses PR-controlled cache or artifact data

- Severity: high
- Rationale: PR-controlled cache keys or artifact names can poison privileged workflow state.
- Example: `key: ${{ github.head_ref }}`
- Remediation: avoid cache/artifact identifiers derived from untrusted PR data.

## Containers And Secrets

### CF-IMG-001: job container image is not pinned by digest

- Severity: medium
- Rationale: tags can point to different image contents over time.
- Example: `container: node:24`
- Remediation: pin job container images with `@sha256:` digests.

### CF-IMG-002: service container image is not pinned by digest

- Severity: medium
- Rationale: service images influence test and release behavior.
- Example: `services.redis.image: redis:7`
- Remediation: pin service images with `@sha256:` digests.

### CF-IMG-003: container image uses latest tag

- Severity: high
- Rationale: `latest` is intentionally mutable.
- Example: `image: redis:latest`
- Remediation: use a fixed tag and digest.

### CF-SEC-001: reusable workflow inherits all secrets

- Severity: high
- Rationale: `secrets: inherit` exposes all caller secrets to the reusable workflow boundary.
- Example: a reusable workflow call with `secrets: inherit`.
- Remediation: pass only explicit secrets required by the called workflow.

## Workflow Boundaries And Runners

### CF-ART-001: workflow_run artifact is executed

- Severity: high
- Rationale: artifacts from a lower-trust workflow can become code or data inputs in a privileged workflow.
- Example: a `workflow_run` job downloads an artifact and then runs `bash artifact/script.sh`.
- Remediation: verify artifact provenance and contents before execution, or avoid privileged artifact execution.

### CF-RUN-001: workflow_run grants dangerous write permissions

- Severity: high
- Rationale: `workflow_run` can cross a privilege boundary from one workflow to another.
- Example: `on: workflow_run` with `contents: write`.
- Remediation: keep `workflow_run` jobs read-only unless producer trust, branch restrictions, and artifact validation are explicit.

### CF-RUNNER-001: self-hosted runner on untrusted trigger

- Severity: high
- Rationale: untrusted contributions can expose persistent self-hosted runners to code execution risks.
- Example: `on: pull_request` with `runs-on: [self-hosted, linux]`.
- Remediation: use GitHub-hosted runners for untrusted workflows or isolate self-hosted runners with trusted labels and protected triggers.

## Governance

### CF-SUP-001: expired suppression

- Severity: high
- Rationale: suppressions need a review window and an owner-visible reason.
- Example: a suppression whose `expires` date is in the past.
- Remediation: remove the suppression or renew it with a current reason and expiry date.

### CF-PARSE-001: workflow YAML could not be parsed

- Severity: high
- Rationale: unparsable workflow files cannot be analyzed safely.
- Example: malformed YAML syntax.
- Remediation: fix the YAML syntax and rerun CIFence.
