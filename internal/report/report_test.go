package report

import (
	"strings"
	"testing"

	"github.com/oaslananka/cifence/internal/githubactions"
)

func TestMarkdownZeroFindings(t *testing.T) {
	output := Markdown(githubactions.NewReport("0.1.0", 1, nil))
	if !strings.Contains(output, "No findings.") {
		t.Fatalf("expected zero finding message, got %s", output)
	}
}

func TestJSONStable(t *testing.T) {
	output, err := JSON(githubactions.NewReport("0.1.0", 1, []githubactions.Finding{
		{RuleID: "CF-ACT-001", Severity: githubactions.SeverityMedium},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), "\"rule_id\": \"CF-ACT-001\"") {
		t.Fatalf("expected rule in json: %s", output)
	}
}
