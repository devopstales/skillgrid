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
