package parser

import (
	"bytes"
	"errors"
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

func DiscoverWorkflows(root string) ([]string, error) {
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
			return nil
		}
		if isWorkflowFile(path) {
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
	return Document{
		Root:    &root,
		File:    displayPath,
		Content: content,
	}, nil
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
