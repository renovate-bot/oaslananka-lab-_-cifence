package parser

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Document struct {
	Root    *yaml.Node
	File    string
	Content []byte
}

type DiscoverOptions struct {
	Include []string
	Exclude []string
}

var ErrMultipleDocuments = errors.New("workflow YAML contains multiple documents")

func DiscoverWorkflows(root string) ([]string, error) {
	return DiscoverWorkflowsWithOptions(root, DiscoverOptions{})
}

func DiscoverWorkflowsWithOptions(root string, options DiscoverOptions) ([]string, error) {
	cleanRoot := filepath.Clean(root)
	info, err := os.Stat(cleanRoot)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		if isWorkflowFile(cleanRoot) {
			return []string{cleanRoot}, nil
		}
		return nil, nil
	}

	workflowsDir := filepath.Join(cleanRoot, ".github", "workflows")
	if _, err := os.Stat(workflowsDir); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if filepath.Base(cleanRoot) != "workflows" || filepath.Base(filepath.Dir(cleanRoot)) != ".github" {
				return nil, nil
			}
			workflowsDir = cleanRoot
		} else {
			return nil, err
		}
	}

	var files []string
	err = filepath.WalkDir(workflowsDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != workflowsDir {
				return filepath.SkipDir
			}
			return nil
		}
		if isWorkflowFile(path) && includedPath(cleanRoot, path, options) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func ParseFile(path string, displayPath string) (Document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Document{}, err
	}

	var root yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	if err := decoder.Decode(&root); err != nil {
		return Document{File: displayPath, Content: content}, err
	}
	for {
		var extra yaml.Node
		err := decoder.Decode(&extra)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return Document{File: displayPath, Content: content}, err
		}
		if !isEmptyDocument(&extra) {
			return Document{Root: &root, File: displayPath, Content: content}, ErrMultipleDocuments
		}
	}
	return Document{
		Root:    &root,
		File:    displayPath,
		Content: content,
	}, nil
}

func isEmptyDocument(node *yaml.Node) bool {
	if node == nil || node.Kind == 0 {
		return true
	}
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) == 0 {
			return true
		}
		return isEmptyDocument(node.Content[0])
	}
	return node.Kind == yaml.ScalarNode && node.Tag == "!!null" && strings.TrimSpace(node.Value) == ""
}

func DisplayPath(root string, file string) string {
	if info, err := os.Stat(filepath.Clean(root)); err == nil && !info.IsDir() {
		return filepath.ToSlash(filepath.Base(file))
	}
	relative, err := filepath.Rel(filepath.Clean(root), file)
	if err == nil && !strings.HasPrefix(relative, "..") {
		return filepath.ToSlash(relative)
	}
	return filepath.ToSlash(filepath.Clean(file))
}

func isWorkflowFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yml" || ext == ".yaml"
}

func includedPath(root string, path string, options DiscoverOptions) bool {
	display := DisplayPath(root, path)
	if len(options.Include) > 0 && !matchesAny(display, options.Include) {
		return false
	}
	if matchesAny(display, options.Exclude) {
		return false
	}
	return true
}

func matchesAny(path string, patterns []string) bool {
	path = filepath.ToSlash(path)
	for _, pattern := range patterns {
		pattern = filepath.ToSlash(strings.TrimSpace(pattern))
		if pattern == "" {
			continue
		}
		if ok, _ := filepath.Match(pattern, path); ok {
			return true
		}
		if ok, _ := filepath.Match(strings.TrimPrefix(pattern, "./"), path); ok {
			return true
		}
	}
	return false
}
