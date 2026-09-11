package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestExtractionLLMConfigDefaultOff is 05.3: the mnemonic.extraction.llm key
// defaults to false (the regex floor) and opts in only when explicitly true.
func TestExtractionLLMConfigDefaultOff(t *testing.T) {
	dir := t.TempDir()

	// No config file at all → defaults: LLM extraction off.
	if got := Load(dir); got.Extraction.LLM {
		t.Fatalf("default: extraction.llm must be false, got true")
	}

	// A config without the extraction section → still off.
	if err := os.MkdirAll(filepath.Join(dir, "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  ttl: 72h\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); got.Extraction.LLM {
		t.Fatalf("absent section: extraction.llm must be false, got true")
	}

	// Explicit opt-in.
	if err := os.WriteFile(filepath.Join(dir, "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  extraction:\n    llm: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); !got.Extraction.LLM {
		t.Fatalf("extraction.llm: true must opt in, got false")
	}
}

// TestImprovementConfigDefaultOff is 08.3: the mnemonic.improve key defaults
// to disabled (the self-improvement feedback loop is opt-in) and opts in only
// when explicitly enabled; rate fields parse from the YAML section.
func TestImprovementConfigDefaultOff(t *testing.T) {
	dir := t.TempDir()

	// No config file at all → defaults: improve disabled.
	if got := Load(dir); got.Improvement.Enabled {
		t.Fatalf("default: improve.enabled must be false, got true")
	}

	// A config without the improve section → still off.
	if err := os.MkdirAll(filepath.Join(dir, "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  ttl: 72h\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); got.Improvement.Enabled {
		t.Fatalf("absent section: improve.enabled must be false, got true")
	}

	// Explicit opt-in with tunable rates.
	if err := os.WriteFile(filepath.Join(dir, "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  improve:\n    enabled: true\n    threshold: 10\n    max_usage: 100\n    boost_rate: \"0.2\"\n    decay_rate: \"0.5\"\n    cooldown: 30s\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := Load(dir)
	if !got.Improvement.Enabled {
		t.Fatalf("improve.enabled: true must opt in, got false")
	}
	if got.Improvement.Threshold != 10 || got.Improvement.MaxUsage != 100 {
		t.Fatalf("improve rates: got threshold=%d max_usage=%d, want 10/100", got.Improvement.Threshold, got.Improvement.MaxUsage)
	}
	if got.Improvement.BoostRate != 0.2 || got.Improvement.DecayRate != 0.5 {
		t.Fatalf("improve rates: got boost=%.2f decay=%.2f, want 0.2/0.5", got.Improvement.BoostRate, got.Improvement.DecayRate)
	}
	if got.Improvement.Cooldown != 30*time.Second {
		t.Fatalf("improve.cooldown: got %v, want 30s", got.Improvement.Cooldown)
	}
}
