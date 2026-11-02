package extract

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
)

// StampFor is the content hash of a single extractor-fingerprint string
// (FNV-1a hex). It is the D-STAMP primitive: two different input strings
// produce two different stamps; the same input always produces the same stamp
// (determinism — a re-walk only fires when the extractor itself changes).
func StampFor(fingerprint string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(fingerprint))
	return fmt.Sprintf("%016x", h.Sum64())
}

// Stamp is the real extractor stamp (D-STAMP, D4): a content hash over the
// PINNED input set — the extractor code's structural fingerprint (the rich-6
// one-pass coverage + the map-tier data tables' shape + the regex battery
// shape + the chunker constants). It is NOT a const (a const never changes, so
// the full re-walk signal never fires) and it EXCLUDES config (include/exclude
// globs, MaxFileSize, debounce — editing them must not force a re-walk, M5).
//
// The pinned input set is derived from the data tables and the constants that
// define the extractor's behavior — not from a hard-coded string (a hard-coded
// string is a const in disguise). Stamp() is byte-stable across repeated
// calls (it hashes only in-process constants).
func Stamp() string {
	return StampFor(pinnedInputSet())
}

// pinnedInputSet serializes the extractor's pinned input set (D-STAMP, M5).
// It is the data that defines extraction behavior:
//
//   - the rich-6 coverage set (which languages use the one-pass tier);
//   - the map-tier data tables' structural shape (each language's def-node and
//     call-node type names, sorted — the 25-row table's shape, NOT its
//     per-language behavior strings);
//   - the regex battery's count (the fallback floor's shape);
//   - the chunker constants (target / merge / window / overlap).
//
// Config (include/exclude globs, MaxFileSize, debounce) is deliberately
// EXCLUDED: a user editing include globs must not trigger a full re-walk.
func pinnedInputSet() string {
	var b strings.Builder

	// 1. the rich-6 coverage set (sorted for determinism).
	rich := make([]string, 0, len(coveredLanguages))
	for l := range coveredLanguages {
		rich = append(rich, l)
	}
	sort.Strings(rich)
	b.WriteString("rich=")
	b.WriteString(strings.Join(rich, ","))
	b.WriteString(";")

	// 2. the map-tier data tables' structural shape: for each language, the
	// sorted def-node type names + sorted call-node type names. This is the
	// 25-row table's shape — a change to a node-type mapping changes the
	// stamp (the extractor's behavior changed); a change to a name-extraction
	// closure that does not change the node-type set does not (documented).
	langs := make([]string, 0, len(langMaps))
	for l := range langMaps {
		langs = append(langs, l)
	}
	sort.Strings(langs)
	for _, l := range langs {
		def := sortedKeys(langMaps[l].defNodes)
		call := sortedKeys(langMaps[l].callNodes)
		b.WriteString(l)
		b.WriteString("{")
		b.WriteString(strings.Join(def, ","))
		b.WriteString("|")
		b.WriteString(strings.Join(call, ","))
		b.WriteString("}")
	}
	b.WriteString(";")

	// 3. the regex battery's shape (the fallback floor).
	b.WriteString("regex=")
	b.WriteString(fmt.Sprintf("%d", len(fallbackFuncRegexes)+len(fallbackImportRegexes)))
	b.WriteString(";")

	// 4. the chunker constants.
	b.WriteString(fmt.Sprintf("chunk=%d/%d/%d/%d",
		semanticChunkTargetChars, minChunkChars, chunkWindowLines, chunkOverlapLines))

	return b.String()
}

// sortedKeys returns a map's keys sorted (deterministic serialization).
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
