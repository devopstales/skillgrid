package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic/internal/ctx/judgment"
)

// The five savings headers added to every proxied response (plan 16.5).
const (
	HeaderSavings    = "X-CTX-Savings"
	HeaderOriginal   = "X-CTX-Original-Tokens"
	HeaderCompressed = "X-CTX-Compressed-Tokens"
	HeaderMode       = "X-CTX-Mode"
	HeaderCacheHit   = "X-CTX-Cache-Hit"
)

// SessionHeader is the header that carries a stable session id. If absent, the
// proxy derives one from the client.
const SessionHeader = "X-CTX-Session"

// EstimateTokens is the deterministic token estimator used for the proxy's own
// savings math. It is deliberately simple and STABLE so the AC fixture's
// before/after delta is exact: a token is one whitespace-delimited word,
// counted on the JSON-encoded message blob (the exact bytes the proxy forwards
// on the wire). This is the delta that drives X-CTX-Savings. The upstream's
// own usage block (input_tokens/output_tokens, prompt_tokens/completion_tokens)
// is passed through to the client unchanged and is NOT used here.
func EstimateTokens(b []byte) int {
	s := string(b)
	return len(strings.Fields(s))
}

// savings is the per-request savings computation, fully determined by the
// before/after token counts and the effective mode.
type savings struct {
	Before   int
	After    int
	Mode     Mode
	CacheHit bool
}

// delta is the compression delta: tokens_before - tokens_after (>= 0 after a
// successful transform, 0 on passthrough / fail-open).
func (s savings) delta() int { return s.Before - s.After }

// savingsRatio is the fraction of tokens removed (0..1); guarded against
// division by zero.
func (s savings) savingsRatio() float64 {
	if s.Before == 0 {
		return 0
	}
	return float64(s.Before-s.After) / float64(s.Before)
}

// writeHeaders writes the five savings headers onto w. When cacheHit is set
// and the mode is cache, X-CTX-Cache-Hit is "true"; every other case is
// "false" (passthrough and token never report a cache hit).
func (s savings) writeHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set(HeaderSavings, fmt.Sprint(s.delta()))
	h.Set(HeaderOriginal, fmt.Sprint(s.Before))
	h.Set(HeaderCompressed, fmt.Sprint(s.After))
	h.Set(HeaderMode, s.Mode.String())
	hit := "false"
	if s.Mode == ModeCache && s.CacheHit {
		hit = "true"
	}
	h.Set(HeaderCacheHit, hit)
}

// transform is the pure compression function shared by every mode. It takes
// the normalized messages and returns the compressed messages (a new slice;
// the input is not mutated). The mode selects how much is transformed:
//
//   - passthrough: the messages are returned unchanged (delta 0).
//   - token: every message's content is reduced to a deterministic summary.
//   - cache: only the mutable tail (messages after the frozen head) is
//     reduced; the stable prefix is left intact.
//
// A transform fault is reported as a non-nil error; the caller fails open to
// passthrough (ADR-0016).
//
// The reducer is a deterministic placeholder so the delta is measurable and
// non-zero on the AC fixture. A later ticket swaps in real compression behind
// this same signature without touching the transport.
func transform(mode Mode, msgs []message) ([]message, error) {
	switch mode {
	case ModePassthrough:
		out := make([]message, len(msgs))
		copy(out, msgs)
		return out, nil
	case ModeToken:
		out := make([]message, len(msgs))
		for i, m := range msgs {
			if len(m.Content) == 0 {
				out[i] = m
				continue
			}
			out[i] = message{Role: m.Role, Content: summarize(m)}
		}
		return out, nil
	case ModeCache:
		head := headSplit(msgs)
		out := make([]message, len(msgs))
		for i, m := range msgs {
			if i < head {
				out[i] = m // frozen stable prefix
			} else if len(m.Content) == 0 {
				out[i] = m
			} else {
				out[i] = message{Role: m.Role, Content: summarize(m)}
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("proxy: unknown mode %q", mode)
	}
}

// summarize is the deterministic reducer: it replaces a message's content with
// a fixed-width placeholder that preserves the role and a small constant
// marker. It is intentionally lossy and stable so the before/after token delta
// is exactly computable from the wire bytes.
func summarize(m message) string {
	return "[ctx-summarized " + m.Role + "]"
}

// judgmentTransform is the ADR-0034 judgment pipeline (prefilter → judge →
// compress) applied to a single message's content. It returns the compressed
// content. Fail-open (ADR-0016): the engine never returns a blocking error; a
// model fault degrades to the deterministic pre-filter decision, and the
// compressor is truncate-only (never expands, never deletes).
func judgmentTransform(ctx context.Context, j *judgment.Judgment, m message) message {
	if len(m.Content) == 0 {
		return m
	}
	d, _ := j.Judge(ctx, judgment.Unit{Role: m.Role, Content: m.Content})
	compressed := judgment.Compress(judgment.Unit{Role: m.Role, Content: m.Content}, d)
	return message{Role: m.Role, Content: compressed.Content}
}

// applyTransformWithJudgment is the judgment-aware variant of applyTransform.
// When the engine is nil it falls back to the pure deterministic transform.
// Otherwise it runs the ADR-0034 pipeline per message, using the request
// context so a client disconnect bounds the (already short-timeout) Ollama call.
func applyTransformWithJudgment(mode Mode, body []byte, j *judgment.Judgment) ([]byte, bool, error) {
	if j == nil {
		return applyTransform(mode, body)
	}
	ctx := context.Background()
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, false, err
	}
	rawMsgs, ok := doc["messages"]
	if !ok || len(rawMsgs) == 0 {
		return nil, false, nil
	}
	var msgs []message
	if err := json.Unmarshal(rawMsgs, &msgs); err != nil {
		return nil, false, err
	}
	cacheHit := false
	if mode == ModeCache {
		cacheHit = headSplit(msgs) > 0
	}
	out := make([]message, len(msgs))
	switch mode {
	case ModePassthrough:
		copy(out, msgs)
	case ModeToken:
		for i, m := range msgs {
			out[i] = judgmentTransform(ctx, j, m)
		}
	case ModeCache:
		head := headSplit(msgs)
		for i, m := range msgs {
			if i < head {
				out[i] = m // frozen stable prefix
			} else {
				out[i] = judgmentTransform(ctx, j, m)
			}
		}
	default:
		return nil, false, fmt.Errorf("proxy: unknown mode %q", mode)
	}
	newRaw, err := json.Marshal(out)
	if err != nil {
		return nil, false, err
	}
	doc["messages"] = newRaw
	outb, err := json.Marshal(doc)
	if err != nil {
		return nil, false, err
	}
	return outb, cacheHit, nil
}

// applyTransform decodes the request body as a JSON object, transforms only
// the messages array in-place, and re-encodes the full object. This preserves
// all non-message fields (model, max_tokens, stream, etc.) instead of the
// lossy re-encoding of a bare {messages:[...]} shape. A parse or transform
// fault is returned as a non-nil error so the caller can fail open to
// passthrough (ADR-0016).
func applyTransform(mode Mode, body []byte) ([]byte, bool, error) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, false, err
	}
	rawMsgs, ok := doc["messages"]
	// A missing or empty messages array means there is nothing to compress.
	if !ok || len(rawMsgs) == 0 {
		return nil, false, nil
	}
	var msgs []message
	if err := json.Unmarshal(rawMsgs, &msgs); err != nil {
		return nil, false, err
	}

	cacheHit := false
	if mode == ModeCache {
		cacheHit = headSplit(msgs) > 0
	}

	tmsgs, err := transform(mode, msgs)
	if err != nil {
		return nil, false, err
	}
	newRaw, err := json.Marshal(tmsgs)
	if err != nil {
		return nil, false, err
	}
	doc["messages"] = newRaw
	out, err := json.Marshal(doc)
	if err != nil {
		return nil, false, err
	}
	return out, cacheHit, nil
}
