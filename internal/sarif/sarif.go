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
	Tool               tool                         `json:"tool"`
	OriginalURIBaseIDs map[string]originalURIBaseID `json:"originalUriBaseIds,omitempty"`
	Results            []result                     `json:"results"`
}

type tool struct {
	Driver driver `json:"driver"`
}

type driver struct {
	Name            string `json:"name"`
	SemanticVersion string `json:"semanticVersion"`
	InformationURI  string `json:"informationUri"`
	Rules           []rule `json:"rules"`
}

type rule struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	HelpURI          string         `json:"helpUri,omitempty"`
	ShortDescription sarifText      `json:"shortDescription"`
	FullDescription  sarifText      `json:"fullDescription"`
	Help             sarifText      `json:"help"`
	Properties       ruleProperties `json:"properties"`
}

type ruleProperties struct {
	SecuritySeverity string   `json:"security-severity"`
	Precision        string   `json:"precision"`
	Problem          problem  `json:"problem"`
	Tags             []string `json:"tags"`
}

type result struct {
	RuleID              string              `json:"ruleId"`
	RuleIndex           int                 `json:"ruleIndex"`
	Rule                ruleReference       `json:"rule"`
	Level               string              `json:"level"`
	Message             sarifText           `json:"message"`
	Locations           []location          `json:"locations"`
	PartialFingerprints partialFingerprints `json:"partialFingerprints,omitempty"`
	Properties          resultProperties    `json:"properties,omitempty"`
}

type location struct {
	PhysicalLocation physicalLocation `json:"physicalLocation"`
	Message          sarifText        `json:"message,omitempty"`
}

type physicalLocation struct {
	ArtifactLocation artifactLocation `json:"artifactLocation"`
	Region           region           `json:"region"`
}

type artifactLocation struct {
	URI string `json:"uri"`
}

type region struct {
	StartLine   int       `json:"startLine"`
	StartColumn int       `json:"startColumn"`
	Snippet     sarifText `json:"snippet,omitempty"`
}

type sarifText struct {
	Text string `json:"text"`
}

type originalURIBaseID struct {
	URI string `json:"uri"`
}

type problem struct {
	Severity string `json:"severity"`
}

type ruleReference struct {
	ID    string `json:"id"`
	Index int    `json:"index"`
}

type partialFingerprints struct {
	CIFenceFingerprint string `json:"cifenceFingerprint,omitempty"`
}

type resultProperties struct {
	Severity string `json:"problem.severity,omitempty"`
}

func JSON(report githubactions.Report) ([]byte, error) {
	file := logFile{
		Version: "2.1.0",
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Runs: []run{{
			Tool: tool{
				Driver: driver{
					Name:            "CIFence",
					SemanticVersion: report.Version,
					InformationURI:  "https://github.com/oaslananka-lab/cifence",
					Rules:           sarifRules(),
				},
			},
			OriginalURIBaseIDs: map[string]originalURIBaseID{
				"%SRCROOT%": {URI: "file:///github/workspace/"},
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
			HelpURI:          definition.HelpURI,
			ShortDescription: sarifText{Text: definition.Title},
			FullDescription:  sarifText{Text: definition.Description},
			Help:             sarifText{Text: definition.Remediation},
			Properties: ruleProperties{
				SecuritySeverity: sarifSecuritySeverity(definition.Severity),
				Precision:        precision(definition.Precision),
				Problem:          problem{Severity: sarifProblemSeverity(definition.Severity)},
				Tags:             tags(definition.Tags),
			},
		})
	}
	return out
}

func sarifResults(findings []githubactions.Finding) []result {
	out := make([]result, 0, len(findings))
	ruleIndexes := map[string]int{}
	for index, definition := range rules.Definitions {
		ruleIndexes[definition.ID] = index
	}
	for _, finding := range findings {
		if finding.Suppressed || finding.BaselineState == "existing" {
			continue
		}
		ruleIndex := ruleIndexes[finding.RuleID]
		out = append(out, result{
			RuleID:    finding.RuleID,
			RuleIndex: ruleIndex,
			Rule:      ruleReference{ID: finding.RuleID, Index: ruleIndex},
			Level:     sarifLevel(finding.Severity),
			Message:   sarifText{Text: finding.Message},
			Locations: []location{{
				PhysicalLocation: physicalLocation{
					ArtifactLocation: artifactLocation{URI: finding.File},
					Region: region{
						StartLine:   finding.Line,
						StartColumn: finding.Column,
						Snippet:     sarifText{Text: finding.Snippet},
					},
				},
				Message: sarifText{Text: finding.Evidence},
			}},
			PartialFingerprints: partialFingerprints{
				CIFenceFingerprint: finding.Fingerprint,
			},
			Properties: resultProperties{Severity: sarifProblemSeverity(finding.Severity)},
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

func sarifProblemSeverity(severity githubactions.Severity) string {
	switch severity {
	case githubactions.SeverityCritical, githubactions.SeverityHigh:
		return "error"
	case githubactions.SeverityMedium:
		return "warning"
	default:
		return "recommendation"
	}
}

func precision(value string) string {
	if value == "" {
		return "medium"
	}
	return value
}

func tags(values []string) []string {
	if len(values) == 0 {
		return []string{"security", "github-actions"}
	}
	return values
}
