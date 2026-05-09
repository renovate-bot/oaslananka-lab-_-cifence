package analyzer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oaslananka/cifence/internal/config"
	"github.com/oaslananka/cifence/internal/githubactions"
)

func TestFixtureGoldenReports(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join("..", "..", "tests", "fixtures", "expected"))
	if err != nil {
		t.Fatalf("read expected fixtures failed: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		fixture := strings.TrimSuffix(entry.Name(), ".json")
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

func TestBaselineExistingFindingsDoNotFail(t *testing.T) {
	root := t.TempDir()
	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte("name: ci\non: push\njobs:\n  test:\n    runs-on: ubuntu-24.04\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	baselinePath := filepath.Join(root, "cifence.baseline.json")
	first, err := ScanWithOptions(root, ScanOptions{
		Config:         config.Default(),
		HasConfig:      true,
		BaselinePath:   baselinePath,
		UpdateBaseline: true,
		Now:            time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("baseline update failed: %v", err)
	}
	if first.Summary.New == 0 {
		t.Fatal("expected first scan to mark findings new")
	}
	second, err := ScanWithOptions(root, ScanOptions{
		Config:       config.Default(),
		HasConfig:    true,
		BaselinePath: baselinePath,
		Now:          time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("baseline scan failed: %v", err)
	}
	if second.Summary.Baselined == 0 {
		t.Fatal("expected existing baseline findings")
	}
	if EnforceFailsAt(second, githubactions.SeverityMedium) {
		t.Fatal("baseline findings should not fail enforce mode")
	}
}

func TestExpiredSuppressionReportsFinding(t *testing.T) {
	cfg := config.Default()
	cfg.Suppressions = []config.Suppression{
		{
			Rule:     "CF-PERM-002",
			Path:     "missing-permissions.yml",
			YAMLPath: stringPtr(""),
			Evidence: "workflow permissions missing",
			Reason:   "migration window",
			Expires:  "2026-01-01",
		},
		{
			Rule:     "CF-PERM-002",
			Path:     "missing-permissions.yml",
			YAMLPath: stringPtr(""),
			Evidence: "job \"test\" permissions missing",
			Reason:   "migration window",
			Expires:  "2026-01-01",
		},
	}
	report, err := ScanWithOptions(filepath.Join("..", "..", "tests", "fixtures", "workflows", "missing-permissions.yml"), ScanOptions{
		Config:    cfg,
		HasConfig: true,
		Now:       time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	found := false
	original := false
	for _, finding := range report.Findings {
		if finding.RuleID == "CF-SUP-001" {
			found = true
		}
		if finding.RuleID == "CF-PERM-002" {
			original = true
		}
	}
	if !found {
		t.Fatal("expected expired suppression finding")
	}
	if original {
		t.Fatal("expired suppression finding should replace the original suppressed finding")
	}
}

func stringPtr(value string) *string {
	return &value
}

func trimTrailingNewline(value []byte) []byte {
	for len(value) > 0 && (value[len(value)-1] == '\n' || value[len(value)-1] == '\r') {
		value = value[:len(value)-1]
	}
	return value
}
