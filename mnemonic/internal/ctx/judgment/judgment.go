// Package judgment implements the ADR-0034 local-model judgment layer for CTX.
//
// It makes per-unit truncate decisions via the plan §12.13 pipeline:
//
//	prefilter (deterministic) → judge (Ollama HTTP) → compress (truncate-only)
//
// The deterministic pre-filter runs even when the model is down and classifies
// each context unit as pin (keep as-is), dead (structural dead weight →
// truncate), or remainder (needs a model decision). For remainder units the
// judge calls the local Ollama endpoint; a keep-score below the threshold
// truncates, otherwise it keeps. Truncate-only: a truncated unit is shortened
// (capped to a budget), NEVER deleted and NEVER rewritten. Fail-open (ADR-0016):
// any model/HTTP/network fault leaves the unit in its pre-judgment state —
// nothing blocks the request and no unit is dropped.
package judgment

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic/internal/ctx/config"
)

// Action is the judgment outcome for a single context unit.
type Action int

const (
	// ActionKeep leaves the unit as-is.
	ActionKeep Action = iota
	// ActionTruncate shortens the unit (truncate-only; never deletes/rewrites).
	ActionTruncate
)

// String renders the action for logging and the judgment cache.
func (a Action) String() string {
	if a == ActionTruncate {
		return "trunc"
	}
	return "keep"
}

// Unit is one context unit to judge: a single message.
type Unit struct {
	Role    string
	Content string
}

// Decision is the outcome of judging a unit.
type Decision struct {
	// Action is keep or truncate.
	Action Action
	// KeepScore is the model's keep-score in [0,1] (0 for deterministic).
	KeepScore float64
	// FromCache is true when the decision came from the fingerprint cache
	// without a model call.
	FromCache bool
}

// CacheStore persists and retrieves judgment decisions by fingerprint. It is the
// narrow seam over the ctx_judgment_cache table; a nil implementation or any
// error degrades to a cache miss (fail-open, ADR-0016).
type CacheStore interface {
	GetJudgmentCache(fingerprint string) (action string, keepScore float64, ok bool)
	PutJudgmentCache(fingerprint, action string, keepScore float64, judgedAt int64)
}

// Judgment is the judgment engine. hc is injectable so tests can point it at an
// httptest.Server; cache may be nil (cache disabled). All config comes from the
// resolved config.Judgment — nothing is hardcoded.
type Judgment struct {
	endpoint  string
	model     string
	threshold float64
	client    *http.Client
	cache     CacheStore
}

// New builds a Judgment from the resolved config (the config.Judgment block)
// and an injectable HTTP client. A nil hc uses a default client with a short
// timeout. The cache may be nil (cache disabled). Nothing is hardcoded: the
// endpoint, model and threshold all come from the passed config (ADR-0034).
func New(cfg config.Judgment, cache CacheStore, hc *http.Client) *Judgment {
	if hc == nil {
		hc = &http.Client{Timeout: defaultTimeout}
	}
	return &Judgment{
		endpoint:  cfg.Endpoint,
		model:     cfg.Model,
		threshold: cfg.Threshold,
		client:    hc,
		cache:     cache,
	}
}

// Judge makes the per-unit decision for u. It first consults the fingerprint
// cache (no model call on a hit), then the deterministic pre-filter, and only
// for remainder units calls the model. Fail-open: any fault returns a keep (or
// the deterministic pre-filter action), never an error that would block.
func (j *Judgment) Judge(ctx context.Context, u Unit) (Decision, error) {
	fp := Fingerprint(u)

	// 1. Fingerprint cache: a hit serves the decision with NO model call.
	if j.cache != nil {
		if action, score, ok := j.cache.GetJudgmentCache(fp); ok {
			return Decision{
				Action:    actionToAction(action),
				KeepScore: score,
				FromCache: true,
			}, nil
		}
	}

	// 2. Deterministic pre-filter: runs even when the model is down.
	class := PreFilter(u)
	switch class {
	case ClassPin:
		// Keep as-is (facts-first: short, self-contained, high-signal).
		return Decision{Action: ActionKeep, KeepScore: 1.0}, nil
	case ClassDead:
		// Structural dead weight: truncate (the deterministic budget stands;
		// no model needed).
		return Decision{Action: ActionTruncate, KeepScore: 0.0}, nil
	}

	// 3. Remainder: consult the model judge (fail-open).
	score, ok := j.modelJudge(ctx, u)
	if !ok {
		// Model down / fault: fail open to KEEP (never drop, never block).
		return Decision{Action: ActionKeep, KeepScore: 1.0}, nil
	}

	action := ActionKeep
	if score < j.threshold {
		action = ActionTruncate
	}

	// Persist the model decision for future cache hits.
	if j.cache != nil {
		j.cache.PutJudgmentCache(fp, action.String(), score, nowMillis())
	}

	return Decision{Action: action, KeepScore: score}, nil
}

// modelJudge calls the local Ollama endpoint and returns (keepScore, ok). ok is
// false on any fault (conn refused, non-2xx, timeout, bad shape) — the caller
// fails open. The keep-score is expected in [0,1].
func (j *Judgment) modelJudge(ctx context.Context, u Unit) (float64, bool) {
	if j.endpoint == "" {
		return 0, false
	}
	prompt := fmt.Sprintf(
		"Rate how much of this message should be KEPT for context (0.0 = drop most, 1.0 = keep all). Reply with JSON {\"score\": <0.0-1.0>}.\n\nrole: %s\n%s",
		u.Role, u.Content)
	reqBody := map[string]any{
		"model":  j.model,
		"prompt": prompt,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return 0, false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, j.endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, false
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := j.client.Do(req)
	if err != nil {
		return 0, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, false
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, false
	}
	return parseKeepScore(raw)
}

// parseKeepScore extracts a keep-score in [0,1] from an Ollama response body.
// It accepts a direct {"score": x} or an Ollama envelope whose text/field
// contains {"score": x}. It returns (score, ok); ok is false when no valid
// score can be extracted (the caller fails open).
func parseKeepScore(raw []byte) (float64, bool) {
	// Direct JSON object with a numeric score.
	var direct struct {
		Score *float64 `json:"score"`
	}
	if err := json.Unmarshal(raw, &direct); err == nil && direct.Score != nil {
		return clamp01(*direct.Score), true
	}
	// Ollama envelope: {"message":{"content":"..."}} or {"response":"..."}.
	var envelope struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Response string `json:"response"`
		Text     string `json:"text"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil {
		text := envelope.Message.Content
		if text == "" {
			text = envelope.Response
		}
		if text == "" {
			text = envelope.Text
		}
		if text != "" {
			if s, ok := scoreFromText(text); ok {
				return s, true
			}
		}
	}
	return 0, false
}

// scoreFromText extracts a {"score": x} object embedded in free text.
func scoreFromText(text string) (float64, bool) {
	// Find the first '{' ... '}' substring that parses as a score object.
	start := strings.Index(text, "{")
	if start < 0 {
		return 0, false
	}
	end := strings.Index(text[start:], "}")
	if end < 0 {
		return 0, false
	}
	sub := text[start : start+end+1]
	var o struct {
		Score *float64 `json:"score"`
	}
	if err := json.Unmarshal([]byte(sub), &o); err != nil || o.Score == nil {
		return 0, false
	}
	return clamp01(*o.Score), true
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func actionToAction(s string) Action {
	if s == "trunc" || s == "truncate" {
		return ActionTruncate
	}
	return ActionKeep
}

// Fingerprint is a deterministic SHA-256 over the unit (role + content).
func Fingerprint(u Unit) string {
	h := sha256.Sum256([]byte(u.Role + "\x00" + u.Content))
	return hex.EncodeToString(h[:])
}
