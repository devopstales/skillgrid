package memory

// Task 16 — chunk codec (port of Engram internal/cloud/chunkcodec).
//
// The [chunk-id-stable] acceptance scenario (acceptance.feature, G14):
// two payloads that differ only in key order canonicalize to the same
// bytes for the same project and therefore hash to the same 8-hex chunk
// id (the chunk identity is content-address, key-order independent).

import (
	"encoding/json"
	"strings"
	"testing"
)

// [chunk-id-stable] payloads that differ only in key order canonicalize to
// the same bytes for the same project → the same 8-hex chunk id.
func TestChunkIDStable(t *testing.T) {
	p1 := CanonicalizeForProject([]byte(`{"content":"a","project":""}`), "p")
	p2 := CanonicalizeForProject([]byte(`{"project":"","content":"a"}`), "p") // key order differs
	if ChunkID(p1) != ChunkID(p2) {
		t.Fatalf("canonicalization must be key-order stable: %q vs %q", ChunkID(p1), ChunkID(p2))
	}
	if len(ChunkID(p1)) != 8 {
		t.Fatalf("chunk id must be 8 hex, got %q", ChunkID(p1))
	}
	// the canonical form must be a valid JSON object (re-keyed, stable order).
	var doc map[string]any
	if err := json.Unmarshal(p1, &doc); err != nil {
		t.Fatalf("canonical payload must be JSON: %v", err)
	}
	if doc["project"] != "p" {
		t.Fatalf("canonicalization must stamp the export project: %q", string(p1))
	}
}

// TestCanonicalizeBlankProjectInherits (TRIANGULATE [chunk-id-stable],
// different inputs): a blank project field inherits the export project
// (spec 002-schema §3.1 payload rule) — the same bytes as a payload that
// already carried the project, so both hash identically.
func TestCanonicalizeBlankProjectInherits(t *testing.T) {
	blank := CanonicalizeForProject([]byte(`{"sync_id":"obs-1","content":"a","project":""}`), "p")
	stamped := CanonicalizeForProject([]byte(`{"sync_id":"obs-1","content":"a","project":"p"}`), "p")
	if ChunkID(blank) != ChunkID(stamped) {
		t.Fatalf("blank project must inherit the export project: %q vs %q", ChunkID(blank), ChunkID(stamped))
	}
}

// TestCanonicalizeRejectsNonJSON (TRIANGULATE, error path): a payload that
// is not a JSON object is a hard error (the chunk is content-addressed; a
// non-JSON chunk cannot be canonically re-keyed).
func TestCanonicalizeRejectsNonJSON(t *testing.T) {
	if err := CanonicalizeError([]byte(`not-json`), "p"); err == nil {
		t.Fatalf("non-JSON payload must be rejected")
	}
	if err := CanonicalizeError([]byte(`[1,2,3]`), "p"); err == nil {
		t.Fatalf("a JSON array is not an object chunk: must be rejected")
	}
	if !strings.Contains(CanonicalizeError([]byte(`not-json`), "p").Error(), "json") {
		t.Fatalf("error must name the JSON failure: %v", CanonicalizeError([]byte(`not-json`), "p"))
	}
}
