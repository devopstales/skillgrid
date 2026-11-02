package graph

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// AffectedOpts configures the code_affected traversal (spec 004 §6).
type AffectedOpts struct {
	// Changed is the set of changed file paths (repo-relative).
	Changed []string
	// Depth caps the traversal (<= 0 uses 5).
	Depth int
}

// AffectedResult is the code_affected answer.
type AffectedResult struct {
	Changed        []string
	AffectedTests  []string
	Relationships  []Relationship
}

// Relationship is one stored edge hop in the blast radius.
type Relationship struct {
	From       string
	To         string
	Kind       string
	Confidence string
	Depth      int
	Via        string
	DepPath    string
}

// Affected answers "which tests do I run after this change?" (spec 004 §6).
// It seeds from every symbol in the changed files, does a depth-capped reverse
// BFS over the dependent kinds, and collects test files.
func Affected(ctx context.Context, st store.Store, opts AffectedOpts) (AffectedResult, error) {
	depth := opts.Depth
	if depth <= 0 {
		depth = 5
	}

	res := AffectedResult{
		Changed:       opts.Changed,
		AffectedTests: []string{},
	}

	if len(opts.Changed) == 0 {
		return res, nil
	}

	// Seed: every symbol defined in a changed file
	seen := map[int64]bool{}
	type visit struct {
		id    int64
		depth int
	}
	var queue []visit

	for _, path := range opts.Changed {
		file, err := st.FileByPath(ctx, path)
		if err != nil {
			continue
		}
		syms, err := st.SymbolsByFile(ctx, file.ID)
		if err != nil {
			continue
		}
		for _, s := range syms {
			if !seen[s.ID] {
				seen[s.ID] = true
				queue = append(queue, visit{id: s.ID, depth: 0})
			}
		}
	}

	// If the seed queue is empty, return early (no symbols in the changed files)
	if len(queue) == 0 {
		return res, nil
	}

	dependentsKinds := []string{"calls", "imports", "references", "extends", "implements", "tests_for", "tested_by"}
	testFiles := map[string]bool{}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[0:]
		if cur.depth >= depth {
			continue
		}

		// Reverse dependents: edges where to_id = cur.id
		edges, err := st.EdgesByTo(ctx, cur.id, dependentsKinds)
		if err != nil {
			continue
		}

		for _, e := range edges {
			// Check if the from_id is a test file
			from, err := st.SymbolByID(ctx, e.FromID)
			if err != nil {
				continue
			}
			if from.FileID != 0 {
				file, err := st.FileByID(ctx, from.FileID)
				if err == nil && isTestPath(file.Path) {
					testFiles[file.Path] = true
					// Record the relationship
					res.Relationships = append(res.Relationships, Relationship{
						From:       file.Path,
						To:         from.Name,
						Kind:       e.Kind,
						Confidence: e.Confidence,
						Depth:      cur.depth + 1,
						Via:        from.UID,
						DepPath:    file.Path,
					})
				}
			}

			if !seen[e.FromID] {
				seen[e.FromID] = true
				queue = append(queue, visit{id: e.FromID, depth: cur.depth + 1})
			}
		}
	}

	// Convert testFiles to a sorted slice
	for f := range testFiles {
		res.AffectedTests = append(res.AffectedTests, f)
	}
	sortStrings(res.AffectedTests)

	// Record the trail
	RecordTrail(ctx, st, "code_affected", strings.Join(opts.Changed, ","), "")

	return res, nil
}

// isTestPath reports whether a file path is a test file (any language).
// The Java test-convention fix: Foo.java changed → FooTest.java recognized.
func isTestPath(path string) bool {
	lower := strings.ToLower(path)
	base := filepath.Base(lower)
	switch {
	case strings.HasSuffix(base, "_test.go"):
		return true
	case strings.Contains(base, "test_") && strings.HasSuffix(base, ".py"):
		return true
	case strings.HasSuffix(base, "_test.rb"):
		return true
	case strings.HasSuffix(base, ".test.js"), strings.HasSuffix(base, ".test.ts"),
		strings.HasSuffix(base, ".test.tsx"), strings.HasSuffix(base, ".test.jsx"):
		return true
	case strings.Contains(base, "test.") && (strings.HasSuffix(base, ".js") || strings.HasSuffix(base, ".ts")):
		return true
	// The Java test-convention fix: FooTest.java is the test for Foo.java
	case strings.HasSuffix(base, "test.java"):
		return true
	}
	return false
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
