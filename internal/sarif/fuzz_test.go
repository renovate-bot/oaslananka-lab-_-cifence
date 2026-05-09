package sarif

import (
	"encoding/json"
	"testing"

	"github.com/oaslananka/cifence/internal/githubactions"
)

func FuzzSARIFGeneration(f *testing.F) {
	f.Add("CF-ACT-001", ".github/workflows/ci.yml", "evidence")
	f.Fuzz(func(t *testing.T, ruleID string, file string, evidence string) {
		content, err := JSON(githubactions.NewReport("0.0.0-dev", 1, []githubactions.Finding{{
			RuleID:      ruleID,
			Severity:    githubactions.SeverityMedium,
			Title:       "title",
			Message:     "message",
			File:        file,
			Line:        1,
			Column:      1,
			Evidence:    evidence,
			Remediation: "remediate",
			Fingerprint: "fingerprint",
		}}))
		if err != nil {
			t.Fatalf("sarif failed: %v", err)
		}
		var parsed map[string]any
		if err := json.Unmarshal(content, &parsed); err != nil {
			t.Fatalf("invalid json: %v", err)
		}
		if parsed["version"] != "2.1.0" {
			t.Fatalf("unexpected version: %#v", parsed["version"])
		}
	})
}
