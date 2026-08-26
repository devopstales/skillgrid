package config

// Indexing holds code index settings from indexing.yaml mnemonic section.
type Indexing struct {
	Include      []string
	Exclude      []string
	ChunkLines   int
	ChunkOverlap int
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
	}
}

// Load returns indexing settings for repoRoot.
// Full indexing.yaml parsing is deferred to Task 10.
func Load(_ string) Indexing {
	return DefaultIndexing()
}
