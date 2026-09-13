// Package config loads skillgrid's indexing/profile YAML configuration.
package config

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// WebCache holds cached web research settings from indexing.yaml.
type WebCache struct {
	Enabled       bool
	MaxEntryBytes int
	TTL           map[string]time.Duration
	Sources       []string
}

// MaxFileSizeDefault is the first-class size-skip threshold (bytes) for
// generated bundles / vendored blobs. Files larger are skipped (counted in
// stats), not an error, not a fallback.
const MaxFileSizeDefault = 500 * 1024

// EmbedderConfig is the mnemonic.embedder.* section of indexing.yaml. The
// provider selects the embedder (onnx default | external | off); indexing and
// query params are asymmetric (separate treatment of corpus vs. query).
type EmbedderConfig struct {
	Provider  string
	Dimension int
	Indexing  EmbedderParams
	Query     EmbedderParams
	// External / ollama-only.
	BaseURL string
	Model   string
	APIKey  string
	// Local-only: directory holding <Model>.onnx (default ~/.skillgrid/models/).
	ModelDir string
}

// EmbedderParams is one side (corpus or query) of the asymmetric embedder.
type EmbedderParams struct {
	Instructions string
	InputType    string
	MaxTokens    int
}

// RetrievalBudget is the tunable read budget (change 013, step 03): the
// item-count cap, the per-snippet char budget, and the context timeout. A
// zero field falls back to its default in memory.DefaultBudget.
type RetrievalBudget struct {
	Items     int
	Chars     int
	TimeoutNs int64
}

// DefaultMemoryTTL is the default soft expiry stamped on every saved
// observation when the caller does not supply one (014 step 04). Operators can
// override it via the mnemonic.ttl config key.
const DefaultMemoryTTL = 7 * 24 * time.Hour

// Extraction is the mnemonic.extraction section (014 step 05). The LLM
// passive-extraction pass is OPT-IN: LLM defaults to false so passive capture
// stays on the deterministic regex floor unless an operator enables it.
type Extraction struct {
	LLM bool
}

// Improvement is the mnemonic.improve section (014 step 08): the
// self-improvement feedback loop (retrieval-usage boost/decay re-ranking in
// mem_search). OPT-IN: Enabled defaults to false so the default search
// behavior is byte-identical to the pre-improve SQL ordering. The rates are
// tunable; zero fields fall back to the memory package defaults.
type Improvement struct {
	Enabled   bool
	Threshold int
	MaxUsage  int
	BoostRate float64
	DecayRate float64
	Cooldown  time.Duration
}

// Promotion is the mnemonic.promotion section (014 step 09): the quality
// threshold for the session-close graph promotion. Zero fields fall back to
// the memory package defaults (MinLength 100 runes, MinSections 1 heading).
// The promotion itself is always on when a summary meets the threshold; the
// config only tunes the bar.
type Promotion struct {
	MinLength   int
	MinSections int
}

// Importance is the mnemonic.importance section (014 step 13): the AKL
// importance scoring + recency decay. DecayRate is the per-day exponential
// decay (default 0.05/day — a 30-day-old observation with 100 retrievals
// scores ≈22.3); the TierThresholds are the maturity-tier age cutoffs in
// days (default 7/30/14). Zero fields fall back to the memory package
// defaults in SetImportance, so a malformed or absent section keeps the
// production behavior.
type Importance struct {
	DecayRate      float64
	TierThresholds TierThresholdsConfig
}

// Federated is the mnemonic.federated section (014 step 16): the merge
// weights for the federated cross-store query. The composite score is
// rank_weight*(1/(1+rank)) + importance_weight*(importance/max_importance);
// zero (or malformed/negative) weights fall back to the defaults below (0.5 /
// 0.5), so an absent section keeps the production balance.
type Federated struct {
	RankWeight       float64
	ImportanceWeight float64
}

// DefaultFederated is the 014 step 16 default: rank and importance carry
// equal weight in the federated composite score.
func DefaultFederated() Federated {
	return Federated{RankWeight: 0.5, ImportanceWeight: 0.5}
}

// TierThresholdsConfig is the mnemonic.importance.tier_thresholds section
// (014 step 13.4): the maturity-tier age cutoffs, in days. Zero fields fall
// back to the memory package defaults (MatureAgeDays 7, ArchivalAgeDays 30,
// UnusedArchivalDays 14).
type TierThresholdsConfig struct {
	MatureAgeDays    int
	ArchivalAgeDays  int
	UnusedArchivalDays int
}

// Indexing holds code index settings from indexing.yaml mnemonic section.
type Indexing struct {
	Include      []string
	Exclude      []string
	ChunkLines   int
	ChunkOverlap int
	MaxFileSize  int
	WebCache     WebCache
	Embedder     EmbedderConfig
	// RetrievalBudget is the tunable mem_* read budget (change 013, step 03).
	// Zero fields fall back to the memory package defaults.
	RetrievalBudget RetrievalBudget
	// TTL is the default observation soft expiry (mnemonic.ttl, 014 step 04).
	// Zero means "use DefaultMemoryTTL".
	TTL time.Duration
	// Extraction is the mnemonic.extraction section (014 step 05): LLM passive
	// extraction is opt-in (LLM defaults to false → regex floor).
	Extraction Extraction
	// Improvement is the mnemonic.improve section (014 step 08): the
	// self-improvement feedback loop. OPT-IN (Enabled defaults to false).
	Improvement Improvement
	// Promotion is the mnemonic.promotion section (014 step 09): the
	// session-close graph promotion quality threshold.
	Promotion Promotion
	// Importance is the mnemonic.importance section (014 step 13): the AKL
	// importance scoring decay rate + maturity-tier thresholds.
	Importance Importance
	// Federated is the mnemonic.federated section (014 step 16): the merge
	// weights for the federated cross-store query composite score.
	Federated Federated
	// SnapshotRetention is the mnemonic.snapshot.retention key (014 step
	// 20.3): the auto-prune retention for store snapshots (how many most
	// recent snapshots to keep after each capture). Zero means "use the
	// memory package default" (10).
	SnapshotRetention int
}

type indexingFile struct {
	Profile  string          `yaml:"profile"`
	Mnemonic mnemonicSection `yaml:"mnemonic"`
}

type mnemonicSection struct {
	Include      []string        `yaml:"include"`
	Exclude      []string        `yaml:"exclude"`
	ChunkLines   int             `yaml:"chunk_lines"`
	ChunkOverlap int             `yaml:"chunk_overlap"`
	MaxFileSize  int             `yaml:"max_file_size"`
	WebCache     webCacheSection `yaml:"web_cache"`
	Embedder     embedderSection `yaml:"embedder"`
	// RetrievalBudget is the mnemonic.retrieval_budget section (change 013,
	// step 03): item/char/timeout caps for every mem_* read path.
	RetrievalBudget retrievalBudgetSection `yaml:"retrieval_budget"`
	// TTL is the mnemonic.ttl key (014 step 04): a Go duration string, e.g.
	// "168h" or "7d" is not supported — use hours/seconds. Zero/empty keeps
	// the DefaultMemoryTTL fallback.
	TTL string `yaml:"ttl"`
	// Extraction is the mnemonic.extraction section (014 step 05): the LLM
	// passive-extraction opt-in switch.
	Extraction extractionSection `yaml:"extraction"`
	// Improvement is the mnemonic.improve section (014 step 08): the
	// self-improvement feedback loop opt-in + tunable rates.
	Improvement improvementSection `yaml:"improve"`
	// Promotion is the mnemonic.promotion section (014 step 09): the
	// session-close graph promotion quality threshold.
	Promotion promotionSection `yaml:"promotion"`
	// Importance is the mnemonic.importance section (014 step 13): the AKL
	// importance scoring decay rate + maturity-tier thresholds.
	Importance importanceSection `yaml:"importance"`
	// Federated is the mnemonic.federated section (014 step 16): the merge
	// weights for the federated cross-store query composite score.
	Federated federatedSection `yaml:"federated"`
	// Snapshot is the mnemonic.snapshot section (014 step 20.3): the auto-
	// prune retention for store snapshots (how many most recent snapshots to
	// keep after each capture).
	Snapshot snapshotSection `yaml:"snapshot"`
}

type retrievalBudgetSection struct {
	Items     int    `yaml:"items"`
	Chars     int    `yaml:"chars"`
	Timeout   string `yaml:"timeout"` // Go duration string, e.g. "3s"
}

// extractionSection is the mnemonic.extraction section (014 step 05). LLM
// defaults to false: passive capture stays on the deterministic regex floor
// unless an operator opts into the LLM pass.
type extractionSection struct {
	LLM bool `yaml:"llm"`
}

// improvementSection is the mnemonic.improve section (014 step 08). Enabled
// defaults to false: the self-improvement feedback loop is opt-in, so the
// default search behavior is byte-identical to the pre-improve SQL ordering.
type improvementSection struct {
	Enabled   bool   `yaml:"enabled"`
	Threshold int    `yaml:"threshold"`
	MaxUsage  int    `yaml:"max_usage"`
	BoostRate string `yaml:"boost_rate"`
	DecayRate string `yaml:"decay_rate"`
	Cooldown  string `yaml:"cooldown"` // Go duration string, e.g. "60s"
}

type embedderSection struct {
	Provider  string         `yaml:"provider"`
	Dimension int            `yaml:"dimension"`
	Indexing  embedderParams `yaml:"indexing_params"`
	Query     embedderParams `yaml:"query_params"`
	BaseURL   string         `yaml:"base_url"`
	Model     string         `yaml:"model"`
	APIKey    string         `yaml:"api_key"`
	ModelDir  string         `yaml:"model_dir"` // local provider only
}

type embedderParams struct {
	Instructions string `yaml:"instructions"`
	InputType    string `yaml:"input_type"`
	MaxTokens    int    `yaml:"max_tokens"`
}

type webCacheSection struct {
	Enabled       *bool             `yaml:"enabled"`
	MaxEntryBytes int               `yaml:"max_entry_bytes"`
	TTL           map[string]string `yaml:"ttl"`
	Sources       []string          `yaml:"sources"`
}

// DefaultWebCache returns TTL and size defaults matching config.d/indexing.yaml.
func DefaultWebCache() WebCache {
	return WebCache{
		Enabled:       true,
		MaxEntryBytes: 262144,
		TTL: map[string]time.Duration{
			"context7": 720 * time.Hour,
			"exa":      168 * time.Hour,
			"deepwiki": 336 * time.Hour,
			"fetch":    168 * time.Hour,
			"manual":   0,
		},
		Sources: []string{"context7", "exa", "deepwiki", "fetch", "manual"},
	}
}

// DefaultOnnxModel is the default ONNX embedder (nomic-embed-code, 768-dim).
const DefaultOnnxModel = "nomic-embed-code"

// DefaultOnnxDim is the default output dimension for the ONNX provider.
const DefaultOnnxDim = 768

// DefaultIndexing returns defaults matching config.d/indexing.yaml.
func DefaultIndexing() Indexing {
	return Indexing{
		Include: []string{
			"**/*.go",
			"**/*.ts",
			"**/*.tsx",
			"**/*.md",
		},
		Exclude: []string{
			"**/node_modules/**",
			"**/.git/**",
			"**/dist/**",
			"**/.skillgrid/**",
		},
		ChunkLines:   80,
		ChunkOverlap: 10,
		MaxFileSize:  MaxFileSizeDefault,
		WebCache:     DefaultWebCache(),
		Embedder:     DefaultEmbedder(),
		TTL:          DefaultMemoryTTL,
		Federated:    DefaultFederated(),
	}
}

// DefaultEmbedder returns the default embedder config: ONNX nomic-embed-code
// (768-dim), with separate indexing (corpus) and query param sets.
func DefaultEmbedder() EmbedderConfig {
	return EmbedderConfig{
		Provider:  "onnx",
		Dimension: DefaultOnnxDim,
		Indexing:  EmbedderParams{InputType: "passage"},
		Query:     EmbedderParams{InputType: "query"},
		Model:     DefaultOnnxModel,
	}
}

// Load returns indexing settings for startDir, walking up to find config.d/indexing.yaml.
func Load(startDir string) Indexing {
	defaults := DefaultIndexing()
	path, ok := findIndexingYAML(startDir)
	if !ok {
		return defaults
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return defaults
	}
	var file indexingFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return defaults
	}
	return mergeIndexing(defaults, file.Mnemonic)
}

func findIndexingYAML(startDir string) (string, bool) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", false
	}
	for {
		candidate := filepath.Join(dir, "config.d", "indexing.yaml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

func mergeIndexing(defaults Indexing, section mnemonicSection) Indexing {
	out := defaults
	if len(section.Include) > 0 {
		out.Include = append([]string(nil), section.Include...)
	}
	if len(section.Exclude) > 0 {
		out.Exclude = append([]string(nil), section.Exclude...)
	}
	if section.ChunkLines > 0 {
		out.ChunkLines = section.ChunkLines
	}
	if section.ChunkOverlap > 0 {
		out.ChunkOverlap = section.ChunkOverlap
	}
	if section.MaxFileSize > 0 {
		out.MaxFileSize = section.MaxFileSize
	}
	out.WebCache = mergeWebCache(defaults.WebCache, section.WebCache)
	out.Embedder = mergeEmbedder(defaults.Embedder, section.Embedder)
	out.RetrievalBudget = mergeRetrievalBudget(defaults.RetrievalBudget, section.RetrievalBudget)
	// TTL (014 step 04): a mnemonic.ttl duration overrides the default; a blank
	// or malformed value keeps the fallback so a bad key never disables expiry.
	if section.TTL != "" {
		if d, err := time.ParseDuration(section.TTL); err == nil && d > 0 {
			out.TTL = d
		}
	}
	// Extraction (014 step 05): the LLM opt-in switch. The section has no
	// non-zero default, so the YAML value applies as-is (absent → false).
	out.Extraction = Extraction{LLM: section.Extraction.LLM}
	// Improvement (014 step 08): the self-improvement feedback loop. OPT-IN —
	// Enabled defaults to false (absent → byte-identical search). Zero rate
	// fields fall back to the memory package defaults in SetImprove.
	out.Improvement = mergeImprovement(section.Improvement)
	// Promotion (014 step 09): the session-close graph promotion threshold.
	// Zero fields fall back to the memory package defaults in SetPromotion.
	out.Promotion = mergePromotion(section.Promotion)
	// Importance (014 step 13): the AKL importance scoring decay rate + tier
	// thresholds. Zero/malformed fields fall back to the memory package
	// defaults in SetImportance, so a malformed section never changes the
	// production scoring.
	out.Importance = mergeImportance(section.Importance)
	// Federated (014 step 16): the federated merge weights. Malformed or
	// non-positive values fall back to the 0.5/0.5 defaults, so a bad key
	// never skews the composite.
	out.Federated = mergeFederated(section.Federated)
	// Snapshot retention (014 step 20.3): a non-positive value is left zero so
	// SetSnapshotRetention applies the memory package default (10).
	out.SnapshotRetention = section.Snapshot.Retention
	return out
}

// mergeFederated maps the mnemonic.federated YAML section (014 step 16) to the
// Federated struct. Weights are float strings (YAML mirrors the importance
// section's pattern); absent or malformed values are left zero so the
// DefaultFederated fallback applies.
func mergeFederated(section federatedSection) Federated {
	out := DefaultFederated()
	if f, err := strconv.ParseFloat(section.RankWeight, 64); err == nil && f > 0 {
		out.RankWeight = f
	}
	if f, err := strconv.ParseFloat(section.ImportanceWeight, 64); err == nil && f > 0 {
		out.ImportanceWeight = f
	}
	return out
}

// mergeImportance maps the mnemonic.importance YAML section (014 step 13) to
// the Importance struct. The decay rate is a float string; absent or
// malformed values are left zero so SetImportance applies the 0.05/day
// default. Negative values are rejected the same way (the default applies).
func mergeImportance(section importanceSection) Importance {
	out := Importance{
		TierThresholds: TierThresholdsConfig{
			MatureAgeDays:    section.TierThresholds.MatureAgeDays,
			ArchivalAgeDays:  section.TierThresholds.ArchivalAgeDays,
			UnusedArchivalDays: section.TierThresholds.UnusedArchivalDays,
		},
	}
	if f, err := strconv.ParseFloat(section.DecayRate, 64); err == nil && f > 0 {
		out.DecayRate = f
	}
	return out
}

// mergeImprovement maps the mnemonic.improve YAML section to the Improvement
// struct. Enabled is applied as-is (absent → false = opt-in off). Rate fields
// are parsed as floats/durations; a missing or malformed value leaves the
// field zero so SetImprove applies its default.
func mergeImprovement(section improvementSection) Improvement {
	out := Improvement{Enabled: section.Enabled, Threshold: section.Threshold, MaxUsage: section.MaxUsage}
	if f, err := strconv.ParseFloat(section.BoostRate, 64); err == nil && f > 0 {
		out.BoostRate = f
	}
	if f, err := strconv.ParseFloat(section.DecayRate, 64); err == nil && f > 0 {
		out.DecayRate = f
	}
	if section.Cooldown != "" {
		if d, err := time.ParseDuration(section.Cooldown); err == nil && d > 0 {
			out.Cooldown = d
		}
	}
	return out
}

// promotionSection is the mnemonic.promotion section (014 step 09). Zero
// fields fall back to the memory package defaults in SetPromotion.
type promotionSection struct {
	MinLength   int `yaml:"min_length"`
	MinSections int `yaml:"min_sections"`
}

// importanceSection is the mnemonic.importance section (014 step 13). The
// decay rate is parsed as a float (absent/malformed → 0, so SetImportance
// applies the 0.05/day default); the tier thresholds are day counts (absent
// → 0, so SetImportance applies the 7/30/14-day defaults).
type importanceSection struct {
	DecayRate      string                 `yaml:"decay"`
	TierThresholds tierThresholdsSection  `yaml:"tier_thresholds"`
}

// federatedSection is the mnemonic.federated section (014 step 16). Weights
// are parsed as floats (absent/malformed → the 0.5/0.5 defaults).
type federatedSection struct {
	RankWeight       string `yaml:"rank_weight"`
	ImportanceWeight string `yaml:"importance_weight"`
}

// snapshotSection is the mnemonic.snapshot section (014 step 20.3): the auto-
// prune retention for store snapshots (Retention = how many most recent
// snapshots to keep after each capture). Zero means "use the memory package
// default" (10) — SetSnapshotRetention falls back to defaultSnapshotRetention
// when the value is non-positive.
type snapshotSection struct {
	Retention int `yaml:"retention"`
}

// tierThresholdsSection is the mnemonic.importance.tier_thresholds section
// (014 step 13.4): maturity-tier age cutoffs, in days.
type tierThresholdsSection struct {
	MatureAgeDays    int `yaml:"mature_age_days"`
	ArchivalAgeDays  int `yaml:"archival_age_days"`
	UnusedArchivalDays int `yaml:"unused_archival_days"`
}

// mergePromotion maps the mnemonic.promotion YAML section to the Promotion
// struct. Zero fields are left zero so SetPromotion applies its defaults.
func mergePromotion(section promotionSection) Promotion {
	return Promotion{MinLength: section.MinLength, MinSections: section.MinSections}
}

// mergeRetrievalBudget merges the mnemonic.retrieval_budget section over the
// defaults. Each field is applied independently when present (a partial budget
// only overrides the knobs it sets).
func mergeRetrievalBudget(defaults RetrievalBudget, section retrievalBudgetSection) RetrievalBudget {
	out := defaults
	if section.Items > 0 {
		out.Items = section.Items
	}
	if section.Chars > 0 {
		out.Chars = section.Chars
	}
	if section.Timeout != "" {
		if d, err := time.ParseDuration(section.Timeout); err == nil {
			out.TimeoutNs = int64(d)
		}
	}
	return out
}

func mergeEmbedder(defaults EmbedderConfig, section embedderSection) EmbedderConfig {
	out := defaults
	if section.Provider != "" {
		out.Provider = section.Provider
	}
	if section.Dimension > 0 {
		out.Dimension = section.Dimension
	}
	if section.BaseURL != "" {
		out.BaseURL = section.BaseURL
	}
	if section.Model != "" {
		out.Model = section.Model
	}
	if section.APIKey != "" {
		out.APIKey = section.APIKey
	}
	if section.ModelDir != "" {
		out.ModelDir = section.ModelDir
	}
	if section.Indexing.Instructions != "" || section.Indexing.InputType != "" || section.Indexing.MaxTokens > 0 {
		out.Indexing = EmbedderParams{
			Instructions: section.Indexing.Instructions,
			InputType:    section.Indexing.InputType,
			MaxTokens:    section.Indexing.MaxTokens,
		}
	}
	if section.Query.Instructions != "" || section.Query.InputType != "" || section.Query.MaxTokens > 0 {
		out.Query = EmbedderParams{
			Instructions: section.Query.Instructions,
			InputType:    section.Query.InputType,
			MaxTokens:    section.Query.MaxTokens,
		}
	}
	return out
}

func mergeWebCache(defaults WebCache, section webCacheSection) WebCache {
	out := defaults
	if section.Enabled != nil {
		out.Enabled = *section.Enabled
	}
	if section.MaxEntryBytes > 0 {
		out.MaxEntryBytes = section.MaxEntryBytes
	}
	if len(section.Sources) > 0 {
		out.Sources = append([]string(nil), section.Sources...)
	}
	if len(section.TTL) > 0 {
		out.TTL = make(map[string]time.Duration, len(section.TTL))
		for source, raw := range section.TTL {
			d, err := time.ParseDuration(raw)
			if err != nil {
				if fallback, ok := defaults.TTL[source]; ok {
					out.TTL[source] = fallback
				}
				continue
			}
			out.TTL[source] = d
		}
	}
	return out
}
