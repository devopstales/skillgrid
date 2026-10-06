package service

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/codeindex"
)

// TestRunCodeIndexPDGWithProgress: the progress callback threaded through the
// service reaches the indexer — the CLI relies on this to render live status.
func TestRunCodeIndexPDGWithProgress(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)
	root := t.TempDir()
	mustWriteSvc(t, filepath.Join(root, "main.go"), "package main\n\nfunc alpha() {\n\tbeta()\n}\n\nfunc beta() {}\n")

	var events []codeindex.Event
	stats, err := svc.RunCodeIndexPDGWithProgress(context.Background(), root, false, false, func(e codeindex.Event) {
		events = append(events, e)
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if stats.FilesIndexed != 1 {
		t.Fatalf("expected 1 file indexed, got %d", stats.FilesIndexed)
	}
	var sawScan, sawComplete bool
	for _, e := range events {
		if e.Phase == "scan" {
			sawScan = true
		}
		if e.Phase == "complete" {
			sawComplete = true
		}
	}
	if !sawScan {
		t.Error("no scan event reached the callback")
	}
	if !sawComplete {
		t.Error("no complete event reached the callback")
	}
}
