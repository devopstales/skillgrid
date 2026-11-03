package secondbrain

import (
	"regexp"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
)

// InferType deterministically infers an observation type from the text. It
// reuses the existing type heuristic — the same classifier mem_capture_passive
// uses (memory.ShapePassiveItem) — and adds two verb-bridges ("decided"/"we
// chose" → decision, "discovered" → discovery) so natural-language captures
// like "we decided to use X" infer their type without the agent spelling the
// type word. It is the deterministic floor for mem_save.infer: no LLM, no new
// taxonomy. Returns a valid type (default "learning") when no signal.
func InferType(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "decided") || strings.Contains(lower, "we chose"):
		return "decision"
	case strings.Contains(lower, "discovered"):
		return "discovery"
	}
	_, typ := memory.ShapePassiveItem(memory.PassiveItem{Text: text})
	return typ
}

// InferTopicKey derives a stable upsert key for an inferred observation. It
// delegates to the existing SuggestTopicKey seam (family/segment) so a
// rephrased capture keeps one key across saves. SuggestTopicKey lives in the
// mcp package (which imports secondbrain), so it cannot be called directly
// here without an import cycle; the handler computes the key with it and hands
// it to ApplyInfer. This wrapper keeps the inference contract (type → key) in
// one place and is the unit-testable seam for that contract.
func InferTopicKey(typ, title string) string {
	segment := topicKeySegment(title)
	if segment == "" {
		segment = "untitled"
	}
	return topicKeyFamily(typ) + "/" + segment
}

// ApplyInfer fills Type and TopicKey on the SaveInput ONLY when empty.
// Agent-provided values are never overwritten. The type/topicKey functions are
// passed in so the call site (the mem_save handler) supplies the exact
// inference implementation — including the mcp-side SuggestTopicKey seam —
// without secondbrain importing mcp. Returns true when any field changed.
func ApplyInfer(in *memory.SaveInput, inferType func(text string) string, inferTopicKey func(typ, title string) string) bool {
	changed := false
	if strings.TrimSpace(in.Type) == "" {
		in.Type = inferType(in.Title + " " + in.Content)
		changed = true
	}
	if strings.TrimSpace(in.TopicKey) == "" {
		in.TopicKey = inferTopicKey(in.Type, in.Title)
		changed = true
	}
	return changed
}

// topicKeyFamily/topicKeySegment mirror the mcp SuggestTopicKey segment logic
// (family = normalized type, segment = slugified title) so the inference key
// is byte-identical to the mem_suggest_topic_key output the handler uses. The
// slug rule is the same as mcp's slugRe (runes outside [a-z0-9] → "-").
var topicKeySlugRe = regexp.MustCompile(`[^a-z0-9]+`)

func topicKeyFamily(typ string) string {
	normalized := strings.ToLower(strings.TrimSpace(typ))
	switch normalized {
	case "architecture", "decision", "bugfix", "bug", "pattern", "config",
		"discovery", "learning", "lesson", "preference", "convention":
		if normalized == "lesson" {
			return "learning"
		}
		return normalized
	default:
		return "topic"
	}
}

func topicKeySegment(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if idx := strings.Index(s, "["); idx >= 0 {
		s = strings.TrimSpace(s[:idx])
	}
	return strings.Trim(topicKeySlugRe.ReplaceAllString(s, "-"), "-")
}
