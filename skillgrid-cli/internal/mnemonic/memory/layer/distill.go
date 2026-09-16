// Package layer implements the session-close distillation of change 013,
// step 02: a session's raw record (L0) is refined upward into L1 atoms, an L2
// scenario block, and an L3 persona delta. The ladder is PROVENANCE-LINKED —
// every derived record links to a resolvable L0 source (a live session row and
// a non-empty source topic); a record whose source cannot be resolved is never
// created (no orphan layers), and a session with no new L1-able content is a
// no-op (no empty atoms/scenarios/persona-delta fabricated).
//
// The pass is LLM-optional with a deterministic no-LLM floor: the floor reuses
// 005's CapturePassive heuristics (Key Learnings / Lesson / Discovery) so the
// ladder works offline; an optional LLM pass refines L2/L3 and is cached by
// content-hash so an unchanged L0 source is never re-distilled.
package layer

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// LLM is the optional distiller for the L2 scenario and L3 persona delta. A
// nil LLM (or one that errors) leaves those layers to the deterministic floor
// — they are still provenance-linked, never fabricated. This mirrors the
// process.LLM seam (no CGo LLM client in this package).
type LLM interface {
	// Summarise produces an L2 scenario block + L3 persona delta from a session
	// summary. A blank return for either field means "use the floor".
	Summarise(ctx context.Context, summary string) (scenario string, personaDelta string, err error)
}

// DistillOptions tunes the pass.
type DistillOptions struct {
	LLM LLM
}

// DistillResult reports what the pass produced.
type DistillResult struct {
	// Created is true when at least one derived layer record was written.
	Created bool
	// L1Count is the number of L1 atoms linked this pass (new + idempotent).
	L1Count int
	// ScenarioCount is the number of L2 scenario records linked this pass.
	ScenarioCount int
	// PersonaDeltaCount is the number of L3 persona-delta records linked.
	PersonaDeltaCount int
	// SourceHash is the content hash of the L0 source this pass ran on.
	SourceHash string
	// LLMUsed is true when the optional LLM pass ran (false on the no-LLM floor).
	LLMUsed bool
	// Err captures a best-effort distill error (set by the session-close hook
	// so a failure is surfaced but never breaks the caller). Empty on success.
	Err error `json:"-"`
}

// Distill refines the L0 record of sessionID into L1 atoms + an L2 scenario +
// an L3 persona delta, each linked to its L0 source. It is the deterministic,
// best-effort core: a missing session (unresolvable source) is a no-op that
// returns a zero result, not an error; a session with no L1-able content is a
// no-op. The session-close hook runs this asynchronously and swallows errors.
func Distill(ctx context.Context, svc *memory.Service, sessionID string, opts DistillOptions) (DistillResult, error) {
	res := DistillResult{}
	if svc == nil {
		return res, fmt.Errorf("memory service is nil")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return res, fmt.Errorf("session_id is required")
	}
	// Load the L0 session record. A missing row means the source is
	// unresolvable → nothing is created (a layer is never orphaned).
	var summary sql.NullString
	var startedAt string
	err := svc.DB().QueryRowContext(ctx, `
		SELECT summary, started_at FROM sessions WHERE id = ? AND project = ?`,
		sessionID, svc.ProjectID()).Scan(&summary, &startedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return res, nil // no-op: unresolvable source
		}
		return res, fmt.Errorf("load session: %w", err)
	}
	sourceText := summary.String
	res.SourceHash = fnvHex("session:" + sessionID + "|" + sourceText)

	// L1 atoms — the deterministic no-LLM floor (005 CapturePassive heuristics).
	items := memory.ExtractLearnings(sourceText)
	newAtoms := 0
	for _, item := range items {
		title, typ := memory.ShapePassiveItem(item)
		title = truncateTitle(title, 120)
		if title == "" || !memory.IsValidTypeExported(typ) {
			continue
		}
		content := memory.ShapePassiveContent(item, title, typ)
		id, err := svc.Save(ctx, memory.SaveInput{
			Title:     title,
			Type:      typ,
			Content:   content,
			Scope:     "project",
			SessionID: sessionID,
			TopicKey:  "layer/L1/" + slugify(title),
			Source:    "distill",
		})
		if err != nil {
			return res, fmt.Errorf("save L1 atom: %w", err)
		}
		if err := linkAtom(ctx, svc, sessionID, sourceText, id); err != nil {
			return res, err
		}
		res.L1Count++
		newAtoms++
	}
	if newAtoms == 0 {
		// No new L1-able content → no-op: no scenario or persona-delta fabricated.
		return res, nil
	}
	res.Created = true

	// L2 scenario + L3 persona delta. The LLM refines these when available and
	// is cached by content-hash; the floor (no LLM) derives them from the
	// source text so the ladder is complete offline.
	scenario, delta := floorLayers(sourceText, items)
	if opts.LLM != nil {
		// Reuse the cached LLM output when the L0 source is unchanged; otherwise
		// call the LLM and cache by content-hash.
		if cachedS, cachedD, ok := loadLLMCache(ctx, svc, res.SourceHash); ok {
			scenario, delta = cachedS, cachedD
		} else if llmS, llmD, lerr := opts.LLM.Summarise(ctx, sourceText); lerr == nil {
			if s := strings.TrimSpace(llmS); s != "" {
				scenario = s
			}
			if d := strings.TrimSpace(llmD); d != "" {
				delta = d
			}
			res.LLMUsed = true
			saveLLMCache(ctx, svc, res.SourceHash, scenario, delta)
		}
	}

	pS, err := upsertPersona(ctx, svc, "scenario", "Scenario: "+sessionID[:min(8, len(sessionID))], scenario)
	if err != nil {
		return res, fmt.Errorf("link scenario: %w", err)
	}
	if err := linkPersona(ctx, svc, "L2", sessionID, sourceText, pS); err != nil {
		return res, err
	}
	res.ScenarioCount++

	pD, err := upsertPersona(ctx, svc, "persona_delta", "Persona delta: "+sessionID[:min(8, len(sessionID))], delta)
	if err != nil {
		return res, fmt.Errorf("link persona delta: %w", err)
	}
	if err := linkPersona(ctx, svc, "L3", sessionID, sourceText, pD); err != nil {
		return res, err
	}
	res.PersonaDeltaCount++
	return res, nil
}

// floorLayers derives the deterministic L2 scenario + L3 persona delta from the
// source text + extracted atoms. Nothing is invented beyond what is present.
func floorLayers(sourceText string, items []memory.PassiveItem) (scenario, delta string) {
	var b strings.Builder
	b.WriteString("**What**: session working context distilled from the L0 record.\n")
	if g := goalLine(sourceText); g != "" {
		fmt.Fprintf(&b, "**Goal**: %s\n", g)
	}
	if len(items) > 0 {
		fmt.Fprintf(&b, "**Key facts**: %d atom(s) extracted.\n", len(items))
	}
	scenario = b.String()

	var d strings.Builder
	d.WriteString("**What**: persona profile increment.\n")
	if len(items) > 0 {
		d.WriteString("**Signals**: ")
		for i, it := range items {
			if i > 0 {
				d.WriteString("; ")
			}
			t, _ := memory.ShapePassiveItem(it)
			d.WriteString(truncateTitle(t, 60))
		}
		d.WriteString("\n")
	}
	delta = d.String()
	return scenario, delta
}

// linkAtom links an L1 atom observation to its resolvable L0 source. It is
// idempotent: an identical (project, L1, target, session, content-hash) link is
// a no-op, so re-distilling an unchanged source does not duplicate rows.
func linkAtom(ctx context.Context, svc *memory.Service, sessionID, sourceText string, atomID int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	ch := fnvHex("L1|atom|" + itoa(atomID) + "|" + sourceText)
	_, err := svc.DB().ExecContext(ctx, `
		INSERT OR IGNORE INTO observation_layers
			(project, layer, target_kind, target_id, source_session, source_topic, content_hash, created_at)
		VALUES (?, 'L1', 'observation', ?, ?, ?, ?, ?)`,
		svc.ProjectID(), atomID, sessionID, truncateTopic(sourceText), ch, now)
	if err != nil {
		return fmt.Errorf("link L1: %w", err)
	}
	return nil
}

// linkPersona links an L2/L3 persona record to its resolvable L0 source.
func linkPersona(ctx context.Context, svc *memory.Service, layer, sessionID, sourceText string, personaID int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	ch := fnvHex(layer + "|persona|" + itoa(personaID) + "|" + sourceText)
	_, err := svc.DB().ExecContext(ctx, `
		INSERT OR IGNORE INTO observation_layers
			(project, layer, target_kind, target_id, source_session, source_topic, content_hash, created_at)
		VALUES (?, ?, 'persona', ?, ?, ?, ?, ?)`,
		svc.ProjectID(), layer, personaID, sessionID, truncateTopic(sourceText), ch, now)
	if err != nil {
		return fmt.Errorf("link %s: %w", layer, err)
	}
	return nil
}

// upsertPersona returns the id of a personas row for (kind, title, content),
// inserting when absent. Re-distilling an unchanged content reuses the same row
// (the ladder is idempotent).
func upsertPersona(ctx context.Context, svc *memory.Service, kind, title, content string) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	ch := fnvHex(kind + "|" + content)
	var id int64
	err := svc.DB().QueryRowContext(ctx, `
		SELECT id FROM personas WHERE project = ? AND kind = ? AND content_hash = ?`,
		svc.ProjectID(), kind, ch).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("lookup persona: %w", err)
	}
	res, err := svc.DB().ExecContext(ctx, `
		INSERT INTO personas (project, kind, title, content, content_hash, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		svc.ProjectID(), kind, title, content, ch, now)
	if err != nil {
		return 0, fmt.Errorf("insert persona: %w", err)
	}
	return res.LastInsertId()
}

// truncateTopic returns a short, stable representation of the L0 source used as
// the provenance topic. It must be non-empty for a real session summary so the
// link is a resolvable L0, not an orphan.
func truncateTopic(sourceText string) string {
	t := collapse(sourceText)
	if len(t) > 200 {
		t = t[:200]
	}
	return t
}

func collapse(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func truncateTitle(s string, n int) string {
	s = collapse(s)
	if len(s) <= n {
		return s
	}
	return strings.TrimRight(s[:n], " ,;") + "…"
}

func slugify(s string) string {
	seg := strings.ToLower(strings.TrimSpace(s))
	seg = strings.ReplaceAll(seg, " ", "-")
	if len(seg) > 60 {
		seg = seg[:60]
	}
	if seg == "" {
		seg = "untitled"
	}
	return seg
}

func itoa(n int64) string { return fmt.Sprintf("%d", n) }

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// goalLine extracts the first non-header line after a "## Goal" heading from the
// session summary — the session's stated purpose. Empty when absent.
func goalLine(summary string) string {
	lines := strings.Split(summary, "\n")
	for i, l := range lines {
		if strings.EqualFold(strings.TrimSpace(l), "## Goal") {
			for _, next := range lines[i+1:] {
				t := strings.TrimSpace(next)
				if t == "" {
					continue
				}
				if strings.HasPrefix(t, "#") {
					break
				}
				return t
			}
		}
	}
	return ""
}

// fnvHex returns the hex digest of a single FNV-1a hash of s (pure Go, CGo-free).
func fnvHex(s string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum64())
}
