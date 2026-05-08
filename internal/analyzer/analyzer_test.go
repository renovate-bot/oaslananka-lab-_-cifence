package analyzer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/oaslananka/cifence/internal/githubactions"
)

func TestFixtureGoldenReports(t *testing.T) {
	fixtures := []string{
		"safe",
		"write-all",
		"missing-permissions",
		"mutable-action-ref",
		"unpinned-action-ref",
		"unsafe-pr-target",
		"invalid-yaml",
	}

	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			report, err := Scan(filepath.Join("..", "..", "tests", "fixtures", "workflows", fixture+".yml"))
			if err != nil {
				t.Fatalf("scan failed: %v", err)
			}
			actual, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				t.Fatalf("json failed: %v", err)
			}
			expected, err := os.ReadFile(filepath.Join("..", "..", "tests", "fixtures", "expected", fixture+".json"))
			if err != nil {
				t.Fatalf("read expected failed: %v", err)
			}
			if string(actual) != string(trimTrailingNewline(expected)) {
				t.Fatalf("report mismatch\nactual:\n%s\nexpected:\n%s", actual, expected)
			}
		})
	}
}

func TestWorkflowDiscovery(t *testing.T) {
	root := t.TempDir()
	workflowsDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a.yml", "b.yaml"} {
		if err := os.WriteFile(filepath.Join(workflowsDir, name), []byte("name: safe\non: push\npermissions:\n  contents: read\njobs:\n  test:\n    permissions:\n      contents: read\n    runs-on: ubuntu-24.04\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	report, err := Scan(root)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if report.Summary.FilesScanned != 2 {
		t.Fatalf("expected workflow discovery under .github/workflows, got %d", report.Summary.FilesScanned)
	}
}

func TestEnforceFails(t *testing.T) {
	report := githubactions.NewReport(Version, 1, []githubactions.Finding{
		{Severity: githubactions.SeverityHigh},
	})
	if !EnforceFails(report) {
		t.Fatal("expected high finding to fail enforce mode")
	}

	report = githubactions.NewReport(Version, 1, []githubactions.Finding{
		{Severity: githubactions.SeverityMedium},
	})
	if EnforceFails(report) {
		t.Fatal("expected medium finding not to fail enforce mode")
	}
}

func trimTrailingNewline(value []byte) []byte {
	for len(value) > 0 && (value[len(value)-1] == '\n' || value[len(value)-1] == '\r') {
		value = value[:len(value)-1]
	}
	return value
}
