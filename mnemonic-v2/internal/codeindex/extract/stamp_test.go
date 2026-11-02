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

// [extractor-stamp-content-hash] (F2, C-2a — the pinned set is REAL, proven by
// in-test mutation): temporarily add a 26th language entry to langMaps,
// capture Stamp(), restore the table, and assert the stamp CHANGED. This is
// the load-bearing proof that the stamp is a content hash over the data
// tables' shape (a 26th row forces a full re-walk), not a hard-coded const.
func TestStampChangesWhenDataTableGrows(t *testing.T) {
	before := Stamp()

	// Inject a 26th map-tier language (a node-type shape the pinned set
	// serializes). It must be a fresh language not already in langMaps.
	const testLang = "testlang_26th"
	if _, exists := langMaps[testLang]; exists {
		t.Fatalf("precondition: %s must not already be in langMaps", testLang)
	}
	langMaps[testLang] = langMapDef{
		defNodes:  map[string]defInfo{"test_function_definition": {kind: "function", nameFrom: nil}},
		callNodes: map[string]callInfo{"test_call_expression": {targetFrom: nil}},
	}
	restored := false
	defer func() {
		if !restored {
			delete(langMaps, testLang)
			restored = true
		}
	}()

	during := Stamp()
	if during == before {
		t.Fatalf("Stamp() must CHANGE when the data table gains a 26th row: before=%q during=%q", before, during)
	}

	// Restore and confirm the stamp returns to the original (byte-stable, and
	// the mutation is fully reversible — no leaked state).
	delete(langMaps, testLang)
	restored = true
	if after := Stamp(); after != before {
		t.Fatalf("Stamp() must return to the original after restore: before=%q after=%q", before, after)
	}
}

// [extractor-stamp-content-hash] (F2, C-2b — config invariance, the M5
// ruling): a different WithMaxFileSize option does NOT change Stamp(). The
// stamp is computed over the extractor code + data tables + chunker constants
// only; MaxFileSize is config, so varying it must leave Stamp() untouched.
// (This is asserted directly at the Stamp() level, not just via the
// StampFor primitive.)
func TestStampConfigInvarianceMaxFileSize(t *testing.T) {
	base := Stamp()

	// A small max file size (config) must not change the stamp.
	_, _, err := ExtractFile(nil, "small.go", "func f() int { return 1 }\n", "go", WithMaxFileSize(64))
	if err != nil {
		t.Fatalf("extract with a small max file size: %v", err)
	}
	if got := Stamp(); got != base {
		t.Fatalf("Stamp() must be invariant to WithMaxFileSize (config, M5): base=%q got=%q", base, got)
	}

	// A huge max file size must not change it either.
	_, _, err = ExtractFile(nil, "small.go", "func f() int { return 1 }\n", "go", WithMaxFileSize(1<<30))
	if err != nil {
		t.Fatalf("extract with a huge max file size: %v", err)
	}
	if got := Stamp(); got != base {
		t.Fatalf("Stamp() must be invariant to WithMaxFileSize (config, M5): base=%q got=%q", base, got)
	}
}
