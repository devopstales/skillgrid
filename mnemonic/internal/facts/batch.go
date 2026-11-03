// Batch operations + char limit + substring matching (change
// 2026-10-24-global-llm-wiki, Change A, TICKET-04).
//
// A batch is an operations array (add/patch/remove) applied as a UNIT to one
// category of one store. The per-category char budget (memory 2200 / user
// 1375, config mnemonic.global_memory) is checked on the batch's FINAL
// result only — never per-op — so a batch that frees 200 chars and adds 250
// to a 2100/2200 store succeeds (final 2150/2200) even though the add alone
// would overshoot. On overflow the whole batch is rolled back and the error
// is a *OverflowError carrying current_entries + usage ("2,150/2,200") so
// the agent can free space and retry.
//
// add: exact-content dedup — content matching an existing live fact is a
// success ("no duplicate added"), not an error, and adds no row.
//
// patch/remove: old_text is a SUBSTRING match against live facts. Zero
// matches → *NoMatchError with closest_entries (the most similar existing
// facts); more than one → *AmbiguousMatchError. patch replaces the old_text
// portion of the matched fact with content; remove soft-deletes it.
package facts

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Operation actions for the batch operations array.
const (
	OpAdd    = "add"
	OpPatch  = "patch"
	OpRemove = "remove"
)

// Operation is one entry of a batch's operations array. add needs Content;
// patch needs OldText + Content (replace the matched substring with
// Content); remove needs OldText.
type Operation struct {
	Action  string `json:"action"`
	Content string `json:"content,omitempty"`
	OldText string `json:"old_text,omitempty"`
}

// BatchResult is the structured outcome of a successful Batch.
type BatchResult struct {
	Success          bool     `json:"success"`
	NoDuplicateAdded bool     `json:"no_duplicate_added,omitempty"`
	Usage            string   `json:"usage,omitempty"`
	CurrentEntries   []string `json:"current_entries,omitempty"`
}

// OverflowError is returned when a batch's FINAL result exceeds the
// category's char budget. CurrentEntries are the existing live fact contents
// (so the agent can see what to free) and Usage is the pre-batch usage
// string ("2,150/2,200"). No ops were applied.
type OverflowError struct {
	CurrentEntries []string
	Usage          string
	OverBy         int
}

func (e *OverflowError) Error() string {
	return fmt.Sprintf("memory at %s chars; adding this entry would exceed the limit by %d chars (current_entries: %s)",
		e.Usage, e.OverBy, strings.Join(e.CurrentEntries, " | "))
}

// AmbiguousMatchError is returned when a patch/remove old_text matches MORE
// than one live fact. No op was applied.
type AmbiguousMatchError struct {
	OldText    string
	MatchCount int
}

func (e *AmbiguousMatchError) Error() string {
	return fmt.Sprintf("ambiguous match: old_text %q matches %d facts (use a longer, unique substring)", e.OldText, e.MatchCount)
}

// NoMatchError is returned when a patch/remove old_text matches ZERO live
// facts. ClosestEntries are the most similar existing facts (by shared
// words, most first). No op was applied.
type NoMatchError struct {
	OldText        string
	ClosestEntries []string
}

func (e *NoMatchError) Error() string {
	return fmt.Sprintf("no match: old_text %q matches no fact (closest_entries: %s)",
		e.OldText, strings.Join(e.ClosestEntries, " | "))
}

// CharLimit returns the category's char budget: memory 2200 / user 1375
// (the Hermes MEMORY.md / USER.md budgets).
//
// ponytail: the budgets are constants; the config wiring (mnemonic.global_memory
// in config.d/indexing.yaml → facts.Store) is deferred because the MCP handler
// does not currently thread config.Load into the facts layer. The config
// loader already parses the section (config.GlobalMemory) — a one-line seam
// in the handler would let an operator override these.
func CharLimit(category string) int {
	if category == CategoryUser {
		return 1375
	}
	return 2200
}

// usageString renders "used/limit" with thousands separators (the
// acceptance contract: "2,150/2,200").
func usageString(used, limit int) string {
	return fmt.Sprintf("%s/%s", withCommas(used), withCommas(limit))
}

// withCommas renders an int with US thousands separators (no locale):
// 2150 → "2,150".
func withCommas(n int) string {
	s := fmt.Sprintf("%d", n)
	rem := len(s) % 3
	if rem == 0 {
		rem = 3
	}
	// The first group has `rem` digits; the rest are 3. Insert a comma
	// before every digit that starts a new group.
	out := s[:rem]
	for i := rem; i < len(s); i += 3 {
		out += "," + s[i:i+3]
	}
	return out
}

// Batch applies the operations array as a unit to one category of the store.
// The char budget is always enforced at the category's production limit
// (memory 2200 / user 1375 via CharLimit) — a zero limit would make the
// bounded-store contract vacuous, and the only unbounded store (project
// facts) is never batched (project writes go through Add/AddWith).
//
// Contract:
//   - Validation (action names, required params) runs first; a bad op is a
//     clean error and nothing is applied.
//   - The budget is checked on the FINAL result only. Overflow → the whole
//     batch is rolled back (a single transaction covers every row change)
//     and a *OverflowError carries current_entries + pre-batch usage.
//   - add dedups on EXACT content against live facts: a match is a success
//     with NoDuplicateAdded, no row inserted.
//   - patch/remove match old_text as a SUBSTRING: zero matches →
//     *NoMatchError with closest_entries; more than one →
//     *AmbiguousMatchError.
func (s *Store) Batch(ctx context.Context, category string, ops []Operation) (BatchResult, error) {
	if s == nil || s.db == nil {
		return BatchResult{}, fmt.Errorf("facts store not initialized")
	}
	if category != CategoryMemory && category != CategoryUser {
		return BatchResult{}, fmt.Errorf("fact category %q is not one of memory|user", category)
	}
	limit := CharLimit(category)
	if len(ops) == 0 {
		return BatchResult{}, fmt.Errorf("batch requires at least one operation")
	}
	for i, op := range ops {
		switch op.Action {
		case OpAdd:
			if strings.TrimSpace(op.Content) == "" {
				return BatchResult{}, fmt.Errorf("operation %d (add): content is required", i)
			}
		case OpPatch:
			if strings.TrimSpace(op.OldText) == "" {
				return BatchResult{}, fmt.Errorf("operation %d (patch): old_text is required", i)
			}
		case OpRemove:
			if strings.TrimSpace(op.OldText) == "" {
				return BatchResult{}, fmt.Errorf("operation %d (remove): old_text is required", i)
			}
		default:
			return BatchResult{}, fmt.Errorf("operation %d: unknown action %q (want add, patch, or remove)", i, op.Action)
		}
	}

	// The live working set: category-filtered so the budget and the
	// substring/dedup matching both see exactly the facts they act on.
	// preBatch keeps the untouched state for the overflow error's
	// current_entries; live is the evolving simulation set.
	live, err := s.categoryFacts(ctx, category)
	if err != nil {
		return BatchResult{}, err
	}
	preBatch := append([]liveFact(nil), live...)
	used := 0
	for _, f := range live {
		used += len(f.Content)
	}

	// Simulate the ops against the working set (no DB writes yet): the
	// budget check runs on the FINAL result, substring matches are resolved
	// against the evolving set (an earlier remove makes a later old_text
	// unique), and dedup runs against the set including this batch's adds.
	// Each resolved op becomes a planned row change (factID anchors it to the
	// real row in the persist phase).
	var plan []plannedOp
	for _, op := range ops {
		switch op.Action {
		case OpAdd:
			if containsContent(live, op.Content) {
				plan = append(plan, plannedOp{Action: OpAdd, Content: op.Content, Duplicate: true})
				continue // exact duplicate: no row, no growth
			}
			plan = append(plan, plannedOp{Action: OpAdd, Content: op.Content})
			live = append(live, liveFact{Content: op.Content})
		case OpPatch:
			matches := matchOldText(live, op.OldText)
			switch len(matches) {
			case 0:
				return BatchResult{}, &NoMatchError{OldText: op.OldText, ClosestEntries: closestEntries(op.OldText, live, 5)}
			case 1:
				f := &live[matches[0]]
				newContent := replaceFirst(f.Content, op.OldText, op.Content)
				plan = append(plan, plannedOp{Action: OpPatch, FactID: f.ID, Content: newContent})
				f.Content = newContent
			default:
				return BatchResult{}, &AmbiguousMatchError{OldText: op.OldText, MatchCount: len(matches)}
			}
		case OpRemove:
			matches := matchOldText(live, op.OldText)
			switch len(matches) {
			case 0:
				return BatchResult{}, &NoMatchError{OldText: op.OldText, ClosestEntries: closestEntries(op.OldText, live, 5)}
			case 1:
				plan = append(plan, plannedOp{Action: OpRemove, FactID: live[matches[0]].ID})
				live = removeIndex(live, matches[0])
			default:
				return BatchResult{}, &AmbiguousMatchError{OldText: op.OldText, MatchCount: len(matches)}
			}
		}
	}

	finalUsed := 0
	for _, f := range live {
		finalUsed += len(f.Content)
	}
	// The budget is checked on the FINAL result only (never per-op).
	// current_entries is the PRE-batch store state (loaded at the top,
	// before the simulation) — the agent's "what to free" list; including
	// the overflowed add itself would be useless guidance.
	if limit > 0 && finalUsed > limit {
		entries := make([]string, 0, len(preBatch))
		for _, f := range preBatch {
			entries = append(entries, f.Content)
		}
		return BatchResult{}, &OverflowError{
			CurrentEntries: entries,
			Usage:          usageString(used, limit),
			OverBy:         finalUsed - limit,
		}
	}

	// Persist: one transaction covers every planned row change (atomic
	// batch). The plan is the simulation's output — no re-resolution, so a
	// fact whose old_text survives its own patch cannot "drift" out from
	// under the remove that follows it.
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return BatchResult{}, fmt.Errorf("begin batch: %w", err)
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339)
	dup := false
	for _, p := range plan {
		switch p.Action {
		case OpAdd:
			if p.Duplicate {
				dup = true
				continue
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO facts (content, category, created_at, updated_at)
				VALUES (?, ?, ?, ?)`, p.Content, category, now, now); err != nil {
				return BatchResult{}, fmt.Errorf("batch insert: %w", err)
			}
		case OpPatch:
			if _, err := tx.ExecContext(ctx, `
				UPDATE facts SET content = ?, updated_at = ? WHERE id = ?`,
				p.Content, now, p.FactID); err != nil {
				return BatchResult{}, fmt.Errorf("batch patch update: %w", err)
			}
		case OpRemove:
			if _, err := tx.ExecContext(ctx, `
				UPDATE facts SET deleted_at = ?, updated_at = ? WHERE id = ?`,
				now, now, p.FactID); err != nil {
				return BatchResult{}, fmt.Errorf("batch remove update: %w", err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return BatchResult{}, fmt.Errorf("commit batch: %w", err)
	}

	entries := make([]string, 0, len(live))
	for _, f := range live {
		entries = append(entries, f.Content)
	}
	return BatchResult{
		Success:          true,
		NoDuplicateAdded: dup,
		Usage:            usageString(finalUsed, limit),
		CurrentEntries:   entries,
	}, nil
}

// liveFact is one live fact row in the batch's working set (the id anchors
// the persist phase back to the real row).
type liveFact struct {
	ID      int64
	Content string
}

// plannedOp is one resolved row change from the simulation phase: the
// persist phase applies these verbatim inside one transaction (no
// re-resolution — a fact whose old_text survives its own patch cannot drift
// out from under the remove that follows it).
type plannedOp struct {
	Action    string // OpAdd | OpPatch | OpRemove
	FactID    int64  // patch/remove target (0 for add)
	Content   string // add content / patched content
	Duplicate bool   // add that matched an existing exact duplicate
}

// categoryFacts loads every live fact of the category (oldest first — a
// stable order for matching and closest-entry ranking).
func (s *Store) categoryFacts(ctx context.Context, category string) ([]liveFact, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, content FROM facts
		WHERE category = ? AND deleted_at IS NULL
		ORDER BY created_at, id`, category)
	if err != nil {
		return nil, fmt.Errorf("list facts for batch: %w", err)
	}
	defer rows.Close()
	var out []liveFact
	for rows.Next() {
		var f liveFact
		if err := rows.Scan(&f.ID, &f.Content); err != nil {
			return nil, fmt.Errorf("scan fact for batch: %w", err)
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// containsContent reports whether any working-set fact has EXACTLY this
// content (the dedup check for add).
func containsContent(live []liveFact, content string) bool {
	for _, f := range live {
		if f.Content == content {
			return true
		}
	}
	return false
}

// matchOldText returns the indices of the working-set facts whose content
// CONTAINS oldText (substring match, case-sensitive — the Hermes pattern).
func matchOldText(live []liveFact, oldText string) []int {
	var idx []int
	for i, f := range live {
		if strings.Contains(f.Content, oldText) {
			idx = append(idx, i)
		}
	}
	return idx
}

// replaceFirst replaces the first occurrence of old in s with repl.
func replaceFirst(s, old, repl string) string {
	i := strings.Index(s, old)
	if i < 0 {
		return s
	}
	return s[:i] + repl + s[i+len(old):]
}

// removeIndex drops the element at idx from the slice.
func removeIndex(live []liveFact, idx int) []liveFact {
	return append(live[:idx], live[idx+1:]...)
}

// closestEntries ranks the live facts by similarity to oldText (shared word
// count, most first; ties by content order) and returns at most n. Empty
// when the store is empty.
func closestEntries(oldText string, live []liveFact, n int) []string {
	type scored struct {
		content string
		shared  int
	}
	scoredList := make([]scored, 0, len(live))
	words := wordSet(oldText)
	for _, f := range live {
		shared := 0
		for w := range words {
			if strings.Contains(strings.ToLower(f.Content), w) {
				shared++
			}
		}
		scoredList = append(scoredList, scored{content: f.Content, shared: shared})
	}
	sort.SliceStable(scoredList, func(i, j int) bool {
		return scoredList[i].shared > scoredList[j].shared
	})
	if n > len(scoredList) {
		n = len(scoredList)
	}
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = scoredList[i].content
	}
	return out
}

// wordSet lowercases oldText into a set of its words (the similarity
// vocabulary; stopword-free by design — the fact store is small and any
// shared word is a useful closeness signal).
func wordSet(s string) map[string]bool {
	out := make(map[string]bool)
	for _, w := range strings.Fields(strings.ToLower(s)) {
		out[strings.Trim(w, ".,;:!?()[]{}\"'")] = true
	}
	return out
}
