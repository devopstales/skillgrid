// Package config loads skillgrid's indexing/profile YAML configuration.
package config

import (
	"os"
	"path/filepath"
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
	// External-only.
	BaseURL string
	Model   string
	APIKey  string
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

type embedderSection struct {
	Provider  string         `yaml:"provider"`
	Dimension int            `yaml:"dimension"`
	Indexing  embedderParams `yaml:"indexing_params"`
	Query     embedderParams `yaml:"query_params"`
	BaseURL   string         `yaml:"base_url"`
	Model     string         `yaml:"model"`
	APIKey    string         `yaml:"api_key"`
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
	return out
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
