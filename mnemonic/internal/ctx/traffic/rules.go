// Package traffic is the zero-LLM, rule-based, deterministic traffic learner
// (plan §18). It turns a session-event stream into candidate ctx_traffic_memories
// rows using a human-editable traffic/rules.yaml. Extraction is a pure function
// of (events, rules): same input, same memories, same order. There is no
// network, no model, no wall clock and no randomness in the candidate identity
// or ordering — the only injectable dependencies are the ctx store (for
// EvictToBound) and a clock (for created_at, testability).
//
// Fail-open (ADR-0016): a rules-file parse error degrades the learner to an
// empty candidate set (never a panic); a store error degrades eviction to a
// no-op. The store methods are already fail-open; this package adds no
// hard-fail path.
package traffic

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ruleCategories is the closed set of categories a rule may declare (plan
// §18.2). A rule outside this set is a data error, not a match-time decision.
var ruleCategories = map[string]bool{
	"error_recovery": true,
	"environment":    true,
	"preference":     true,
	"architecture":   true,
}

// PatternKind names the deterministic matchers a rule may use. The set is
// closed; an unknown kind is a data error at LoadRules time.
type PatternKind string

const (
	// PatternFailThenOK matches a tool_result{status:error} followed, on the
	// same non-empty Target, by a tool_result{status:ok} (error_recovery).
	PatternFailThenOK PatternKind = "fail_then_ok"
	// PatternRepetition matches a Tool/Target pair (or a Tool with an empty
	// Target) occurring at least MinTimes times (environment).
	PatternRepetition PatternKind = "repetition"
	// PatternMessageRepetition matches a non-empty user_message Message
	// appearing at least MinTimes times (preference).
	PatternMessageRepetition PatternKind = "message_repetition"
	// PatternPathPrefix matches reads/edits of files under one common path
	// prefix, at least MinTimes times (architecture).
	PatternPathPrefix PatternKind = "path_prefix"
)

// Rule is one human-authored entry of traffic/rules.yaml. PatternKind and
// MinTimes are the matcher descriptor; Description is free, human-auditable
// text carried into the generated memory.
type Rule struct {
	ID          string      `yaml:"id"`
	Category    string      `yaml:"category"`
	Pattern     PatternKind `yaml:"pattern"`
	Description string      `yaml:"description"`
	Confidence  float64     `yaml:"confidence"`
	MinTimes    int         `yaml:"min_times"`
}

// rulesFile is the top-level shape of traffic/rules.yaml.
type rulesFile struct {
	Rules []Rule `yaml:"rules"`
}

// LoadRules reads and validates a traffic/rules.yaml file. Rules are returned
// in file order (the first matching rule wins at extraction time). A missing
// or unparseable file, or a malformed rule (empty id, unknown category or
// pattern, confidence outside [0,1], non-positive min_times for a counting
// pattern), is a data error: LoadRules returns a non-nil error and the caller
// (Learner) degrades gracefully (ADR-0016).
func LoadRules(path string) ([]Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("traffic: read rules: %w", err)
	}
	var rf rulesFile
	if err := yaml.Unmarshal(data, &rf); err != nil {
		return nil, fmt.Errorf("traffic: parse rules: %w", err)
	}
	out := make([]Rule, 0, len(rf.Rules))
	for i, r := range rf.Rules {
		if r.ID == "" {
			return nil, fmt.Errorf("traffic: rule %d: missing id", i)
		}
		if !ruleCategories[r.Category] {
			return nil, fmt.Errorf("traffic: rule %s: unknown category %q", r.ID, r.Category)
		}
		if !validPattern(r.Pattern) {
			return nil, fmt.Errorf("traffic: rule %s: unknown pattern %q", r.ID, r.Pattern)
		}
		if r.Confidence < 0 || r.Confidence > 1 {
			return nil, fmt.Errorf("traffic: rule %s: confidence %v out of [0,1]", r.ID, r.Confidence)
		}
		if needsMinTimes(r.Pattern) && r.MinTimes < 1 {
			return nil, fmt.Errorf("traffic: rule %s: min_times must be >= 1 for pattern %q", r.ID, r.Pattern)
		}
		out = append(out, r)
	}
	return out, nil
}

func validPattern(p PatternKind) bool {
	switch p {
	case PatternFailThenOK, PatternRepetition, PatternMessageRepetition, PatternPathPrefix:
		return true
	default:
		return false
	}
}

func needsMinTimes(p PatternKind) bool {
	switch p {
	case PatternRepetition, PatternMessageRepetition, PatternPathPrefix:
		return true
	default:
		return false
	}
}
