package analyzer

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"time"

	"github.com/oaslananka/cifence/internal/baseline"
	"github.com/oaslananka/cifence/internal/config"
	"github.com/oaslananka/cifence/internal/githubactions"
	"github.com/oaslananka/cifence/internal/parser"
	"github.com/oaslananka/cifence/internal/rules"
)

var Version = "0.0.0-dev"

type ScanOptions struct {
	Config         config.Config
	HasConfig      bool
	BaselinePath   string
	UpdateBaseline bool
	Now            time.Time
}

func Scan(path string) (githubactions.Report, error) {
	return ScanWithOptions(path, ScanOptions{})
}

func ScanWithOptions(path string, options ScanOptions) (githubactions.Report, error) {
	cfg := options.Config
	if !options.HasConfig {
		cfg = config.Default()
	}
	now := options.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	files, err := parser.DiscoverWorkflowsWithOptions(path, cfg.DiscoverOptions())
	if err != nil {
		return githubactions.Report{}, err
	}

	findings := make([]githubactions.Finding, 0)
	for _, file := range files {
		displayPath := parser.DisplayPath(path, file)
		doc, err := parser.ParseFile(file, displayPath)
		snippets := newSnippetIndex(doc.Content)
		if err != nil {
			findings = append(findings, enrichFindingWithSnippets(rules.ParseFinding(displayPath, err), snippets))
			continue
		}
		for _, finding := range rules.Analyze(doc) {
			findings = append(findings, enrichFindingWithSnippets(finding, snippets))
		}
	}

	findings = applyConfig(findings, cfg, now)
	if options.BaselinePath != "" {
		currentBaseline, err := baseline.Load(options.BaselinePath)
		if err != nil {
			return githubactions.Report{}, err
		}
		findings = baseline.Apply(findings, currentBaseline)
		if options.UpdateBaseline {
			nextBaseline := baseline.FromFindings(findings, currentBaseline, now)
			if err := baseline.Write(options.BaselinePath, nextBaseline); err != nil {
				return githubactions.Report{}, err
			}
		}
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
	return EnforceFailsAt(report, githubactions.SeverityHigh)
}

func EnforceFailsAt(report githubactions.Report, threshold githubactions.Severity) bool {
	thresholdRank := githubactions.SeverityRank(threshold)
	for _, finding := range report.Findings {
		if finding.Suppressed || finding.BaselineState == "existing" {
			continue
		}
		if githubactions.SeverityRank(finding.Severity) >= thresholdRank {
			return true
		}
	}
	return false
}

func applyConfig(findings []githubactions.Finding, cfg config.Config, now time.Time) []githubactions.Finding {
	out := make([]githubactions.Finding, 0, len(findings))
	for _, finding := range findings {
		if !cfg.RuleEnabled(finding.RuleID) || cfg.IsAllowed(finding) {
			continue
		}
		finding.Severity = cfg.OverrideSeverity(finding.RuleID, finding.Severity)
		if suppression, ok, expired := cfg.SuppressionFor(finding, now); ok {
			if expired {
				out = append(out, expiredSuppressionFinding(finding, suppression))
				continue
			} else {
				finding.Suppressed = true
				finding.SuppressionReason = suppression.Reason
				finding.SuppressionExpires = suppression.Expires
			}
		}
		out = append(out, finding)
	}
	return out
}

func expiredSuppressionFinding(base githubactions.Finding, suppression config.Suppression) githubactions.Finding {
	definition := rules.DefinitionByID("CF-SUP-001")
	finding := githubactions.Finding{
		RuleID:      definition.ID,
		Severity:    definition.Severity,
		Title:       definition.Title,
		Message:     "Configured suppression has expired.",
		File:        suppression.Path,
		Line:        max(1, base.Line),
		Column:      max(1, base.Column),
		YAMLPath:    base.YAMLPath,
		Evidence:    suppression.Rule + " expires " + suppression.Expires,
		Remediation: definition.Remediation,
		Snippet:     base.Snippet,
	}
	return enrichFinding(finding, nil)
}

func enrichFinding(finding githubactions.Finding, content []byte) githubactions.Finding {
	return enrichFindingWithSnippets(finding, newSnippetIndex(content))
}

func enrichFindingWithSnippets(finding githubactions.Finding, snippets snippetIndex) githubactions.Finding {
	if finding.Line <= 0 {
		finding.Line = 1
	}
	if finding.Column <= 0 {
		finding.Column = 1
	}
	if finding.Snippet == "" {
		finding.Snippet = snippets.forLine(finding.Line)
	}
	if finding.Fingerprint == "" {
		finding.Fingerprint = fingerprint(finding)
	}
	return finding
}

func fingerprint(finding githubactions.Finding) string {
	parts := []string{
		finding.RuleID,
		strings.ToLower(strings.ReplaceAll(finding.File, "\\", "/")),
		strings.TrimSpace(finding.Evidence),
		strings.TrimSpace(finding.YAMLPath),
	}
	hash := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(hash[:])
}

type snippetIndex []string

func newSnippetIndex(content []byte) snippetIndex {
	if len(content) == 0 {
		return nil
	}
	return strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
}

func (lines snippetIndex) forLine(line int) string {
	if line <= 0 || line > len(lines) {
		return ""
	}
	return strings.TrimSpace(lines[line-1])
}
