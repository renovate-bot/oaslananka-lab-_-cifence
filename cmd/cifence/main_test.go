package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIWarnAndEnforceExitCodes(t *testing.T) {
	fixture := filepath.Join("..", "..", "tests", "fixtures", "workflows", "mutable-action-ref.yml")
	if code := run([]string{"scan", fixture, "--mode", "warn", "--format", "json"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("warn mode exit = %d", code)
	}
	if code := run([]string{"scan", fixture, "--mode", "enforce", "--format", "json"}, &bytes.Buffer{}, &bytes.Buffer{}); code == 0 {
		t.Fatal("expected enforce mode to fail")
	}
}

func TestCLIInvalidArguments(t *testing.T) {
	var stderr bytes.Buffer
	code := run([]string{"scan", "--mode", "strict"}, &bytes.Buffer{}, &stderr)
	if code == 0 {
		t.Fatal("expected invalid arguments to fail")
	}
	if !strings.Contains(stderr.String(), "invalid mode") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestCLIRules(t *testing.T) {
	var stdout bytes.Buffer
	if code := run([]string{"rules"}, &stdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("rules exit = %d", code)
	}
	if !strings.Contains(stdout.String(), "CF-PERM-001") {
		t.Fatalf("rules output missing rule: %s", stdout.String())
	}
}
