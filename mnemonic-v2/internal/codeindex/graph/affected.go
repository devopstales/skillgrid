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
	Changed       []string
	AffectedTests []string
	Relationships []Relationship
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

	// Seed: every symbol defined in a changed file. Staged like the BFS
	// below — first resolve all changed paths to file rows (each query fully
	// consumed), then fetch each file's symbols — so no query is issued while
	// another statement is open. A seed symbol is marked in `seen` the moment
	// it is queued: the BFS visits each node exactly once, so a changed symbol
	// that depends on itself (a self-edge, e.g. a test that references its own
	// subject) cannot re-enter the queue and loop.
	seen := map[int64]bool{}
	type visit struct {
		id    int64
		depth int
	}
	var queue []visit

	var changedFiles []store.FileRow
	for _, path := range opts.Changed {
		if f, err := st.FileByPath(ctx, path); err == nil {
			changedFiles = append(changedFiles, f)
		}
	}
	for _, f := range changedFiles {
		syms, err := st.SymbolsByFile(ctx, f.ID)
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

	// The SQLite store is single-connection (MaxOpenConns(1)) with
	// _txlock=immediate: a second query issued while a first statement is
	// open blocks itself. So the BFS issues NO query while another is in
	// flight — each frontier's edges are fetched in one fully-consumed query,
	// then the next batch of from-symbols is preloaded in one fully-consumed
	// query, then the next batch of their files. No nested queries, so no
	// self-deadlock. (Dialect-agnostic: Store read methods only, no *sql.DB.)
	dependentsKinds := []string{"calls", "imports", "references", "extends", "implements", "tests_for", "tested_by"}
	testFiles := map[string]bool{}

	// trailIDs accumulates the test symbols flagged by this read (for the
	// trail).
	var trailIDs []int64
	trailSeen := map[int64]bool{}
	addTrailID := func(id int64) {
		if id != 0 && !trailSeen[id] {
			trailSeen[id] = true
			trailIDs = append(trailIDs, id)
		}
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.depth >= depth {
			continue
		}

		// Staged 1: fetch this node's reverse dependents (one query, fully
		// consumed into memory before any next query).
		edges, err := st.EdgesByTo(ctx, cur.id, dependentsKinds)
		if err != nil {
			continue
		}

		// Collect the distinct from_ids this frontier reached.
		var fromIDs []int64
		{
			idSeen := map[int64]bool{}
			for _, e := range edges {
				if !idSeen[e.FromID] {
					idSeen[e.FromID] = true
					fromIDs = append(fromIDs, e.FromID)
				}
			}
		}

		// Staged 2: preload every distinct from-symbol in one fully-consumed
		// query (no per-edge nested lookup while a statement is open).
		fromByID := make(map[int64]store.SymbolRow, len(fromIDs))
		for _, id := range fromIDs {
			if s, err := st.SymbolByID(ctx, id); err == nil {
				fromByID[id] = s
			}
		}

		// Staged 3: preload the distinct files those symbols belong to (one
		// query per file id, fully consumed before the next).
		fileIDSeen := map[int64]bool{}
		fileByID := map[int64]store.FileRow{}
		for _, s := range fromByID {
			if s.FileID != 0 && !fileIDSeen[s.FileID] {
				fileIDSeen[s.FileID] = true
				if f, err := st.FileByID(ctx, s.FileID); err == nil {
					fileByID[s.FileID] = f
				}
			}
		}

		// Now everything needed is in memory: classify + queue, no I/O.
		for _, e := range edges {
			from := fromByID[e.FromID]
			if from.FileID != 0 {
				if file, ok := fileByID[from.FileID]; ok && isTestPath(file.Path) {
					testFiles[file.Path] = true
					addTrailID(from.ID)
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

	// Record the trail with the affected test symbols' ids (the audit answer
	// to "which test symbols did this read flag?").
	RecordTrail(ctx, st, "code_affected", strings.Join(opts.Changed, ","), "", trailIDs...)

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
