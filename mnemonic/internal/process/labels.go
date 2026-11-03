package process

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
)

// labelFlows assigns LLM labels to the freshly-traced flows, caching by the
// flow's content-hash. It first reuses any stored label whose content hash
// matches (so an unchanged re-index does NOT re-call the LLM — the label is
// restored from processes.label). Only flows with a new content hash are
// labeled, and only up to limit calls per pass (the rest stay cached
// unlabeled). A nil LLM or one that errors leaves the flow cached UNLABELED
// (empty label), never fabricated.
func labelFlows(ctx context.Context, db *sql.DB, procs []Process, llm LLM, limit int) {
	if llm == nil {
		return
	}
	stored, err := storedLabels(db)
	if err != nil {
		return // best-effort: on a read error, just label fresh (bounded)
	}
	calls := 0
	for i := range procs {
		p := &procs[i]
		if calls >= limit {
			break // over the per-pass budget: keep unlabeled (never fabricate)
		}
		if label, ok := stored[p.ContentHash]; ok && label != "" {
			// Reused from cache — the LLM is NOT called again.
			p.Label = label
			p.LabelStatus = "cached"
			continue
		}
		calls++
		summary, err := flowSummary(db, p)
		if err != nil {
			p.LabelStatus = "unlabeled"
			continue
		}
		label, err := llm.Label(ctx, summary)
		if err != nil || strings.TrimSpace(label) == "" {
			// LLM down (or returned blank): cache the flow UNLABELED — the
			// structure is still stored, the label is empty, never invented.
			p.Label = ""
			p.LabelStatus = "unlabeled"
			continue
		}
		p.Label = label
		p.LabelStatus = "labeled"
	}
	for i := range procs {
		if procs[i].LabelStatus == "unlabeled" && procs[i].Label != "" {
			// keep consistency: a labeled flow is never marked unlabeled
			procs[i].LabelStatus = "labeled"
		}
		if err := saveLabel(db, &procs[i]); err != nil {
			return
		}
	}
}

// storedLabels returns the current label cache keyed by content hash.
func storedLabels(db *sql.DB) (map[string]string, error) {
	rows, err := db.Query(`SELECT content_hash, label FROM processes`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var hash, label string
		if err := rows.Scan(&hash, &label); err != nil {
			return nil, err
		}
		out[hash] = label
	}
	return out, rows.Err()
}

// saveLabel persists a flow's label + label status (the structure was already
// written by persistProcess; here we only set the label columns).
func saveLabel(db *sql.DB, p *Process) error {
	if p.LabelStatus == "" {
		p.LabelStatus = "unlabeled"
	}
	_, err := db.Exec(`UPDATE processes SET label = ?, label_status = ? WHERE content_hash = ?`,
		p.Label, p.LabelStatus, p.ContentHash)
	return err
}

// flowSummary renders the flow's structure (entry + ordered step names) as the
// prompt input to the LLM. It is deterministic for a given flow, so the label
// is a pure function of the flow content (modulo the LLM).
func flowSummary(db *sql.DB, p *Process) (string, error) {
	var b strings.Builder
	b.WriteString("Entry: ")
	b.WriteString(p.Name)
	b.WriteString(" (" + p.EntryKind + ")\nSteps:\n")
	for i, s := range p.Steps {
		b.WriteString("  ")
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(". ")
		if s.Name != "" {
			b.WriteString(s.Name)
		} else {
			b.WriteString(strconv.FormatInt(s.SymbolID, 10))
		}
		if s.Confidence != "" {
			b.WriteString(" [" + s.Confidence + "]")
		}
		b.WriteString("\n")
	}
	if p.Stop != nil {
		b.WriteString("Stops at: ")
		b.WriteString(p.Stop.Symbol)
		b.WriteString(" (")
		b.WriteString(p.Stop.Reason)
		b.WriteString(")\n")
	}
	return b.String(), nil
}
