package judgment

import (
	"strings"
	"time"
)

// defaultTimeout bounds a single Ollama call so a hanging endpoint never stalls
// the request (ADR-0016 fail-open).
const defaultTimeout = 2 * time.Second

// nowMillis returns the current unix time in milliseconds for judged_at stamps.
func nowMillis() int64 { return time.Now().UnixMilli() }

// PreFilterClass is the deterministic pre-filter classification of a unit.
type PreFilterClass int

const (
	// ClassPin: keep as-is — a short, self-contained, high-signal unit
	// (facts-first: a fact, a decision, a short answer).
	ClassPin PreFilterClass = iota
	// ClassDead: structural dead weight — empty, pure whitespace, or a pure
	// boilerplate/label unit.
	ClassDead
	// ClassRemainder: needs a model decision.
	ClassRemainder
)

// String renders the class for logging.
func (c PreFilterClass) String() string {
	switch c {
	case ClassPin:
		return "pin"
	case ClassDead:
		return "dead"
	default:
		return "remainder"
	}
}

// pinMaxWords is the facts-first bound: a unit at or below this many words is
// treated as a self-contained, high-signal unit (pin) and kept as-is.
const pinMaxWords = 24

// PreFilter deterministically classifies a unit without a model call. It is a
// pure function: the same unit always yields the same class (pre-filter
// determinism AC). It runs even when the model is down.
//
// Rules (plan §18 facts-first / structural-dead / label-remainder):
//   - dead: empty content, pure whitespace, or a pure label/boilerplate unit
//     (no sentence content — e.g. a lone heading marker, a role tag, or a
//     single non-word token).
//   - pin: short (<= pinMaxWords), self-contained, high-signal — a fact,
//     decision, or short answer.
//   - remainder: everything else — needs a model decision.
func PreFilter(u Unit) PreFilterClass {
	content := u.Content

	// Structural dead weight: empty or pure whitespace.
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ClassDead
	}

	// Pure boilerplate/label: a lone marker with no sentence content. A unit
	// that is a single token of non-letters (e.g. "##", "[label]", "---") or a
	// pure label line (all-caps tag, no spaces, no sentence punctuation) is
	// dead weight.
	if isLabelOnly(trimmed) {
		return ClassDead
	}

	// Facts-first: a short, self-contained, high-signal unit is pinned as-is.
	words := strings.Fields(trimmed)
	if len(words) <= pinMaxWords {
		return ClassPin
	}

	// Otherwise it needs a model decision.
	return ClassRemainder
}

// isLabelOnly reports whether s is a pure label/boilerplate unit: a single
// token made of non-letters (markers like "##", "---", "[label]") or an
// all-caps tag with no spaces and no sentence punctuation. These carry no
// compressible content.
func isLabelOnly(s string) bool {
	// A single token with no spaces.
	if strings.ContainsAny(s, " \t\n") {
		return false
	}
	// Marker of pure non-letters (e.g. "##", "---", "***").
	allNonLetter := true
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
			allNonLetter = false
			break
		}
	}
	if allNonLetter {
		return true
	}
	// An all-caps single word (a label/tag) with no sentence punctuation.
	upper := s
	lower := strings.ToLower(s)
	if upper == strings.ToUpper(upper) && upper != lower && !strings.ContainsAny(lower, ".?!") {
		// Must contain at least one letter to be a label (not a number).
		hasLetter := false
		for _, r := range lower {
			if r >= 'a' && r <= 'z' {
				hasLetter = true
				break
			}
		}
		if hasLetter {
			return true
		}
	}
	return false
}
