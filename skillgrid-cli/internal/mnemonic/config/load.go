package config

import "time"

// WebCache holds cached web research settings from indexing.yaml.
type WebCache struct {
	Enabled       bool
	MaxEntryBytes int
	TTL           map[string]time.Duration
	Sources       []string
}

// Indexing holds code index settings from indexing.yaml mnemonic section.
type Indexing struct {
	Include      []string
	Exclude      []string
	ChunkLines   int
	ChunkOverlap int
	WebCache     WebCache
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

// DefaultIndexing returns defaults matching config.d/indexing.yaml (Task 10).
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
		WebCache:     DefaultWebCache(),
	}
}

// Load returns indexing settings for repoRoot.
// Full indexing.yaml parsing is deferred to Task 10.
func Load(_ string) Indexing {
	return DefaultIndexing()
}
