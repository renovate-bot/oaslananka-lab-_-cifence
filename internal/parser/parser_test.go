package parser

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverWorkflows(t *testing.T) {
	root := t.TempDir()
	workflows := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflows, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workflows, "b.yaml"), []byte("name: b\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workflows, "a.yml"), []byte("name: a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workflows, "notes.txt"), []byte("ignore\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	files, err := DiscoverWorkflows(root)
	if err != nil {
		t.Fatalf("discover failed: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 workflow files, got %d", len(files))
	}
	if filepath.Base(files[0]) != "a.yml" || filepath.Base(files[1]) != "b.yaml" {
		t.Fatalf("files not deterministic: %#v", files)
	}
}

func TestDiscoverWorkflowsMissingDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "workflow.yml"), []byte("name: a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	files, err := DiscoverWorkflows(root)
	if err != nil {
		t.Fatalf("discover failed: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected no files when .github/workflows is absent, got %#v", files)
	}
}

func TestDiscoverExplicitWorkflowsDirectory(t *testing.T) {
	root := t.TempDir()
	workflows := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflows, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workflows, "workflow.yml"), []byte("name: a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	files, err := DiscoverWorkflows(workflows)
	if err != nil {
		t.Fatalf("discover failed: %v", err)
	}
	if len(files) != 1 || filepath.Base(files[0]) != "workflow.yml" {
		t.Fatalf("unexpected workflow directory files: %#v", files)
	}
}

func TestDiscoverWorkflowsSkipsNestedDirectories(t *testing.T) {
	root := t.TempDir()
	workflows := filepath.Join(root, ".github", "workflows")
	nested := filepath.Join(workflows, "sub")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workflows, "workflow.yml"), []byte("name: a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "ignored.yml"), []byte("name: ignored\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	files, err := DiscoverWorkflows(root)
	if err != nil {
		t.Fatalf("discover failed: %v", err)
	}
	if len(files) != 1 || filepath.Base(files[0]) != "workflow.yml" {
		t.Fatalf("nested workflow-like files should not be discovered by repository scan: %#v", files)
	}
}

func TestParseInvalidYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.yml")
	if err := os.WriteFile(path, []byte("name: [unterminated\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseFile(path, "invalid.yml"); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseMultiDocumentWorkflowFailsClosed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "multi.yml")
	if err := os.WriteFile(path, []byte("name: safe\n---\npermissions: write-all\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseFile(path, "multi.yml"); !errors.Is(err, ErrMultipleDocuments) {
		t.Fatalf("expected multi-document error, got %v", err)
	}
}
