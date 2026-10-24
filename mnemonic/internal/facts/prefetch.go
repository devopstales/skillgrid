// Prefetch injection — the frozen snapshot renderer (change
// 2026-10-24-global-llm-wiki, Change A, TICKET-05).
//
// The snapshot is the system-prompt injection block: every live global fact,
// split into the two labeled sections (GLOBAL MEMORY = category memory, USER
// PROFILE = category user), each headed by its usage header (percent +
// used/limit char count, the 2200/1375 budgets from CharLimit), entries
// separated by exactly "\n§\n" and sorted by importance_score DESC (ties by
// id).
//
// "Frozen" is a property of the INJECTION MOMENT, not the renderer (blueprint
// M5): RenderFrozenSnapshot is a pure function of the store — it renders
// everything that is in the store at call time. The per-session cache is
// Snapshot: the OpenCode plugin hook fires on every LLM request, so it
// memoizes the first render per session id and returns the cached block on
// every later request (mid-session fact_add does not change the prompt until
// the next session).
//
// Fail-open: the adapters (OpenCode plugin hook, MCP resource memory/global,
// CLI `memory global prefetch`) map ANY render error to an empty string —
// the system prompt injection must never break the session.
package facts

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// snapshotRule is the 40-char header rule that opens and closes every
// section header (the acceptance shape).
const snapshotRule = "════════════════════════════════════════"

// snapshotSections is the fixed section order and labels: GLOBAL MEMORY
// (category memory) first, USER PROFILE (category user) second.
var snapshotSections = []struct {
	label    string
	category string
}{
	{"GLOBAL MEMORY", CategoryMemory},
	{"USER PROFILE", CategoryUser},
}

// RenderFrozenSnapshot renders the frozen injection block for the store:
// both sections, usage headers (percent + used/limit), entries separated by
// "\n§\n", sorted by importance_score DESC (ties by id). A nil store is a
// clean error; an empty store renders both headers at 0% with no entries and
// no error (G6). The adapters fail open: any error here becomes an empty
// string at the injection seam.
func RenderFrozenSnapshot(ctx context.Context, s *Store) (string, error) {
	if s == nil || s.db == nil {
		return "", fmt.Errorf("facts: prefetch requires the global store handle")
	}
	byCategory := map[string][]snapshotEntry{}
	usedByCategory := map[string]int{}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, content, category, importance_score
		FROM facts
		WHERE deleted_at IS NULL
		ORDER BY importance_score DESC, id`)
	if err != nil {
		return "", fmt.Errorf("prefetch list facts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var e snapshotEntry
		var category string
		if err := rows.Scan(&e.id, &e.content, &category, &e.score); err != nil {
			return "", fmt.Errorf("prefetch scan fact: %w", err)
		}
		if category == "" {
			category = CategoryMemory
		}
		byCategory[category] = append(byCategory[category], e)
		usedByCategory[category] += len(e.content)
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("prefetch iterate facts: %w", err)
	}
	var b strings.Builder
	for _, sec := range snapshotSections {
		entries := byCategory[sec.category]
		// The query order (importance DESC, id) already sorts; re-assert it
		// defensively so the contract holds even if the query changes.
		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].score != entries[j].score {
				return entries[i].score > entries[j].score
			}
			return entries[i].id < entries[j].id
		})
		limit := CharLimit(sec.category)
		b.WriteByte('\n')
		b.WriteString(snapshotRule)
		b.WriteByte('\n')
		b.WriteString(fmt.Sprintf("%s [%d%% — %s/%s chars]", sec.label, snapshotPct(usedByCategory[sec.category], limit),
			withCommas(usedByCategory[sec.category]), withCommas(limit)))
		b.WriteByte('\n')
		b.WriteString(snapshotRule)
		b.WriteByte('\n')
		b.WriteString(entryJoin(entryContents(entries)))
	}
	return b.String(), nil
}

// entryJoin joins a section's entries so the rendered bytes between any two
// entries are exactly "\n§\n" (the acceptance contract): every entry is
// written on its own line, preceded by the section sign. A non-empty section
// ends with a trailing newline (before the next section's opening rule); an
// empty section renders nothing after its closing rule.
func entryJoin(contents []string) string {
	var b strings.Builder
	for i, c := range contents {
		if i > 0 {
			b.WriteString("§\n")
		}
		b.WriteString(c)
		b.WriteByte('\n')
	}
	return b.String()
}

// snapshotPct is the usage-header percent: round-half-up of used*100/limit,
// capped at 100; 0 for a zero used or limit.
func snapshotPct(used, limit int) int {
	if limit <= 0 || used <= 0 {
		return 0
	}
	p := (used*100 + limit/2) / limit
	if p > 100 {
		p = 100
	}
	return p
}

// snapshotEntry is one live fact row in the rendered block (id anchors the
// stable tie-break, score is the sort key).
type snapshotEntry struct {
	content string
	score   float64
	id      int64
}

// entryContents flattens a sorted entry list to its contents (for the §-join).
func entryContents(entries []snapshotEntry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.content
	}
	return out
}

// Snapshot is the per-session frozen cache in front of RenderFrozenSnapshot.
// The frozen contract (blueprint M8: render once, cache, return the cached
// block on every later call for the same session) is enforced here for the
// Go seams — the CLI `memory global prefetch` and the MCP resource — and by
// the OpenCode plugin's own in-memory cache (the TS hook cannot call Go
// directly; it re-fetches and memoizes per session id, the same contract).
type Snapshot struct {
	mu   sync.Mutex
	s    *Store
	cash map[string]string
}

// NewSnapshot builds a per-session cache over the global store. A nil store
// renders a clean error on every Request (the adapters map it to "").
func NewSnapshot(s *Store) *Snapshot {
	return &Snapshot{s: s, cash: map[string]string{}}
}

// Request returns the frozen block for sessionID: the first call renders
// (and caches) the snapshot; later calls for the same id return the cached
// result verbatim. A different id renders fresh. Render errors are returned
// to the caller (NOT cached) and are not retried against the cache — the
// caller fails open.
func (snap *Snapshot) Request(ctx context.Context, sessionID string) (string, error) {
	if snap == nil || snap.s == nil || snap.s.db == nil {
		return "", fmt.Errorf("facts: snapshot requires the global store handle")
	}
	snap.mu.Lock()
	defer snap.mu.Unlock()
	if cached, ok := snap.cash[sessionID]; ok {
		return cached, nil
	}
	out, err := RenderFrozenSnapshot(ctx, snap.s)
	if err != nil {
		return "", err
	}
	snap.cash[sessionID] = out
	return out, nil
}
