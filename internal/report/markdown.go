package report

import (
	"fmt"
	"strings"

	"github.com/oaslananka/cifence/internal/githubactions"
)

func Markdown(report githubactions.Report) string {
	var builder strings.Builder
	builder.WriteString("# CIFence Report\n\n")
	builder.WriteString("| Files scanned | Findings | Critical | High | Medium | Low | Suppressed | Baselined | New |\n")
	builder.WriteString("| ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	builder.WriteString(fmt.Sprintf("| %d | %d | %d | %d | %d | %d | %d | %d | %d |\n\n",
		report.Summary.FilesScanned,
		report.Summary.Findings,
		report.Summary.Critical,
		report.Summary.High,
		report.Summary.Medium,
		report.Summary.Low,
		report.Summary.Suppressed,
		report.Summary.Baselined,
		report.Summary.New,
	))

	if len(report.Findings) == 0 {
		builder.WriteString("No findings.\n")
		return builder.String()
	}

	builder.WriteString("| Rule | Severity | Location | Message | Remediation |\n")
	builder.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, finding := range report.Findings {
		location := fmt.Sprintf("%s:%d:%d", finding.File, finding.Line, finding.Column)
		builder.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
			escapeMarkdown(finding.RuleID),
			escapeMarkdown(string(finding.Severity)),
			escapeMarkdown(location),
			escapeMarkdown(finding.Message),
			escapeMarkdown(finding.Remediation),
		))
	}
	return builder.String()
}

func escapeMarkdown(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "|", "\\|")
	return value
}
