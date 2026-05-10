package sarif

import (
	"encoding/json"
	"testing"

	"github.com/oaslananka/cifence/internal/githubactions"
)

func TestSARIFGeneration(t *testing.T) {
	content, err := JSON(githubactions.NewReport("0.1.0", 1, []githubactions.Finding{
		{
			RuleID:   "CF-ACT-001",
			Severity: githubactions.SeverityMedium,
			Message:  "message",
			File:     ".github/workflows/ci.yml",
			Line:     1,
			Column:   2,
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(content, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["version"] != "2.1.0" {
		t.Fatalf("unexpected SARIF version: %#v", parsed["version"])
	}
	runs := parsed["runs"].([]any)
	driver := runs[0].(map[string]any)["tool"].(map[string]any)["driver"].(map[string]any)
	if driver["name"] != "CIFence" {
		t.Fatalf("unexpected tool name: %#v", driver["name"])
	}
	if driver["semanticVersion"] != "0.1.0" {
		t.Fatalf("unexpected tool version: %#v", driver["semanticVersion"])
	}
}

func TestSARIFExcludesSuppressedAndExistingBaselineFindings(t *testing.T) {
	content, err := JSON(githubactions.NewReport("0.1.0", 1, []githubactions.Finding{
		{
			RuleID:   "CF-ACT-001",
			Severity: githubactions.SeverityMedium,
			Message:  "active",
			File:     ".github/workflows/ci.yml",
			Line:     1,
			Column:   1,
		},
		{
			RuleID:     "CF-ACT-001",
			Severity:   githubactions.SeverityMedium,
			Message:    "suppressed",
			File:       ".github/workflows/ci.yml",
			Line:       2,
			Column:     1,
			Suppressed: true,
		},
		{
			RuleID:        "CF-ACT-001",
			Severity:      githubactions.SeverityMedium,
			Message:       "baselined",
			File:          ".github/workflows/ci.yml",
			Line:          3,
			Column:        1,
			BaselineState: "existing",
		},
	}))
	if err != nil {
		t.Fatal(err)
	}

	var parsed struct {
		Runs []struct {
			Results []struct {
				Message struct {
					Text string `json:"text"`
				} `json:"message"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(content, &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Runs) != 1 {
		t.Fatalf("unexpected run count: %d", len(parsed.Runs))
	}
	if len(parsed.Runs[0].Results) != 1 {
		t.Fatalf("expected one active SARIF result, got %d", len(parsed.Runs[0].Results))
	}
	if parsed.Runs[0].Results[0].Message.Text != "active" {
		t.Fatalf("unexpected result message: %q", parsed.Runs[0].Results[0].Message.Text)
	}
}
