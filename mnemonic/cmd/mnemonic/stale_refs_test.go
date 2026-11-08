package main

// stale_refs_test.go — the shared-convention drift boundary (T-12 /
// TICKET-06): no LIVE skill, doc, or vault page names the pre-rename
// command paths or module directory. The patterns are boundary-tolerant
// (a dead `mnemonic mcp`/`mnemonic mem `/`mnemonic index `/`mnemonic
// serve` command, or the old `mnemonic-v2/` path); the exclusion set
// keeps historical records out of scope (change-archive/, the historical
// mnemonic-v2 change-specs folders, the shipped ADR files). The vault's
// LIVE pages (notably ARCHITECTURE.md) are NOT excluded — they must be
// rewritten, not excused.

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var staleRefPatterns = []*regexp.Regexp{
	regexp.MustCompile(`mnemonic (mcp|mem |index |serve)([ \t]|$|` + "`" + `)`),
	regexp.MustCompile(`mnemonic-v2/`),
}

// isExcludedStaleRef is T-12's exclusion set: the archive, the
// historical change-specs folders, and the shipped ADR decision files
// (immutable once shipped).
func isExcludedStaleRef(rel string) bool {
	rel = filepath.ToSlash(rel)
	if strings.Contains(rel, "change-archive/") {
		return true
	}
	if regexp.MustCompile(`change-specs/2026-\d+-\d+-mnemonic-v2-[^/]+/`).MatchString(rel) {
		return true
	}
	return regexp.MustCompile(`projectwiki/decisions/04-adr-[^/]+\.md`).MatchString(rel)
}

func TestNoStalePathReferences(t *testing.T) {
	// The repo root from this package (mnemonic/cmd/mnemonic -> root).
	root := filepath.Join("..", "..", "..")
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err != nil {
		t.Fatalf("repo root not found at %s: %v", root, err)
	}
	var hits []string
	for _, sub := range []string{".agents/skills", "docs", ".skillgrid/projectwiki"} {
		base := filepath.Join(root, sub)
		err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, rerr := filepath.Rel(root, path)
			if rerr != nil {
				return rerr
			}
			if isExcludedStaleRef(rel) {
				return nil
			}
			data, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			for i, line := range strings.Split(string(data), "\n") {
				for _, p := range staleRefPatterns {
					if p.MatchString(line) {
						hits = append(hits, rel+":"+strconv.Itoa(i+1)+": "+strings.TrimSpace(line))
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", sub, err)
		}
	}
	if len(hits) > 0 {
		t.Errorf("%d stale reference(s) outside the exclusion set:\n%s", len(hits), strings.Join(hits, "\n"))
	}
}
