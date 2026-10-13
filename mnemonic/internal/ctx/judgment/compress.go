package judgment

import (
	"strings"
)

// truncateBudgetWords caps a truncated unit at this many words. Truncate-only:
// the unit is shortened to the budget, never deleted and never rewritten
// (the kept prefix is byte-identical to the original's start).
const truncateBudgetWords = 12

// Compress applies truncate-only compression to a unit according to the
// decision. For ActionKeep it returns the unit unchanged (byte-identical). For
// ActionTruncate it returns the unit with Content capped to the budget — a
// strict prefix of the original words, so no stored message is rewritten and
// the unit is never deleted.
func Compress(u Unit, d Decision) Unit {
	if d.Action != ActionTruncate {
		return u
	}
	words := strings.Fields(u.Content)
	if len(words) <= truncateBudgetWords {
		// Already within budget: keep as-is (truncation would be a no-op).
		return u
	}
	capped := strings.Join(words[:truncateBudgetWords], " ")
	// Never expand: if the capped result is not strictly shorter, keep original.
	if len(capped) >= len(u.Content) {
		return u
	}
	return Unit{Role: u.Role, Content: capped}
}
