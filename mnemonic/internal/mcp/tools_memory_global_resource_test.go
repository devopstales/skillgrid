// memory/global MCP resource (change 2026-10-24-global-llm-wiki, TICKET-05).
//
// RED notes (the failures that prove each test exercises new code):
//   - TestMemoryGlobalResourceRegistered / TestMemoryGlobalResourceReads
//     Snapshot: compile error — registerMemoryGlobalResource does not exist
//     yet; NewServer() lists no resources.
package mcp

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	mcplib "github.com/mark3labs/mcp-go/mcp"

	"github.com/devopstales/skillgrid/mnemonic/internal/facts"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

// writeFile is a tiny os.WriteFile wrapper (the test seam keeps os out of
// the read paths above).
func writeFile(p, content string) error { return os.WriteFile(p, []byte(content), 0o644) }

// readResourceReq builds a ReadResourceRequest for the resource test seam.
func readResourceReq(uri string) mcplib.ReadResourceRequest {
	var req mcplib.ReadResourceRequest
	req.Params.URI = uri
	return req
}

// TestMemoryGlobalResourceRegistered: NewServer() exposes the memory/global
// resource.
func TestMemoryGlobalResourceRegistered(t *testing.T) {
	s := NewServer()
	resources := s.ListResources()
	if len(resources) == 0 {
		t.Fatal("no resources registered")
	}
	if _, ok := resources[memoryGlobalResourceURI]; !ok {
		t.Errorf("resource %q not registered; got %v", memoryGlobalResourceURI, resources)
	}
}

// TestMemoryGlobalResourceReadsSnapshot: a read of memory/global returns the
// rendered snapshot (both section headers + the seeded fact) for the global
// store; an unreadable store yields empty text, not an error (fail-open).
func TestMemoryGlobalResourceReadsSnapshot(t *testing.T) {
	dataDir := t.TempDir()
	SetService(service.New(dataDir))
	t.Cleanup(func() { SetService(nil) })

	// Seed one live fact directly into the global store (bypassing the
	// session trail: the resource is a read path).
	svc := service.New(dataDir)
	gh, cleanup, err := svc.OpenGlobal()
	if err != nil {
		t.Fatalf("OpenGlobal: %v", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := gh.Store().DB.Exec(
		`INSERT INTO facts (content, category, created_at, updated_at)
		 VALUES (?, ?, ?, ?)`,
		"prefetch resource probe fact", facts.CategoryMemory, now, now,
	); err != nil {
		t.Fatalf("seed fact: %v", err)
	}
	cleanup()

	s := NewServer()
	resources := s.ListResources()
	r, ok := resources[memoryGlobalResourceURI]
	if !ok {
		t.Fatalf("resource %q missing", memoryGlobalResourceURI)
	}
	contents, err := r.Handler(context.Background(), readResourceReq(memoryGlobalResourceURI))
	if err != nil {
		t.Fatalf("resource read: %v", err)
	}
	if len(contents) != 1 {
		t.Fatalf("resource returned %d contents, want 1", len(contents))
	}
	text, ok := contents[0].(mcplib.TextResourceContents)
	if !ok {
		t.Fatalf("resource content type %T, want text contents", contents[0])
	}
	if !strings.Contains(text.Text, "GLOBAL MEMORY") || !strings.Contains(text.Text, "USER PROFILE") {
		t.Errorf("resource text missing a section header:\n%s", text.Text)
	}
	if !strings.Contains(text.Text, "prefetch resource probe fact") {
		t.Errorf("resource text missing the seeded fact:\n%s", text.Text)
	}
}

// TestMemoryGlobalResourceFailOpen: with a service whose data directory
// cannot be created, the handler must return empty text, not a protocol
// error — the injection never breaks the client.
func TestMemoryGlobalResourceFailOpen(t *testing.T) {
	// A data dir pointing at a path under an existing FILE: OpenGlobal
	// cannot create a store there, so every read fails open to "".
	blocker := t.TempDir()
	blockerFile := blocker + "/not-a-dir"
	if err := writeFile(blockerFile, "x"); err != nil {
		t.Fatalf("write blocker file: %v", err)
	}
	SetService(service.New(blockerFile))
	t.Cleanup(func() { SetService(nil) })

	s := NewServer()
	r, ok := s.ListResources()[memoryGlobalResourceURI]
	if !ok {
		t.Fatalf("resource %q missing", memoryGlobalResourceURI)
	}
	contents, err := r.Handler(context.Background(), readResourceReq(memoryGlobalResourceURI))
	if err != nil {
		t.Fatalf("fail-open resource read must not error, got: %v", err)
	}
	if len(contents) != 1 {
		t.Fatalf("fail-open resource returned %d contents, want 1", len(contents))
	}
	text, ok := contents[0].(mcplib.TextResourceContents)
	if !ok {
		t.Fatalf("resource content type %T, want text contents", contents[0])
	}
	if text.Text != "" {
		t.Errorf("fail-open text should be empty, got:\n%s", text.Text)
	}
}
