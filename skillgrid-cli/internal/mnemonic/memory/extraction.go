package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// logWriter is the sink for passive-extraction warnings (the LLM-failure
// fallback notice). stdout keeps it visible without pulling a logging
// dependency into this package; tests do not assert on it.
func logWriter() io.Writer { return os.Stderr }

// ExtractionLLM is the pluggable LLM seam for passive extraction (014,
// step 05). It mirrors the process.LLM / layer.LLM seam pattern: a small
// interface a backend implements, with NO CGo LLM client in this package. The
// method takes the raw session text and returns the LLM's structured response
// (a JSON document of learnings, see extractionPrompt). A nil seam (or one
// that errors) leaves extraction to the deterministic regex floor — the
// fallback is always available.
type ExtractionLLM interface {
	Extract(ctx context.Context, text string) (string, error)
}

// extractionPrompt is the instruction sent with the session text. It asks for
// nuanced learnings the regex floor misses (free-form text, multi-clause
// sentences, implied decisions/gotchas) — never invented content — and for a
// strict JSON response so the parse is deterministic.
const extractionPrompt = `You are extracting structured learnings from a block of session text.
Return ONLY a JSON object: {"learnings": [{"text": "<one learning>", "type": "<decision|bugfix|architecture|pattern|config|preference|discovery|learning|lesson>"}]}.
One entry per learning. If a learning is written in a structured section, quote its item VERBATIM. For a nuanced learning in free-form prose, consolidate the relevant sentences into a single complete sentence. Do not invent content that is not present in the text.
Text:
`

// llmExtraction is the LLM's structured response: a list of learnings, each a
// short text plus an optional type hint (validated and defaulted downstream).
type llmExtraction struct {
	Learnings []llmLearning `json:"learnings"`
}

type llmLearning struct {
	Text string `json:"text"`
	Type string `json:"type,omitempty"`
}

// ExtractWithLLM calls the LLM seam on the raw text and parses the structured
// JSON response into PassiveItem values (014, step 05.1). The caller decides
// the fallback: a nil seam, a seam error, or a malformed response is returned
// as an error so CapturePassive can fall back to the regex floor. Items are
// trimmed and dropped when blank; an empty parse is an error too (the LLM
// found nothing → let the regex floor decide).
func ExtractWithLLM(ctx context.Context, text string, llm ExtractionLLM) ([]PassiveItem, error) {
	if llm == nil {
		return nil, fmt.Errorf("no extraction LLM configured")
	}
	resp, err := llm.Extract(ctx, extractionPrompt+text)
	if err != nil {
		return nil, fmt.Errorf("extraction LLM: %w", err)
	}
	var out llmExtraction
	if err := json.Unmarshal([]byte(resp), &out); err != nil {
		return nil, fmt.Errorf("parse extraction LLM response: %w", err)
	}
	items := make([]PassiveItem, 0, len(out.Learnings))
	for _, l := range out.Learnings {
		body := strings.TrimSpace(l.Text)
		if body == "" {
			continue
		}
		// No Heading: an LLM item identical in text to a regex item must shape
		// (and content-hash) identically so dedupePassiveItems merges it.
		items = append(items, PassiveItem{Text: body})
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("extraction LLM returned no learnings")
	}
	return items, nil
}

// passiveItemHash is the content hash of an item's normalized text — the
// dedup key for combining LLM and regex extraction outputs (014, step 05.3).
// It normalizes the text (case, punctuation, whitespace) and caps the length
// so a near-duplicate item the LLM and the regex both produce (identical or
// only lightly reworded) hashes to the same key, while a genuinely new
// learning hashes differently.
func passiveItemHash(item PassiveItem, title, typ string) string {
	norm := extractDedupKey(item.Text)
	sum := sha256.Sum256([]byte(norm))
	return hex.EncodeToString(sum[:])
}

// extractDedupKey normalizes item text into a stable dedup fingerprint:
// lowercased, punctuation stripped, whitespace collapsed, length-capped so a
// trailing fragment difference does not split one learning into two keys.
func extractDedupKey(text string) string {
	var b strings.Builder
	prevSpace := true
	for _, r := range strings.ToLower(text) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			prevSpace = false
		case r == ' ' || r == '\t' || r == '\n':
			if !prevSpace && b.Len() > 0 {
				b.WriteRune(' ')
				prevSpace = true
			}
		default:
			// punctuation → boundary
			prevSpace = true
		}
	}
	key := strings.TrimRight(b.String(), " ")
	if len(key) > 90 {
		key = key[:90]
	}
	return key
}

// dedupePassiveItems merges LLM and regex extraction outputs into one
// de-duplicated set (014, step 05.3). Items are keyed by their content hash
// (passiveItemHash) so an item both passes produce is stored once. The LLM's
// order is preserved first, then any regex-only items.
func dedupePassiveItems(primary, secondary []PassiveItem) []PassiveItem {
	seen := make(map[string]bool, len(primary)+len(secondary))
	out := make([]PassiveItem, 0, len(primary)+len(secondary))
	for _, group := range [][]PassiveItem{primary, secondary} {
		for _, item := range group {
			title, typ := shapePassiveItem(item)
			k := passiveItemHash(item, title, typ)
			if seen[k] {
				continue
			}
			seen[k] = true
			out = append(out, item)
		}
	}
	return out
}
