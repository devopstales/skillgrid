package traffic

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	ctxstore "github.com/devopstales/skillgrid/mnemonic/internal/ctx/store"
)

// Event is one entry of a session-event stream (plan §18.1). The learner is a
// pure function over a slice of these plus the rules; nothing here reaches the
// network or a model.
type Event struct {
	Kind    string // e.g. "tool_call", "tool_result", "file_read", "file_edit", "user_message"
	Tool    string // tool name when Kind=="tool_call" / tool_result
	Target  string // path/command/arg the tool acted on
	Status  string // "ok" | "error" | "retry" | "" (tool_result)
	Message string // free text (user_message / tool_result message)
	Seq     int    // monotonic event order within the session
}

// Learner is the deterministic extractor. Rules is applied in file order;
// MinConfidence is the floor below which a candidate is dropped; MaxMemories
// is advisory here (the store-side bound is enforced by EvictToBound); Clock
// supplies created_at (determinism of identity/ordering never depends on it).
// The ONLY injectable dependencies anywhere in the extract path are this
// struct's fields and the *ctxstore.Store passed to EvictToBound — there is no
// http client and no model field, so the extract path cannot make an LLM call.
type Learner struct {
	Rules         []Rule
	MinConfidence float64
	MaxMemories   int
	Clock         func() int64
}

// NewLearner builds a Learner with a fixed clock (for deterministic runs).
func NewLearner(rules []Rule, minConfidence float64, maxMemories int) *Learner {
	return &Learner{
		Rules:         rules,
		MinConfidence: minConfidence,
		MaxMemories:   maxMemories,
		Clock:         func() int64 { return 0 },
	}
}

// evidenceItem is the compact per-event record stored in evidence_json so a
// human can audit what produced a memory.
type evidenceItem struct {
	Seq     int    `json:"seq"`
	Kind    string `json:"kind"`
	Tool    string `json:"tool"`
	Target  string `json:"target"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// Extract is the core pure function: for each rule, in file order, it scans the
// events and emits one candidate *ctxstore.TrafficMemory per match, in a
// deterministic order. A candidate below MinConfidence is NOT emitted. The
// result is ordered by (rule file order, ascending seq) — no map iteration, no
// wall clock, no randomness leak into identity or order.
func (l *Learner) Extract(sessionID string, events []Event) []*ctxstore.TrafficMemory {
	if l == nil {
		return nil
	}
	clock := l.Clock
	if clock == nil {
		clock = func() int64 { return 0 }
	}
	var out []*ctxstore.TrafficMemory
	for _, rule := range l.Rules {
		for _, cand := range matchRule(rule, events) {
			if cand.conf < l.MinConfidence {
				continue
			}
			m := &ctxstore.TrafficMemory{
				MemoryID:     memoryID(sessionID, rule.ID, cand.key),
				SessionID:    sessionID,
				Category:     rule.Category,
				MemoryText:   memoryText(rule, cand),
				EvidenceJSON: evidenceJSON(cand.events),
				Confidence:   cand.conf,
				CreatedAt:    clock(),
			}
			out = append(out, m)
		}
	}
	return out
}

// candidate is a single rule match awaiting emission. key is the stable
// identity suffix (target / fragment / prefix); events are the evidence.
type candidate struct {
	key    string
	conf   float64
	events []Event
}

// matchRule scans events for a single rule's pattern and returns its candidates
// in deterministic (ascending seq) order. Only the first match per logical
// group is emitted (see the per-kind guards) so a repeated trigger does not
// fan out into duplicate memories.
func matchRule(rule Rule, events []Event) []candidate {
	switch rule.Pattern {
	case PatternFailThenOK:
		return failThenOK(rule, events)
	case PatternRepetition:
		return repetition(rule, events)
	case PatternMessageRepetition:
		return messageRepetition(rule, events)
	case PatternPathPrefix:
		return pathPrefix(rule, events)
	default:
		return nil
	}
}

// failThenOK finds tool_result{status:error} events that are later followed by
// a tool_result{status:ok} on the SAME non-empty Target. Each error event is a
// distinct candidate (evidence = the error event + the first ok recovery after
// it on the same target). Confidence is the rule weight, scaled up (capped at
// 1) by the number of error→ok cycles observed for that target.
func failThenOK(rule Rule, events []Event) []candidate {
	// Precompute, for each target, the ok recovery event (lowest seq after a
	// given error) and the total number of error→ok cycles per target.
	// First pass: group ok events by target in seq order.
	okByTarget := map[string][]int{} // target -> seqs of ok tool_result
	for _, e := range events {
		if e.Kind == "tool_result" && e.Status == "ok" && e.Target != "" {
			okByTarget[e.Target] = append(okByTarget[e.Target], e.Seq)
		}
	}
	// Second pass: for each error event, find the first ok with seq > error seq
	// on the same target, and count cycles.
	type recovery struct {
		target string
		errSeq int
		okSeq  int
		errIdx int
		okIdx  int
	}
	var recs []recovery
	cycles := map[string]int{}
	for i, e := range events {
		if e.Kind != "tool_result" || e.Status != "error" || e.Target == "" {
			continue
		}
		for _, okSeq := range okByTarget[e.Target] {
			if okSeq > e.Seq {
				recs = append(recs, recovery{target: e.Target, errSeq: e.Seq, okSeq: okSeq, errIdx: i, okIdx: idxOfSeq(events, okSeq)})
				cycles[e.Target]++
				break
			}
		}
	}
	// Deterministic order: by error seq.
	sort.Slice(recs, func(a, b int) bool { return recs[a].errSeq < recs[b].errSeq })
	out := []candidate{}
	for _, r := range recs {
		conf := rule.Confidence
		if cycles[r.target] > 1 {
			conf = min1(conf + 0.05*(float64(cycles[r.target])-1))
		}
		out = append(out, candidate{
			key:    "target=" + r.target,
			conf:   conf,
			events: []Event{events[r.errIdx], events[r.okIdx]},
		})
	}
	return out
}

// idxOfSeq returns the index of the first event with the given seq, or -1.
func idxOfSeq(events []Event, seq int) int {
	for i, e := range events {
		if e.Seq == seq {
			return i
		}
	}
	return -1
}

// repetition finds a Tool (with a non-empty Target, or any Tool when Target is
// empty) that is used at least MinTimes times, emitting one candidate for the
// single most-frequent qualifying key. This is the "stable fact" signal
// (environment). The key is tool/target so the memory is auditable.
func repetition(rule Rule, events []Event) []candidate {
	counts := map[string]int{}
	var keys []string
	for _, e := range events {
		if e.Kind == "tool_call" && e.Tool != "" {
			key := e.Tool + "|" + e.Target
			if _, ok := counts[key]; !ok {
				keys = append(keys, key)
			}
			counts[key]++
		}
	}
	// Pick the single highest-count key (ties broken by key string for
	// determinism).
	var best string
	bestCount := 0
	for _, k := range keys {
		if counts[k] > bestCount || (counts[k] == bestCount && k < best) {
			best = k
			bestCount = counts[k]
		}
	}
	if bestCount < rule.MinTimes {
		return nil
	}
	// Evidence: all events for the winning key, in seq order.
	parts := strings.SplitN(best, "|", 2)
	tool, target := parts[0], parts[1]
	var ev []Event
	for _, e := range events {
		if e.Kind == "tool_call" && e.Tool == tool && e.Target == target {
			ev = append(ev, e)
		}
	}
	keyStr := tool
	if target != "" {
		keyStr = tool + " target=" + target
	}
	return []candidate{{key: keyStr, conf: rule.Confidence, events: ev}}
}

// messageRepetition finds a non-empty user_message Message repeated at least
// MinTimes times (preference signal — a repeated user correction or usage
// pattern). One candidate per qualifying distinct fragment.
func messageRepetition(rule Rule, events []Event) []candidate {
	counts := map[string][]int{} // fragment -> seqs
	var frags []string
	for _, e := range events {
		if e.Kind == "user_message" && strings.TrimSpace(e.Message) != "" {
			frag := e.Message
			if _, ok := counts[frag]; !ok {
				frags = append(frags, frag)
			}
			counts[frag] = append(counts[frag], e.Seq)
		}
	}
	var out []candidate
	for _, f := range frags {
		if len(counts[f]) < rule.MinTimes {
			continue
		}
		seqs := append([]int(nil), counts[f]...)
		sort.Ints(seqs)
		var ev []Event
		for _, s := range seqs {
			ev = append(ev, events[idxOfSeq(events, s)])
		}
		out = append(out, candidate{
			key:    "msg=" + fragmentKey(f),
			conf:   rule.Confidence,
			events: ev,
		})
	}
	// Deterministic order by fragment key.
	sort.Slice(out, func(a, b int) bool { return out[a].key < out[b].key })
	return out
}

// pathPrefix finds a common path prefix under which files are read or edited
// at least MinTimes times (architecture signal — a file-structure/convention
// pattern). The prefix is the longest common path segment shared by ≥MinTimes
// distinct targets.
func pathPrefix(rule Rule, events []Event) []candidate {
	// Collect distinct targets from file_read / file_edit / file_write events.
	counts := map[string]int{}
	var targets []string
	for _, e := range events {
		if (e.Kind == "file_read" || e.Kind == "file_edit" || e.Kind == "file_write") && e.Target != "" {
			if _, ok := counts[e.Target]; !ok {
				targets = append(targets, e.Target)
			}
			counts[e.Target]++
		}
	}
	if len(targets) < rule.MinTimes {
		return nil
	}
	// Find the longest directory prefix shared by at least MinTimes targets.
	// Try progressively shorter prefixes (from the longest common ancestor).
	bestPrefix := ""
	for i := 0; i <= len(targets[0]); i++ {
		// Candidate prefix = targets[0][:i], must be a dir boundary or full path.
		cand := targets[0][:i]
		// Only consider prefixes that end at a path boundary or are a full path.
		if i == len(targets[0]) || (i > 0 && targets[0][i-1] == '/') {
			n := 0
			for _, t := range targets {
				if strings.HasPrefix(t, cand) {
					n++
				}
			}
			if n >= rule.MinTimes && len(cand) > len(bestPrefix) {
				bestPrefix = cand
			}
		}
	}
	if bestPrefix == "" {
		return nil
	}
	// Evidence: all events under the winning prefix, in seq order.
	var ev []Event
	seen := map[int]bool{}
	for _, e := range events {
		if (e.Kind == "file_read" || e.Kind == "file_edit" || e.Kind == "file_write") &&
			strings.HasPrefix(e.Target, bestPrefix) && !seen[e.Seq] {
			seen[e.Seq] = true
			ev = append(ev, e)
		}
	}
	sort.Slice(ev, func(a, b int) bool { return ev[a].Seq < ev[b].Seq })
	return []candidate{{
		key:    "prefix=" + bestPrefix,
		conf:   rule.Confidence,
		events: ev,
	}}
}

// ---- deterministic identity + text helpers ----

// memoryID is a stable, deterministic identity: hex(sha1(sessionID + "\x00" +
// ruleID + "\x00" + key)). Same input → same id, so re-runs dedupe via
// INSERT OR REPLACE. No wall clock, no randomness.
func memoryID(sessionID, ruleID, key string) string {
	h := sha1.Sum([]byte(sessionID + "\x00" + ruleID + "\x00" + key))
	return hex.EncodeToString(h[:])
}

// fragmentKey is a stable short identity for a free-text fragment (the first
// 64 rune-safe bytes, length-prefixed to avoid collisions).
func fragmentKey(f string) string {
	b := []byte(f)
	if len(b) > 64 {
		b = b[:64]
	}
	return fmt.Sprintf("%d:%s", len([]byte(f)), string(b))
}

// memoryText is the short human-auditable sentence stored in memory_text.
func memoryText(rule Rule, c candidate) string {
	what := ""
	switch rule.Pattern {
	case PatternFailThenOK:
		what = "recovered from a tool failure on " + c.key
	case PatternRepetition:
		what = "stable fact: repeated use of " + c.key
	case PatternMessageRepetition:
		what = "repeated user message pattern: " + c.key
	case PatternPathPrefix:
		what = "file-structure convention under " + c.key
	default:
		what = c.key
	}
	if rule.Description != "" {
		return rule.Description + " — " + what
	}
	return what
}

// evidenceJSON renders the evidence events as a compact JSON array.
func evidenceJSON(events []Event) string {
	items := make([]evidenceItem, 0, len(events))
	for _, e := range events {
		items = append(items, evidenceItem{
			Seq:     e.Seq,
			Kind:    e.Kind,
			Tool:    e.Tool,
			Target:  e.Target,
			Status:  e.Status,
			Message: e.Message,
		})
	}
	b, _ := json.Marshal(items)
	return string(b)
}

func min1(v float64) float64 {
	if v > 1 {
		return 1
	}
	return v
}
