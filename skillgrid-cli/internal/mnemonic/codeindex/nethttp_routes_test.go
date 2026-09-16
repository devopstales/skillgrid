package codeindex

import (
	"context"
	"path/filepath"
	"testing"
)

const nethttpServerFixture = `package main

import "net/http"

func handleHealth() {}

func handleSessionCreate() {}

func requireWriteAuth(h http.HandlerFunc) http.HandlerFunc {
	return h
}

func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("POST /sessions", requireWriteAuth(handleSessionCreate))
	return mux
}
`

const nethttpUniqueHandlerFixture = `package main

import "net/http"

var sessionHandler http.HandlerFunc

func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /sessions", sessionHandler)
	return mux
}
`

const nethttpUniqueHandlerDef = `package main

func sessionHandler(w http.ResponseWriter, r *http.Request) {}
`

const nethttpAmbiguousHandlerFixture = `package main

import "net/http"

var dupHandler http.HandlerFunc

func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /dup", dupHandler)
	return mux
}
`

const nethttpAmbiguousCollidingFixture = `package main

type A struct{}

type B struct{}

func (a *A) dupHandler(w http.ResponseWriter, r *http.Request) {}

func (b *B) dupHandler(w http.ResponseWriter, r *http.Request) {}
`

// TestIndexNetHTTPRoutesResolved covers the nethttp extractor end-to-end: a
// stdlib HandleFunc server with same-file handler methods yields route nodes,
// route_meta rows, and EXTRACTED references edges to the handler symbols (the
// receiver-qualified refs strip to the bare method name), with no unresolved
// refs logged.
func TestIndexNetHTTPRoutesResolved(t *testing.T) {
	ResetFileFirstSymbol()
	idx, clean := newTestIndexer(t)
	defer clean()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "server.go"), nethttpServerFixture)
	if _, err := idx.Run(context.Background(), root, routeCfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	db := idx.store.DB

	var routeSymbols int
	if err := db.QueryRow(`SELECT COUNT(*) FROM symbols WHERE kind = 'route'`).Scan(&routeSymbols); err != nil {
		t.Fatalf("count route symbols: %v", err)
	}
	if routeSymbols != 2 {
		t.Fatalf("expected 2 route symbols (/health, /sessions), got %d", routeSymbols)
	}
	var meta int
	if err := db.QueryRow(`SELECT COUNT(*) FROM route_meta`).Scan(&meta); err != nil {
		t.Fatalf("count route_meta: %v", err)
	}
	if meta != 2 {
		t.Errorf("expected 2 route_meta rows, got %d", meta)
	}
	// The wrapped handler arg resolves to the inner handler method.
	for _, handler := range []string{"handleHealth", "handleSessionCreate"} {
		var refs int
		if err := db.QueryRow(`
			SELECT COUNT(*) FROM edges e
			JOIN symbols r ON r.id = e.from_id
			JOIN symbols h ON h.id = e.to_id
			WHERE e.kind = 'references' AND r.kind = 'route' AND h.name = ?
		`, handler).Scan(&refs); err != nil {
			t.Fatalf("count references %s: %v", handler, err)
		}
		if refs != 1 {
			t.Errorf("expected 1 references edge to %s, got %d", handler, refs)
		}
		var conf string
		if err := db.QueryRow(`
			SELECT e.confidence FROM edges e
			JOIN symbols r ON r.id = e.from_id
			WHERE e.kind = 'references' AND r.kind = 'route'
		`).Scan(&conf); err != nil {
			t.Fatalf("reference confidence: %v", err)
		}
		if conf != "EXTRACTED" {
			t.Errorf("same-file nethttp handler should be EXTRACTED, got %q", conf)
		}
	}
	// Nothing was dropped for this file.
	var unresolved int
	if err := db.QueryRow(`SELECT COUNT(*) FROM unresolved_refs`).Scan(&unresolved); err != nil {
		t.Fatalf("count unresolved_refs: %v", err)
	}
	if unresolved != 0 {
		t.Errorf("expected no unresolved refs (all resolved same-file), got %d", unresolved)
	}
}

// TestIndexNetHTTPUniqueGlobalHandler covers the AMBIGUOUS path: a handler
// defined in a DIFFERENT file with a unique global name resolves name-only
// (stored low-confidence), and no unresolved ref is logged.
func TestIndexNetHTTPUniqueGlobalHandler(t *testing.T) {
	ResetFileFirstSymbol()
	idx, clean := newTestIndexer(t)
	defer clean()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "server.go"), nethttpUniqueHandlerFixture)
	mustWrite(t, filepath.Join(root, "handlers.go"), nethttpUniqueHandlerDef)
	if _, err := idx.Run(context.Background(), root, routeCfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	db := idx.store.DB

	var refs int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM edges e
		JOIN symbols r ON r.id = e.from_id
		JOIN symbols h ON h.id = e.to_id
		WHERE e.kind = 'references' AND r.kind = 'route' AND h.name = 'sessionHandler'
	`).Scan(&refs); err != nil {
		t.Fatalf("count references: %v", err)
	}
	if refs != 1 {
		t.Fatalf("expected 1 AMBIGUOUS references edge (unique global), got %d", refs)
	}
	var conf string
	if err := db.QueryRow(`
		SELECT e.confidence FROM edges e
		JOIN symbols r ON r.id = e.from_id
		JOIN symbols h ON h.id = e.to_id
		WHERE e.kind = 'references' AND r.kind = 'route' AND h.name = 'sessionHandler'
	`).Scan(&conf); err != nil {
		t.Fatalf("reference confidence: %v", err)
	}
	if conf != "AMBIGUOUS" {
		t.Errorf("name-only unique-global handler should be AMBIGUOUS, got %q", conf)
	}
	var unresolved int
	if err := db.QueryRow(`SELECT COUNT(*) FROM unresolved_refs`).Scan(&unresolved); err != nil {
		t.Fatalf("count unresolved_refs: %v", err)
	}
	if unresolved != 0 {
		t.Errorf("a unique global match resolves; expected 0 unresolved refs, got %d", unresolved)
	}
}

// TestIndexNetHTTPAmbiguousHandlerDropped covers the drop log: a handler name
// matching 2 global symbols is unresolvable (drop-not-guess) — no references
// edge, and the ref is logged in unresolved_refs with candidates=2.
func TestIndexNetHTTPAmbiguousHandlerDropped(t *testing.T) {
	ResetFileFirstSymbol()
	idx, clean := newTestIndexer(t)
	defer clean()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "server.go"), nethttpAmbiguousHandlerFixture)
	mustWrite(t, filepath.Join(root, "handlers1.go"), nethttpAmbiguousCollidingFixture)
	if _, err := idx.Run(context.Background(), root, routeCfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	db := idx.store.DB

	var refs int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM edges e
		JOIN symbols r ON r.id = e.from_id
		WHERE e.kind = 'references' AND r.kind = 'route'
	`).Scan(&refs); err != nil {
		t.Fatalf("count references: %v", err)
	}
	if refs != 0 {
		t.Errorf("expected 0 references edges (ambiguous handler dropped), got %d", refs)
	}
	var routeNodes int
	if err := db.QueryRow(`SELECT COUNT(*) FROM symbols WHERE kind = 'route'`).Scan(&routeNodes); err != nil {
		t.Fatalf("count route nodes: %v", err)
	}
	if routeNodes != 1 {
		t.Errorf("the route node still exists even when the handler is dropped, got %d", routeNodes)
	}
	var candidates int
	var refName string
	err := db.QueryRow(`SELECT candidates, reference_name FROM unresolved_refs LIMIT 1`).Scan(&candidates, &refName)
	if err != nil {
		t.Fatalf("expected an unresolved_refs row, got err: %v", err)
	}
	if candidates != 2 {
		t.Errorf("unresolved ref candidates = %d, want 2", candidates)
	}
	if refName != "dupHandler" {
		t.Errorf("unresolved ref name = %q, want dupHandler", refName)
	}
	var dropped int
	if err := db.QueryRow(`
		SELECT COALESCE(d.dropped,0) FROM route_drops d
		JOIN files f ON f.id = d.file_id
		WHERE f.path = 'server.go'
	`).Scan(&dropped); err != nil {
		t.Fatalf("count drops: %v", err)
	}
	if dropped != 1 {
		t.Errorf("expected 1 dropped ref for server.go, got %d", dropped)
	}
}
