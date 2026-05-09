package config

import (
	"testing"
	"time"

	"github.com/oaslananka/cifence/internal/githubactions"
)

func TestValidateRequiresSuppressionReasonAndExpiry(t *testing.T) {
	cfg := Default()
	cfg.Suppressions = []Suppression{{Rule: "CF-ACT-001", Path: ".github/workflows/ci.yml", YAMLPath: stringPtr("jobs.test.steps[0].uses"), Evidence: "actions/checkout@v6"}}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected invalid suppression to fail validation")
	}
}

func TestValidateRequiresPreciseSuppressionMatchKey(t *testing.T) {
	cfg := Default()
	cfg.Suppressions = []Suppression{{
		Rule:    "CF-ACT-001",
		Path:    ".github/workflows/ci.yml",
		Reason:  "migration",
		Expires: "2026-12-31",
	}}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected suppression without fingerprint or yaml_path/evidence to fail validation")
	}
}

func TestValidateRejectsUnknownRule(t *testing.T) {
	cfg := Default()
	cfg.Rules = map[string]RuleConfig{"CF-UNKNOWN": {}}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected unknown rule to fail validation")
	}
}

func TestSuppressionRequiresExactYAMLPathAndEvidence(t *testing.T) {
	cfg := Default()
	cfg.Suppressions = []Suppression{{
		Rule:     "CF-PERM-006",
		Path:     ".github/workflows/release.yml",
		YAMLPath: stringPtr("jobs.release.permissions"),
		Evidence: "job \"release\" third-party action with write token",
		Reason:   "release boundary",
		Expires:  "2026-12-31",
	}}
	finding := githubactions.Finding{
		RuleID:   "CF-PERM-006",
		File:     ".github/workflows/release.yml",
		YAMLPath: "jobs.other.permissions",
		Evidence: "job \"release\" third-party action with write token",
	}
	if _, ok, _ := cfg.SuppressionFor(finding, time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)); ok {
		t.Fatal("suppression must not match different yaml_path")
	}
	finding.YAMLPath = "jobs.release.permissions"
	finding.Evidence = "different evidence"
	if _, ok, _ := cfg.SuppressionFor(finding, time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)); ok {
		t.Fatal("suppression must not match different evidence")
	}
	finding.Evidence = "job \"release\" third-party action with write token"
	if _, ok, expired := cfg.SuppressionFor(finding, time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)); !ok || expired {
		t.Fatalf("expected exact suppression match, ok=%v expired=%v", ok, expired)
	}
}

func TestValidateAllowsExplicitEmptyYAMLPathForFileLevelSuppression(t *testing.T) {
	cfg := Default()
	cfg.Suppressions = []Suppression{{
		Rule:     "CF-PERM-002",
		Path:     ".github/workflows/ci.yml",
		YAMLPath: stringPtr(""),
		Evidence: "workflow permissions missing",
		Reason:   "migration",
		Expires:  "2026-12-31",
	}}
	if err := Validate(cfg); err != nil {
		t.Fatalf("expected explicit empty yaml_path to be valid for file-level finding: %v", err)
	}
}

func stringPtr(value string) *string {
	return &value
}
