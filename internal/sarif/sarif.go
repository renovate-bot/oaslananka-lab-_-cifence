package sarif

import (
	"encoding/json"

	"github.com/oaslananka/cifence/internal/githubactions"
	"github.com/oaslananka/cifence/internal/rules"
)

type logFile struct {
	Version string `json:"version"`
	Schema  string `json:"$schema"`
	Runs    []run  `json:"runs"`
}

type run struct {
	Tool    tool     `json:"tool"`
	Results []result `json:"results"`
}

type tool struct {
	Driver driver `json:"driver"`
}

type driver struct {
	Name           string `json:"name"`
	InformationURI string `json:"informationUri"`
	Rules          []rule `json:"rules"`
}

type rule struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	ShortDescription sarifText      `json:"shortDescription"`
	FullDescription  sarifText      `json:"fullDescription"`
	Help             sarifText      `json:"help"`
	Properties       ruleProperties `json:"properties"`
}

type ruleProperties struct {
	SecuritySeverity string   `json:"security-severity"`
	Tags             []string `json:"tags"`
}

type result struct {
	RuleID    string     `json:"ruleId"`
	Level     string     `json:"level"`
	Message   sarifText  `json:"message"`
	Locations []location `json:"locations"`
}

type location struct {
	PhysicalLocation physicalLocation `json:"physicalLocation"`
}

type physicalLocation struct {
	ArtifactLocation artifactLocation `json:"artifactLocation"`
	Region           region           `json:"region"`
}

type artifactLocation struct {
	URI string `json:"uri"`
}

type region struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
}

type sarifText struct {
	Text string `json:"text"`
}

func JSON(report githubactions.Report) ([]byte, error) {
	file := logFile{
		Version: "2.1.0",
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Runs: []run{{
			Tool: tool{
				Driver: driver{
					Name:           "CIFence",
					InformationURI: "https://github.com/oaslananka-lab/cifence",
					Rules:          sarifRules(),
				},
			},
			Results: sarifResults(report.Findings),
		}},
	}
	return json.MarshalIndent(file, "", "  ")
}

func sarifRules() []rule {
	definitions := rules.Definitions
	out := make([]rule, 0, len(definitions))
	for _, definition := range definitions {
		out = append(out, rule{
			ID:               definition.ID,
			Name:             definition.Title,
			ShortDescription: sarifText{Text: definition.Title},
			FullDescription:  sarifText{Text: definition.Description},
			Help:             sarifText{Text: definition.Remediation},
			Properties: ruleProperties{
				SecuritySeverity: sarifSecuritySeverity(definition.Severity),
				Tags:             []string{"security", "github-actions"},
			},
		})
	}
	return out
}

func sarifResults(findings []githubactions.Finding) []result {
	out := make([]result, 0, len(findings))
	for _, finding := range findings {
		out = append(out, result{
			RuleID:  finding.RuleID,
			Level:   sarifLevel(finding.Severity),
			Message: sarifText{Text: finding.Message},
			Locations: []location{{
				PhysicalLocation: physicalLocation{
					ArtifactLocation: artifactLocation{URI: finding.File},
					Region: region{
						StartLine:   finding.Line,
						StartColumn: finding.Column,
					},
				},
			}},
		})
	}
	return out
}

func sarifLevel(severity githubactions.Severity) string {
	switch severity {
	case githubactions.SeverityCritical, githubactions.SeverityHigh:
		return "error"
	case githubactions.SeverityMedium:
		return "warning"
	default:
		return "note"
	}
}

func sarifSecuritySeverity(severity githubactions.Severity) string {
	switch severity {
	case githubactions.SeverityCritical:
		return "9.0"
	case githubactions.SeverityHigh:
		return "7.0"
	case githubactions.SeverityMedium:
		return "5.0"
	default:
		return "2.0"
	}
}
