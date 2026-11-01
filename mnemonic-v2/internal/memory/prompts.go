package memory

import (
	"context"
	"fmt"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// prompts.go — the captured-prompt write (spec 004-transport-mcp mem_save
// capture, spec 002-schema §2.4). The save path captures the current turn's
// prompt through the gate's transactional insert (same txn as the
// observation); SavePrompt is the session-side surface (Task 7 builds its
// surface around this).

// promptSeam is the store's prompt-write seam (the SQLite store's
// test-only InsertPrompt — Task 7's prompts surface switches the save
// path to the gate's transactional prompt insert, which also carries
// source_inbox_id).
type promptSeam interface {
	InsertPrompt(ctx context.Context, sessionID, content, project string) (int64, error)
}

// SavePrompt persists the current turn's prompt (redacted + truncated like
// observation content, sync_id prefix prm-) for sessionID in project.
// Returns the prompt's local id (0 when nothing was stored).
func (s *Service) SavePrompt(ctx context.Context, sessionID, content, project, sourceInboxID string) (int64, error) {
	if s == nil || s.st == nil {
		return 0, errNoStore
	}
	prompt := s.redactPrompt(content)
	if prompt == "" {
		return 0, nil // an empty capture stores nothing
	}
	// ponytail: the store's InsertPrompt seam mints its own prm- sync id and
	// does not take source_inbox_id; the save path's prompt lands through
	// the gate's transactional insert instead (prompts_test's assertions
	// run via the gate). Ceiling: Task 7's prompts surface switches this
	// seam to the gate's path (which also sets source_inbox_id).
	_ = sourceInboxID
	backend, ok := s.st.(promptSeam)
	if !ok {
		return 0, fmt.Errorf("memory: store %T does not implement the prompt seam", s.st)
	}
	return backend.InsertPrompt(ctx, sessionID, prompt, project)
}

var _ promptSeam = nil
var _ store.Store

var errNoStore = errNoStoreValue{}

type errNoStoreValue struct{}

func (errNoStoreValue) Error() string { return "memory: no store configured" }
