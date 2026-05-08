package rules

import "github.com/oaslananka/cifence/internal/githubactions"

const ParseRuleID = "CF-PARSE-001"

var Definitions = []githubactions.RuleDefinition{
	{
		ID:          "CF-PERM-001",
		Title:       "permissions: write-all is used",
		Severity:    githubactions.SeverityCritical,
		Description: "Detects workflow-level or job-level permissions: write-all.",
		Remediation: "Replace write-all with explicit least-privilege permissions.",
	},
	{
		ID:          "CF-PERM-002",
		Title:       "missing explicit permissions block",
		Severity:    githubactions.SeverityMedium,
		Description: "Detects workflows and jobs without explicit permissions blocks.",
		Remediation: "Add explicit permissions, starting from permissions: contents: read.",
	},
	{
		ID:          "CF-ACT-001",
		Title:       "action reference is not pinned to a full commit SHA",
		Severity:    githubactions.SeverityMedium,
		Description: "Detects remote GitHub Actions references that use tags or branches instead of a full commit SHA.",
		Remediation: "Pin to a full commit SHA.",
	},
	{
		ID:          "CF-ACT-002",
		Title:       "mutable action reference is used",
		Severity:    githubactions.SeverityHigh,
		Description: "Detects mutable GitHub Action refs and Docker latest tags.",
		Remediation: "Pin GitHub actions to full SHA and Docker images to immutable digest.",
	},
	{
		ID:          "CF-TRG-001",
		Title:       "pull_request_target checks out untrusted pull request code",
		Severity:    githubactions.SeverityCritical,
		Description: "Detects pull_request_target workflows that checkout untrusted pull request head refs or SHAs.",
		Remediation: "Avoid pull_request_target for untrusted code. Use pull_request with read-only permissions, or never checkout and execute the untrusted head while secrets/write token are available.",
	},
	{
		ID:          ParseRuleID,
		Title:       "workflow YAML could not be parsed",
		Severity:    githubactions.SeverityHigh,
		Description: "Reports a workflow file that could not be parsed as YAML.",
		Remediation: "Fix the YAML syntax so the workflow can be analyzed.",
	},
}

func DefinitionByID(id string) githubactions.RuleDefinition {
	for _, definition := range Definitions {
		if definition.ID == id {
			return definition
		}
	}
	return githubactions.RuleDefinition{
		ID:          id,
		Title:       id,
		Severity:    githubactions.SeverityLow,
		Description: "Unknown rule.",
		Remediation: "Review the finding.",
	}
}
