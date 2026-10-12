package traffic

import (
	"os"
	"path/filepath"
	"testing"
)

// writeRulesFile writes content to a temp file and returns its path.
func writeRulesFile(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "rules.yaml")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write rules: %v", err)
	}
	return p
}

const goodRules = `
rules:
  - id: er_fail_then_ok
    category: error_recovery
    pattern: fail_then_ok
    confidence: 0.8
    description: "fail then recover"
  - id: env_repeated_command
    category: environment
    pattern: repetition
    confidence: 0.7
    min_times: 3
    description: "repeated command"
  - id: pref_repeated_message
    category: preference
    pattern: message_repetition
    confidence: 0.7
    min_times: 2
    description: "repeated message"
  - id: arch_common_prefix
    category: architecture
    pattern: path_prefix
    confidence: 0.7
    min_times: 3
    description: "common prefix"
`

// TestLoadRules_GoodFile: a well-formed file loads, preserves file order, and
// keeps every field (including min_times).
func TestLoadRules_GoodFile(t *testing.T) {
	rules, err := LoadRules(writeRulesFile(t, goodRules))
	if err != nil {
		t.Fatalf("LoadRules: %v", err)
	}
	if len(rules) != 4 {
		t.Fatalf("len(rules) = %d, want 4", len(rules))
	}
	want := []struct {
		id       string
		category string
		pattern  PatternKind
		conf     float64
		minTimes int
	}{
		{"er_fail_then_ok", "error_recovery", PatternFailThenOK, 0.8, 0},
		{"env_repeated_command", "environment", PatternRepetition, 0.7, 3},
		{"pref_repeated_message", "preference", PatternMessageRepetition, 0.7, 2},
		{"arch_common_prefix", "architecture", PatternPathPrefix, 0.7, 3},
	}
	for i, w := range want {
		got := rules[i]
		if got.ID != w.id || got.Category != w.category || got.Pattern != w.pattern ||
			got.Confidence != w.conf || got.MinTimes != w.minTimes {
			t.Errorf("rule %d = %+v, want %+v", i, got, w)
		}
	}
}

// TestLoadRules_MissingFile: a missing file is a data error (fail-open concern
// is on the caller).
func TestLoadRules_MissingFile(t *testing.T) {
	_, err := LoadRules(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err == nil {
		t.Fatal("LoadRules on a missing file: want error, got nil")
	}
}

func TestLoadRules_BadYAML(t *testing.T) {
	if _, err := LoadRules(writeRulesFile(t, "rules: [not a mapping")); err == nil {
		t.Fatal("LoadRules on bad yaml: want error, got nil")
	}
}

func TestLoadRules_UnknownCategory(t *testing.T) {
	body := `
rules:
  - id: x
    category: bogus
    pattern: fail_then_ok
    confidence: 0.8
`
	if _, err := LoadRules(writeRulesFile(t, body)); err == nil {
		t.Fatal("LoadRules with unknown category: want error, got nil")
	}
}

func TestLoadRules_UnknownPattern(t *testing.T) {
	body := `
rules:
  - id: x
    category: environment
    pattern: nonsense
    confidence: 0.8
`
	if _, err := LoadRules(writeRulesFile(t, body)); err == nil {
		t.Fatal("LoadRules with unknown pattern: want error, got nil")
	}
}

func TestLoadRules_ConfidenceOutOfRange(t *testing.T) {
	body := `
rules:
  - id: x
    category: environment
    pattern: repetition
    confidence: 1.5
    min_times: 2
`
	if _, err := LoadRules(writeRulesFile(t, body)); err == nil {
		t.Fatal("LoadRules with confidence 1.5: want error, got nil")
	}
}

func TestLoadRules_RepetitionRequiresMinTimes(t *testing.T) {
	body := `
rules:
  - id: x
    category: environment
    pattern: repetition
    confidence: 0.7
`
	if _, err := LoadRules(writeRulesFile(t, body)); err == nil {
		t.Fatal("LoadRules repetition without min_times: want error, got nil")
	}
}

func TestLoadRules_MissingID(t *testing.T) {
	body := `
rules:
  - category: environment
    pattern: repetition
    confidence: 0.7
    min_times: 2
`
	if _, err := LoadRules(writeRulesFile(t, body)); err == nil {
		t.Fatal("LoadRules with missing id: want error, got nil")
	}
}

// TestLoadRules_ErrorDoesNotPanic: a parse/data error returns a non-nil error
// and a nil slice (the fail-open contract). It must never panic, and the
// caller can degrade to an empty candidate set from the returned nil.
func TestLoadRules_ErrorDoesNotPanic(t *testing.T) {
	var rules []Rule
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("LoadRules panicked on bad input: %v", r)
			}
		}()
		rules, err = LoadRules(writeRulesFile(t, "rules: not-a-mapping"))
	}()
	if err == nil {
		t.Fatal("LoadRules: want error for malformed yaml, got nil")
	}
	if rules != nil {
		t.Errorf("rules = %v, want nil on error", rules)
	}
	// Degrade: nil rules → a learner with no rules extracts nothing, no panic.
	l := NewLearner(rules, 0, 0)
	events := []Event{
		{Kind: "tool_result", Tool: "bash", Target: "x", Status: "error", Seq: 1},
		{Kind: "tool_result", Tool: "bash", Target: "x", Status: "ok", Seq: 2},
	}
	if got := l.Extract("sess", events); len(got) != 0 {
		t.Errorf("degraded learner extracted %d candidates, want 0", len(got))
	}
}

// TestLoadRules_PackageRulesYAML: the shipped package rules.yaml loads and has
// at least one rule per category (the §18.2 requirement).
func TestLoadRules_PackageRulesYAML(t *testing.T) {
	rules, err := LoadRules("rules.yaml")
	if err != nil {
		t.Fatalf("LoadRules(rules.yaml): %v", err)
	}
	cats := map[string]bool{}
	for _, r := range rules {
		cats[r.Category] = true
	}
	for _, c := range []string{"error_recovery", "environment", "preference", "architecture"} {
		if !cats[c] {
			t.Errorf("package rules.yaml has no rule for category %q", c)
		}
	}
}
