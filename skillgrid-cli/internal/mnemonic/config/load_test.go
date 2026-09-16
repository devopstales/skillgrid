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

// TestImportanceConfigurableDecay covers 014 step 13.4 (config side): the
// mnemonic.importance.decay key defaults to 0.05/day, parses from the YAML
// section, falls back to the default on a malformed value, and the
// mnemonic.importance.tier_thresholds key tunes the tier cutoffs.
func TestImportanceConfigurableDecay(t *testing.T) {
	dir := t.TempDir()

	// No config file at all → default decay 0.05/day + 7/30/14-day
	// thresholds (the documented production defaults).
	got := Load(dir)
	if got.Importance.DecayRate != 0 {
		t.Fatalf("default: importance.decay must be unset (0 → memory default), got %v", got.Importance.DecayRate)
	}

	if err := os.MkdirAll(filepath.Join(dir, "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Absent section → still the default.
	if err := os.WriteFile(filepath.Join(dir, "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  ttl: 72h\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); got.Importance.DecayRate != 0 {
		t.Fatalf("absent section: importance.decay must be unset, got %v", got.Importance.DecayRate)
	}

	// Explicit decay + thresholds.
	if err := os.WriteFile(filepath.Join(dir, "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  importance:\n    decay: \"0.2\"\n    tier_thresholds:\n      mature_age_days: 3\n      archival_age_days: 21\n      unused_archival_days: 10\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = Load(dir)
	if got.Importance.DecayRate != 0.2 {
		t.Fatalf("importance.decay: got %v, want 0.2", got.Importance.DecayRate)
	}
	if got.Importance.TierThresholds.MatureAgeDays != 3 ||
		got.Importance.TierThresholds.ArchivalAgeDays != 21 ||
		got.Importance.TierThresholds.UnusedArchivalDays != 10 {
		t.Fatalf("importance.tier_thresholds: got %+v, want 3/21/10", got.Importance.TierThresholds)
	}

	// Malformed decay → falls back to the default (unset → memory 0.05).
	if err := os.WriteFile(filepath.Join(dir, "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  importance:\n    decay: \"not-a-number\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); got.Importance.DecayRate != 0 {
		t.Fatalf("malformed importance.decay must fall back to the default, got %v", got.Importance.DecayRate)
	}
}

// TestFederatedConfigWeights covers 014 step 16 (config side): the
// mnemonic.federated section defaults to the 0.5/0.5 balance, parses explicit
// weights from YAML, and falls back to the defaults on a malformed value.
func TestFederatedConfigWeights(t *testing.T) {
	dir := t.TempDir()

	// No config file at all → the 0.5/0.5 production default.
	if got := Load(dir); got.Federated != DefaultFederated() {
		t.Fatalf("default: federated weights must be 0.5/0.5, got %+v", got.Federated)
	}

	if err := os.MkdirAll(filepath.Join(dir, "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Absent section → still the default.
	if err := os.WriteFile(filepath.Join(dir, "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  ttl: 72h\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); got.Federated != DefaultFederated() {
		t.Fatalf("absent section: federated weights must be 0.5/0.5, got %+v", got.Federated)
	}

	// Explicit weights.
	if err := os.WriteFile(filepath.Join(dir, "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  federated:\n    rank_weight: \"0.8\"\n    importance_weight: \"0.2\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := Load(dir)
	if got.Federated.RankWeight != 0.8 || got.Federated.ImportanceWeight != 0.2 {
		t.Fatalf("federated weights: got rank=%.2f importance=%.2f, want 0.8/0.2", got.Federated.RankWeight, got.Federated.ImportanceWeight)
	}

	// Malformed weight → falls back to the 0.5/0.5 default.
	if err := os.WriteFile(filepath.Join(dir, "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  federated:\n    rank_weight: \"not-a-number\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); got.Federated != DefaultFederated() {
		t.Fatalf("malfederated weight must fall back to the 0.5/0.5 default, got %+v", got.Federated)
	}
}

// TestHomeLocalConfigFallback covers the ~/.skillgrid/config.d/indexing.yaml
// per-key fallback: a machine-local embedder block applies when the repo-local
// file leaves the embedder unset, and a repo-local embedder block wins when both
// are set. Uses an isolated HOME so it does not touch the operator's real home.
func TestHomeLocalConfigFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".skillgrid", "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	homeCfg := filepath.Join(home, ".skillgrid", "config.d", "indexing.yaml")

	// Write a machine-local Ollama embedder into the home config.
	homeYAML := "mnemonic:\n  embedder:\n    provider: ollama\n    base_url: http://10.0.0.70:11434\n    model: nomic-embed-text\n    dimension: 768\n"
	if err := os.WriteFile(homeCfg, []byte(homeYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	// Case 1: repo-local file present WITHOUT an embedder → home fallback applies.
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.d", "indexing.yaml"),
		[]byte("chunk_lines: 80\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := Load(dir)
	if got.Embedder.Provider != "ollama" || got.Embedder.BaseURL != "http://10.0.0.70:11434" || got.Embedder.Model != "nomic-embed-text" {
		t.Fatalf("home fallback: expected ollama embedder from ~/.skillgrid, got provider=%q base_url=%q model=%q",
			got.Embedder.Provider, got.Embedder.BaseURL, got.Embedder.Model)
	}

	// Case 2: repo-local file WITH its own embedder → repo-local wins (no leak).
	if err := os.WriteFile(filepath.Join(dir, "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  embedder:\n    provider: onnx\n    model: nomic-embed-code\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = Load(dir)
	if got.Embedder.Provider != "onnx" || got.Embedder.Model != "nomic-embed-code" {
		t.Fatalf("repo-local must win over home: got provider=%q model=%q, want onnx/nomic-embed-code",
			got.Embedder.Provider, got.Embedder.Model)
	}
}
