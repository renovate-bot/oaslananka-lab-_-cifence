package rules

import (
	"testing"

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

func parseYAML(t *testing.T, content string) parser.Document {
	t.Helper()
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(content), &root); err != nil {
		t.Fatal(err)
	}
	return parser.Document{Root: &root, File: "workflow.yml", Content: []byte(content)}
}
