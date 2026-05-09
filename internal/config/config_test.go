package config

import "testing"

func TestValidateRequiresSuppressionReasonAndExpiry(t *testing.T) {
	cfg := Default()
	cfg.Suppressions = []Suppression{{Rule: "CF-ACT-001", Path: ".github/workflows/ci.yml"}}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected invalid suppression to fail validation")
	}
}

func TestValidateRejectsUnknownRule(t *testing.T) {
	cfg := Default()
	cfg.Rules = map[string]RuleConfig{"CF-UNKNOWN": {}}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected unknown rule to fail validation")
	}
}
