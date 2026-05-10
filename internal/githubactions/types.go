package githubactions

type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
)

const (
	BaselineStateExisting = "existing"
	BaselineStateNew      = "new"
)

type RuleDefinition struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Severity    Severity `json:"severity"`
	Description string   `json:"description"`
	Remediation string   `json:"remediation"`
	HelpURI     string   `json:"help_uri,omitempty"`
	Precision   string   `json:"precision,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type Finding struct {
	RuleID             string   `json:"rule_id"`
	Severity           Severity `json:"severity"`
	Title              string   `json:"title"`
	Message            string   `json:"message"`
	File               string   `json:"file"`
	Line               int      `json:"line"`
	Column             int      `json:"column"`
	YAMLPath           string   `json:"yaml_path,omitempty"`
	Evidence           string   `json:"evidence"`
	Remediation        string   `json:"remediation"`
	Fingerprint        string   `json:"fingerprint"`
	Snippet            string   `json:"snippet,omitempty"`
	Suppressed         bool     `json:"suppressed,omitempty"`
	SuppressionReason  string   `json:"suppression_reason,omitempty"`
	SuppressionExpires string   `json:"suppression_expires,omitempty"`
	BaselineState      string   `json:"baseline_state,omitempty"`
}

type Summary struct {
	FilesScanned int `json:"files_scanned"`
	Findings     int `json:"findings"`
	Critical     int `json:"critical"`
	High         int `json:"high"`
	Medium       int `json:"medium"`
	Low          int `json:"low"`
	Suppressed   int `json:"suppressed"`
	Baselined    int `json:"baselined"`
	New          int `json:"new"`
}

type Report struct {
	Version  string    `json:"version"`
	Summary  Summary   `json:"summary"`
	Findings []Finding `json:"findings"`
}

func NewReport(version string, filesScanned int, findings []Finding) Report {
	summary := Summary{FilesScanned: filesScanned, Findings: len(findings)}
	for _, finding := range findings {
		if finding.Suppressed {
			summary.Suppressed++
		}
		if finding.BaselineState == BaselineStateExisting {
			summary.Baselined++
		}
		if finding.BaselineState == BaselineStateNew {
			summary.New++
		}
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

func ValidSeverity(value string) bool {
	switch Severity(value) {
	case SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow:
		return true
	default:
		return false
	}
}

func SeverityRank(severity Severity) int {
	switch severity {
	case SeverityCritical:
		return 4
	case SeverityHigh:
		return 3
	case SeverityMedium:
		return 2
	case SeverityLow:
		return 1
	default:
		return 0
	}
}
