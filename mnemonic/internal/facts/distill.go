// Background distill + write approval (change
// 2026-10-24-global-llm-wiki, Change A, TICKET-06).
//
// One LLM pass at session end (ADR-0023) extracts durable facts from the
// conversation, classifying each as project or global scope. The extracted
// facts are routed by scope: project facts go to the caller's project store,
// global facts to the __global__ store (TICKET-02 scope routing). The
// LLM seam mirrors skills.Store.LLM — a func field the MCP layer wires from
// service.AskLLMSeam().Complete; tests inject a stub.
//
// Coalescing: if a second trigger fires for the same session while the first
// is mid-flight, it does NOT start a second LLM call. It records the latest
// conversation snapshot and returns a zero result; the in-flight call, once
// its LLM response arrives, writes the LATEST snapshot's facts (the
// distiller re-reads the latest snapshot at write time). Latest snapshot
// wins.
//
// Fail-open: if the LLM client is unreachable (or the response is malformed),
// Distill returns a NIL error — the session closes normally. No facts are
// written, and the failure is recorded on DistillResult.Err (the caller logs
// it; it is not returned).
//
// Write approval (the Hermes write_approval pattern): when the Distiller's
// WriteApproval flag is set, PASSIVE (distill) writes are staged in the
// pending_facts table (migration 073) instead of going directly to facts.
// A staged row survives server restart (it is a table, not a memory map).
// ApprovePending moves a staged row into facts and deletes it from
// pending_facts (two-phase: approve → apply). RejectPending discards it.
// Explicit writes (agent-initiated fact_add → Add/AddWith/Batch) bypass the
// gate: they are not staged.
package facts

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// DistillLLM is the distill's LLM seam (ADR-0023). It mirrors
// skills.Store.LLM: a func field the MCP layer wires from
// service.AskLLMSeam().Complete; tests inject a stub. Nil (the default)
// means no LLM is attached: Distill fails open with a recorded no-LLM
// condition.
type DistillLLM func(ctx context.Context, system, user string) (string, error)

// distillSystemPrompt is the scope-classifying extraction prompt (ONE pass:
// the LLM extracts every durable fact and labels each with its scope —
// project (this project's operational notes) or global (cross-project
// knowledge) — and its target (memory | user). It answers with a single
// JSON object: {"facts": [{"scope": "project|global", "target":
// "memory|user", "content": "..."}]}.
const distillSystemPrompt = `You extract durable, reusable facts from a coding session's conversation.
For each fact, classify its SCOPE:
- "project": operational notes specific to THIS project (conventions, file locations, build/test commands, decisions)
- "global": cross-project knowledge (user preferences, tool behavior, environment facts that hold everywhere)
and its TARGET:
- "memory": agent operational notes (the default)
- "user": user profile (preferences, habits, how the user works)

Extract only DURABLE facts (true across sessions). Skip transient state,
one-off commands, and anything already obvious from the code. Do not
invent facts the conversation does not support.

Answer with a single JSON object only — no prose, no markdown:
{"facts": [{"scope": "project" | "global", "target": "memory" | "user", "content": "one durable fact"}]}`

// Distiller is the background distill handle: one LLM pass at session end
// over the conversation snapshot, scope-classifying, with the write
// approval gate for passive writes. A Distiller is bound to the two store
// roles (global = the __global__ data store for global-scope facts,
// project = the caller's project store for project-scope facts and the
// session trail) and is reusable across sessions: the per-session
// coalescing state is keyed by session id, not by the Distiller.
type Distiller struct {
	global  *Store // global-scope facts (the __global__ store)
	project *Store // project-scope facts (the caller's project store)

	// LLM is the extraction seam (ADR-0023). Nil = no LLM (fail-open).
	LLM DistillLLM

	// WriteApproval gates passive (distill) writes: when true, distill
	// writes are staged in pending_facts (approval required) instead of
	// going directly to facts. Explicit writes bypass the gate (they are
	// Add/AddWith/Batch, not Distill).
	WriteApproval bool

	mu       sync.Mutex
	inFlight map[string]*distillSession
}

// distillSession is the per-session coalescing state: one in-flight LLM call
// per session (the mutex guard), and the latest conversation snapshot (a
// coalesced trigger records it; the next pass runs on it — see the ponytail
// note in Distill for the mid-flight boundary).
type distillSession struct {
	inFlight bool
	latest   string // the latest conversation snapshot (coalescing)
}

// NewDistiller builds a Distiller over the two store roles. global is the
// __global__ store (global-scope facts); project is the caller's project
// store (project-scope facts + session trail).
func NewDistiller(global, project *Store) *Distiller {
	return &Distiller{global: global, project: project, inFlight: make(map[string]*distillSession)}
}

// DistillResult reports what the pass produced.
type DistillResult struct {
	// ProjectWritten is the number of project-scope facts written (or
	// staged, when WriteApproval is on).
	ProjectWritten int
	// GlobalWritten is the number of global-scope facts written (or
	// staged, when WriteApproval is on).
	GlobalWritten int
	// Staged is the number of facts staged in pending_facts (WriteApproval
	// on); 0 when the gate is off or no facts were extracted.
	Staged int
	// Err records a best-effort distill failure (LLM down, malformed
	// response, no LLM attached, a superseded trigger). It is LOGGED, not
	// returned — Distill returns a nil error on these so the session
	// closes normally (fail-open). Nil on success.
	Err error `json:"-"`
}

// Distill runs the one-LLM-pass extraction for sessionID over the given
// conversation snapshot and routes the extracted facts by scope (project
// facts → project store, global facts → global store). Coalescing: if the
// session already has an in-flight pass, this call records the latest
// snapshot and returns a zero result (the in-flight call writes the latest
// snapshot's facts on completion). Fail-open: any LLM/parse/write error
// returns a NIL error with the condition on DistillResult.Err — the
// session closes normally and no facts are written.
func (d *Distiller) Distill(ctx context.Context, sessionID, trailSessionID, conversation string) (DistillResult, error) {
	if d == nil || d.global == nil || d.project == nil {
		return DistillResult{}, errors.New("distiller not initialized (both the global and the project store handles are required)")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return DistillResult{}, errors.New("session_id is required")
	}
	conversation = strings.TrimSpace(conversation)
	if conversation == "" {
		return DistillResult{}, errors.New("conversation is required (the distill pass runs on the session's conversation snapshot)")
	}

	// Coalescing: one LLM call per session. If a pass is already in flight
	// for this session, record the LATEST snapshot (the in-flight call
	// re-reads it at LLM time, so its single call carries the newest
	// conversation) and return a zero result — this trigger adds no second
	// LLM call and writes nothing of its own.
	d.mu.Lock()
	sess, ok := d.inFlight[sessionID]
	if ok && sess.inFlight {
		sess.latest = conversation
		d.mu.Unlock()
		log.Printf("mnemonic: distill coalesced for session %q (an in-flight pass already holds the LLM call; the latest snapshot wins)", sessionID)
		return DistillResult{Err: fmt.Errorf("distill coalesced into the in-flight pass for session %q", sessionID)}, nil
	}
	if !ok {
		sess = &distillSession{}
		d.inFlight[sessionID] = sess
	}
	sess.inFlight = true
	sess.latest = conversation
	d.mu.Unlock()

	// ponytail: the LLM call carries the snapshot this trigger captured at
	// entry. A trigger that coalesces WHILE the call is in flight records a
	// newer snapshot, which this response no longer reflects — "latest
	// snapshot wins" then degrades to "first trigger's snapshot wins"
	// (still one LLM call, still the newest snapshot the call itself saw).
	// A trigger that coalesces BEFORE the call is issued is reflected: the
	// next pass runs on the latest snapshot. Upgrading to true latest-wins
	// for a mid-flight coalesce needs a per-session LLM call queue (the
	// in-flight call defers to the newest pending snapshot and re-issues) —
	// deferred because the session-close trigger fires once per session, so
	// a mid-flight coalesce is a rare race, not the common path.
	res := DistillResult{}
	defer d.finishSession(sessionID)

	// Fail-open: no LLM attached → the session closes, nothing is written,
	// the condition is recorded (and logged) on the result.
	if d.LLM == nil {
		res.Err = errors.New("no LLM attached: background distill needs the shared LLM (mnemonic.llm.enabled)")
		log.Printf("mnemonic: distill skipped for session %q: %v", sessionID, res.Err)
		return res, nil
	}

	// One LLM call per session (the coalescing contract): the call carries
	// the snapshot this trigger captured at entry (see the ponytail note
	// above for the mid-flight coalesce boundary).
	out, err := d.LLM(ctx, distillSystemPrompt, conversation)
	if err != nil {
		// LLM down (unreachable / timeout / HTTP error): fail-open. The
		// session closes normally, nothing is written, the error is logged
		// and recorded — not returned.
		res.Err = fmt.Errorf("distill LLM: %w", err)
		log.Printf("mnemonic: distill LLM failed for session %q: %v (fail-open: no facts written)", sessionID, res.Err)
		return res, nil
	}

	extracted, err := parseDistillResponse(out)
	if err != nil {
		// Malformed LLM response: fail-open, same contract.
		res.Err = fmt.Errorf("distill response: %w", err)
		log.Printf("mnemonic: distill response malformed for session %q: %v (fail-open: no facts written)", sessionID, res.Err)
		return res, nil
	}

	// The response is this pass's snapshot's facts — write them.
	for _, f := range extracted {
		switch f.Scope {
		case string(ScopeProject), "":
			if d.WriteApproval {
				if err := d.stagePending(ctx, f.Content, f.Target, string(ScopeProject)); err != nil {
					res.Err = fmt.Errorf("stage project fact: %w", err)
					log.Printf("mnemonic: distill stage failed for session %q: %v (fail-open)", sessionID, err)
					return res, nil
				}
				res.ProjectWritten++
				res.Staged++
			} else {
				if _, err := d.project.AddWith(ctx, trailSessionID, f.Content, f.Target, d.project); err != nil {
					res.Err = fmt.Errorf("write project fact: %w", err)
					log.Printf("mnemonic: distill write failed for session %q: %v (fail-open: later facts skipped)", sessionID, err)
					return res, nil
				}
				res.ProjectWritten++
			}
		case string(ScopeGlobal):
			if d.WriteApproval {
				if err := d.stagePending(ctx, f.Content, f.Target, string(ScopeGlobal)); err != nil {
					res.Err = fmt.Errorf("stage global fact: %w", err)
					log.Printf("mnemonic: distill stage failed for session %q: %v (fail-open)", sessionID, err)
					return res, nil
				}
				res.GlobalWritten++
				res.Staged++
			} else {
				// C3: the fact lands in the global store; the trail lands in
				// the caller's project store (the session lives there).
				if _, err := d.global.AddWith(ctx, trailSessionID, f.Content, f.Target, d.project); err != nil {
					res.Err = fmt.Errorf("write global fact: %w", err)
					log.Printf("mnemonic: distill write failed for session %q: %v (fail-open: later facts skipped)", sessionID, err)
					return res, nil
				}
				res.GlobalWritten++
			}
		default:
			// An out-of-contract scope is skipped (not an error): the LLM
			// occasionally labels a fact with a non-canonical scope, and the
			// pass must not fail the session close over one bad label.
			res.Err = fmt.Errorf("distill fact skipped: unknown scope %q (want project or global)", f.Scope)
			log.Printf("mnemonic: %v (content %q)", res.Err, truncate(f.Content, 64))
		}
	}
	return res, nil
}

// finishSession clears the per-session coalescing state.
func (d *Distiller) finishSession(sessionID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if sess, ok := d.inFlight[sessionID]; ok {
		sess.inFlight = false
	}
}

// stagePending stages one passive write in pending_facts (the write
// approval gate, migration 073): the row lands in the store matching its
// scope (project facts → project store, global facts → global store) so
// ApprovePending can apply it to the same store. status='pending' until
// approved or rejected.
func (d *Distiller) stagePending(ctx context.Context, content, target, scope string) error {
	store := d.project
	if scope == string(ScopeGlobal) {
		store = d.global
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := store.db.ExecContext(ctx, `
		INSERT INTO pending_facts (content, target, scope, created_at, status)
		VALUES (?, ?, ?, ?, 'pending')`, content, target, scope, now)
	if err != nil {
		return fmt.Errorf("stage pending fact: %w", err)
	}
	return nil
}

// distillFact is one extracted fact from the LLM's response (the
// scope-classifying prompt's contract).
type distillFact struct {
	Scope   string `json:"scope"`
	Target  string `json:"target"`
	Content string `json:"content"`
}

// parseDistillResponse parses the LLM's JSON response into the extracted
// facts. Blank content is dropped (no empty fact); an unknown target is
// normalized to the default (the LLM is not a validator; the write path
// re-validates). A blank or unparseable response is an error (fail-open at
// the caller).
func parseDistillResponse(out string) ([]distillFact, error) {
	out = strings.TrimSpace(out)
	// Strip a markdown fence if the LLM wrapped its JSON in one (a cheap
	// robustness floor for weaker models).
	if strings.HasPrefix(out, "```") {
		if i := strings.Index(out[3:], "\n"); i >= 0 {
			out = out[3+i+1:]
		}
		if i := strings.LastIndex(out, "```"); i > 0 {
			out = out[:i]
		}
		out = strings.TrimSpace(out)
	}
	var envelope struct {
		Facts []distillFact `json:"facts"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		return nil, fmt.Errorf("not a JSON object with a facts array: %w", err)
	}
	var out2 []distillFact
	for _, f := range envelope.Facts {
		f.Content = strings.TrimSpace(f.Content)
		if f.Content == "" {
			continue
		}
		f.Scope = strings.TrimSpace(f.Scope)
		f.Target = strings.TrimSpace(f.Target)
		if _, err := CategoryForTarget(f.Target); err != nil {
			f.Target = CategoryMemory // unknown target → the default, not a skip
		}
		out2 = append(out2, f)
	}
	return out2, nil
}

// PendingFact is one staged (passive, write-approval-gated) fact row from
// pending_facts.
type PendingFact struct {
	ID        int64  `json:"id"`
	Content   string `json:"content"`
	Target    string `json:"target"`
	Scope     string `json:"scope"`
	CreatedAt string `json:"created_at"`
	Status    string `json:"status"`
}

// PendingList returns every staged pending fact (all stores the Distiller is
// bound to — global + project — oldest first). The read for the MCP
// fact_pending_list tool.
func (d *Distiller) PendingList(ctx context.Context) ([]PendingFact, error) {
	if d == nil || d.global == nil || d.project == nil {
		return nil, errors.New("distiller not initialized")
	}
	var out []PendingFact
	seen := map[*sql.DB]bool{}
	for _, store := range []*Store{d.global, d.project} {
		if store == nil || store.db == nil || seen[store.db] {
			continue
		}
		seen[store.db] = true
		rows, err := store.db.QueryContext(ctx, `
			SELECT id, content, target, scope, created_at, status
			FROM pending_facts WHERE status = 'pending'
			ORDER BY created_at, id`)
		if err != nil {
			return nil, fmt.Errorf("list pending facts: %w", err)
		}
		for rows.Next() {
			var p PendingFact
			if err := rows.Scan(&p.ID, &p.Content, &p.Target, &p.Scope, &p.CreatedAt, &p.Status); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan pending fact: %w", err)
			}
			out = append(out, p)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, fmt.Errorf("iterate pending facts: %w", err)
		}
	}
	return out, nil
}

// ApprovePending moves one staged fact (by its pending_facts id) into the
// facts table of the store it was staged in, and deletes the pending row —
// two-phase approval (approve → apply). It returns the new fact id. A
// pending row that is already gone (approved or rejected by someone else) is
// a clean error, not a silent no-op.
func (d *Distiller) ApprovePending(ctx context.Context, pendingID int64) (int64, error) {
	if d == nil || d.global == nil || d.project == nil {
		return 0, errors.New("distiller not initialized")
	}
	for _, store := range []*Store{d.global, d.project} {
		var content, target, scope string
		err := store.db.QueryRowContext(ctx, `
			SELECT content, target, scope FROM pending_facts WHERE id = ? AND status = 'pending'`,
			pendingID).Scan(&content, &target, &scope)
		switch {
		case err == nil:
			return d.applyPending(ctx, store, pendingID, content, target, scope)
		case errors.Is(err, sql.ErrNoRows):
			continue // not staged in this store
		default:
			return 0, fmt.Errorf("load pending fact: %w", err)
		}
	}
	return 0, fmt.Errorf("pending fact %d not found (already approved or rejected?)", pendingID)
}

// applyPending applies one staged fact to its store (insert into facts +
// delete the pending row in one transaction — a crash cannot leave a
// staged row that was also applied, or an applied row whose stage survives).
func (d *Distiller) applyPending(ctx context.Context, store *Store, pendingID int64, content, target, scope string) (int64, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin approve: %w", err)
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := tx.ExecContext(ctx, `
		INSERT INTO facts (content, category, created_at, updated_at)
		VALUES (?, ?, ?, ?)`, content, target, now, now)
	if err != nil {
		return 0, fmt.Errorf("insert approved fact: %w", err)
	}
	factID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("approved fact id: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM pending_facts WHERE id = ?`, pendingID); err != nil {
		return 0, fmt.Errorf("delete staged fact: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit approve: %w", err)
	}
	return factID, nil
}

// RejectPending discards one staged fact (by its pending_facts id): the row
// is deleted from pending_facts and never reaches facts. A pending row that
// is already gone is a clean error, not a silent no-op.
func (d *Distiller) RejectPending(ctx context.Context, pendingID int64) error {
	if d == nil || d.global == nil || d.project == nil {
		return errors.New("distiller not initialized")
	}
	for _, store := range []*Store{d.global, d.project} {
		var n int
		if err := store.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM pending_facts WHERE id = ? AND status = 'pending'`,
			pendingID).Scan(&n); err != nil {
			return fmt.Errorf("load pending fact: %w", err)
		}
		if n == 0 {
			continue // not in this store
		}
		if _, err := store.db.ExecContext(ctx, `DELETE FROM pending_facts WHERE id = ?`, pendingID); err != nil {
			return fmt.Errorf("reject pending fact: %w", err)
		}
		return nil
	}
	return fmt.Errorf("pending fact %d not found (already approved or rejected?)", pendingID)
}

// truncate caps a log string at n runes (a fact content preview).
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
