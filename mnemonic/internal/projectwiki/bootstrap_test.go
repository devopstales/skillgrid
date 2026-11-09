package projectwiki

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestBootstrapScaffoldsContextualTypes pins the 013 (wiki-autolearning,
// ADR-0060) bootstrap scaffold: the four contextual type folders, the four
// templates, and the sentinel's page_types (10) + recall block (001 §8)
// with preference in inline_types.
func TestBootstrapScaffoldsContextualTypes(t *testing.T) {
	root := t.TempDir()
	sc, err := Bootstrap(context.Background(), nil, root, Options{Project: "t", AnchorEngine: "auto"})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	for _, d := range []string{"sources", "concepts", "syntheses", "preferences"} {
		fi, err := os.Stat(filepath.Join(root, d))
		if err != nil || !fi.IsDir() {
			t.Fatalf("missing scaffolded folder %s: %v", d, err)
		}
	}
	for _, f := range []string{"source.md", "concept.md", "synthesis.md", "preference.md"} {
		if _, err := os.Stat(filepath.Join(root, "templates", f)); err != nil {
			t.Fatalf("missing templates/%s: %v", f, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(root, VaultConfigFile))
	if err != nil {
		t.Fatalf("read %s: %v", VaultConfigFile, err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	pt, _ := cfg["page_types"].([]any)
	if len(pt) != 10 {
		t.Fatalf("page_types has %d entries, want 10 (6 core + 4 contextual): %v", len(pt), cfg["page_types"])
	}
	recall, _ := cfg["recall"].(map[string]any)
	if recall == nil {
		t.Fatalf("recall block missing from %s: %v", VaultConfigFile, cfg)
	}
	it, _ := recall["inline_types"].([]any)
	found := false
	for _, v := range it {
		if v == "preference" {
			found = true
		}
	}
	if !found {
		t.Fatalf("recall.inline_types missing preference: %v", it)
	}
	created := 0
	for _, c := range sc.Created {
		if filepath.FromSlash(c) == "templates/source.md" {
			created++
		}
	}
	if created != 1 {
		t.Fatalf("Created should record templates/source.md exactly once: %v", sc.Created)
	}
}
