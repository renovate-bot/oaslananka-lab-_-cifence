package rules

import (
	"strings"

	"gopkg.in/yaml.v3"
)

func documentMapping(root *yaml.Node) *yaml.Node {
	if root == nil {
		return nil
	}
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		return asMapping(root.Content[0])
	}
	return asMapping(root)
}

func asMapping(node *yaml.Node) *yaml.Node {
	if node != nil && node.Kind == yaml.MappingNode {
		return node
	}
	return nil
}

func asSequence(node *yaml.Node) *yaml.Node {
	if node != nil && node.Kind == yaml.SequenceNode {
		return node
	}
	return nil
}

func lookup(mapping *yaml.Node, key string) (*yaml.Node, *yaml.Node, bool) {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil, nil, false
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		keyNode := mapping.Content[i]
		if keyNode.Value == key {
			return keyNode, mapping.Content[i+1], true
		}
	}
	return nil, nil, false
}

func mappingPairs(mapping *yaml.Node) []pair {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	pairs := make([]pair, 0, len(mapping.Content)/2)
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		pairs = append(pairs, pair{Key: mapping.Content[i], Value: mapping.Content[i+1]})
	}
	return pairs
}

func scalarString(node *yaml.Node) (string, bool) {
	if node == nil || node.Kind != yaml.ScalarNode {
		return "", false
	}
	return strings.TrimSpace(node.Value), true
}

type pair struct {
	Key   *yaml.Node
	Value *yaml.Node
}
