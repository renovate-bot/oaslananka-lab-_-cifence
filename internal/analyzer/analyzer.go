package analyzer

import (
	"sort"

	"github.com/oaslananka/cifence/internal/githubactions"
	"github.com/oaslananka/cifence/internal/parser"
	"github.com/oaslananka/cifence/internal/rules"
)

const Version = "0.1.1"

func Scan(path string) (githubactions.Report, error) {
	files, err := parser.DiscoverWorkflows(path)
	if err != nil {
		return githubactions.Report{}, err
	}

	findings := make([]githubactions.Finding, 0)
	for _, file := range files {
		displayPath := parser.DisplayPath(path, file)
		doc, err := parser.ParseFile(file, displayPath)
		if err != nil {
			findings = append(findings, rules.ParseFinding(displayPath, err))
			continue
		}
		findings = append(findings, rules.Analyze(doc)...)
	}

	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].File != findings[j].File {
			return findings[i].File < findings[j].File
		}
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		if findings[i].Column != findings[j].Column {
			return findings[i].Column < findings[j].Column
		}
		return findings[i].RuleID < findings[j].RuleID
	})

	return githubactions.NewReport(Version, len(files), findings), nil
}

func EnforceFails(report githubactions.Report) bool {
	return report.Summary.Critical > 0 || report.Summary.High > 0
}
