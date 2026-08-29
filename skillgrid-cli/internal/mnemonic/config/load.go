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

// Indexing holds code index settings from indexing.yaml mnemonic section.
type Indexing struct {
	Include      []string
	Exclude      []string
	ChunkLines   int
	ChunkOverlap int
	WebCache     WebCache
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
	WebCache     webCacheSection `yaml:"web_cache"`
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
		WebCache:     DefaultWebCache(),
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
	out.WebCache = mergeWebCache(defaults.WebCache, section.WebCache)
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
