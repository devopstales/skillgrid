package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex"
)

// TestIndexStatus covers 03.14 (Scenario: index status shows pending sync):
// `skillgrid index status` renders index health + the `### Pending sync:`
// section + the watcher state (enabled / disabled = manual index).
func TestIndexStatus(t *testing.T) {
	// A non-empty pending set + watcher enabled.
	var buf bytes.Buffer
	printIndexStatus(&buf, codeindex.Status{FileCount: 3, ChunkCount: 10, LastIndexed: "2026-09-10T00:00:00Z"},
		codeindex.WatchStatus{Enabled: true, Pending: []string{"main.go", "helper.ts"}})
	out := buf.String()
	if !strings.Contains(out, "files: 3, chunks: 10") {
		t.Errorf("expected file/chunk counts in the status; got:\n%s", out)
	}
	if !strings.Contains(out, "### Pending sync:") {
		t.Errorf("expected the `### Pending sync:` section; got:\n%s", out)
	}
	if !strings.Contains(out, "main.go") || !strings.Contains(out, "helper.ts") {
		t.Errorf("expected the pending files in the section; got:\n%s", out)
	}
	if !strings.Contains(out, "watcher: enabled") {
		t.Errorf("expected watcher: enabled; got:\n%s", out)
	}

	// Empty pending set → "(none)".
	buf.Reset()
	printIndexStatus(&buf, codeindex.Status{}, codeindex.WatchStatus{Enabled: true})
	if !strings.Contains(buf.String(), "(none)") {
		t.Errorf("expected (none) for an empty pending set; got:\n%s", buf.String())
	}

	// Watcher disabled (SKILLGRID_NO_WATCH=1) → manual-index note.
	t.Setenv(codeindex.NoWatchEnv, "1")
	buf.Reset()
	printIndexStatus(&buf, codeindex.Status{}, codeindex.WatchStatus{Enabled: false})
	if !strings.Contains(buf.String(), "watcher: disabled") {
		t.Errorf("expected watcher: disabled with %s=1; got:\n%s", codeindex.NoWatchEnv, buf.String())
	}

	// JSON shape.
	var jbuf bytes.Buffer
	printIndexStatusJSON(&jbuf, codeindex.Status{FileCount: 1}, codeindex.WatchStatus{Enabled: true, Pending: []string{"a.go"}})
	var decoded map[string]any
	if err := json.Unmarshal(jbuf.Bytes(), &decoded); err != nil {
		t.Fatalf("index status JSON did not parse: %v (%s)", err, jbuf.String())
	}
	if !strings.Contains(jbuf.String(), "pending_sync") || !strings.Contains(jbuf.String(), "watch_disabled") {
		t.Errorf("expected pending_sync + watch_disabled keys in JSON; got:\n%s", jbuf.String())
	}
}
