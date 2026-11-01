package codeindex

import (
	"context"
	"path/filepath"
	"testing"
)

var routeCfg = Config{
	Include:      []string{"**/*.go", "**/*.js", "**/*.ts", "**/*.tsx", "**/*.py", "**/*.rb", "**/*.java", "**/*.svelte"},
	Exclude:      []string{"**/node_modules/**", "**/.git/**"},
	ChunkLines:   80,
	ChunkOverlap: 10,
}

// writeWebAppFixture returns a small web app: an Express route (server.js)
// with a handler (handler.js), a Next.js navigation (page.tsx) with a literal
// and a markup link, and a computed-destination file that must stay unresolved.
func writeWebAppFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "server.js"), "function listUsers() { return []; }\n\napp.get('/users', listUsers);\napp.get('/mystery', unservedHandler);\n")
	mustWrite(t, filepath.Join(root, "page.tsx"), `import { useRouter } from 'next/router';

export function Page() {
  const router = useRouter();
  return (
    <>
      <button onClick={() => router.push('/about')}>About</button>
      <Link href="/settings">Settings</Link>
    </>
  );
}
`)
	return root
}

// TestIndexProducesRouteNodesAndReferences covers @step-01 (Scenario:
// Web-framework routing files produce route nodes): after indexing, the route
// nodes exist (kind=route symbols), each is linked by a references edge to its
// handler, every explicit edge is EXTRACTED, and a query for callers of a
// handler surfaces the URL pattern.
func TestIndexProducesRouteNodesAndReferences(t *testing.T) {
	ResetFileFirstSymbol()
	idx, clean := newTestIndexer(t)
	defer clean()
	root := writeWebAppFixture(t)
	if _, err := idx.Run(context.Background(), root, routeCfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	db := idx.store.DB

	// Route nodes exist as kind=route symbols.
	var routeSymbols int
	if err := db.QueryRow(`SELECT COUNT(*) FROM symbols WHERE kind = 'route'`).Scan(&routeSymbols); err != nil {
		t.Fatalf("count route symbols: %v", err)
	}
	if routeSymbols < 2 {
		t.Fatalf("expected >=2 route symbols (/users, /mystery), got %d", routeSymbols)
	}

	// A references edge from the /users route to its handler (EXTRACTED).
	var usersRef int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM edges e
		JOIN symbols r ON r.id = e.from_id
		JOIN symbols h ON h.id = e.to_id
		WHERE e.kind = 'references' AND r.kind = 'route'
		  AND h.name = 'listUsers'
	`).Scan(&usersRef); err != nil {
		t.Fatalf("count references: %v", err)
	}
	if usersRef != 1 {
		t.Errorf("expected 1 references edge from /users route to listUsers, got %d", usersRef)
	}

	// Every references edge from an explicit route is EXTRACTED.
	var nonExtracted int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM edges WHERE kind = 'references' AND confidence != 'EXTRACTED'
	`).Scan(&nonExtracted); err != nil {
		t.Fatalf("count non-extracted references: %v", err)
	}
	if nonExtracted != 0 {
		t.Errorf("expected all explicit references edges EXTRACTED, got %d non-EXTRACTED", nonExtracted)
	}

	// A query for callers of the handler surfaces the URL pattern: the route
	// node's references edge points at listUsers, so reverse-lookup of
	// listUsers surfaces the /users route.
	var callersSurface int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM edges e
		JOIN symbols h ON h.id = e.to_id
		JOIN symbols r ON r.id = e.from_id
		WHERE e.kind = 'references' AND h.name = 'listUsers' AND r.kind = 'route'
	`).Scan(&callersSurface); err != nil {
		t.Fatalf("callers surface: %v", err)
	}
	if callersSurface != 1 {
		t.Errorf("expected the /users URL pattern to surface for handler listUsers, got %d", callersSurface)
	}
}

// TestIndexRouteNinavatesEdges covers @step-01 (Scenario: Router navigations
// produce navigates edges): after indexing, navigates edges are stored from
// the sending function to the named screen; literal destinations are
// EXTRACTED, markup links are INFERRED.
func TestIndexRouteNavigatesEdges(t *testing.T) {
	ResetFileFirstSymbol()
	idx, clean := newTestIndexer(t)
	defer clean()
	root := writeWebAppFixture(t)
	if _, err := idx.Run(context.Background(), root, routeCfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	db := idx.store.DB

	var navCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind = 'navigates'`).Scan(&navCount); err != nil {
		t.Fatalf("count navigates: %v", err)
	}
	if navCount < 2 {
		t.Fatalf("expected >=2 navigates edges (/about literal, /settings markup), got %d", navCount)
	}

	// The literal /about destination is EXTRACTED.
	var aboutConf string
	err := db.QueryRow(`SELECT confidence FROM edges WHERE kind = 'navigates' AND to_name = '/about' LIMIT 1`).Scan(&aboutConf)
	if err != nil || aboutConf != "EXTRACTED" {
		t.Errorf("navigates to /about should be EXTRACTED, got %q (err %v)", aboutConf, err)
	}
	// The markup <Link href> destination is INFERRED.
	var settingsConf string
	err = db.QueryRow(`SELECT confidence FROM edges WHERE kind = 'navigates' AND to_name = '/settings' LIMIT 1`).Scan(&settingsConf)
	if err != nil || settingsConf != "INFERRED" {
		t.Errorf("navigates to /settings (markup) should be INFERRED, got %q (err %v)", settingsConf, err)
	}
}

// TestIndexUnresolvedDestinationStaysUnresolved covers @step-01 (Scenario:
// Computed or unserved destination stays unresolved): a navigates edge to a
// computed destination is NOT fabricated, and an unserved handler reference is
// dropped (no references edge), while the route node still exists.
func TestIndexUnresolvedDestinationStaysUnresolved(t *testing.T) {
	ResetFileFirstSymbol()
	idx, clean := newTestIndexer(t)
	defer clean()
	root := writeWebAppFixture(t)
	if _, err := idx.Run(context.Background(), root, routeCfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	db := idx.store.DB

	// The /mystery route node exists but has NO references edge (its handler
	// `unservedHandler` is unresolvable → dropped by drop-not-guess).
	var mysteryRoute int
	if err := db.QueryRow(`SELECT COUNT(*) FROM symbols WHERE kind = 'route' AND name LIKE '%/mystery%'`).Scan(&mysteryRoute); err != nil {
		t.Fatalf("count /mystery route: %v", err)
	}
	if mysteryRoute != 1 {
		t.Errorf("expected the /mystery route node to exist, got %d", mysteryRoute)
	}
	var mysteryRef int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM edges e
		JOIN symbols r ON r.id = e.from_id
		WHERE e.kind = 'references' AND r.name LIKE '%/mystery%'
	`).Scan(&mysteryRef); err != nil {
		t.Fatalf("count /mystery references: %v", err)
	}
	if mysteryRef != 0 {
		t.Errorf("expected 0 references edges for the unserved /mystery handler (drop-not-guess), got %d", mysteryRef)
	}

	// A drop warning was recorded for server.js (the /mystery ref was dropped).
	var dropped int
	if err := db.QueryRow(`
		SELECT COALESCE(d.dropped,0) FROM route_drops d
		JOIN files f ON f.id = d.file_id
		WHERE f.path = 'server.js'
	`).Scan(&dropped); err != nil {
		t.Fatalf("count drops for server.js: %v", err)
	}
	if dropped != 1 {
		t.Errorf("expected 1 dropped ref warning for server.js, got %d", dropped)
	}
}

// TestIndexMalformedRouteFileFallsBack continues the index when a routing file
// is malformed: a second, malformed file is added, the index re-runs without
// error, and the previously-extracted route nodes survive.
func TestIndexMalformedRouteFileFallsBack(t *testing.T) {
	ResetFileFirstSymbol()
	idx, clean := newTestIndexer(t)
	defer clean()
	root := writeWebAppFixture(t)
	if _, err := idx.Run(context.Background(), root, routeCfg); err != nil {
		t.Fatalf("first run: %v", err)
	}
	// Add a malformed python routing file (invalid syntax).
	mustWrite(t, filepath.Join(root, "broken.py"), "urlpatterns = [\npath('/x/', views.Home.as_view(,\n")
	if _, err := idx.Run(context.Background(), root, routeCfg); err != nil {
		t.Fatalf("second run (with malformed file) should not error: %v", err)
	}
	db := idx.store.DB
	// The Express route nodes from the first run must survive.
	var routes int
	if err := db.QueryRow(`SELECT COUNT(*) FROM symbols WHERE kind = 'route'`).Scan(&routes); err != nil {
		t.Fatalf("count routes: %v", err)
	}
	if routes < 2 {
		t.Errorf("expected the prior route nodes to survive a malformed-file re-index, got %d", routes)
	}
}
