// handler_kb_test.go — the AGENTS.md KnowledgeBase block tests (the
// kb_bootstrap repo-level wiring): insertion into the Skillgrid section,
// idempotent refresh, missing-file creation, and the Bootstrap
// integration (the block is ensured on a fresh AND a no-op bootstrap).

package handler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/kb"
)

// agentsFixture builds a temp repo root with an AGENTS.md carrying the
// Skillgrid section (start/end markers + subsections) — the shape this
// repo's AGENTS.md has.
func agentsFixture(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	agents := "# Repo\n\n" +
		"<!-- skillgrid:start -->\n" +
		"## Skillgrid\n\n" +
		"### Artifacts\n\nsome artifacts\n\n" +
		"### Memory\n\nMnemonic\n\n" +
		"### Workflow\n\nbrainstorming -> ship\n\n" +
		"<!-- skillgrid:end -->\n"
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte(agents), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	return repo
}

func agentsContent(t *testing.T, repo string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repo, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	return string(b)
}

// TestEnsureAgentsBlockInsertsBeforeSkillgridEnd pins the placement: the
// block goes inside the Skillgrid section, before its end marker (after the
// last subsection), without disturbing the rest of the file.
func TestEnsureAgentsBlockInsertsBeforeSkillgridEnd(t *testing.T) {
	repo := agentsFixture(t)
	if err := ensureAgentsBlock(repo); err != nil {
		t.Fatalf("ensureAgentsBlock: %v", err)
	}
	s := agentsContent(t, repo)
	if !strings.Contains(s, "### KnowledgeBase") {
		t.Fatalf("the block must carry the ### KnowledgeBase heading, got: %s", s)
	}
	if !strings.Contains(s, "kb_query") {
		t.Fatalf("the block must route the read verbs (kb_query), got: %s", s)
	}
	// Placement: the block sits between the Workflow subsection and the
	// skillgrid end marker.
	iWorkflow := strings.Index(s, "brainstorming -> ship")
	iBlock := strings.Index(s, "### KnowledgeBase")
	iEnd := strings.Index(s, "<!-- skillgrid:end -->")
	if !(iWorkflow < iBlock && iBlock < iEnd) {
		t.Fatalf("the block must be inside the Skillgrid section (before the end marker); workflow=%d block=%d end=%d", iWorkflow, iBlock, iEnd)
	}
	// The existing content is untouched.
	for _, want := range []string{"### Artifacts", "### Memory", "### Workflow", "Mnemonic"} {
		if !strings.Contains(s, want) {
			t.Fatalf("existing content lost: %q", want)
		}
	}
}

// TestEnsureAgentsBlockIdempotentAndRefreshes: a second run never
// duplicates the block, and a hand-edited (drifted) block is replaced with
// the canonical text (a refresh, not an append).
func TestEnsureAgentsBlockIdempotentAndRefreshes(t *testing.T) {
	repo := agentsFixture(t)
	if err := ensureAgentsBlock(repo); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if err := ensureAgentsBlock(repo); err != nil {
		t.Fatalf("second run: %v", err)
	}
	s := agentsContent(t, repo)
	if n := strings.Count(s, "### KnowledgeBase"); n != 1 {
		t.Fatalf("a repeat bootstrap must not duplicate the block (count=%d)", n)
	}
	// Drift: hand-edit the block body, then refresh.
	drifted := strings.Replace(s, "kb_query", "kb_query_DRIFTED", 1)
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte(drifted), 0o644); err != nil {
		t.Fatalf("write drift: %v", err)
	}
	if err := ensureAgentsBlock(repo); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	s = agentsContent(t, repo)
	if strings.Contains(s, "kb_query_DRIFTED") {
		t.Fatal("a drifted block must be replaced with the canonical text")
	}
	if n := strings.Count(s, "### KnowledgeBase"); n != 1 {
		t.Fatalf("the refresh must not duplicate the block (count=%d)", n)
	}
}

// TestEnsureAgentsBlockCreatesMissing: no AGENTS.md at the repo root → the
// file is created with the block.
func TestEnsureAgentsBlockCreatesMissing(t *testing.T) {
	repo := t.TempDir()
	if err := ensureAgentsBlock(repo); err != nil {
		t.Fatalf("ensureAgentsBlock: %v", err)
	}
	s := agentsContent(t, repo)
	if !strings.Contains(s, "### KnowledgeBase") {
		t.Fatalf("a missing AGENTS.md must be created with the block, got: %s", s)
	}
}

// TestEnsureAgentsBlockIgnoresInlineProseMarkers: this repo's AGENTS.md
// QUOTES both markers inline in the intro paragraph (in backticks). The
// insertion must anchor on the standalone section-end line, not the
// inline prose occurrence, and must leave the prose line intact.
func TestEnsureAgentsBlockIgnoresInlineProseMarkers(t *testing.T) {
	repo := t.TempDir()
	agents := "# Repo\n\n" +
		"The onboarding skill renders the `<!-- skillgrid:start -->` … `<!-- skillgrid:end -->` block below into this file.\n\n" +
		"<!-- skillgrid:start -->\n" +
		"## Skillgrid\n\n" +
		"### Workflow\n\nbrainstorming -> ship\n\n" +
		"<!-- skillgrid:end -->\n"
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte(agents), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	if err := ensureAgentsBlock(repo); err != nil {
		t.Fatalf("ensureAgentsBlock: %v", err)
	}
	s := agentsContent(t, repo)
	// The prose line is intact (no block inserted into it).
	if !strings.Contains(s, "renders the `<!-- skillgrid:start -->` … `<!-- skillgrid:end -->` block below") {
		t.Fatalf("the prose-quoted markers must stay intact, got: %s", s)
	}
	// The block anchors on the real section end (after the Workflow
	// subsection, before the standalone end marker line).
	iWorkflow := strings.Index(s, "brainstorming -> ship")
	iBlock := strings.Index(s, "### KnowledgeBase")
	iEnd := strings.Index(s, "\n<!-- skillgrid:end -->")
	if !(iWorkflow < iBlock && iBlock < iEnd) {
		t.Fatalf("the block must be inside the real Skillgrid section; workflow=%d block=%d end=%d", iWorkflow, iBlock, iEnd)
	}
	if n := strings.Count(s, "### KnowledgeBase"); n != 1 {
		t.Fatalf("exactly one block, got %d", n)
	}
}

// TestEnsureAgentsBlockAppendsWithoutSkillgridSection: a repo whose
// AGENTS.md has no Skillgrid markers gets the block appended (no marker to
// anchor to).
func TestEnsureAgentsBlockAppendsWithoutSkillgridSection(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("# Repo\n\nsome rules\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	if err := ensureAgentsBlock(repo); err != nil {
		t.Fatalf("ensureAgentsBlock: %v", err)
	}
	s := agentsContent(t, repo)
	if !strings.Contains(s, "### KnowledgeBase") {
		t.Fatalf("the block must be appended, got: %s", s)
	}
	if !strings.HasPrefix(s, "# Repo\n\nsome rules") {
		t.Fatalf("the existing content must stay at the top, got: %s", s)
	}
}

// TestKbBootstrapWritesAgentsBlock: the Bootstrap integration — the shared
// handler (the K6 seam both surfaces call) writes the block at the repo
// root on a fresh bootstrap, and still ensures it when the vault
// bootstrap is a no-op (the sentinel already exists).
func TestKbBootstrapWritesAgentsBlock(t *testing.T) {
	st := openTestStore(t, t.TempDir())
	kh := NewKbHandler(st, config.Config{}, fixedDetector{})
	vault := t.TempDir()
	repo := t.TempDir()
	kh.SetRoots(vault, repo)
	ctx := context.Background()

	// Fresh: no vault sentinel, no AGENTS.md.
	if _, err := kh.Bootstrap(ctx, "", ""); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	s := agentsContent(t, repo)
	if !strings.Contains(s, "### KnowledgeBase") {
		t.Fatalf("a fresh bootstrap must write the AGENTS.md block, got: %s", s)
	}

	// No-op: the vault sentinel now exists; the block is still ensured
	// (idempotent refresh — no duplication).
	if _, err := kh.Bootstrap(ctx, "", ""); err != nil {
		t.Fatalf("no-op bootstrap: %v", err)
	}
	s = agentsContent(t, repo)
	if n := strings.Count(s, "### KnowledgeBase"); n != 1 {
		t.Fatalf("the no-op bootstrap must refresh, not duplicate, the block (count=%d)", n)
	}
}

// TestKbBootstrapNoRepoRootSkipsAgentsBlock: without a repo root (no git
// worktree known to the adapter) the bootstrap still scaffolds the vault
// and skips the AGENTS.md block (no crash, no write to an unknown path).
func TestKbBootstrapNoRepoRootSkipsAgentsBlock(t *testing.T) {
	st := openTestStore(t, t.TempDir())
	kh := NewKbHandler(st, config.Config{}, fixedDetector{})
	vault := t.TempDir()
	kh.SetRoots(vault, "")
	if _, err := kh.Bootstrap(context.Background(), "", ""); err != nil {
		t.Fatalf("bootstrap without a repo root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(vault, kb.VaultConfigFile)); err != nil {
		t.Fatalf("the vault must still be scaffolded: %v", err)
	}
}
