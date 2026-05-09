package rules

import (
	"testing"

	"github.com/oaslananka/cifence/internal/githubactions"
	"github.com/oaslananka/cifence/internal/parser"
	"gopkg.in/yaml.v3"
)

func TestActionReferenceParsing(t *testing.T) {
	action, ref, ok := splitActionRef("actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd")
	if !ok || action != "actions/checkout" || ref != "de0fac2e4500dabe0009e67214ff5f5447ce83dd" {
		t.Fatalf("unexpected parse: %q %q %v", action, ref, ok)
	}
}

func TestLocalActionIsIgnored(t *testing.T) {
	doc := parseYAML(t, `
name: Local
on: push
permissions:
  contents: read
jobs:
  test:
    permissions:
      contents: read
    runs-on: ubuntu-24.04
    steps:
      - uses: ./action
      - uses: ./.github/actions/foo
      - uses: ./actions/foo
`)
	findings := Analyze(doc)
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %#v", findings)
	}
}

func TestPullRequestTargetUnsafeCheckout(t *testing.T) {
	doc := parseYAML(t, `
name: Unsafe
on:
  pull_request_target:
permissions:
  contents: read
jobs:
  test:
    permissions:
      contents: read
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd
        with:
          ref: refs/pull/1/head
`)
	findings := Analyze(doc)
	if len(findings) != 1 || findings[0].RuleID != "CF-TRG-001" {
		t.Fatalf("expected CF-TRG-001, got %#v", findings)
	}
}

func TestNullPermissionsAndNonSequenceStepsDoNotPanic(t *testing.T) {
	doc := parseYAML(t, `
name: Null
on: push
permissions:
jobs:
  test:
    permissions:
    runs-on: ubuntu-24.04
    steps:
`)
	findings := Analyze(doc)
	if len(findings) < 2 {
		t.Fatalf("expected null permissions findings, got %#v", findings)
	}
}

func TestDockerTagWithoutDigestIsReported(t *testing.T) {
	doc := parseYAML(t, `
name: Docker
on: push
permissions:
  contents: read
jobs:
  test:
    permissions:
      contents: read
    runs-on: ubuntu-24.04
    steps:
      - uses: docker://alpine:3.20
`)
	findings := Analyze(doc)
	if len(findings) != 1 || findings[0].RuleID != "CF-ACT-001" {
		t.Fatalf("expected CF-ACT-001, got %#v", findings)
	}
}

func TestFullSHAIsAcceptedAndVersionTagIsUnpinned(t *testing.T) {
	doc := parseYAML(t, `
name: Actions
on: push
permissions:
  contents: read
jobs:
  test:
    permissions:
      contents: read
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd
      - uses: actions/setup-go@v6.4.0
`)
	findings := Analyze(doc)
	if len(findings) != 1 || findings[0].RuleID != "CF-ACT-001" {
		t.Fatalf("expected one unpinned version tag finding, got %#v", findings)
	}
	if findings[0].RuleID == "CF-ACT-002" {
		t.Fatalf("v-prefixed semver tag must not be treated as mutable: %#v", findings)
	}
}

func TestJobPermissionsOverrideWorkflowWriteForThirdPartyAction(t *testing.T) {
	doc := parseYAML(t, `
name: Override
on: push
permissions:
  contents: write
jobs:
  test:
    permissions:
      contents: read
    runs-on: ubuntu-24.04
    steps:
      - uses: owner/action@de0fac2e4500dabe0009e67214ff5f5447ce83dd
`)
	findings := Analyze(doc)
	for _, finding := range findings {
		if finding.RuleID == "CF-PERM-006" {
			t.Fatalf("job-level read permissions should override workflow write permissions: %#v", findings)
		}
	}
}

func TestPinnedReusableWorkflowSecretsInheritIsReported(t *testing.T) {
	doc := parseYAML(t, `
name: Reusable
on: push
permissions:
  contents: read
jobs:
  call:
    permissions:
      contents: read
    uses: org/repo/.github/workflows/ci.yml@de0fac2e4500dabe0009e67214ff5f5447ce83dd
    secrets: inherit
`)
	findings := Analyze(doc)
	if len(findings) != 1 || findings[0].RuleID != "CF-SEC-001" {
		t.Fatalf("expected CF-SEC-001 for pinned reusable workflow secrets inherit, got %#v", findings)
	}
}

func TestDangerousPermissionScopeCoverage(t *testing.T) {
	doc := parseYAML(t, `
name: Dangerous scopes
on:
  pull_request:
permissions:
  contents: read
jobs:
  scan:
    permissions:
      actions: write
      artifact-metadata: write
      attestations: write
      checks: write
      contents: write
      deployments: write
      discussions: write
      id-token: write
      issues: write
      packages: write
      pages: write
      pull-requests: write
      security-events: write
      statuses: write
    runs-on: ubuntu-24.04
`)
	findings := Analyze(doc)
	required := map[string]bool{
		"actions: write":           false,
		"artifact-metadata: write": false,
		"attestations: write":      false,
		"checks: write":            false,
		"contents: write":          false,
		"deployments: write":       false,
		"discussions: write":       false,
		"id-token: write":          false,
		"issues: write":            false,
		"packages: write":          false,
		"pages: write":             false,
		"pull-requests: write":     false,
		"security-events: write":   false,
		"statuses: write":          false,
	}
	for _, finding := range findings {
		if finding.RuleID == "CF-PERM-003" {
			required[finding.Evidence] = true
		}
	}
	for evidence, found := range required {
		if !found {
			t.Fatalf("missing CF-PERM-003 for %s in %#v", evidence, findings)
		}
	}
}

func TestUnknownPermissionScopeIsReported(t *testing.T) {
	doc := parseYAML(t, `
name: Unknown permission
on: push
permissions:
  contents: read
  typo-scope: write
jobs:
  test:
    permissions:
      contents: read
    runs-on: ubuntu-24.04
`)
	assertRulePresent(t, Analyze(doc), "CF-PERM-007")
}

func TestBroadPushDoesNotTrustIDToken(t *testing.T) {
	doc := parseYAML(t, `
name: Broad push
on:
  push:
permissions:
  id-token: write
jobs:
  publish:
    permissions:
      contents: read
    runs-on: ubuntu-24.04
`)
	assertRulePresent(t, Analyze(doc), "CF-PERM-004")
}

func TestEnvAssignmentAloneDoesNotReportInjection(t *testing.T) {
	doc := parseYAML(t, `
name: Safe env indirection
on:
  pull_request:
permissions:
  contents: read
jobs:
  test:
    permissions:
      contents: read
    runs-on: ubuntu-24.04
    steps:
      - env:
          TITLE: ${{ github.event.pull_request.title }}
        run: printf '%s\n' "$TITLE"
`)
	for _, finding := range Analyze(doc) {
		if finding.RuleID == "CF-INJ-001" || finding.RuleID == "CF-INJ-003" {
			t.Fatalf("safe env indirection should not be reported as injection: %#v", finding)
		}
	}
}

func TestExpandedUntrustedContextSinks(t *testing.T) {
	doc := parseYAML(t, `
name: Expanded injection
on:
  workflow_dispatch:
    inputs:
      shell:
        required: true
permissions:
  contents: read
jobs:
  test:
    permissions:
      contents: read
    runs-on: ubuntu-24.04
    steps:
      - shell: ${{ github.event.inputs.shell }}
        run: echo safe
      - uses: ./local-action
        with:
          args: ${{ github.event.release.body }}
`)
	findings := Analyze(doc)
	assertRulePresent(t, findings, "CF-INJ-001")
	assertRulePresent(t, findings, "CF-INJ-003")
}

func TestEnvironmentFileInjectionIsReported(t *testing.T) {
	doc := parseYAML(t, `
name: Environment file injection
on:
  pull_request:
permissions:
  contents: read
jobs:
  test:
    permissions:
      contents: read
    runs-on: ubuntu-24.04
    steps:
      - run: echo "NAME=${{ github.event.pull_request.title }}" >> "$GITHUB_ENV"
`)
	assertRulePresent(t, Analyze(doc), "CF-ENV-001")
}

func TestWorkflowRunArtifactExecutionAndSelfHostedRunner(t *testing.T) {
	doc := parseYAML(t, `
name: Workflow run boundary
on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]
permissions:
  contents: write
jobs:
  replay:
    permissions:
      contents: write
    runs-on: [self-hosted, linux]
    steps:
      - uses: actions/download-artifact@0123456789abcdef0123456789abcdef01234567
      - run: bash artifact/script.sh
`)
	findings := Analyze(doc)
	assertRulePresent(t, findings, "CF-RUN-001")
	assertRulePresent(t, findings, "CF-ART-001")
	assertRulePresent(t, findings, "CF-RUNNER-001")
}

func assertRulePresent(t *testing.T, findings []githubactions.Finding, ruleID string) {
	t.Helper()
	for _, finding := range findings {
		if finding.RuleID == ruleID {
			return
		}
	}
	t.Fatalf("expected %s in %#v", ruleID, findings)
}

func parseYAML(t *testing.T, content string) parser.Document {
	t.Helper()
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(content), &root); err != nil {
		t.Fatal(err)
	}
	return parser.Document{Root: &root, File: "workflow.yml", Content: []byte(content)}
}
