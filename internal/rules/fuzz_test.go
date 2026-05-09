package rules

import (
	"encoding/json"
	"testing"

	"github.com/oaslananka/cifence/internal/parser"
	"gopkg.in/yaml.v3"
)

func FuzzAnalyzeWorkflowNoPanic(f *testing.F) {
	f.Add([]byte("name: ci\non: pull_request\npermissions:\n  contents: read\njobs:\n  test:\n    permissions:\n      contents: read\n    runs-on: ubuntu-24.04\n    steps:\n      - run: echo ok\n"))
	f.Add([]byte("name: ci\non:\n  pull_request_target:\npermissions:\n  contents: write\njobs:\n  test:\n    runs-on: ubuntu-24.04\n    steps:\n      - run: echo '${{ github.head_ref }}'\n"))
	f.Fuzz(func(t *testing.T, content []byte) {
		var root yaml.Node
		_ = yaml.Unmarshal(content, &root)
		findings := Analyze(parser.Document{Root: &root, File: "workflow.yml", Content: content})
		first, err := json.Marshal(findings)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}
		second, err := json.Marshal(Analyze(parser.Document{Root: &root, File: "workflow.yml", Content: content}))
		if err != nil {
			t.Fatalf("second marshal failed: %v", err)
		}
		if string(first) != string(second) {
			t.Fatal("findings are not deterministic")
		}
		for _, finding := range findings {
			if finding.Line <= 0 || finding.Column <= 0 {
				t.Fatalf("non-positive location: %#v", finding)
			}
		}
	})
}

func FuzzActionRefParser(f *testing.F) {
	f.Add("actions/checkout@0123456789abcdef0123456789abcdef01234567")
	f.Add("docker://alpine:latest")
	f.Add("./local-action")
	f.Fuzz(func(t *testing.T, value string) {
		_, _, _ = splitActionRef(value)
		_ = isLocalAction(value)
		_ = isThirdPartyAction(value)
	})
}
