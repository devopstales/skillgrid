package codeindex

import (
	"context"
	"path/filepath"
	"testing"
)

// TestIndexPersistsSymbolTypesAndAudit covers 036: after indexing a Go file,
// function symbols carry derived type/visibility columns, the
// resolution_audit ledger records per-language call counts, and a
// receiver-qualified member call lands in unresolved_members with an
// internal/external split.
func TestIndexPersistsSymbolTypesAndAudit(t *testing.T) {
	ResetFileFirstSymbol()
	idx, clean := newTestIndexer(t)
	defer clean()

	root := t.TempDir()
	// A typed function + a lowercase-receiver member call (rows.Scan) that the
	// extractor cannot statically bind → external unresolved member.
	mustWrite(t, filepath.Join(root, "svc.go"), `package svc

import "context"

func Load(ctx context.Context, id int) (*Doc, error) {
	return nil, nil
}

func run(rows *rows) error {
	return rows.Scan()
}
`)
	if _, err := idx.Run(context.Background(), root, routeCfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	db := idx.store.DB

	// Load has a derived return type, param types, and is exported.
	var ret, params, vis string
	var exported int
	err := db.QueryRow(`SELECT return_type, param_types, visibility, is_exported
		FROM symbols WHERE name = 'Load'`).Scan(&ret, &params, &vis, &exported)
	if err != nil {
		t.Fatalf("Load symbol: %v", err)
	}
	if ret != "*Doc" {
		t.Errorf("Load return_type: got %q want *Doc", ret)
	}
	if params != "context.Context, int" {
		t.Errorf("Load param_types: got %q want 'context.Context, int'", params)
	}
	if vis != "export" || exported != 1 {
		t.Errorf("Load visibility: got %q exported=%d want export/1", vis, exported)
	}

	// run is private/unexported.
	var runVis string
	var runExported int
	if err := db.QueryRow(`SELECT visibility, is_exported FROM symbols WHERE name = 'run'`).Scan(&runVis, &runExported); err != nil {
		t.Fatalf("run symbol: %v", err)
	}
	if runVis != "private" || runExported != 0 {
		t.Errorf("run visibility: got %q exported=%d want private/0", runVis, &runExported)
	}

	// The resolution_audit ledger records Go call sites (Load + run + the
	// Scan member call are all extracted calls).
	var callSites, unres int
	if err := db.QueryRow(`SELECT call_sites, unresolved FROM resolution_audit WHERE language = 'go'`).Scan(&callSites, &unres); err != nil {
		t.Fatalf("resolution_audit go: %v", err)
	}
	if callSites < 1 {
		t.Errorf("go call_sites: got %d want >=1", callSites)
	}

	// The schema fingerprint is recorded so code_status can detect a stale
	// index.
	var fp string
	if err := db.QueryRow(`SELECT value FROM index_meta_kv WHERE key = 'schema_fingerprint'`).Scan(&fp); err != nil {
		t.Fatalf("schema_fingerprint: %v", err)
	}
	if fp == "" {
		t.Error("schema_fingerprint is empty; expected a non-empty hash")
	}
}

// TestIndexPersistsUnresolvedMembers verifies the receiver-qualified member
// backlog is logged with the internal/external split (036).
func TestIndexPersistsUnresolvedMembers(t *testing.T) {
	ResetFileFirstSymbol()
	idx, clean := newTestIndexer(t)
	defer clean()

	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "db.go"), `package db

func scan(rows *rows) error {
	return rows.Scan()
}
`)
	if _, err := idx.Run(context.Background(), root, routeCfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	db := idx.store.DB

	// rows.Scan is a lowercase-receiver member → external, member=Scan.
	var member, receiver string
	var external int
	err := db.QueryRow(`SELECT member, receiver, external FROM unresolved_members WHERE member = 'Scan'`).Scan(&member, &receiver, &external)
	if err != nil {
		t.Fatalf("unresolved Scan row: %v", err)
	}
	if receiver != "rows" {
		t.Errorf("receiver: got %q want rows", receiver)
	}
	if external != 1 {
		t.Errorf("external: got %d want 1", external)
	}
}
