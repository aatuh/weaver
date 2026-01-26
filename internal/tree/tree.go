package tree

import (
	"sort"
	"strings"
)

// Node represents a JSON-serializable directory tree.
type Node struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Children []*Node `json:"children,omitempty"`
}

type node struct {
	name     string
	nodeType string
	children map[string]*node
}

// Build constructs a tree from relative paths.
func Build(rootName string, paths []string) *Node {
	root := &node{name: rootName, nodeType: "dir", children: map[string]*node{}}
	for _, rel := range paths {
		if rel == "" {
			continue
		}
		parts := strings.Split(rel, "/")
		current := root
		for i, part := range parts {
			if part == "" {
				continue
			}
			isFile := i == len(parts)-1
			child, ok := current.children[part]
			if !ok {
				nodeType := "dir"
				if isFile {
					nodeType = "file"
				}
				child = &node{name: part, nodeType: nodeType, children: map[string]*node{}}
				current.children[part] = child
			}
			current = child
		}
	}

	return toPublic(root)
}

func toPublic(n *node) *Node {
	result := &Node{Name: n.name, Type: n.nodeType}
	if len(n.children) == 0 {
		return result
	}

	names := make([]string, 0, len(n.children))
	for name := range n.children {
		names = append(names, name)
	}
	// Sort directories before files, then by name.
	sort.Slice(names, func(i, j int) bool {
		left := n.children[names[i]]
		right := n.children[names[j]]
		if left.nodeType != right.nodeType {
			return left.nodeType == "dir"
		}
		return left.name < right.name
	})

	children := make([]*Node, 0, len(names))
	for _, name := range names {
		children = append(children, toPublic(n.children[name]))
	}
	result.Children = children
	return result
}
