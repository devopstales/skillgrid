// Package memory — dream executor (014, step 12).
//
// The DreamExecutor decomposes the session-close distillation into three
// phases: consolidate (merge same-topic facts into one coherent record),
// synthesize (lift lower-tier session summaries into a higher-tier summary),
// and prune (soft-delete low-importance rows). It is additive on the same
// store as *Service (composing via Service.DB(), like the layer package) and
// never changes Save/Search/Distill. The LLM is opt-in and injectable (the
// DreamLLM seam, mirroring ExtractionLLM from step 05); the no-LLM path is a
// deterministic, lossless floor.
package memory

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// dreamStatus is the status flag consolidate() stamps on a source observation
// after its facts have been folded into a consolidated record. It is a NEW
// status value that coexists with the existing "active" default (change 013
// step 01) — it is never written by any pre-dream path, so it is purely
// additive and does not re-read as a lifecycle state.
const dreamStatus = "consolidated"

// dreamSynthesisSource is the source tag on an observation produced by
// synthesize(), so a reader can tell a derived higher-tier summary apart from
// a live observation.
const dreamSynthesisSource = "synthesis"

// DreamTierL2 is the maturity tier of a synthesized cross-session summary
// (one level above the L0/L1 session summaries that are its input).
const DreamTierL2 = "L2"

// PrunedResult reports what prune() removed.
type PrunedResult struct {
	Pruned int    `json:"pruned"`
	IDs    []int64 `json:"ids"`
}

// dreamTypeWeights are the on-the-fly importance weights for the prune phase
// (014 step 12.3). importance = wUsage*usageNorm + wRecency*recency +
// wType*typeWeight, clamped to [0,1]. usageNorm is min(retrieval_usage, 50)/50
// (the same cap the step-08 improve loop uses, so one hot observation cannot
// outrank the recency/type terms). wType weights durable knowledge types
// (decision, architecture, convention, config) higher than transient ones, so a
// never-retrieved architectural decision survives a fresh-but-never-retrieved
// learning. No importance_score / maturity_tier column is added — the score is
// computed on-the-fly from existing columns (the brief's preferred option).
//
// The weights (wUsage=0.6, wRecency=0.37, wType=0.2) are chosen so that at a
// 0.5 threshold: a fresh never-retrieved non-durable note scores 0.45 (pruned),
// while the least-valued keeper — a fresh non-durable note retrieved even once
// (usage 5) — scores 0.51 (kept) and a fresh never-retrieved durable note
// scores 0.55 (kept). Usage dominates, durability breaks the tie at equal
// usage, and age (via the recency grace window) only ever lowers the score.
const (
	wUsage      = 0.6
	wRecency    = 0.37
	wType       = 0.2
	typeWeightDurable = 0.9
	typeWeightDefault = 0.4
)

// durableTypes are the observation types the prune phase weights as durable
// (they carry the higher typeWeight). This is the on-the-fly stand-in for a
// maturity tier: durable knowledge resists pruning more than transient notes.
var durableTypes = map[string]bool{
	"decision":     true,
	"architecture": true,
	"convention":   true,
	"config":       true,
}

// Memory is a lower-tier record synthesize() lifts. For session summaries the
// SessionID is the source session (the provenance the L2 record references);
// for a generic observation it is that observation's session_id.
type Memory struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
}

// Summary is the higher-tier record synthesize() creates.
type Summary struct {
	ID          int64    `json:"id"`
	Tier        string   `json:"tier"`
	Content     string   `json:"content"`
	SourceCount int      `json:"source_count"`
	SessionIDs  []string `json:"session_ids"`
}

// DreamLLM is the pluggable LLM seam for the dream's consolidate/synthesize
// phases (014 step 12). It mirrors the ExtractionLLM / layer.LLM seam pattern
// from step 05: a tiny interface a backend implements, with NO CGo LLM client
// in this package. Both methods take a prompt and return a single text block.
// A nil seam (or one that errors) leaves the phase to its deterministic
// fallback (a lossless join), so the no-LLM path is always available and the
// tests can mock the LLM by supplying a stub.
type DreamLLM interface {
	// ConsolidatePrompt folds a group of same-topic observations into one
	// coherent record. The fallback (no LLM) is a lossless join.
	ConsolidatePrompt(ctx context.Context, topic string, contents []string) (string, error)
	// SynthesizePrompt lifts a set of lower-tier session summaries into a
	// single higher-tier summary. The fallback (no LLM) is a lossless join.
	SynthesizePrompt(ctx context.Context, summaries []string) (string, error)
}

// ConsolidatedGroup reports one topic group that consolidate() merged.
type ConsolidatedGroup struct {
	Topic         string `json:"topic"`
	NewID         int64  `json:"new_id"`
	SourceIDs     []int64 `json:"source_ids"`
	SourceCount   int    `json:"source_count"`
	Consolidated  bool   `json:"consolidated"` // true when merged (group had >1 source)
	MergedContent string `json:"merged_content"`
}

// ConsolidatedResult reports what consolidate() produced.
type ConsolidatedResult struct {
	Groups []ConsolidatedGroup `json:"groups"`
}

// DreamExecutor decomposes the session-close distillation into three phases:
// consolidate (this file, 12.1), synthesize (12.2), and prune (12.3). It is
// additive on the same store as *Service (composing via Service.DB(), like the
// layer package) and never changes Save/Search/Distill. The LLM is opt-in and
// injectable; the no-LLM path is a deterministic, lossless floor.
type DreamExecutor struct {
	svc *Service
	// llm is the optional seam; nil = the deterministic floor (default).
	llm DreamLLM
	// now is the clock seam (default time.Now) used for recency/importance and
	// the lock TTL; tests pin it to make age deterministic.
	now func() time.Time
}

// NewDreamExecutor builds a DreamExecutor bound to the service's store and
// project. The LLM is NOT attached here: a nil seam is the default (the
// deterministic floor), and callers opt in via SetLLM — the same shape as
// Service.SetExtractionLLM (step 05).
func NewDreamExecutor(svc *Service) *DreamExecutor {
	return &DreamExecutor{svc: svc, now: time.Now}
}

// SetLLM attaches the optional LLM seam for consolidate/synthesize. A nil seam
// (the default) keeps the deterministic lossless-join floor.
func (de *DreamExecutor) SetLLM(llm DreamLLM) {
	if de != nil {
		de.llm = llm
	}
}

// consolidate merges observations that share a topic into a single coherent
// record per topic (014 step 12.1). Grouping is by topic_key (an empty
// topic_key is its own group, so a topic-less observation is never merged with
// a different topic's observations). Within a group with >1 member, the group's
// contents are folded into one coherent body via the LLM (or the deterministic
// lossless-join floor), a NEW observation is created with that body, and every
// source is marked consolidated (status = dreamStatus). A singleton group is a
// no-op (nothing to merge). No fact is lost: the merged body contains each
// source's content verbatim (the LLM path keeps the source list alongside the
// summary; the floor is a pure join).
func (de *DreamExecutor) consolidate(ctx context.Context, observations []Observation) (ConsolidatedResult, error) {
	res := ConsolidatedResult{}
	if de == nil || de.svc == nil || de.svc.DB() == nil {
		return res, fmt.Errorf("dream executor not initialized")
	}

	// Group by topic, preserving input order within each group.
	order := make([]string, 0)
	byTopic := make(map[string][]Observation)
	for _, o := range observations {
		topic := topicOf(o)
		if _, ok := byTopic[topic]; !ok {
			order = append(order, topic)
		}
		byTopic[topic] = append(byTopic[topic], o)
	}

	for _, topic := range order {
		group := byTopic[topic]
		if len(group) == 1 {
			g := ConsolidatedGroup{Topic: topic, NewID: group[0].ID, SourceIDs: []int64{group[0].ID}, SourceCount: 1, Consolidated: false, MergedContent: group[0].Content}
			res.Groups = append(res.Groups, g)
			continue
		}
		contents := make([]string, 0, len(group))
		srcIDs := make([]int64, 0, len(group))
		for _, o := range group {
			contents = append(contents, o.Content)
			srcIDs = append(srcIDs, o.ID)
		}
		body, err := de.consolidateBody(ctx, topic, contents)
		if err != nil {
			return res, err
		}
		title := dreamDerivedTitle(topic)
		mergedID, err := de.svc.Save(ctx, SaveInput{
			Title:     title,
			Type:      "learning",
			Content:   body,
			Scope:     "project",
			SessionID: group[0].SessionID,
			TopicKey:  topic,
			Source:    "dream",
			// No ExpiresAt: a consolidated record is a durable summary, not a
			// transient observation, so it does not carry the auto-TTL stamp.
		})
		if err != nil {
			return res, fmt.Errorf("save consolidated %q: %w", topic, err)
		}
		if err := de.markConsolidated(ctx, srcIDs); err != nil {
			return res, err
		}
		g := ConsolidatedGroup{Topic: topic, NewID: mergedID, SourceIDs: srcIDs, SourceCount: len(group), Consolidated: true, MergedContent: body}
		res.Groups = append(res.Groups, g)
	}
	return res, nil
}

// consolidateBody folds a topic's contents into one coherent body. With an LLM
// it asks the seam for a coherent merge (then appends the verbatim source list
// so no fact is silently dropped by the summary); without an LLM (or on an LLM
// error) it is the deterministic lossless join.
func (de *DreamExecutor) consolidateBody(ctx context.Context, topic string, contents []string) (string, error) {
	var b strings.Builder
	if de.llm != nil {
		if merged, err := de.llm.ConsolidatePrompt(ctx, topic, contents); err == nil {
			if m := strings.TrimSpace(merged); m != "" {
				b.WriteString(m)
				b.WriteString("\n")
			}
		}
	}
	b.WriteString(dreamJoin("Sources", contents))
	return b.String(), nil
}

// markConsolidated stamps every source observation with the consolidated
// status flag. It never touches deleted_at (the sources stay readable; they are
// just no longer "active" as standalone facts).
func (de *DreamExecutor) markConsolidated(ctx context.Context, ids []int64) error {
	for _, id := range ids {
		if _, err := de.svc.DB().ExecContext(ctx, `
			UPDATE observations SET status = ?
			WHERE id = ? AND project = ?`,
			dreamStatus, id, de.svc.ProjectID()); err != nil {
			return fmt.Errorf("mark consolidated %d: %w", id, err)
		}
	}
	return nil
}

// synthesize lifts a set of lower-tier (L0/L1) session summaries into a single
// higher-tier (L2) summary (014 step 12.2). It stores the summary as a NEW
// observation with tier = DreamTierL2 and records the source sessions in its
// content (the provenance link) and in the returned Summary.SessionIDs.
func (de *DreamExecutor) synthesize(ctx context.Context, memories []Memory) (Summary, error) {
	var out Summary
	if de == nil || de.svc == nil || de.svc.DB() == nil {
		return out, fmt.Errorf("dream executor not initialized")
	}
	if len(memories) == 0 {
		return out, fmt.Errorf("synthesize: at least one memory is required")
	}
	contents := make([]string, 0, len(memories))
	sessions := make([]string, 0, len(memories))
	for _, m := range memories {
		contents = append(contents, m.Content)
		sessions = append(sessions, m.SessionID)
	}
	body, err := de.synthesizeBody(ctx, contents)
	if err != nil {
		return out, err
	}
	body = body + "\n" + dreamSourceSessions(sessions)
	title := "L2 synthesis: " + dreamShortID(sessions[0])
	id, err := de.svc.Save(ctx, SaveInput{
		Title:     title,
		Type:      "learning",
		Content:   body,
		Scope:     "project",
		SessionID: sessions[0],
		TopicKey:  "dream/L2/" + dreamShortID(sessions[0]),
		Source:    dreamSynthesisSource,
	})
	if err != nil {
		return out, fmt.Errorf("save synthesis: %w", err)
	}
	out.ID = id
	out.Tier = DreamTierL2
	out.Content = body
	out.SessionIDs = sessions
	out.SourceCount = len(memories)
	return out, nil
}

// synthesizeBody produces the higher-tier summary body. With an LLM it asks the
// seam; without one (or on error) it is the deterministic lossless join of the
// lower-tier summaries.
func (de *DreamExecutor) synthesizeBody(ctx context.Context, contents []string) (string, error) {
	if de.llm != nil {
		if merged, err := de.llm.SynthesizePrompt(ctx, contents); err == nil {
			if m := strings.TrimSpace(merged); m != "" {
				return m + "\n", nil
			}
		}
	}
	return dreamJoin("Lower-tier summaries", contents), nil
}

// dreamSourceSessions renders the provenance line that ties a synthesized L2
// record back to its source sessions.
func dreamSourceSessions(sessions []string) string {
	return "Source sessions: " + strings.Join(sessions, ", ")
}

// prune soft-deletes observations whose on-the-fly importance score is below
// threshold (014 step 12.3). It loads the live rows of the project, scores each
// with importance(), and soft-deletes the ones below the threshold. It never
// touches a row that is already deleted, and returns the count and the ids it
// pruned. Maturity tier is not a stored column, so the tier condition is
// expressed through the durable-type weight (durable types are harder to
// prune) — see the weights above.
func (de *DreamExecutor) prune(ctx context.Context, threshold float64) (PrunedResult, error) {
	res := PrunedResult{}
	if de == nil || de.svc == nil || de.svc.DB() == nil {
		return res, fmt.Errorf("dream executor not initialized")
	}
	rows, err := de.svc.DB().QueryContext(ctx, `
		SELECT id, type, created_at, COALESCE(retrieval_usage, 0)
		FROM observations
		WHERE project = ? AND deleted_at IS NULL`,
		de.svc.ProjectID())
	if err != nil {
		return res, fmt.Errorf("prune query: %w", err)
	}
	defer rows.Close()
	var toPrune []int64
	for rows.Next() {
		var id int64
		var typ, created string
		var usage int
		if err := rows.Scan(&id, &typ, &created, &usage); err != nil {
			return res, fmt.Errorf("prune scan: %w", err)
		}
		if de.importance(typ, created, usage) < threshold {
			toPrune = append(toPrune, id)
		}
	}
	if err := rows.Err(); err != nil {
		return res, fmt.Errorf("prune iterate: %w", err)
	}
	now := de.now().UTC().Format(time.RFC3339)
	for _, id := range toPrune {
		if _, err := de.svc.DB().ExecContext(ctx, `
			UPDATE observations SET deleted_at = ?
			WHERE id = ? AND project = ? AND deleted_at IS NULL`,
			now, id, de.svc.ProjectID()); err != nil {
			return res, fmt.Errorf("soft-delete %d: %w", id, err)
		}
		res.Pruned++
		res.IDs = append(res.IDs, id)
	}
	return res, nil
}

// importance is the on-the-fly AKL-style importance score used by prune (014
// step 12.3). It is computed from existing columns — no new schema:
//
//	usageNorm = min(retrieval_usage, 50) / 50               (0..1)
//	age       = now - created_at
//	recency   = 1 when age <= 30d (fresh, no penalty); max(0, 1 - age/30d)
//	            once past the 30-day grace window (decays to 0 at 60d)
//	typeWeight = 0.9 durable types, else 0.4
//	importance = clamp(0.6*usageNorm + 0.37*recency + 0.2*typeWeight, 0, 1)
//
// The recency term is a GATED age penalty, not a raw freshness boost: a row
// younger than the 30-day grace window keeps full recency (1.0) regardless of
// its usage, so a fresh row is scored on usage + durability alone — fresh
// content is given a minimum age before age can drag it below the threshold
// (per the brief's "importance + a minimum age"). Only once a row is old does
// recency decay, dragging a never-retrieved, non-durable old note below the
// threshold (a 60d old one scores 0 + 0 + 0.08 = 0.08). Durable types and
// retrieval usage keep their rows above the line at any age.
const pruneGraceWindow = 30 * 24 * time.Hour

func (de *DreamExecutor) importance(typ, createdAt string, usage int) float64 {
	usageNorm := 0.0
	if usage > 0 {
		usageNorm = float64(usage) / 50.0
		if usageNorm > 1.0 {
			usageNorm = 1.0
		}
	}
	recency := 1.0
	if t, err := parseObsTimestamp(createdAt); err == nil && !t.IsZero() {
		age := de.now().Sub(t)
		if age > pruneGraceWindow {
			r := 1.0 - age.Hours()/24.0/30.0
			if r < 0 {
				r = 0
			}
			recency = r
		}
	}
	typeWeight := typeWeightDefault
	if durableTypes[typ] {
		typeWeight = typeWeightDurable
	}
	s := wUsage*usageNorm + wRecency*recency + wType*typeWeight
	if s < 0 {
		s = 0
	}
	if s > 1 {
		s = 1
	}
	return s
}

// dreamShortID is a short, stable identifier (first 8 chars) used in titles.
func dreamShortID(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 8 {
		return s[:8]
	}
	if s == "" {
		return "no-session"
	}
	return s
}

// dreamJoin builds the deterministic lossless body for a group: a header line
// plus one numbered block per source. Every source's content appears verbatim,
// so no fact is dropped on the no-LLM path.
func dreamJoin(header string, contents []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s (%d):\n", header, len(contents))
	for i, c := range contents {
		fmt.Fprintf(&b, "%d. %s\n", i+1, strings.TrimSpace(c))
	}
	return b.String()
}

// dreamDerivedTitle names a consolidated record from its topic.
func dreamDerivedTitle(topic string) string {
	if t := strings.TrimSpace(topic); t != "" {
		return "Consolidated: " + t
	}
	return "Consolidated memory"
}

// topicOf returns the grouping key for an observation. An empty topic_key is
// its own group (a unique per-id key) so a topic-less observation is never
// merged with a different observation.
func topicOf(o Observation) string {
	if t := strings.TrimSpace(o.TopicKey); t != "" {
		return t
	}
	return fmt.Sprintf("__id__%d", o.ID)
}
