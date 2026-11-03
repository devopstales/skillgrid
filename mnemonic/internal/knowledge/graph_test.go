package knowledge

import (
	"os"
	"path/filepath"
	"testing"
)

// writeConceptBundle writes the fixture knowledge bundle (TICKET-09,
// @step-10 knowledge-graph-view): four concepts — a decision (links to the
// invariant and to a broken .md), a deprecated invariant (links to the
// workflow), a workflow (linked by the invariant), and a reference (links
// to the decision and to the broken .md; no inbound links, hence the
// orphan) — plus index.md, log.md and reverse-index.md catalogs.
func writeConceptBundle(t *testing.T, root string) {
	t.Helper()
	files := map[string]string{
		"index.md":         "# Index\n\n- [D-1](decisions/d1.md)\n- [I-1](invariants/i1.md)\n- [W-1](workflows/w1.md)\n- [R-1](reference/r1.md)\n",
		"log.md":           "# Log\n\n- 2026-01-01 d1\n- 2026-01-02 i1\n",
		"reverse-index.md": "# Reverse index\n\n- d1\n",
		"decisions/d1.md": `---
type: decision
title: D-1
description: Use Go stdlib only.
tags: [dep, go]
status: active
---

# D-1

See [I-1](../invariants/i1.md) and [Ghost](ghost.md).
`,
		"invariants/i1.md": `---
type: invariant
title: I-1
description: Tests never write.
tags: []
status: deprecated
---

# I-1

See [W-1](../workflows/w1.md).
`,
		"workflows/w1.md": `---
type: workflow
title: W-1
description: Run tests before commit.
tags: [ci]
status: active
---

# W-1

No links.
`,
		"reference/r1.md": `---
type: reference
title: R-1
description: Terminology.
tags: [glossary]
status: active
---

# R-1

See [D-1](../decisions/d1.md) and [Ghost](../ghost.md) and [ext](https://example.com/a.md).
`,
	}
	for path, content := range files {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
}

// TestKnowledgeGraph_Fixture covers @step-10 knowledge-graph-view (G11):
// the fixture bundle derives exactly 4 concept nodes carrying their
// frontmatter fields, edges only for links that resolve to another concept
// (broken links tolerated without error), the orphan flagged, the deprecated
// page flagged, and index.md / log.md / reverse-index.md excluded from nodes.
func TestKnowledgeGraph_Fixture(t *testing.T) {
	root := t.TempDir()
	writeConceptBundle(t, root)

	g, err := DeriveConceptGraph(root)
	if err != nil {
		t.Fatalf("DeriveConceptGraph: %v (broken links must be tolerated, not errored)", err)
	}

	byID := map[string]ConceptNode{}
	for _, n := range g.Nodes {
		byID[n.ID] = n
	}

	// 4 concept nodes; the three catalogs are excluded.
	if len(g.Nodes) != 4 {
		t.Fatalf("expected 4 nodes (index/log/reverse-index excluded), got %d: %v", len(g.Nodes), g.Nodes)
	}
	for _, id := range []string{"index", "log", "reverse-index"} {
		if _, ok := byID[id]; ok {
			t.Errorf("catalog %q must be excluded from nodes", id)
		}
	}

	// Frontmatter fields round-trip per node.
	want := map[string]struct {
		title, typ, desc, status string
		tags                     []string
	}{
		"decisions/d1":  {"D-1", "decision", "Use Go stdlib only.", "active", []string{"dep", "go"}},
		"invariants/i1": {"I-1", "invariant", "Tests never write.", "deprecated", nil},
		"workflows/w1":  {"W-1", "workflow", "Run tests before commit.", "active", []string{"ci"}},
		"reference/r1":  {"R-1", "reference", "Terminology.", "active", []string{"glossary"}},
	}
	for id, w := range want {
		n, ok := byID[id]
		if !ok {
			t.Fatalf("missing node %q", id)
		}
		if n.Title != w.title || n.Type != w.typ || n.Description != w.desc || n.Status != w.status {
			t.Errorf("node %q = (title %q, type %q, desc %q, status %q), want (%q, %q, %q, %q)",
				id, n.Title, n.Type, n.Description, n.Status, w.title, w.typ, w.desc, w.status)
		}
		if len(n.Tags) != len(w.tags) {
			t.Errorf("node %q tags = %v, want %v", id, n.Tags, w.tags)
		} else {
			for i := range w.tags {
				if n.Tags[i] != w.tags[i] {
					t.Errorf("node %q tags = %v, want %v", id, n.Tags, w.tags)
				}
			}
		}
		if n.Deprecated != (w.status == "deprecated") {
			t.Errorf("node %q deprecated = %v, want %v", id, n.Deprecated, w.status == "deprecated")
		}
	}

	// Edges: only the resolving links — d1→i1, i1→w1, r1→d1. The broken
	// ghost.md links (x2) and the external URL link produce no edge.
	wantEdges := map[string]bool{
		"decisions/d1->invariants/i1": true,
		"invariants/i1->workflows/w1": true,
		"reference/r1->decisions/d1":  true,
	}
	if len(g.Edges) != len(wantEdges) {
		t.Fatalf("expected %d edges, got %d: %v", len(wantEdges), len(g.Edges), g.Edges)
	}
	for _, e := range g.Edges {
		key := e.Source + "->" + e.Target
		if !wantEdges[key] {
			t.Errorf("unexpected edge %s", key)
		}
	}

	// Orphans: nodes with no inbound edges — w1 (inbound i1→w1? yes). Recompute
	// from edges: w1 has an inbound (i1→w1), i1 has d1→i1, d1 has r1→d1, r1 has
	// none → r1 is the orphan.
	inbound := map[string]int{}
	for _, e := range g.Edges {
		inbound[e.Target]++
	}
	if byID["reference/r1"].Orphan != true {
		t.Errorf("reference/r1 has no inbound links, orphan = %v, want true", byID["reference/r1"].Orphan)
	}
	for _, id := range []string{"decisions/d1", "invariants/i1", "workflows/w1"} {
		if byID[id].Orphan {
			t.Errorf("node %q has inbound %d, orphan = true, want false", id, inbound[id])
		}
	}

	// Backlinks: reverse of edges.
	if n := len(g.Backlinks["decisions/d1"]); n != 1 || g.Backlinks["decisions/d1"][0] != "reference/r1" {
		t.Errorf("backlinks[decisions/d1] = %v, want [reference/r1]", g.Backlinks["decisions/d1"])
	}
	if n := len(g.Backlinks["invariants/i1"]); n != 1 || g.Backlinks["invariants/i1"][0] != "decisions/d1" {
		t.Errorf("backlinks[invariants/i1] = %v, want [decisions/d1]", g.Backlinks["invariants/i1"])
	}
	if n := len(g.Backlinks["workflows/w1"]); n != 1 || g.Backlinks["workflows/w1"][0] != "invariants/i1" {
		t.Errorf("backlinks[workflows/w1] = %v, want [invariants/i1]", g.Backlinks["workflows/w1"])
	}
	if _, ok := g.Backlinks["reference/r1"]; ok {
		t.Errorf("backlinks[reference/r1] present: %v, want absent (it is the orphan)", g.Backlinks["reference/r1"])
	}
}

// TestKnowledgeGraph_Empty covers @step-10 (an empty bundle yields an empty
// state, not an error): an empty root and a root whose only files are the
// excluded catalogs both derive an empty graph without error.
func TestKnowledgeGraph_Empty(t *testing.T) {
	root := t.TempDir()
	g, err := DeriveConceptGraph(root)
	if err != nil {
		t.Fatalf("empty root: DeriveConceptGraph = %v, want no error", err)
	}
	if len(g.Nodes) != 0 || len(g.Edges) != 0 {
		t.Errorf("empty root: got %d nodes / %d edges, want 0/0", len(g.Nodes), len(g.Edges))
	}

	// Different inputs: a root containing ONLY the excluded catalogs is still
	// an empty graph (catalogs are not concepts).
	catalogOnly := t.TempDir()
	writeConceptBundle(t, catalogOnly)
	if err := os.RemoveAll(filepath.Join(catalogOnly, "decisions")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(catalogOnly, "invariants")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(catalogOnly, "workflows")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(catalogOnly, "reference")); err != nil {
		t.Fatal(err)
	}
	g2, err := DeriveConceptGraph(catalogOnly)
	if err != nil {
		t.Fatalf("catalog-only root: DeriveConceptGraph = %v, want no error", err)
	}
	if len(g2.Nodes) != 0 || len(g2.Edges) != 0 {
		t.Errorf("catalog-only root: got %d nodes / %d edges, want 0/0", len(g2.Nodes), len(g2.Edges))
	}
}

// TestKnowledgeGraph_LinkResolution covers @step-10 edge resolution at the
// boundaries: a link that resolves (relative + ./ + anchor-suffixed) becomes
// an edge; links that do not resolve (outside the bundle, a non-concept file,
// a non-markdown target) are tolerated without an edge; links in excluded
// catalogs produce no edges (the catalog is not a node).
func TestKnowledgeGraph_LinkResolution(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"decisions/a.md": "---\ntype: decision\ntitle: A\nstatus: active\n---\n# A\n\n[anchor](./b.md#sec) [outside](../out.md) [notmd](notes.txt) [broken](nope.md)\n",
		"decisions/b.md": "---\ntype: invariant\ntitle: B\nstatus: active\n---\n# B\n\ntext\n",
		"notes.txt":      "a non-markdown file in the bundle\n",
	}
	for path, content := range files {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	g, err := DeriveConceptGraph(root)
	if err != nil {
		t.Fatalf("DeriveConceptGraph: %v", err)
	}
	if len(g.Nodes) != 2 {
		t.Fatalf("expected 2 nodes (a.md, b.md), got %d: %v", len(g.Nodes), g.Nodes)
	}
	if len(g.Edges) != 1 {
		t.Fatalf("expected 1 edge (a→b via the anchor-suffixed relative link), got %d: %v", len(g.Edges), g.Edges)
	}
	if e := g.Edges[0]; e.Source != "decisions/a" || e.Target != "decisions/b" {
		t.Errorf("edge = %v, want decisions/a -> decisions/b", e)
	}
}

// TestKnowledgeGraph_MissingRoot covers the root-not-found case: it is an
// error (the caller named a directory that is not there) — distinct from an
// empty-but-present bundle, which derives empty.
func TestKnowledgeGraph_MissingRoot(t *testing.T) {
	_, err := DeriveConceptGraph(filepath.Join(t.TempDir(), "absent"))
	if err == nil {
		t.Fatal("missing root: DeriveConceptGraph = nil error, want an error")
	}
}
