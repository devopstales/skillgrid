package memory

import (
	"context"
	"testing"
)

// seedLayeredStore creates a session + L2 scenario + L3 persona-delta personas
// records (the step-02 bootstrap sources) and an L1 atom observation (the RRF
// fallback source), each provenance-linked. It returns the session id.
func seedLayeredStore(t *testing.T, fx *fixture) {
	t.Helper()
	ctx := context.Background()
	// L0 source: the fixture's session1 (a resolvable live session row).
	l0 := session1
	// L1 atom observation (the RRF fallback source — reachable via FTS).
	if _, err := fx.svc.Save(ctx, SaveInput{
		SessionID: l0, Type: "architecture",
		Title:   "Persona delta marker PERSONA_DELTA_XYZ",
		Content: "Persona profile increment PERSONA_DELTA_XYZ",
	}); err != nil {
		t.Fatalf("save persona atom: %v", err)
	}
	// L2 scenario + L3 persona-delta personas records (step-02 bootstrap).
	if _, err := fx.st.DB.Exec(`
		INSERT INTO personas (project, kind, title, content, content_hash, created_at)
		VALUES (?, 'persona_delta', 'Persona delta marker PERSONA_DELTA_XYZ', 'Persona profile increment PERSONA_DELTA_XYZ', 'h-delta', '2026-01-02T00:00:00Z')`,
		fx.svc.ProjectID()); err != nil {
		t.Fatalf("insert persona_delta: %v", err)
	}
	if _, err := fx.st.DB.Exec(`
		INSERT INTO personas (project, kind, title, content, content_hash, created_at)
		VALUES (?, 'scenario', 'Scenario marker SCENARIO_ABC', 'Working context scenario SCENARIO_ABC', 'h-scen', '2026-01-02T00:00:00Z')`,
		fx.svc.ProjectID()); err != nil {
		t.Fatalf("insert scenario: %v", err)
	}
	deltaID := lastPersonaID(t, fx, "persona_delta")
	scenID := lastPersonaID(t, fx, "scenario")
	if _, err := fx.st.DB.Exec(`
		INSERT INTO observation_layers (project, layer, target_kind, target_id, source_session, source_topic, content_hash, created_at)
		VALUES (?, 'L3', 'persona', ?, ?, 'delta-topic', 'h-delta', '2026-01-02T00:00:00Z')`,
		fx.svc.ProjectID(), deltaID, l0); err != nil {
		t.Fatalf("link L3: %v", err)
	}
	if _, err := fx.st.DB.Exec(`
		INSERT INTO observation_layers (project, layer, target_kind, target_id, source_session, source_topic, content_hash, created_at)
		VALUES (?, 'L2', 'persona', ?, ?, 'scen-topic', 'h-scen', '2026-01-02T00:00:00Z')`,
		fx.svc.ProjectID(), scenID, l0); err != nil {
		t.Fatalf("link L2: %v", err)
	}
}

func lastPersonaID(t *testing.T, fx *fixture, kind string) int64 {
	t.Helper()
	var id int64
	if err := fx.st.DB.QueryRow(`
		SELECT id FROM personas WHERE project = ? AND kind = ? ORDER BY id DESC LIMIT 1`,
		fx.svc.ProjectID(), kind).Scan(&id); err != nil {
		t.Fatalf("last persona %s: %v", kind, err)
	}
	return id
}

// TestRetrieve is 03.2 [RED] — the "Mnemonic tool surface" threat. It proves:
// (a) layered retrieval returns L2/L3 first (bootstrap) and falls back to L1/L0
// via the existing RRF path for a specific fact; (b) every in-list hit carries
// its full-content fetch id (mem_get_observation is the only full-content
// path); and (c) bad retrieval args are rejected clearly.
//
// Scenarios: layered-retrieval-l2-l3-first-with-l1-l0-rrf-fallback,
// mem-get-observation-is-only-full-content-path.
func TestRetrieve(t *testing.T) {
	fx := newFixture(t, "retrieve-proj")
	ctx := context.Background()
	seedLayeredStore(t, fx)

	t.Run("bootstrap-returns-l2-l3-first", func(t *testing.T) {
		hits, err := fx.svc.Retrieve(ctx, RetrieveOpts{Mode: "bootstrap", Limit: 10})
		if err != nil {
			t.Fatalf("retrieve bootstrap: %v", err)
		}
		if len(hits) < 2 {
			t.Fatalf("expected L2+L3 bootstrap hits, got %d", len(hits))
		}
		// L3 (persona_delta) must come before L2 (scenario): the most stable
		// layer first.
		if hits[0].Layer != "L3" {
			t.Fatalf("expected L3 first, got %q (hit %+v)", hits[0].Layer, hits[0])
		}
		// Every in-list hit carries its full-content fetch id.
		for _, h := range hits {
			if h.GetObservationID == 0 {
				t.Fatalf("in-list hit missing full-content fetch id: %+v", h)
			}
		}
	})

	t.Run("fact-falls-back-to-l1-l0-rrf", func(t *testing.T) {
		// A specific-fact query that matches the L1 atom content falls back to
		// the existing RRF path (BlendedSearch / FTS) and returns L1 hits.
		hits, err := fx.svc.Retrieve(ctx, RetrieveOpts{Mode: "fact", Query: "persona delta marker", Limit: 10})
		if err != nil {
			t.Fatalf("retrieve fact: %v", err)
		}
		if len(hits) == 0 {
			t.Fatal("expected an L1/L0 RRF fallback hit for the specific fact")
		}
		for _, h := range hits {
			if h.Layer != "L1" {
				t.Fatalf("RRF fallback should return L1 hits, got layer %q", h.Layer)
			}
			if h.GetObservationID == 0 {
				t.Fatalf("RRF fallback hit missing full-content fetch id: %+v", h)
			}
		}
	})

	t.Run("bad-mode-rejected", func(t *testing.T) {
		_, err := fx.svc.Retrieve(ctx, RetrieveOpts{Mode: "nonsense", Limit: 5})
		if err == nil {
			t.Fatal("unknown retrieve mode should be rejected")
		}
	})
}

// TestLayered is 03.3 [AFK] — proves the layered-retrieval contract in full:
// a bootstrap read returns L2/L3 first (cheap, stable), and a specific-fact
// query falls back to L1/L0 via the EXISTING RRF path (BlendedSearch). It is
// the standalone scenario test for
// layered-retrieval-l2-l3-first-with-l1-l0-rrf-fallback.
func TestLayered(t *testing.T) {
	fx := newFixture(t, "layered-proj")
	ctx := context.Background()
	seedLayeredStore(t, fx)

	// Bootstrap: L2/L3 first. The first hit must be L3 (persona_delta), then L2
	// (scenario) — the most stable, cheapest layers before any L1/L0.
	boot, err := fx.svc.Retrieve(ctx, RetrieveOpts{Mode: "bootstrap", Limit: 10})
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if len(boot) < 2 {
		t.Fatalf("expected >=2 bootstrap layers, got %d", len(boot))
	}
	layers := make([]string, 0, len(boot))
	for _, h := range boot {
		layers = append(layers, h.Layer)
	}
	if layers[0] != "L3" || layers[1] != "L2" {
		t.Fatalf("bootstrap must return L3 then L2 first, got %v", layers)
	}
	// No L1/L0 in the bootstrap (those are the RRF fallback, not bootstrap).
	for _, h := range boot {
		if h.Layer == "L1" || h.Layer == "L0" {
			t.Fatalf("bootstrap must not include L1/L0, got layer %q", h.Layer)
		}
	}

	// Specific fact: falls back to L1/L0 via the existing RRF path. The fact
	// matches the L1 atom's content, so the RRF leg (BlendedSearch/FTS) returns
	// it as an L1 hit.
	fact, err := fx.svc.Retrieve(ctx, RetrieveOpts{Mode: "fact", Query: "persona profile increment", Limit: 10})
	if err != nil {
		t.Fatalf("fact: %v", err)
	}
	if len(fact) == 0 {
		t.Fatal("specific-fact query must fall back to L1/L0 RRF and return a hit")
	}
	for _, h := range fact {
		if h.Layer != "L1" {
			t.Fatalf("fact fallback must be L1/L0 RRF, got layer %q", h.Layer)
		}
		// Every RRF fallback hit carries its full-content fetch id.
		if h.GetObservationID == 0 {
			t.Fatalf("fact hit missing full-content fetch id: %+v", h)
		}
	}
}
