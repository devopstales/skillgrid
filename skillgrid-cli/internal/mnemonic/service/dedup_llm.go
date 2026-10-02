package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// dedupLLMFunc is the pluggable LLM completion for the pre-write
// semantic-dedup backend (TICKET-05). It mirrors the AskLLM seam pattern: a
// small function interface a backend implements, NO CGo LLM client in this
// package. The caller (secondbrain) wires a real completion; a nil function
// leaves the seam disabled so the deterministic hash dedup floor applies.
type dedupLLMFunc func(ctx context.Context, system, user string) (string, error)

var (
	dedupLLMFuncFn dedupLLMFunc
	dedupLLMFuncMu sync.RWMutex
)

// SetDedupLLMFunc attaches the LLM completion the dedup backend calls. Passing
// nil restores the default (no LLM → the seam errors → hash fallback).
// Package-level, like SetAskLLM: a single CLI process has one LLM backend.
func SetDedupLLMFunc(f dedupLLMFunc) {
	dedupLLMFuncMu.Lock()
	dedupLLMFuncFn = f
	dedupLLMFuncMu.Unlock()
}

// dedupPrompt is the instruction sent to the LLM. It asks for a strict 4-way
// JSON verdict (add / update / delete / noop) plus the candidate index the
// verdict applies to, so the parse is deterministic (mirrors extractionPrompt).
const dedupPrompt = `You are classifying a new observation against existing candidates.
Return ONLY a JSON object: {"verdict": "add"|"update"|"delete"|"noop", "candidate": <0-based index or 0>}.
- "add": the new observation is genuinely new.
- "update": it updates an existing candidate (set candidate to that index).
- "delete": it supersedes/deletes an existing candidate (set candidate to that index).
- "noop": it matches an existing candidate and changes nothing.
Set candidate to null for "add"/"noop". Do not invent candidates that are not listed.
New observation:
`

// dedupLLMResponse is the LLM's strict 4-way verdict. Candidate is the 0-based
// index into the candidates slice (observation id = index+1); nil means "no
// candidate" (add / noop). A pointer distinguishes an explicit 0 (first
// candidate) from an absent field.
type dedupLLMResponse struct {
	Verdict   string `json:"verdict"`
	Candidate *int   `json:"candidate"`
}

// dedupLLMBackend is the memory.DedupLLM seam implementation (TICKET-05). It
// satisfies the interface's Classify (the 4-way pre-write path) and the
// deprecated binary Dedup (the async-extraction path) by reducing the 4-way
// verdict. A nil LLM function or a malformed response returns an error so the
// caller falls back to the deterministic hash floor (mirrors the step-05
// LLM-failure fallback).
type dedupLLMBackend struct{}

// Classify 4-way classifies newContent against the candidate contents.
// candidate is the 0-based index into the candidates slice; the 1-based
// observation id (index+1) is returned for VerdictUpdate / VerdictDelete. The
// deterministic hash floor remains the caller's fallback on error.
func (b *dedupLLMBackend) Classify(ctx context.Context, newContent string, candidates []string) (memory.DedupDecision, error) {
	fn := dedupLLMFuncFn
	if fn == nil {
		return memory.DedupDecision{}, fmt.Errorf("no dedup LLM configured")
	}
	user := dedupPrompt + newContent + candidateBlock(candidates)
	resp, err := fn(ctx, dedupSystemPrompt, user)
	if err != nil {
		return memory.DedupDecision{}, fmt.Errorf("dedup LLM: %w", err)
	}
	var out dedupLLMResponse
	if err := json.Unmarshal([]byte(resp), &out); err != nil {
		return memory.DedupDecision{}, fmt.Errorf("parse dedup LLM response: %w", err)
	}
	verdict, ok := mapDedupVerdict(strings.ToLower(strings.TrimSpace(out.Verdict)))
	if !ok {
		return memory.DedupDecision{}, fmt.Errorf("dedup LLM: unknown verdict %q", out.Verdict)
	}
	decision := memory.DedupDecision{Verdict: verdict}
	if (verdict == memory.VerdictUpdate || verdict == memory.VerdictDelete) && out.Candidate != nil && *out.Candidate >= 0 && *out.Candidate < len(candidates) {
		decision.CandidateID = int64(*out.Candidate + 1)
	}
	return decision, nil
}

// Dedup is the deprecated binary async-extraction dedup path. It reduces the
// 4-way verdict: "update" (or "noop") against a candidate reports a duplicate;
// the observation id is the 1-based candidate index.
func (b *dedupLLMBackend) Dedup(ctx context.Context, newContent string, candidates []string) (bool, int64, error) {
	d, err := b.Classify(ctx, newContent, candidates)
	if err != nil {
		return false, 0, err
	}
	switch d.Verdict {
	case memory.VerdictUpdate, memory.VerdictNoop:
		return true, d.CandidateID, nil
	default:
		return false, 0, nil
	}
}

// dedupSystemPrompt is the system instruction for the dedup LLM call.
const dedupSystemPrompt = "You classify new observations against existing candidates. Return only strict JSON."

// candidateBlock renders the candidate contents as a numbered block the LLM
// can reference by 0-based index.
func candidateBlock(candidates []string) string {
	var b strings.Builder
	for i, c := range candidates {
		fmt.Fprintf(&b, "[%d] %s\n", i, c)
	}
	return b.String()
}

// mapDedupVerdict maps the LLM's raw verdict string to the canonical
// memory.DedupVerdict. It returns ok=false on an unrecognized value so the
// caller can fall back to the hash floor.
func mapDedupVerdict(s string) (memory.DedupVerdict, bool) {
	switch s {
	case "add":
		return memory.VerdictAdd, true
	case "update":
		return memory.VerdictUpdate, true
	case "delete":
		return memory.VerdictDelete, true
	case "noop":
		return memory.VerdictNoop, true
	default:
		return "", false
	}
}

// newDedupLLMBackend constructs the dedup LLM seam for the pre-write
// 4-way semantic-dedup check (TICKET-05). It is attached to the memory service
// (SetDedupLLM + EnableDedupLLM) when mnemonic.dedup.llm is true. The backend
// calls the package-level dedupLLMFunc (wired by SetDedupLLMFunc); when that
// function is nil, Classify errors and the deterministic hash floor applies.
func newDedupLLMBackend() memory.DedupLLM {
	return &dedupLLMBackend{}
}
