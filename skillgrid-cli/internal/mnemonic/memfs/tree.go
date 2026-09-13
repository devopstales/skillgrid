package memfs

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// TreeNode is one node in the hierarchical tree view.
type TreeNode struct {
	Name     string     `json:"name"`
	Children []*TreeNode `json:"children,omitempty"`
	// Leaf is true when this node represents an observation (a leaf file),
	// false when it is a directory.
	Leaf    bool   `json:"leaf"`
	Title   string `json:"title,omitempty"`
}

// Tree returns a rendered hierarchical tree of the topic_key namespace
// under the given scope. The output is indented text using ├── and └──.
//
// Example for scope "project/A/":
//
//	project/A/
//	├── entities/
//	│   └── user_model.go
//	└── preferences/
//	    ├── sub1
//	    └── sub2
func (fs *MemFS) Tree(ctx context.Context, scope string) (string, error) {
	if fs == nil || fs.db == nil {
		return "", fmt.Errorf("memfs: not initialized")
	}
	f, err := ResolveURI(scope)
	if err != nil {
		return "", err
	}
	prefix := f.Prefix()

	rows, err := fs.db.QueryContext(ctx, `
		SELECT topic_key, title
		FROM observations
		WHERE project = ? AND deleted_at IS NULL
		  AND topic_key IS NOT NULL AND topic_key != ''
		  AND topic_key LIKE ?
		ORDER BY topic_key`,
		fs.projectID, prefix+"/%",
	)
	if err != nil {
		return "", fmt.Errorf("memfs tree: %w", err)
	}
	defer rows.Close()

	// Collect all topic_keys under the prefix.
	type keyRow struct {
		key   string
		title string
	}
	var keys []keyRow
	for rows.Next() {
		var k, t string
		if err := rows.Scan(&k, &t); err != nil {
			return "", fmt.Errorf("memfs tree scan: %w", err)
		}
		keys = append(keys, keyRow{key: k, title: t})
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	// Build the tree.
	root := &TreeNode{Name: prefix + "/", Children: nil, Leaf: false}
	for _, kr := range keys {
		// Strip the prefix to get the relative path.
		rel := strings.TrimPrefix(kr.key, prefix+"/")
		segs := strings.Split(rel, "/")
		insertNode(root, segs, kr.title)
	}

	return renderTree(root, prefix+"/", ""), nil
}

// insertNode inserts a leaf (title) at the path segs under parent.
func insertNode(parent *TreeNode, segs []string, title string) {
	if len(segs) == 0 {
		return
	}
	cur := parent
	for i, seg := range segs {
		last := i == len(segs)-1
		var child *TreeNode
		if len(cur.Children) > 0 {
			for _, c := range cur.Children {
				if c.Name == seg {
					child = c
					break
				}
			}
		}
		if child == nil {
			child = &TreeNode{Name: seg}
			cur.Children = append(cur.Children, child)
		}
		if last {
			child.Leaf = true
			child.Title = title
		}
		cur = child
	}
	// Sort children alphabetically for deterministic output.
	sortChildren(cur)
}

func sortChildren(n *TreeNode) {
	if n == nil || len(n.Children) == 0 {
		return
	}
	sort.Slice(n.Children, func(i, j int) bool {
		return n.Children[i].Name < n.Children[j].Name
	})
	for _, c := range n.Children {
		sortChildren(c)
	}
}

// renderTree renders the tree as indented text.
func renderTree(node *TreeNode, path, indent string) string {
	var b strings.Builder
	children := node.Children
	if len(children) == 0 {
		b.WriteString(indent + leafLabel(node))
		return b.String()
	}
	// The root node's name ends with "/" — its children are memory_type
	// directories and should always render with a trailing slash.
	isRoot := strings.HasSuffix(node.Name, "/")
	for i, c := range children {
		last := i == len(children)-1
		connector := "├── "
		childIndent := indent + "│   "
		if last {
			connector = "└── "
			childIndent = indent + "    "
		}
		if c.Leaf && len(c.Children) == 0 && !isRoot {
			b.WriteString(indent + connector + leafLabel(c) + "\n")
		} else {
			b.WriteString(indent + connector + c.Name + "/\n")
			if len(c.Children) > 0 {
				b.WriteString(renderTree(c, path+c.Name+"/", childIndent))
			}
		}
	}
	return b.String()
}

func leafLabel(n *TreeNode) string {
	label := n.Name
	if !strings.HasSuffix(label, "/") && n.Title != "" {
		label = n.Title
	}
	return label
}
