package memory

// relations.go — the conflict-candidate detection for the save pipeline
// (spec 01-memory/003 §3): step 4, the lexical pre-filter, and the
// deterministic relevance gate. The surviving candidates are materialized as
// pending memory_relations rows by the pipeline (step 7.5) and returned in
// SaveResult.Candidates with JudgmentRequired=true.
//
// This is a lexical heuristic, not semantic detection (Engram, explicitly):
// synonyms and one-term overlaps can miss a real conflict; the ignored-word
// list is English-specific. The LLM (the agent or mem_compare, Task 10) is
// the semantic layer.

import (
	"context"
	"strings"
	"unicode"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// candidateBound is the per-source bound on materialized pending candidates
// (spec 01-memory/003 §3 step 3: "bounded per-source, default 3").
const candidateBound = 3

// gateIgnoredWords are the common connectives + generic change verbs the
// relevance gate ignores (spec 01-memory/003 §3 step 2).
var gateIgnoredWords = map[string]bool{
	"updated": true, "fixed": true, "added": true, "removed": true, "changed": true,
	"and": true, "the": true, "a": true, "an": true, "to": true, "for": true, "of": true,
}

// titleTopic is a (title, topic_key) pair — the input to the gate.
type titleTopic struct {
	Title    string
	TopicKey string
}

// significantTerms tokenizes a title into its significant terms:
// case-insensitive runs of letter/digit characters, with `+`/`#` preserved
// attached to the preceding run so that identifiers stay distinct
// (`C++` ≠ `C#`; both are single terms, and neither equals the other).
// Ignored words (gateIgnoredWords) are dropped.
func significantTerms(s string) []string {
	var out []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		term := strings.ToLower(cur.String())
		cur.Reset()
		if gateIgnoredWords[term] {
			return
		}
		out = append(out, term)
	}
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			cur.WriteRune(r)
		case r == '+' || r == '#':
			// attach to the preceding run (an empty run + a symbol is not a
			// term — a title starting with "#" carries no term for it).
			if cur.Len() > 0 {
				cur.WriteRune(r)
			}
		default:
			flush()
		}
	}
	flush()
	return out
}

// candidateGate is the deterministic relevance gate (spec 01-memory/003 §3
// step 2): accept a candidate iff it shares a nonempty topic_key, OR shares
// ≥2 distinct significant title terms, OR both titles reduce to the same
// single significant term. Content-only matches do not establish relevance.
func candidateGate(a, b titleTopic) bool {
	if a.TopicKey != "" && a.TopicKey == b.TopicKey {
		return true
	}
	ta, tb := map[string]bool{}, map[string]bool{}
	for _, t := range significantTerms(a.Title) {
		ta[t] = true
	}
	for _, t := range significantTerms(b.Title) {
		tb[t] = true
	}
	shared := 0
	for t := range ta {
		if tb[t] {
			shared++
		}
	}
	if shared >= 2 {
		return true
	}
	// the "same single significant term" clause: both titles reduce to
	// exactly one term, and it is the same term.
	if len(ta) == 1 && len(tb) == 1 {
		for t := range ta {
			if tb[t] {
				return true
			}
		}
	}
	return false
}

// conflictPreFilter returns the OR-based FTS candidate set for a save's
// title (spec 01-memory/003 §3 step 1): OR over the title's significant
// terms (the Engram sanitizeFTSCandidates OR mode), raw BM25 order,
// scoped to project+scope, live rows only. A failed or empty query
// returns no candidates (a pre-filter miss is not evidence of absence,
// but it does not fail the save — the lexical heuristic may miss, §3
// note).
//
// The OR set is small (a title's term count); 20 pre-gate rows is plenty
// — the bound on what is materialized is candidateBound, applied after
// the gate.
//
// ponytail: the FTS query runs through the store's RawQuery read seam
// rather than store.Search because Search's hasShortFTSTerm guard counts
// the 2-rune OR keyword as a short term and routes the query to the LIKE
// fallback, which AND-joins terms and would never match the literal "OR"
// token. RawQuery is the narrow read seam for this (the conflict pre-
// filter is the only multi-row, multi-column raw read the memory package
// needs; the Probe/ProbeInt seams return single values only).
func (s *Service) conflictPreFilter(ctx context.Context, project, scope, title string) []store.Observation {
	terms := significantTerms(title)
	if len(terms) == 0 {
		return nil
	}
	quoted := make([]string, len(terms))
	for i, t := range terms {
		quoted[i] = `"` + strings.ReplaceAll(t, `"`, `""`) + `"`
	}
	// ponytail: the RawQuery seam is on the SQLite adapter only; a dialect
	// without it (Task 18's Postgres) degrades to no candidates — a
	// pre-filter miss is not evidence of absence (§3 note), and a panic
	// in the save pipeline would be a regression on the INV-3 posture
	// (the seams may be absent; the deterministic floor still runs).
	rq, ok := s.st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return nil
	}
	q := `SELECT o.id, ifnull(o.sync_id,''), o.title, o.scope, ifnull(o.topic_key,'')
		FROM observations_fts fts
		CROSS JOIN observations o ON o.id = fts.rowid
		WHERE observations_fts MATCH ? AND o.deleted_at IS NULL
		  AND LOWER(o.project) = ? AND o.scope = ?
		ORDER BY bm25(observations_fts, 5.0, 1.0, 0.0, 0.0, 0.0, 3.0)
		LIMIT 20`
	rows, err := rq.RawQuery(q, strings.Join(quoted, " OR "), project, scope)
	if err != nil {
		return nil
	}
	out := make([]store.Observation, 0, len(rows))
	for _, row := range rows {
		out = append(out, store.Observation{
			ID:       row[0].(int64),
			SyncID:   row[1].(string),
			Title:    row[2].(string),
			Scope:    row[3].(string),
			TopicKey: strPtr(row[4].(string)),
		})
	}
	return out
}

// strPtr returns a pointer to s, or nil for the empty string.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// materializeCandidates writes the gate's surviving observations as pending
// memory_relations rows (source = the new save, target = the candidate) and
// returns the stored rows (bounded to candidateBound). One AddRelation per
// candidate — the relations rows are independent; a failure drops that
// candidate and returns the partial list (the save itself has already
// committed — the relations are an after-save step, not part of the save
// transaction; see the ponytail comment for the ceiling).
func (s *Service) materializeCandidates(ctx context.Context, sourceSyncID, sessionID string, cands []store.Observation) []store.Relation {
	out := make([]store.Relation, 0, candidateBound)
	for _, c := range cands {
		if len(out) >= candidateBound {
			break
		}
		rel, err := s.st.AddRelation(ctx, store.AddRelationParams{
			SourceID:  sourceSyncID,
			TargetID:  c.SyncID,
			Relation:  "pending",
			Reason:    "lexical conflict candidate (title pre-filter + relevance gate)",
			SessionID: sessionID,
		})
		if err != nil {
			// ponytail: a relation write failure drops only that candidate
			// (the save is already committed; the wiki reads pending rows,
			// a missing one is a miss, not a wrong verdict). Upgrade path:
			// fold the materialization into the save transaction when the
			// gate grows a transactional method (Task 18's Postgres seam).
			return out
		}
		out = append(out, rel)
	}
	return out
}
