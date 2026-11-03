package codeindex

import (
	"context"
	"slices"
	"testing"
)

func runWithProgress(t *testing.T, cfg Config) []Event {
	t.Helper()
	var events []Event
	cfg.Progress = func(e Event) { events = append(events, e) }
	return events
}

func phasesOf(events []Event) []string {
	var out []string
	for _, e := range events {
		if e.PhaseStart {
			out = append(out, e.Phase)
		}
	}
	return out
}

func extractEvents(events []Event) []Event {
	var out []Event
	for _, e := range events {
		if e.Phase == "extract" {
			out = append(out, e)
		}
	}
	return out
}

// TestProgressNilSafe: a run with Progress unset (the default) must work
// exactly as before — no panic, normal stats.
func TestProgressNilSafe(t *testing.T) {
	idx, clean := newTestIndexer(t)
	defer clean()
	root := writeTestRepo(t)
	stats, err := idx.Run(context.Background(), root, testCfg)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if stats.FilesIndexed != 2 {
		t.Errorf("expected 2 files indexed, got %d", stats.FilesIndexed)
	}
}

// TestProgressScanAndExtract: the scan event reports the scanned file count,
// and the extract phase reports per-file done/total.
func TestProgressScanAndExtract(t *testing.T) {
	idx, clean := newTestIndexer(t)
	defer clean()
	root := writeTestRepo(t)
	cfg := testCfg
	var events []Event
	cfg.Progress = func(e Event) { events = append(events, e) }
	if _, err := idx.Run(context.Background(), root, cfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	var scan *Event
	for i := range events {
		if events[i].Phase == "scan" {
			scan = &events[i]
			break
		}
	}
	if scan == nil {
		t.Fatal("no scan event emitted")
	}
	if scan.Total != 2 {
		t.Errorf("scan total = %d, want 2 (node_modules excluded)", scan.Total)
	}
	ext := extractEvents(events)
	if len(ext) == 0 {
		t.Fatal("no extract progress events")
	}
	if ext[0].Total != 2 {
		t.Errorf("extract total = %d, want 2", ext[0].Total)
	}
	done := make(map[int]bool)
	for _, e := range ext {
		done[e.Done] = true
	}
	if !done[1] || !done[2] {
		t.Errorf("extract done values = %v, want to include 1 and 2", done)
	}
}

// TestProgressNamedPasses: every advisory pass announces its start phase in
// order.
func TestProgressNamedPasses(t *testing.T) {
	idx, clean := newTestIndexer(t)
	defer clean()
	root := writeTestRepo(t)
	cfg := testCfg
	var events []Event
	cfg.Progress = func(e Event) { events = append(events, e) }
	if _, err := idx.Run(context.Background(), root, cfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	want := []string{"community", "import-cycles", "process", "knowledge", "resolution-audit", "complete"}
	got := phasesOf(events)
	for _, p := range want {
		i := slices.Index(got, p)
		if i < 0 {
			t.Errorf("phase %q not announced; got %v", p, got)
			continue
		}
		if j := slices.Index(got, "complete"); i > j {
			t.Errorf("phase %q announced after complete; got %v", p, got)
		}
	}
}

// TestProgressEmbedPass: when an embedder is attached, the embed phase
// announces with a nonzero total and reports per-item progress.
func TestProgressEmbedPass(t *testing.T) {
	idx, clean := newTestIndexerWithEmbedder(t)
	defer clean()
	root := writeEmbedFixture(t)
	cfg := testCfg
	var events []Event
	cfg.Progress = func(e Event) { events = append(events, e) }
	if _, err := idx.Run(context.Background(), root, cfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	var start, item *Event
	for i := range events {
		if events[i].Phase == "embed" {
			if events[i].Done == 0 {
				start = &events[i]
			} else if item == nil {
				item = &events[i]
			}
		}
	}
	if start == nil {
		t.Fatal("no embed start event")
	}
	if start.Total <= 0 {
		t.Errorf("embed start total = %d, want > 0", start.Total)
	}
	if item == nil {
		t.Fatal("no embed item progress events")
	}
	if item.Done > start.Total {
		t.Errorf("embed done %d exceeds total %d", item.Done, start.Total)
	}
}

// TestProgressPDGPass: with --pdg, the pdg phase announces with a nonzero
// total (one per function) and reports per-function progress.
func TestProgressPDGPass(t *testing.T) {
	idx, clean := newTestIndexer(t)
	defer clean()
	root := t.TempDir()
	mustWrite(t, root+"/a.go", "package a\n\nfunc One() { x := 1; _ = x }\nfunc Two() { y := 2; _ = y }\n")
	cfg := testCfg
	cfg.PDG = true
	var events []Event
	cfg.Progress = func(e Event) { events = append(events, e) }
	if _, err := idx.Run(context.Background(), root, cfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	var start, item *Event
	for i := range events {
		if events[i].Phase == "pdg" {
			if events[i].Done == 0 {
				start = &events[i]
			} else if item == nil {
				item = &events[i]
			}
		}
	}
	if start == nil {
		t.Fatal("no pdg start event")
	}
	if start.Total != 2 {
		t.Errorf("pdg total = %d, want 2 functions", start.Total)
	}
	if item == nil {
		t.Fatal("no pdg item progress events")
	}
	if item.Done != 1 && item.Done != 2 {
		t.Errorf("pdg item done = %d, want 1 or 2", item.Done)
	}
}
