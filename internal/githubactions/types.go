package githubactions

type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
)

type RuleDefinition struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Severity    Severity `json:"severity"`
	Description string   `json:"description"`
	Remediation string   `json:"remediation"`
}

type Finding struct {
	RuleID      string   `json:"rule_id"`
	Severity    Severity `json:"severity"`
	Title       string   `json:"title"`
	Message     string   `json:"message"`
	File        string   `json:"file"`
	Line        int      `json:"line"`
	Column      int      `json:"column"`
	Evidence    string   `json:"evidence"`
	Remediation string   `json:"remediation"`
}

type Summary struct {
	FilesScanned int `json:"files_scanned"`
	Findings     int `json:"findings"`
	Critical     int `json:"critical"`
	High         int `json:"high"`
	Medium       int `json:"medium"`
	Low          int `json:"low"`
}

type Report struct {
	Version  string    `json:"version"`
	Summary  Summary   `json:"summary"`
	Findings []Finding `json:"findings"`
}

func NewReport(version string, filesScanned int, findings []Finding) Report {
	summary := Summary{FilesScanned: filesScanned, Findings: len(findings)}
	for _, finding := range findings {
		switch finding.Severity {
		case SeverityCritical:
			summary.Critical++
		case SeverityHigh:
			summary.High++
		case SeverityMedium:
			summary.Medium++
		case SeverityLow:
			summary.Low++
		}
	}
	return Report{
		Version:  version,
		Summary:  summary,
		Findings: findings,
	}
}
