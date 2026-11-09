package projectwiki

import "testing"

// TestValidatePagePathContextualTypes pins 013 (wiki-autolearning,
// ADR-0060): the four contextual type folders pass the plan path contract
// while unknown folders and nested paths stay rejected.
func TestValidatePagePathContextualTypes(t *testing.T) {
	for _, p := range []string{
		"sources/context7-embed-rs-v0.4.md",
		"concepts/fence-sizing.md",
		"syntheses/mcp-seriality.md",
		"preferences/tooling.md",
		"gotchas/queue-fallback.md", // a core folder — unchanged
	} {
		if err := ValidatePagePath(p); err != nil {
			t.Fatalf("ValidatePagePath(%q): %v", p, err)
		}
	}
	for _, p := range []string{
		"reference/findings.md", // the removed pre-013 home
		"templates/source.md",   // scaffolding, not a page type
		"index/catalog.md",      // derived projection
		"sources/a/b.md",        // one file per thing: no nesting
	} {
		if err := ValidatePagePath(p); err == nil {
			t.Fatalf("ValidatePagePath(%q) = nil, want error", p)
		}
	}
}
