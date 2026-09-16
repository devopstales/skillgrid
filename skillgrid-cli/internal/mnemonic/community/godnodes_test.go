package community

import (
	"database/sql"
	"testing"
)

// hubFixture builds a graph with a utility super-hub (a helper called from
// >= HubFileThreshold distinct files) plus a genuinely central concept (a
// dispatcher called from 3 files). This lets us assert that excludeHubs
// suppresses the super-hub while keeping the subsystem concept.
func hubFixture(t *testing.T) *sql.DB {
	t.Helper()
	db := openStore(t)
	// The super-hub: helper, called from 5 distinct files.
	fhub := seedFile(t, db, "util/helper.go")
	helper := seedSymbol(t, db, fhub, "helper", "function", 1)
	for i, p := range []string{"f1/x.go", "f2/y.go", "f3/z.go", "f4/w.go", "f5/v.go"} {
		f := seedFile(t, db, p)
		caller := seedSymbol(t, db, f, "caller"+string(rune('0'+i)), "function", 1)
		seedEdge(t, db, "calls", caller, helper, i+2)
	}
	// A central concept: dispatcher, called from 3 distinct files (below the
	// hub threshold) plus a couple more edges within its cluster.
	fd := seedFile(t, db, "core/dispatch.go")
	dispatcher := seedSymbol(t, db, fd, "dispatcher", "function", 1)
	fa := seedFile(t, db, "core/a.go")
	fb := seedFile(t, db, "core/b.go")
	ca := seedSymbol(t, db, fa, "alpha", "function", 1)
	cb := seedSymbol(t, db, fb, "beta", "function", 1)
	seedEdge(t, db, "calls", ca, dispatcher, 2)
	seedEdge(t, db, "calls", cb, dispatcher, 2)
	seedEdge(t, db, "calls", dispatcher, ca, 3)
	seedEdge(t, db, "calls", dispatcher, cb, 3)
	_ = helper
	return db
}

// TestGodNodesRankByDegree covers @step-01 (Scenario: code_god_nodes ranks
// hubs and exclude-hubs suppresses them): RankGodNodes returns symbols
// ranked by degree (descending), and excludeHubs=true suppresses the utility
// super-hub (referenced across >= HubFileThreshold files) while keeping the
// central concept.
func TestGodNodesRankByDegree(t *testing.T) {
	db := hubFixture(t)
	var ids []int64
	rows, err := db.Query(`SELECT id FROM symbols`)
	if err != nil {
		t.Fatalf("query symbols: %v", err)
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		ids = append(ids, id)
	}
	rows.Close()

	gods, err := RankGodNodes(db, ids, false)
	if err != nil {
		t.Fatalf("RankGodNodes: %v", err)
	}
	if len(gods) == 0 {
		t.Fatal("expected god nodes, got none")
	}
	// Ranked by degree descending.
	for i := 1; i < len(gods); i++ {
		if gods[i-1].Degree < gods[i].Degree {
			t.Errorf("not ranked by degree descending at %d: %d < %d", i, gods[i-1].Degree, gods[i].Degree)
		}
	}
	// The super-hub (helper, degree 5) must be the top god node without
	// exclusion.
	if gods[0].Name != "helper" {
		t.Errorf("expected super-hub 'helper' to rank first, got %q (degree %d)", gods[0].Name, gods[0].Degree)
	}

	// With excludeHubs, the super-hub is suppressed but the central concept
	// (dispatcher) is still present.
	excluded, err := RankGodNodes(db, ids, true)
	if err != nil {
		t.Fatalf("RankGodNodes excludeHubs: %v", err)
	}
	sawHub, sawDispatcher := false, false
	for _, g := range excluded {
		if g.Name == "helper" {
			sawHub = true
		}
		if g.Name == "dispatcher" {
			sawDispatcher = true
		}
	}
	if sawHub {
		t.Errorf("excludeHubs should suppress the utility super-hub 'helper'")
	}
	if !sawDispatcher {
		t.Errorf("excludeHubs should keep the central concept 'dispatcher'")
	}
}
