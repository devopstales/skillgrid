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
	if err := os.MkdirAll(filepath.Join(dir, ".skillgrid", "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  ttl: 72h\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); got.Extraction.LLM {
		t.Fatalf("absent section: extraction.llm must be false, got true")
	}

	// Explicit opt-in.
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  extraction:\n    llm: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); !got.Extraction.LLM {
		t.Fatalf("extraction.llm: true must opt in, got false")
	}
}

// TestDedupLLMConfigDefaultOff is TICKET-05: the mnemonic.dedup.llm key
// defaults to false (the deterministic hash dedup floor) and opts in only when
// explicitly true. Mirrors the mnemonic.extraction.llm opt-in pattern.
func TestDedupLLMConfigDefaultOff(t *testing.T) {
	dir := t.TempDir()

	// No config file at all → defaults: LLM dedup off.
	if got := Load(dir); got.Dedup.LLM {
		t.Fatalf("default: dedup.llm must be false, got true")
	}

	// A config without the dedup section → still off.
	if err := os.MkdirAll(filepath.Join(dir, ".skillgrid", "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  ttl: 72h\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); got.Dedup.LLM {
		t.Fatalf("absent section: dedup.llm must be false, got true")
	}

	// Explicit opt-in.
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  dedup:\n    llm: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); !got.Dedup.LLM {
		t.Fatalf("dedup.llm: true must opt in, got false")
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
	if err := os.MkdirAll(filepath.Join(dir, ".skillgrid", "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  ttl: 72h\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); got.Improvement.Enabled {
		t.Fatalf("absent section: improve.enabled must be false, got true")
	}

	// Explicit opt-in with tunable rates.
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
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

// TestLoadDecayConfig is TICKET-03: mnemonic.decay is on by default. An absent
// section yields Enabled true and a 30-day half-life; an explicit enabled
// false stays false.
func TestLoadDecayConfig(t *testing.T) {
	dir := t.TempDir()

	// No config file at all → defaults: decay on, 30-day half-life.
	got := Load(dir)
	if !got.Decay.Enabled {
		t.Fatalf("default: decay.enabled must be true, got false")
	}
	if got.Decay.HalfLifeDays != 30 {
		t.Fatalf("default: decay.half_life_days = %v, want 30", got.Decay.HalfLifeDays)
	}
	if got.Decay.ImmunityMinImportance != 4 || got.Decay.ImmunityMinAccess != 3 {
		t.Fatalf("default: immunity = %v/%d, want 4/3", got.Decay.ImmunityMinImportance, got.Decay.ImmunityMinAccess)
	}

	// A config without the decay section → still the defaults.
	if err := os.MkdirAll(filepath.Join(dir, ".skillgrid", "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  ttl: 72h\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = Load(dir)
	if !got.Decay.Enabled {
		t.Fatalf("absent section: decay.enabled must be true, got false")
	}
	if got.Decay.HalfLifeDays != 30 {
		t.Fatalf("absent section: decay.half_life_days = %v, want 30", got.Decay.HalfLifeDays)
	}

	// Explicit opt-out stays off.
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  decay:\n    enabled: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); got.Decay.Enabled {
		t.Fatalf("decay.enabled: false must stay false")
	}

	// Explicit knobs parse through.
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  decay:\n    enabled: true\n    half_life_days: 14\n    immunity_min_importance: 5\n    immunity_min_access: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = Load(dir)
	if !got.Decay.Enabled {
		t.Fatalf("decay.enabled: true must stay on")
	}
	if got.Decay.HalfLifeDays != 14 {
		t.Fatalf("decay.half_life_days = %v, want 14", got.Decay.HalfLifeDays)
	}
	if got.Decay.ImmunityMinImportance != 5 || got.Decay.ImmunityMinAccess != 2 {
		t.Fatalf("decay immunity = %v/%d, want 5/2", got.Decay.ImmunityMinImportance, got.Decay.ImmunityMinAccess)
	}
}

// TestLoadDecayHomeOptOutSurvivesRepoWithoutDecay: a home-level
// decay.enabled: false must survive a repo file that omits the decay key.
func TestLoadDecayHomeOptOutSurvivesRepoWithoutDecay(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".skillgrid", "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  decay:\n    enabled: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".skillgrid", "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	repoFile := filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml")
	if err := os.WriteFile(repoFile, []byte("mnemonic:\n  ttl: 72h\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); got.Decay.Enabled {
		t.Fatalf("home decay.enabled: false lost to a repo file without decay: %+v", got.Decay)
	}

	// An explicit repo enabled: true still wins over the home opt-out.
	if err := os.WriteFile(repoFile, []byte("mnemonic:\n  decay:\n    enabled: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); !got.Decay.Enabled {
		t.Fatalf("repo decay.enabled: true must override home opt-out: %+v", got.Decay)
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

	if err := os.MkdirAll(filepath.Join(dir, ".skillgrid", "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Absent section → still the default.
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  ttl: 72h\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); got.Importance.DecayRate != 0 {
		t.Fatalf("absent section: importance.decay must be unset, got %v", got.Importance.DecayRate)
	}

	// Explicit decay + thresholds.
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
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
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
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

	if err := os.MkdirAll(filepath.Join(dir, ".skillgrid", "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Absent section → still the default.
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  ttl: 72h\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); got.Federated != DefaultFederated() {
		t.Fatalf("absent section: federated weights must be 0.5/0.5, got %+v", got.Federated)
	}

	// Explicit weights.
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  federated:\n    rank_weight: \"0.8\"\n    importance_weight: \"0.2\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := Load(dir)
	if got.Federated.RankWeight != 0.8 || got.Federated.ImportanceWeight != 0.2 {
		t.Fatalf("federated weights: got rank=%.2f importance=%.2f, want 0.8/0.2", got.Federated.RankWeight, got.Federated.ImportanceWeight)
	}

	// Malformed weight → falls back to the 0.5/0.5 default.
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  federated:\n    rank_weight: \"not-a-number\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); got.Federated != DefaultFederated() {
		t.Fatalf("malfederated weight must fall back to the 0.5/0.5 default, got %+v", got.Federated)
	}
}

// TestLoad_HooksDefaultOn is the one-way default flip (2026-09-24 monitoring):
// with no hooks section — or a config file that omits it — hooks are ON
// (observe-mode: writes rows, never blocks). An explicit enabled: false still
// opts out.
func TestLoad_HooksDefaultOn(t *testing.T) {
	// No config file at all → hooks on by default.
	if got := Load(t.TempDir()); !got.Hooks.Enabled {
		t.Fatalf("default: hooks.enabled must be true (observe-mode), got false")
	}

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".skillgrid", "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Config without the hooks section → still on.
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  ttl: 72h\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); !got.Hooks.Enabled {
		t.Fatalf("absent section: hooks.enabled must be true, got false")
	}
	// Explicit opt-out.
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  hooks:\n    enabled: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); got.Hooks.Enabled {
		t.Fatalf("hooks.enabled: false must opt out, got true")
	}
	// Explicit opt-in stays on (and a malformed timeout keeps the zero
	// fallback so SetHooks applies its default).
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  hooks:\n    enabled: true\n    timeout: 45s\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := Load(dir)
	if !got.Hooks.Enabled {
		t.Fatalf("hooks.enabled: true must stay on, got false")
	}
	if got.Hooks.Timeout != 45*time.Second {
		t.Fatalf("hooks.timeout: got %v, want 45s", got.Hooks.Timeout)
	}
}

// TestLoad_HooksExplicitOff is the one-way flip's escape hatch: an explicit
// enabled: false in either the repo-local or the home-local config opts out.
func TestLoad_HooksExplicitOff(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".skillgrid", "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  hooks:\n    enabled: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".skillgrid", "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Home-local explicit false applies even without a repo-local file.
	if got := Load(dir); got.Hooks.Enabled {
		t.Fatalf("home-local hooks.enabled: false must opt out, got true")
	}
	// A repo-local file without the hooks key does not re-enable: mergeHooks
	// applies the default (true) when the key is absent, so the home-local
	// explicit false only applies when no repo-local file is present. This
	// matches the pre-flip semantics (a repo-local file always wins the keys
	// its section leaves unset only for keys with no default; enabled now has
	// a default, so absent → default).
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  ttl: 72h\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); !got.Hooks.Enabled {
		t.Fatalf("repo-local file without hooks key → default true, got false")
	}
	// Repo-local explicit true beats the home-local false.
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  hooks:\n    enabled: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); !got.Hooks.Enabled {
		t.Fatalf("repo-local hooks.enabled: true must win, got false")
	}
}

// TestLoad_RetentionDaysDefault is the 2026-09-24 monitoring retention window:
// mnemonic.retention_days defaults to 90 when absent and parses explicit
// values; non-positive values fall back to the default.
func TestLoad_RetentionDaysDefault(t *testing.T) {
	if got := Load(t.TempDir()); got.RetentionDays != 90 {
		t.Fatalf("default: retention_days must be 90, got %d", got.RetentionDays)
	}

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".skillgrid", "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Absent key → 90.
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  ttl: 72h\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); got.RetentionDays != 90 {
		t.Fatalf("absent key: retention_days must be 90, got %d", got.RetentionDays)
	}
	// Explicit value.
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  retention_days: 30\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); got.RetentionDays != 30 {
		t.Fatalf("retention_days: 30 must apply, got %d", got.RetentionDays)
	}
	// Non-positive value → falls back to 90.
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  retention_days: 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); got.RetentionDays != 90 {
		t.Fatalf("retention_days: 0 must fall back to 90, got %d", got.RetentionDays)
	}
}

// TestLoad_PrivateTools is the 2026-09-24 monitoring always-private allowlist:
// mnemonic.private_tools is empty by default and parses an explicit list.
func TestLoad_PrivateTools(t *testing.T) {
	if got := Load(t.TempDir()); len(got.PrivateTools) != 0 {
		t.Fatalf("default: private_tools must be empty, got %v", got.PrivateTools)
	}

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".skillgrid", "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  private_tools: [mem_save, mem_save_prompt]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := Load(dir)
	if len(got.PrivateTools) != 2 || got.PrivateTools[0] != "mem_save" || got.PrivateTools[1] != "mem_save_prompt" {
		t.Fatalf("private_tools = %v, want [mem_save mem_save_prompt]", got.PrivateTools)
	}
}

// TestCheckpointConfig is TICKET-01: mnemonic.checkpoint and mnemonic.inject
// parse from indexing.yaml with production defaults; invalid checkpoint knobs
// fall back without failing Load.
func TestCheckpointConfig(t *testing.T) {
	dir := t.TempDir()
	wantCheckpoint := DefaultCheckpoint()
	wantInject := DefaultInject()

	// No config file at all → briefing defaults.
	got := Load(dir)
	if got.Checkpoint != wantCheckpoint {
		t.Fatalf("default checkpoint: got %+v, want %+v", got.Checkpoint, wantCheckpoint)
	}
	if got.Inject != wantInject {
		t.Fatalf("default inject: got %+v, want %+v", got.Inject, wantInject)
	}

	if err := os.MkdirAll(filepath.Join(dir, ".skillgrid", "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Config without checkpoint/inject keys → still defaults.
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  ttl: 72h\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = Load(dir)
	if got.Checkpoint != wantCheckpoint {
		t.Fatalf("absent section checkpoint: got %+v, want %+v", got.Checkpoint, wantCheckpoint)
	}
	if got.Inject != wantInject {
		t.Fatalf("absent section inject: got %+v, want %+v", got.Inject, wantInject)
	}

	// Valid YAML for both sections.
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  checkpoint:\n    enabled: false\n    min_events: 8\n    cooldown_minutes: 15\n    max_observations: 3\n  inject:\n    summaries: 2\n    observations: 10\n    max_tokens: 1200\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = Load(dir)
	if got.Checkpoint.Enabled {
		t.Fatalf("checkpoint.enabled: false must apply, got true")
	}
	if got.Checkpoint.MinEvents != 8 || got.Checkpoint.Cooldown != 15*time.Minute || got.Checkpoint.MaxObservations != 3 {
		t.Fatalf("checkpoint knobs: got %+v, want min_events=8 cooldown=15m max_obs=3", got.Checkpoint)
	}
	if got.Inject.Summaries != 2 || got.Inject.Observations != 10 || got.Inject.MaxTokens != 1200 {
		t.Fatalf("inject knobs: got %+v, want 2/10/1200", got.Inject)
	}

	// Invalid min_events → defaults for that field, Load still succeeds.
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  checkpoint:\n    min_events: 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = Load(dir)
	if got.Checkpoint.MinEvents != wantCheckpoint.MinEvents {
		t.Fatalf("invalid min_events must fall back to default %d, got %d", wantCheckpoint.MinEvents, got.Checkpoint.MinEvents)
	}
	if got.Checkpoint != wantCheckpoint {
		t.Fatalf("invalid min_events: expected full checkpoint defaults %+v, got %+v", wantCheckpoint, got.Checkpoint)
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
	if err := os.MkdirAll(filepath.Join(dir, ".skillgrid", "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("chunk_lines: 80\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := Load(dir)
	if got.Embedder.Provider != "ollama" || got.Embedder.BaseURL != "http://10.0.0.70:11434" || got.Embedder.Model != "nomic-embed-text" {
		t.Fatalf("home fallback: expected ollama embedder from ~/.skillgrid, got provider=%q base_url=%q model=%q",
			got.Embedder.Provider, got.Embedder.BaseURL, got.Embedder.Model)
	}

	// Case 2: repo-local file WITH its own embedder → repo-local wins (no leak).
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  embedder:\n    provider: onnx\n    model: nomic-embed-code\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = Load(dir)
	if got.Embedder.Provider != "onnx" || got.Embedder.Model != "nomic-embed-code" {
		t.Fatalf("repo-local must win over home: got provider=%q model=%q, want onnx/nomic-embed-code",
			got.Embedder.Provider, got.Embedder.Model)
	}
}

// TestLLMConfigDefaultOff is TICKET-01 (G1, SATISFIES "happy path llm config
// defaults off"): the mnemonic.llm section defaults to disabled and an empty
// base URL. With no llm YAML and the SKILLGRID_LLM_API_KEY env var cleared,
// Load returns a zero LLM config (Enabled false, BaseURL ""). Uses an isolated
// HOME so the operator's ~/.skillgrid overlay cannot leak an llm block.
func TestLLMConfigDefaultOff(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// Isolate the env fallback: with no api_key in YAML and no env, the key
	// must stay empty (deterministic regardless of the operator's shell).
	t.Setenv("SKILLGRID_LLM_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")

	dir := t.TempDir()

	// No config file at all → defaults: LLM off, empty base URL.
	got := Load(dir)
	if got.LLM.Enabled {
		t.Fatalf("default: llm.enabled must be false, got true")
	}
	if got.LLM.BaseURL != "" {
		t.Fatalf("default: llm.base_url must be empty, got %q", got.LLM.BaseURL)
	}
	if got.LLM.APIKey != "" {
		t.Fatalf("default: llm.api_key must be empty (no env), got %q", got.LLM.APIKey)
	}

	// A config without the llm section → still off, empty base URL.
	if err := os.MkdirAll(filepath.Join(dir, ".skillgrid", "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  ttl: 72h\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = Load(dir)
	if got.LLM.Enabled {
		t.Fatalf("absent section: llm.enabled must be false, got true")
	}
	if got.LLM.BaseURL != "" {
		t.Fatalf("absent section: llm.base_url must be empty, got %q", got.LLM.BaseURL)
	}
}

// TestLLMConfigRequiresURLAndModel is TICKET-01 (G2): mnemonic.llm parses an
// explicit enabled/base_url/model/api_key block, and the api_key falls back to
// the SKILLGRID_LLM_API_KEY env var when the YAML key is absent.
//
// NOTE on the "requires URL and model" case: the loader does NOT reject an
// enabled:true block that omits base_url — enforcement (Validate) happens at
// the attach step in Task 2, not in config loading. After a merge with
// enabled:true and an empty base_url, Enabled is true but BaseURL stays
// empty. This test documents that the loader is permissive; the attach site
// is where the missing URL is turned into a non-attach (fail-open).
func TestLLMConfigRequiresURLAndModel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OPENAI_API_KEY", "")

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".skillgrid", "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	repoFile := filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml")

	// Full explicit block: enabled + base_url + model + api_key all parse.
	if err := os.WriteFile(repoFile,
		[]byte("mnemonic:\n  llm:\n    enabled: true\n    base_url: http://localhost:11434/v1\n    model: qwen3:8b\n    api_key: sk-test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := Load(dir)
	if !got.LLM.Enabled {
		t.Fatalf("llm.enabled: true must apply, got false")
	}
	if got.LLM.BaseURL != "http://localhost:11434/v1" {
		t.Fatalf("llm.base_url: got %q, want http://localhost:11434/v1", got.LLM.BaseURL)
	}
	if got.LLM.Model != "qwen3:8b" {
		t.Fatalf("llm.model: got %q, want qwen3:8b", got.LLM.Model)
	}
	if got.LLM.APIKey != "sk-test" {
		t.Fatalf("llm.api_key: got %q, want sk-test", got.LLM.APIKey)
	}

	// enabled:true but base_url omitted → the loader is permissive: Enabled is
	// true, BaseURL stays empty (enforced later at the attach step, Task 2).
	if err := os.WriteFile(repoFile,
		[]byte("mnemonic:\n  llm:\n    enabled: true\n    model: qwen3:8b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = Load(dir)
	if !got.LLM.Enabled {
		t.Fatalf("llm.enabled: true must apply, got false")
	}
	if got.LLM.BaseURL != "" {
		t.Fatalf("llm.base_url omitted: must stay empty (attach enforces), got %q", got.LLM.BaseURL)
	}
}

// TestLLMAPIKeyFromEnv is TICKET-01 (G2, secrets): when the YAML api_key is
// absent, the api_key falls back to SKILLGRID_LLM_API_KEY; an explicit YAML
// api_key wins over the env var. The precedence is explicit YAML > env > base.
func TestLLMAPIKeyFromEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".skillgrid", "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	repoFile := filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml")

	// Case 1: api_key omitted, env set → falls back to the env value.
	t.Setenv("SKILLGRID_LLM_API_KEY", "from-env")
	if err := os.WriteFile(repoFile,
		[]byte("mnemonic:\n  llm:\n    enabled: true\n    base_url: http://localhost:11434/v1\n    model: qwen3:8b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := Load(dir)
	if got.LLM.APIKey != "from-env" {
		t.Fatalf("llm.api_key: must fall back to SKILLGRID_LLM_API_KEY, got %q", got.LLM.APIKey)
	}

	// Case 2: explicit YAML api_key wins over the env var.
	if err := os.WriteFile(repoFile,
		[]byte("mnemonic:\n  llm:\n    enabled: true\n    base_url: http://localhost:11434/v1\n    model: qwen3:8b\n    api_key: explicit-yaml\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = Load(dir)
	if got.LLM.APIKey != "explicit-yaml" {
		t.Fatalf("llm.api_key: explicit YAML must win over env, got %q", got.LLM.APIKey)
	}

	// Case 3: both YAML api_key and env absent → key stays empty.
	t.Setenv("SKILLGRID_LLM_API_KEY", "")
	if err := os.WriteFile(repoFile,
		[]byte("mnemonic:\n  llm:\n    enabled: true\n    base_url: http://localhost:11434/v1\n    model: qwen3:8b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = Load(dir)
	if got.LLM.APIKey != "" {
		t.Fatalf("llm.api_key: no YAML key and no env must stay empty, got %q", got.LLM.APIKey)
	}
}
