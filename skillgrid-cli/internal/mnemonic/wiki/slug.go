package wiki

import "strings"

// reservedSlugs are titles whose slug collides with the bundle's own special
// files (index.md, log.md). They are prefixed to avoid the collision.
var reservedSlugs = map[string]bool{
	"index": true,
	"log":   true,
}

// reservedSlugPrefix is prepended to a reserved slug to make it unique.
const reservedSlugPrefix = "page-"

// Slugify turns a human title into a filesystem-safe slug: lowercase ASCII
// alphanumeric runs separated by single hyphens, no leading/trailing hyphens.
//
// Rules:
//   - Only ASCII letters and digits are kept; everything else (spaces,
//     punctuation, accented letters, CJK, symbols) becomes a hyphen boundary.
//   - Consecutive boundaries collapse to a single hyphen.
//   - No leading hyphen (the first kept rune never emits a separator) and no
//     trailing hyphen (the builder only emits a separator before a kept rune).
//   - Empty / all-non-ASCII input yields the empty string.
//   - Reserved titles ("index", "log") are prefixed with "page-" to avoid
//     colliding with the bundle's own index.md / log.md.
//
// Choosing ASCII-only: the brief says "alnum + -" and the slug is used both as
// a filename and an ID/URL segment. Dropping non-ASCII keeps slugs portable
// across filesystems and URL schemes without transliteration tables.
//
// Slugify is pure and deterministic.
func Slugify(title string) string {
	var b strings.Builder
	prevHyphen := true // treat start as hyphen so we don't emit a leading one
	for _, r := range strings.ToLower(title) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if prevHyphen && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteByte(byte(r))
			prevHyphen = false
		} else {
			// Anything that is not ASCII alnum is a hyphen boundary.
			prevHyphen = true
		}
	}
	out := b.String()
	if reservedSlugs[out] {
		out = reservedSlugPrefix + out
	}
	return out
}

// typeDir maps a concept type to its bundle directory.
//
//	ADR                          → adr
//	Term, Constraint             → concepts
//	Spec, Spike, State,          → entities
//	Architecture, Finding
//	unknown                      → entities
//
// TypeDir is pure and deterministic.
func TypeDir(typ string) string {
	switch strings.TrimSpace(typ) {
	case "ADR":
		return "adr"
	case "Term", "Constraint":
		return "concepts"
	case "Spec", "Spike", "State", "Architecture", "Finding":
		return "entities"
	default:
		return "entities"
	}
}
