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
}
