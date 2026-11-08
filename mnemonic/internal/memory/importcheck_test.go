package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// [memory-does-not-import-wiki] dependency direction (spec 000-architecture
// §3): the wiki consumes the SaveEvent; memory must never import the wiki.
// The wiki package does not exist yet (it lands in a later change), so this
// is a source scan: no .go file in this package may reference the marker
// string "llm" + "wiki" (assembled below so this guard's own file does not
// trip the scan) — a hit is a memory→wiki import, the inverted dependency.
// The wiki_teststub test proves the other direction: a consumer outside
// internal/memory (internal/wiki_teststub) that imports only the event-bus
// package + event type and receives a well-formed SaveEvent after a save
// driven through memory.Service.
func TestMemoryDoesNotImportWiki(t *testing.T) {
	// The wiki package's name. Assembled so this file's source does not
	// contain the literal marker (the guard would otherwise trip on its own
	// comment).
	const marker = "llm" + "wiki"
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	var hits []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(".", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if strings.Contains(string(b), marker) {
			hits = append(hits, name)
		}
	}
	if len(hits) > 0 {
		t.Fatalf("internal/memory references %s (dependency inversion: the wiki consumes the event, memory must not import it): %v", marker, hits)
	}
}
