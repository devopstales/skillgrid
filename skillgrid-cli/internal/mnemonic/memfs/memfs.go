// Package memfs is the virtual filesystem over the mnemonic observation store
// (change 014, step 25). It exposes `mem ls`, `mem tree`, and `mem find` as
// scope-aware directory operations alongside the existing `mem search`.
//
// Scope URIs follow the shape `mem://project/{id}/` and `mem://user/{id}/`.
// The "directories" under a scope are organized by memory_type (e.g.
// `mem://project/A/preferences` = project A's preference-typed observations).
//
// Scope storage: observations are associated with a memfs scope via the
// topic_key column using the pattern `{kind}/{id}/{memory_type}/...`.
// This is consistent with step 19's directory retrieval and requires no
// schema migration.
package memfs

import (
	"database/sql"
	"errors"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
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

// Observation is the memfs-level view of a stored observation. It mirrors
// the fields relevant to scope-based directory listing.
type Observation struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	TopicKey   string `json:"topic_key,omitempty"`
	MemoryType string `json:"memory_type,omitempty"`
	CreatedAt  string `json:"created_at"`
}

// ErrEmptyScope is returned when a scope path is empty or malformed.
var ErrEmptyScope = errors.New("memfs: empty or malformed scope path")
