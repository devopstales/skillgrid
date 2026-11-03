package proxy

import (
	"encoding/json"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic/internal/ctx/traffic"
)

// activeMemoryContext fetches the applied (active) traffic memories for a
// session and renders them as a single context block suitable for prepending
// to a request's message list (the compression context). This is the seam that
// proves "apply marks a candidate promoted and it appears in subsequent
// compression context": a candidate (status="candidate") is NOT returned; after
// Gate.Apply it IS; after Gate.Revoke it is NOT again.
//
// Fail-open (ADR-0016): a nil gate, a nil store behind the gate, or a
// no-active-memories result all yield ("", nil) — the context block is empty
// and the request body is unchanged. A store error also degrades to ("", nil)
// so a ctx_* fault never 500s the proxy (the one human-gate exception lives in
// the CLI, not this request path).
func activeMemoryContext(g *traffic.Gate, sessionID string) (string, error) {
	if g == nil {
		return "", nil
	}
	active, err := g.ActiveMemories(sessionID)
	if err != nil || len(active) == 0 {
		return "", nil
	}
	var b strings.Builder
	b.WriteString("[traffic context]\n")
	for _, m := range active {
		b.WriteString("- ")
		b.WriteString(m.MemoryText)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

// prependMemoryContext inserts the memory context block as a leading system
// message on the request body's messages array, BEFORE the compression
// transform runs. It operates on the full upstream JSON object (preserving all
// non-message fields) so the memory text becomes part of the context the
// transform sees. A missing/empty messages array or a fault yields the body
// unchanged (fail-open, ADR-0016).
func prependMemoryContext(body []byte, ctx string) []byte {
	if ctx == "" {
		return body
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(body, &doc); err != nil {
		return body
	}
	rawMsgs, ok := doc["messages"]
	if !ok || len(rawMsgs) == 0 {
		return body
	}
	var msgs []message
	if err := json.Unmarshal(rawMsgs, &msgs); err != nil {
		return body
	}
	// Insert the context as a leading system message so it precedes the
	// conversation (and the frozen head) the transform operates on.
	out := make([]message, 0, len(msgs)+1)
	out = append(out, message{Role: "system", Content: ctx})
	out = append(out, msgs...)
	newRaw, err := json.Marshal(out)
	if err != nil {
		return body
	}
	doc["messages"] = newRaw
	encoded, err := json.Marshal(doc)
	if err != nil {
		return body
	}
	return encoded
}
