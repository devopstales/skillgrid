package secondbrain

import (
	"regexp"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
)

// Capture classifier (TICKET-05, 2026-10-06-second-brain-learnings):
// a deterministic two-pass filter over free-form text that decides whether
// the text is worth proposing as a memory save, and how confidently. It
// mirrors the correction-detector structure adopted from claude-os
// (change findings):
//
//  1. NEGATIVE patterns ("no worries", ...) suppress the suggestion even when
//     a strong/weak pattern is present — polite refusals are not corrections.
//  2. STRONG patterns ("we decided to", "always use", ...) fire on their own.
//  3. WEAK patterns ("no", "yes", "use", "avoid") fire only when a directive
//     word ("use X", "avoid Y") is present — a bare "no" is not a capture.
//
// The score is keyword weight + position (an earlier match in the text is
// the subject of the turn; buried matches are damped), and the whole
// classifier is pure: same input, same output, every run — no LLM, no time,
// no randomness, no store. The suggestion itself never persists; the agent
// confirms via mem_save.

// CaptureThreshold is the default confidence gate for mem_capture_suggest:
// at or above it the tool returns suggested:true with a draft.
const CaptureThreshold = 0.75

// triggerType maps a capture trigger to the observation type the confirm
// (mem_save) should carry. Durable-rule triggers (decisions, switches,
// conventions) map to decision/convention; the bare weak correction states
// how things should be done, so it maps to convention.
var triggerType = map[string]string{
	"decided":    "decision",
	"rule":       "convention",
	"always":     "convention",
	"never":      "convention",
	"convention": "convention",
	"switching":  "decision",
	"correction": "convention",
	"weak":       "convention",
}

// TriggerType returns the observation type a trigger maps to ("" for an
// unknown trigger — the confirm call still supplies its own type).
func TriggerType(trigger string) string {
	return triggerType[trigger]
}

// SuggestedKnowledgeClass returns the durability class the confirm
// (mem_save) should record for a triggered capture: the same type→class
// heuristic mem_save itself applies on save (memory.InferKnowledgeClass),
// so the draft and the landed row can never disagree.
func SuggestedKnowledgeClass(trigger string) string {
	return memory.InferKnowledgeClass(TriggerType(trigger))
}

// capturePattern is one named pattern of the two-pass classifier. strong
// patterns fire on their own; weak patterns fire only when a directive
// word is present; negative patterns suppress everything.
type capturePattern struct {
	kind    string // "strong" | "weak" | "negative"
	name    string // stable trigger key for the draft
	phrases []string
	weights []float64 // per-phrase keyword weight
}

// capturePatterns is the closed pattern table. Order encodes precedence
// (the first matched strong pattern wins ties), not scoring.
var capturePatterns = []capturePattern{
	{kind: "strong", name: "decided", phrases: []string{"we decided to", "we decided"}, weights: []float64{0.9, 0.9}},
	{kind: "strong", name: "switching", phrases: []string{"switching from", "switch from"}, weights: []float64{0.9, 0.9}},
	{kind: "strong", name: "rule", phrases: []string{"the rule is"}, weights: []float64{0.9}},
	{kind: "strong", name: "always", phrases: []string{"always use"}, weights: []float64{0.85}},
	{kind: "strong", name: "never", phrases: []string{"never use"}, weights: []float64{0.85}},
	{kind: "strong", name: "correction", phrases: []string{"don't do that", "do not do that"}, weights: []float64{0.85, 0.85}},
	{kind: "strong", name: "convention", phrases: []string{"the convention is", "convention is"}, weights: []float64{0.8, 0.8}},
	{kind: "weak", name: "weak", phrases: []string{"no", "yes", "use", "avoid"}, weights: []float64{0.6, 0.6, 0.55, 0.55}},
	{kind: "negative", name: "negative", phrases: []string{"no worries", "no problem", "nope"}, weights: []float64{0, 0, 0}},
}

// directiveWords are the words that promote a weak pattern to a fireable
// one ("use X", "avoid Y" are directives; a bare "no" is not).
var directiveWords = []string{"use", "avoid", "do", "change", "instead", "prefer", "pick", "swap"}

// wordRe builds a word-boundary matcher for a literal word so "nope" does
// not match "no" and "number" does not match "no".
func wordRe(word string) func(string) bool {
	re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(word) + `\b`)
	return re.MatchString
}

// captureMatch is a pattern match plus its position.
type captureMatch struct {
	kind   string // "strong" | "weak"
	name   string // stable trigger key for the draft
	phrase string
	weight float64
	pos    int // rune index of the match in the lowercased text
}

// ClassifyCapture deterministically classifies free-form text for
// real-time capture. It returns the winning trigger name ("" when nothing
// fires) and a confidence in [0,1]; the caller gates on CaptureThreshold
// (0.75). Deterministic: pure over the input string.
func ClassifyCapture(text string) (string, float64) {
	lower := strings.ToLower(text)

	// Pass 0: negative patterns suppress everything, even a strong match.
	for _, p := range capturePatterns {
		if p.kind != "negative" {
			continue
		}
		for _, ph := range p.phrases {
			if strings.Contains(lower, ph) {
				return "", 0.0
			}
		}
	}

	var strongs, weaks []captureMatch
	for _, p := range capturePatterns {
		for i, ph := range p.phrases {
			pos := strings.Index(lower, ph)
			if pos < 0 {
				continue
			}
			m := captureMatch{kind: p.kind, name: p.name, phrase: ph, weight: p.weights[i], pos: pos}
			if p.kind == "strong" {
				strongs = append(strongs, m)
			}
		}
	}

	// Weak patterns fire only when the weak keyword is present AND a
	// directive word distinct from it is present. "no, use pnpm": "no" is
	// the keyword, "use" the directive. "use x" alone: "use" is both, so no
	// self-promotion — a directive without a weak keyword is not a
	// correction.
	for i, ph := range capturePatterns[weakIndex()].phrases {
		if !wordRe(ph)(lower) {
			continue
		}
		if hasDirectiveWord(lower, ph) {
			weaks = append(weaks, captureMatch{kind: "weak", name: "weak", phrase: ph, weight: capturePatterns[weakIndex()].weights[i], pos: strings.Index(lower, ph)})
		}
	}

	best, conf := pickBest(strongs, weaks)
	if best == nil {
		return "", 0.0
	}
	// A weak match that did not reach the gate names no trigger: below
	// threshold the suggestion is simply absent.
	if best.kind == "weak" && conf < CaptureThreshold {
		return "", conf
	}
	return best.name, conf
}

// weakIndex is the position of the weak pattern in capturePatterns (the
// table is fixed; derived once so the table stays the single source).
func weakIndex() int {
	for i, p := range capturePatterns {
		if p.kind == "weak" {
			return i
		}
	}
	return -1
}

// directiveBoost is the confidence a directive word adds to a weak match:
// a weak keyword is not a capture alone, but a weak keyword PLUS a directive
// word ("no, use X") is — the boost lifts it to the weak-pattern confidence
// floor (CaptureThreshold).
const directiveBoost = 0.2

// pickBest returns the highest-scoring match and its score. A strong match
// always beats a weak one at equal base score; a directive word lifts a
// weak match to the confidence floor.
func pickBest(strongs, weaks []captureMatch) (*captureMatch, float64) {
	best := (*captureMatch)(nil)
	bestVal := -1.0
	consider := func(m captureMatch, strong bool) {
		s := matchScore(m)
		if strong {
			s += 0.1
		}
		if best == nil || s > bestVal {
			best = &m
			bestVal = s
		}
	}
	for _, m := range strongs {
		consider(m, true)
	}
	for _, m := range weaks {
		mm := m
		consider(mm, false)
	}
	if best == nil {
		return nil, 0.0
	}
	s := matchScore(*best)
	if best.kind == "weak" {
		s += directiveBoost
	}
	return best, clamp01(s)
}

// matchScore is the keyword + position score: the matched phrase's keyword
// weight, damped by how deep into the text the match sits (early text is
// the subject of the turn; filler buries the signal).
func matchScore(m captureMatch) float64 {
	posScore := 1.0 - 0.5*(float64(m.pos)/posDampenSpan)
	if posScore < 0.5 {
		posScore = 0.5
	}
	return clamp01(m.weight * posScore)
}

// posDampenSpan is the rune distance over which the position score damps
// from 1.0 to its 0.5 floor: a capture phrase in the first ~40 runes of a
// turn is "at the front"; deeper matches are damped.
const posDampenSpan = 40.0

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// hasDirectiveWord reports whether the text contains a directive word other
// than the weak keyword itself.
func hasDirectiveWord(lower, kw string) bool {
	for _, d := range directiveWords {
		if d == kw {
			continue
		}
		if wordRe(d)(lower) {
			return true
		}
	}
	return false
}

// SuggestDraft is the proposed save the suggest tool returns above the
// gate: the winning trigger, its confidence, and the type/knowledge_class
// the confirm (mem_save) should carry.
type SuggestDraft struct {
	Trigger        string  `json:"trigger"`
	Confidence     float64 `json:"confidence"`
	KnowledgeClass string  `json:"knowledge_class"`
	Type           string  `json:"type"`
}

// SuggestOutcome is the confidence-gated projection mem_capture_suggest
// renders: the deterministic classifier result plus the 0.75 gate. It never
// touches a store — idempotent and read-only; the agent confirms via
// mem_save (which persists, with the suggested knowledge_class).
type SuggestOutcome struct {
	Suggested  bool          `json:"suggested"`
	Confidence float64       `json:"confidence"`
	Draft      *SuggestDraft `json:"draft,omitempty"`
}

// SuggestCapture classifies text and applies the 0.75 gate. It never
// touches a store — idempotent and read-only.
func SuggestCapture(text string) SuggestOutcome {
	trigger, conf := ClassifyCapture(text)
	if trigger == "" || conf < CaptureThreshold {
		return SuggestOutcome{Suggested: false, Confidence: round4(conf)}
	}
	return SuggestOutcome{
		Suggested:  true,
		Confidence: round4(conf),
		Draft: &SuggestDraft{
			Trigger:        trigger,
			Confidence:     round4(conf),
			KnowledgeClass: SuggestedKnowledgeClass(trigger),
			Type:           TriggerType(trigger),
		},
	}
}

// round4 rounds a confidence to 4 decimals so the JSON output is stable
// (deterministic) across calls — a float64 repr drift would otherwise break
// the idempotency contract at the byte level.
func round4(v float64) float64 {
	return float64(int64(v*10000+0.5)) / 10000
}
