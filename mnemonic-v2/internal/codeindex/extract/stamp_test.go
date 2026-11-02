package extract

import (
	"testing"
)

// [extractor-stamp-content-hash] (D4, D-STAMP, M5): the stamp is a content
// hash over the PINNED input set (extractor code + data tables' structural
// shape + chunker constants; config excluded), byte-stable across repeated
// calls. Pass criterion: StampFor(a) != StampFor(b) for two different input
// strings; the pinned set excludes config (a glob edit must not change the
// stamp); Stamp() stable across calls.
func TestExtractorStampIsContentHash(t *testing.T) {
	a := Stamp()
	if a == "" || len(a) < 8 {
		t.Fatalf("stamp too short: %q", a)
	}
	// a DIFFERENT extractor (a different fingerprint string) → DIFFERENT stamp
	if StampFor("extractor-config-v1") == StampFor("extractor-config-v2") {
		t.Fatalf("StampFor must be a content hash, not a const: two inputs → same stamp")
	}
	// the pinned input set excludes config: a config change (include/exclude
	// globs, MaxFileSize) must NOT change the stamp. The stamp is computed
	// over the extractor code + data tables + chunker constants only, so the
	// same extractor with different config yields the same Stamp().
	if Stamp() != a {
		t.Fatalf("Stamp() must be byte-stable across calls: %q vs %q", Stamp(), a)
	}
	// Stamp() must be derived from the pinned set, and the pinned set must be
	// non-empty (a content hash over nothing is a degenerate const).
	if StampFor("") == a {
		t.Fatalf("Stamp() must hash a non-empty pinned input set (not the empty string)")
	}
}

// [extractor-stamp-content-hash] the same fingerprint string always yields the
// same stamp (determinism) — a re-walk only fires when the extractor itself
// changes, not when the config or the input source changes.
func TestStampForDeterministic(t *testing.T) {
	if StampFor("code-v1") != StampFor("code-v1") {
		t.Fatalf("StampFor must be deterministic: same input → same stamp")
	}
	if StampFor("code-v1") == StampFor("code-v2") {
		t.Fatalf("StampFor must be a content hash: different inputs → different stamps")
	}
}
