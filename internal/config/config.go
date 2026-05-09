package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/oaslananka/cifence/internal/githubactions"
	"github.com/oaslananka/cifence/internal/parser"
	"github.com/oaslananka/cifence/internal/rules"
	"gopkg.in/yaml.v3"
)

const Version = 1

type Config struct {
	Version      int                   `yaml:"version" json:"version"`
	Severity     SeverityConfig        `yaml:"severity" json:"severity"`
	Rules        map[string]RuleConfig `yaml:"rules" json:"rules"`
	Paths        PathsConfig           `yaml:"paths" json:"paths"`
	Suppressions []Suppression         `yaml:"suppressions" json:"suppressions"`
}

type SeverityConfig struct {
	FailOn string `yaml:"fail_on" json:"fail_on"`
}

type RuleConfig struct {
	Enabled  *bool                  `yaml:"enabled" json:"enabled"`
	Severity githubactions.Severity `yaml:"severity" json:"severity"`
	Allow    []string               `yaml:"allow" json:"allow"`
}

type PathsConfig struct {
	Include []string `yaml:"include" json:"include"`
	Exclude []string `yaml:"exclude" json:"exclude"`
}

type Suppression struct {
	Rule        string  `yaml:"rule" json:"rule"`
	Path        string  `yaml:"path" json:"path"`
	Fingerprint string  `yaml:"fingerprint,omitempty" json:"fingerprint,omitempty"`
	YAMLPath    *string `yaml:"yaml_path,omitempty" json:"yaml_path,omitempty"`
	Evidence    string  `yaml:"evidence,omitempty" json:"evidence,omitempty"`
	Reason      string  `yaml:"reason" json:"reason"`
	Expires     string  `yaml:"expires" json:"expires"`
}

func Default() Config {
	return Config{
		Version: Version,
		Severity: SeverityConfig{
			FailOn: string(githubactions.SeverityHigh),
		},
		Paths: PathsConfig{
			Include: []string{".github/workflows/*.yml", ".github/workflows/*.yaml"},
		},
	}
}

func Load(root string, explicitPath string) (Config, string, error) {
	cfg := Default()
	configPath := explicitPath
	if configPath == "" {
		found, err := findDefault(root)
		if err != nil {
			return cfg, "", err
		}
		configPath = found
	}
	if configPath == "" {
		return cfg, "", nil
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		return cfg, configPath, err
	}
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return cfg, configPath, err
	}
	if err := Validate(cfg); err != nil {
		return cfg, configPath, err
	}
	return cfg, configPath, nil
}

func Validate(cfg Config) error {
	var validationErrs []error
	if cfg.Version != Version {
		validationErrs = append(validationErrs, fmt.Errorf("unsupported config version %d", cfg.Version))
	}
	if cfg.Severity.FailOn != "" && !githubactions.ValidSeverity(cfg.Severity.FailOn) {
		validationErrs = append(validationErrs, fmt.Errorf("invalid severity.fail_on %q", cfg.Severity.FailOn))
	}
	knownRules := map[string]struct{}{}
	for _, definition := range rules.Definitions {
		knownRules[definition.ID] = struct{}{}
	}
	for ruleID, ruleCfg := range cfg.Rules {
		if _, ok := knownRules[ruleID]; !ok {
			validationErrs = append(validationErrs, fmt.Errorf("unknown rule ID %q", ruleID))
		}
		if ruleCfg.Severity != "" && !githubactions.ValidSeverity(string(ruleCfg.Severity)) {
			validationErrs = append(validationErrs, fmt.Errorf("invalid severity %q for %s", ruleCfg.Severity, ruleID))
		}
	}
	for index, suppression := range cfg.Suppressions {
		if _, ok := knownRules[suppression.Rule]; !ok {
			validationErrs = append(validationErrs, fmt.Errorf("unknown suppression rule ID %q", suppression.Rule))
		}
		if suppression.Path == "" {
			validationErrs = append(validationErrs, fmt.Errorf("suppression %d path is required", index))
		}
		if suppression.Fingerprint == "" && (suppression.YAMLPath == nil || suppression.Evidence == "") {
			validationErrs = append(validationErrs, fmt.Errorf("suppression %d requires fingerprint or both yaml_path and evidence", index))
		}
		if suppression.Reason == "" {
			validationErrs = append(validationErrs, fmt.Errorf("suppression %d reason is required", index))
		}
		if suppression.Expires == "" {
			validationErrs = append(validationErrs, fmt.Errorf("suppression %d expires is required", index))
			continue
		}
		if _, err := time.Parse("2006-01-02", suppression.Expires); err != nil {
			validationErrs = append(validationErrs, fmt.Errorf("suppression %d expires must use YYYY-MM-DD", index))
		}
	}
	return errors.Join(validationErrs...)
}

func (cfg Config) DiscoverOptions() parser.DiscoverOptions {
	return parser.DiscoverOptions{
		Include: cfg.Paths.Include,
		Exclude: cfg.Paths.Exclude,
	}
}

func (cfg Config) FailOn() githubactions.Severity {
	if cfg.Severity.FailOn == "" {
		return githubactions.SeverityHigh
	}
	return githubactions.Severity(cfg.Severity.FailOn)
}

func (cfg Config) RuleEnabled(ruleID string) bool {
	ruleCfg, ok := cfg.Rules[ruleID]
	if !ok || ruleCfg.Enabled == nil {
		return true
	}
	return *ruleCfg.Enabled
}

func (cfg Config) OverrideSeverity(ruleID string, current githubactions.Severity) githubactions.Severity {
	if ruleCfg, ok := cfg.Rules[ruleID]; ok && ruleCfg.Severity != "" {
		return ruleCfg.Severity
	}
	return current
}

func (cfg Config) IsAllowed(finding githubactions.Finding) bool {
	ruleCfg, ok := cfg.Rules[finding.RuleID]
	if !ok {
		return false
	}
	for _, allowed := range ruleCfg.Allow {
		if allowed == finding.Evidence {
			return true
		}
	}
	return false
}

func (cfg Config) SuppressionFor(finding githubactions.Finding, now time.Time) (Suppression, bool, bool) {
	for _, suppression := range cfg.Suppressions {
		if suppression.Rule != finding.RuleID || suppression.Path != finding.File {
			continue
		}
		if !suppressionMatches(suppression, finding) {
			continue
		}
		expires, err := time.Parse("2006-01-02", suppression.Expires)
		if err != nil {
			continue
		}
		if now.After(expires.Add(24*time.Hour - time.Nanosecond)) {
			return suppression, true, true
		}
		return suppression, true, false
	}
	return Suppression{}, false, false
}

func suppressionMatches(suppression Suppression, finding githubactions.Finding) bool {
	if suppression.Fingerprint != "" {
		return suppression.Fingerprint == finding.Fingerprint
	}
	if suppression.YAMLPath == nil {
		return false
	}
	return *suppression.YAMLPath == finding.YAMLPath && suppression.Evidence == finding.Evidence
}

func findDefault(root string) (string, error) {
	cleanRoot := filepath.Clean(root)
	info, err := os.Stat(cleanRoot)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		cleanRoot = filepath.Dir(cleanRoot)
	}
	for _, name := range []string{"cifence.yml", "cifence.yaml", ".cifence.yml", ".cifence.yaml"} {
		candidate := filepath.Join(cleanRoot, name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
	}
	return "", nil
}
