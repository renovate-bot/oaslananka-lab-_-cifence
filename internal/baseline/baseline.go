package baseline

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"time"

	"github.com/oaslananka/cifence/internal/githubactions"
)

type Baseline struct {
	Version  int     `json:"version"`
	Findings []Entry `json:"findings"`
}

type Entry struct {
	RuleID      string `json:"rule_id"`
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
	FirstSeen   string `json:"first_seen"`
	LastSeen    string `json:"last_seen"`
}

func Load(path string) (Baseline, error) {
	if path == "" {
		return Baseline{Version: 1}, nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Baseline{Version: 1}, nil
		}
		return Baseline{}, err
	}
	var baseline Baseline
	if err := json.Unmarshal(content, &baseline); err != nil {
		return Baseline{}, err
	}
	if baseline.Version != 1 {
		return Baseline{}, errors.New("unsupported baseline version")
	}
	return baseline, nil
}

func Apply(findings []githubactions.Finding, current Baseline) []githubactions.Finding {
	known := map[string]Entry{}
	for _, entry := range current.Findings {
		known[entry.Fingerprint] = entry
	}
	out := make([]githubactions.Finding, 0, len(findings))
	for _, finding := range findings {
		if _, ok := known[finding.Fingerprint]; ok {
			finding.BaselineState = githubactions.BaselineStateExisting
		} else {
			finding.BaselineState = githubactions.BaselineStateNew
		}
		out = append(out, finding)
	}
	return out
}

func FromFindings(findings []githubactions.Finding, existing Baseline, now time.Time) Baseline {
	firstSeen := map[string]string{}
	for _, entry := range existing.Findings {
		firstSeen[entry.Fingerprint] = entry.FirstSeen
	}
	date := now.UTC().Format("2006-01-02")
	entries := make([]Entry, 0, len(findings))
	seen := map[string]struct{}{}
	for _, finding := range findings {
		if finding.Suppressed {
			continue
		}
		if _, ok := seen[finding.Fingerprint]; ok {
			continue
		}
		seen[finding.Fingerprint] = struct{}{}
		first := firstSeen[finding.Fingerprint]
		if first == "" {
			first = date
		}
		entries = append(entries, Entry{
			RuleID:      finding.RuleID,
			Path:        finding.File,
			Fingerprint: finding.Fingerprint,
			FirstSeen:   first,
			LastSeen:    date,
		})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Path != entries[j].Path {
			return entries[i].Path < entries[j].Path
		}
		if entries[i].RuleID != entries[j].RuleID {
			return entries[i].RuleID < entries[j].RuleID
		}
		return entries[i].Fingerprint < entries[j].Fingerprint
	})
	return Baseline{Version: 1, Findings: entries}
}

func Write(path string, baseline Baseline) error {
	content, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(content, '\n'), 0o600)
}
