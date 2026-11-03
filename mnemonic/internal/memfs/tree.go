package memfs

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// TreeNode is one node in the code-index tree view.
type TreeNode struct {
	Name     string
	Children []*TreeNode
	Leaf     bool
	Meta     string // e.g. "1 symbols" on file leaves
}

// Tree renders the code-index repo directory tree under the given code path.
// Files are leaves annotated with their symbol count, e.g.
//
//	src/
//	├── auth/
//	│   └── login.go (1 symbols)
//	└── util.go (0 symbols)
//
// An empty (unindexed) store returns the "no code index" note.
func (fs *MemFS) Tree(ctx context.Context, codePath string) (string, error) {
	if fs == nil || fs.db == nil {
		return "", fmt.Errorf("memfs: not initialized")
	}
	cp, err := ResolveCodePath(fs.projectID, codePath)
	if err != nil {
		return "", err
	}
	q, args := fileCountQuery(cp.Dir)
	rows, err := fs.db.QueryContext(ctx, q, args...)
	if err != nil {
		return "", fmt.Errorf("memfs tree: %w", err)
	}
	defer rows.Close()

	type fileRow struct {
		path string
		n    int
	}
	var files []fileRow
	for rows.Next() {
		var fr fileRow
		if err := rows.Scan(&fr.path, &fr.n); err != nil {
			return "", fmt.Errorf("memfs tree scan: %w", err)
		}
		files = append(files, fr)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(files) == 0 {
		return fs.noCodeIndexNote(), nil
	}

	relPrefix := cp.Dir
	if relPrefix != "" {
		relPrefix += "/"
	}
	root := &TreeNode{Name: cp.Dir + "/", Leaf: false}
	for _, fr := range files {
		rel := strings.TrimPrefix(fr.path, relPrefix)
		segs := splitPath(rel)
		insertCodeNode(root, segs, fr.n)
	}
	return renderCodeTree(root), nil
}

// insertCodeNode inserts a file leaf (annotated with n symbols) at segs under parent.
func insertCodeNode(parent *TreeNode, segs []string, n int) {
	if len(segs) == 0 {
		return
	}
	cur := parent
	for i, seg := range segs {
		last := i == len(segs)-1
		var child *TreeNode
		for _, c := range cur.Children {
			if c.Name == seg {
				child = c
				break
			}
		}
		if child == nil {
			child = &TreeNode{Name: seg}
			cur.Children = append(cur.Children, child)
		}
		if last {
			child.Leaf = true
			child.Meta = fmt.Sprintf("%d symbols", n)
		}
		cur = child
	}
	sortCodeChildren(cur)
}

func sortCodeChildren(n *TreeNode) {
	if n == nil || len(n.Children) == 0 {
		return
	}
	sort.Slice(n.Children, func(i, j int) bool {
		return n.Children[i].Name < n.Children[j].Name
	})
	for _, c := range n.Children {
		sortCodeChildren(c)
	}
}

func renderCodeTree(node *TreeNode) string {
	var b strings.Builder
	writeCodeChildren(&b, node, "")
	return b.String()
}

func writeCodeChildren(b *strings.Builder, node *TreeNode, indent string) {
	children := node.Children
	for i, c := range children {
		last := i == len(children)-1
		connector := "├── "
		childIndent := indent + "│   "
		if last {
			connector = "└── "
			childIndent = indent + "    "
		}
		if c.Leaf && len(c.Children) == 0 {
			label := c.Name
			if c.Meta != "" {
				label += " (" + c.Meta + ")"
			}
			b.WriteString(indent + connector + label + "\n")
		} else {
			b.WriteString(indent + connector + c.Name + "/\n")
			writeCodeChildren(b, c, childIndent)
		}
	}
}
