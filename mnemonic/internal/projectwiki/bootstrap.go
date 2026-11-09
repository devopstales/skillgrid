package projectwiki

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/devopstales/skillgrid/mnemonic/internal/core/store"
)

const (
	VaultConfigFile       = "config.json"
	VaultSchemaFile       = "PROJECTWIKI_SCHEMA.md"
	VaultIgnoreFile       = ".ignore"
	legacyVaultIgnoreFile = ".knowledgeignore" // pre-rename name (read fallback)
	VaultIndexDir         = "index"
)

var PageTypeDirs = []string{
	"decisions",
	"invariants",
	"workflows",
	"architecture",
	"gotchas",
	"glossary",
	// 013 (wiki-autolearning, ADR-0060): the four contextual types — research
	// results + user preferences enter the vault as flat, type-driven pages.
	"sources",
	"concepts",
	"syntheses",
	"preferences",
}

type Options struct {
	Project      string
	AnchorEngine string
	Strict       bool
}

type Scaffold struct {
	Root    string
	Created []string
	Skipped []string
}

// Bootstrap scaffolds a project wiki vault at root. Idempotent: existing
// sentinel config.json → no-op.
func Bootstrap(ctx context.Context, st store.Store, root string, opts Options) (*Scaffold, error) {
	sc := &Scaffold{Root: root}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("projectwiki: mkdir %s: %w", root, err)
	}
	sentinel := filepath.Join(root, VaultConfigFile)
	if _, err := os.Stat(sentinel); err == nil {
		sc.Skipped = append(sc.Skipped, VaultConfigFile)
		return sc, nil
	}
	created := []string{}
	cfg := map[string]any{
		"schema_version": 1,
		"project":        opts.Project,
		"page_types":     PageTypeDirs,
		"anchor_engine":  opts.AnchorEngine,
		"strict":         opts.Strict,
		// 001 §8 (013): the stage-1 inline carve-out (005 §2.3) — preferences
		// inline (short, immediately actionable); sources/concepts/syntheses
		// stay links-only. The 012 mechanism consumes this block.
		"recall": map[string]any{
			"inline_types":    []string{"invariant", "gotcha", "skill", "case", "preference"},
			"inline_max":      1600,
			"links_threshold": 50,
		},
	}
	cfgBytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("projectwiki: marshal config: %w", err)
	}
	if err := os.WriteFile(sentinel, append(cfgBytes, '\n'), 0o644); err != nil {
		return nil, fmt.Errorf("projectwiki: write %s: %w", VaultConfigFile, err)
	}
	created = append(created, VaultConfigFile)
	schema := "# ProjectWiki Schema\n\nOperating manual for the project wiki vault at this root.\n"
	if err := os.WriteFile(filepath.Join(root, VaultSchemaFile), []byte(schema), 0o644); err != nil {
		return nil, fmt.Errorf("projectwiki: write %s: %w", VaultSchemaFile, err)
	}
	created = append(created, VaultSchemaFile)
	if err := os.WriteFile(filepath.Join(root, VaultIgnoreFile), []byte("# Project wiki read boundary\n"), 0o644); err != nil {
		return nil, fmt.Errorf("projectwiki: write %s: %w", VaultIgnoreFile, err)
	}
	created = append(created, VaultIgnoreFile)
	dirs := append([]string{VaultIndexDir}, PageTypeDirs...)
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			return nil, fmt.Errorf("projectwiki: mkdir %s: %w", d, err)
		}
	}
	// 013 (001 §8): bootstrap scaffolds the four contextual-type templates.
	if err := os.MkdirAll(filepath.Join(root, "templates"), 0o755); err != nil {
		return nil, fmt.Errorf("projectwiki: mkdir templates: %w", err)
	}
	for name, body := range contextualTemplates {
		p := filepath.Join(root, "templates", name)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			return nil, fmt.Errorf("projectwiki: write templates/%s: %w", name, err)
		}
		created = append(created, "templates/"+name)
	}
	return &Scaffold{Root: root, Created: created}, nil
}

// contextualTemplates are the four 013 (wiki-autolearning) page templates
// (ADR-0060) — the skeletons a research result / preference is cut into.
// They mirror the files committed to an existing vault's templates/
// (the same-edit rule keeps the two copies identical).
var contextualTemplates = map[string]string{
	"source.md": `---
type: SourceSummary
title: "<name> <version>"
status: stable
sources:
  - "external:<name> <version>"
---

# <name> <version>

## Summary

<A synthesis of what this source is and says — never a copy of the raw content (WA-INV-2).>

## Claims taken

- <a checkable claim> — used by [[syntheses/<result-slug>]]

## Staleness

Re-verify when the upstream version moves (the build_fingerprint drift flags this page alone, WA-INV-3).
`,
	"concept.md": `---
type: Concept
title: "<concept name>"
status: stable
sources:
  - "syntheses/<result-slug>"
---

# <concept name>

<A reusable idea / pattern / technique that outlives the research result it came from.>

## Where it applies

- [[gotchas/<slug>]] / [[architecture/<slug>]] / [[decisions/<adr>]] — the link that makes the external finding actionable in this repo.

## Evidence

- [[syntheses/<result-slug>]] — the result this concept was cut from.
`,
	"synthesis.md": `---
type: Synthesis
title: "<result question>"
status: draft
sources:
  - "external:<name> <version>"
---

# <result question>

The hub of a research result (WA-INV-1): one file per thing, cross-linked.

## Method

<how the question was researched>

## Findings

- <finding> — from [[sources/<name>-<version>]]

## Answer

<TL;DR + the answer. Links to every sources/ page drawn on and every concepts/ page spawned (≥2 outgoing, 001 §6).>
`,
	"preference.md": `---
type: Preference
title: "<preference area>"
status: stable
sources:
  - "memory:<sync_id>"
---

# <preference area>

Project-scoped user preferences (the global view stays in memory; each bullet cites the observation where the user stated it — INV-1 provenance).

## Preferences

- <the preference, in imperative form> — said in session <S> (memory:<sync_id>)

## Constrains

- [[invariants/<slug>]] / [[workflows/<slug>]] — the convention pages this preference constrains (≥2 outgoing, 001 §6).
`,
}

// Root returns the vault root for cwd: .skillgrid/projectwiki/ when it
// exists, else the legacy pre-rename vault dir, else the new path (the
// bootstrap target).
func Root(cwd string) string {
	newRoot := filepath.Join(cwd, ".skillgrid", "projectwiki")
	if dirExists(newRoot) {
		return newRoot
	}
	legacy := filepath.Join(cwd, ".skillgrid", "knowledge")
	if dirExists(legacy) {
		return legacy
	}
	return newRoot
}

func dirExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}
