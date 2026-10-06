// Package memfs is a virtual filesystem over the project code index
// (files/symbols/chunks). It exposes `mem fs ls`, `mem fs tree`, `mem fs find`,
// and `mem fs cat` as path-aware directory operations, modeled on the
// OpenViking viking:// browse paradigm: the agent walks the indexed repo tree
// instead of querying a black-box store.
//
// Code paths are repo-relative, rooted at the project:
//
//	memfs://project/{id}/                → repo root
//	memfs://project/{id}/src/            → a directory
//	memfs://project/{id}/src/a/b.go      → a file
//	memfs://project/{id}/src/a/b.go::Fn  → a symbol in a file
//
// The observation store is unchanged and remains reachable via `mem search`
// and the MCP mem_* tools; memfs no longer fronts it.
package memfs

import (
	"database/sql"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// MemFS is the virtual filesystem handle bound to one project store.
type MemFS struct {
	db        *sql.DB
	projectID string
}

// New creates a MemFS over the given store for the given project.
func New(st *store.Store, projectID string) *MemFS {
	if st == nil || st.DB == nil {
		return nil
	}
	return &MemFS{db: st.DB, projectID: projectID}
}

// DB returns the underlying *sql.DB.
func (fs *MemFS) DB() *sql.DB {
	if fs == nil {
		return nil
	}
	return fs.db
}
