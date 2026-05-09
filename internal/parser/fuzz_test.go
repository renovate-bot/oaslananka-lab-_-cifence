package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzParseWorkflow(f *testing.F) {
	f.Add([]byte("name: ci\non: push\npermissions:\n  contents: read\njobs:\n  test:\n    runs-on: ubuntu-24.04\n"))
	f.Add([]byte("name: [unterminated\n"))
	f.Fuzz(func(t *testing.T, content []byte) {
		path := filepath.Join(t.TempDir(), "workflow.yml")
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatalf("write failed: %v", err)
		}
		_, _ = ParseFile(path, "workflow.yml")
	})
}
