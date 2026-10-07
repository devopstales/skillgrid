package dep

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(t.TempDir(), "deptest")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("fixtures", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(b)
}

const (
	purlApp      = "pkg:pypi/app@1.0.0"
	purlFlask    = "pkg:pypi/flask@3.0.2"
	purlWerkzeug = "pkg:pypi/werkzeug@3.0.6"
	purlJinja2   = "pkg:pypi/jinja2@3.1.5"
)

// [happy path dep ingest upserts by purl and soft-retires absent]
func TestIngestUpsertsByPurl(t *testing.T) {
	st := openTestStore(t)
	s := New(st.DB)
	ctx := context.Background()

	if err := s.Ingest(ctx, readFixture(t, "sbom.cyclonedx.json")); err != nil {
		t.Fatalf("ingest: %v", err)
	}

	for _, p := range []string{purlApp, purlFlask, purlWerkzeug} {
		var name, version string
		var retired int
		var lastSeen *string
		if err := st.DB.QueryRow(`
			SELECT name, version, retired, last_seen FROM dependencies WHERE purl = ?`, p).
			Scan(&name, &version, &retired, &lastSeen); err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		if retired != 0 {
			t.Errorf("%s retired = %d, want 0", p, retired)
		}
		if lastSeen == nil || *lastSeen == "" {
			t.Errorf("%s last_seen not set", p)
		}
		if name == "" || version == "" {
			t.Errorf("%s name/version empty (name=%q version=%q)", p, name, version)
		}
	}

	var edgeCount int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM dep_edges`).Scan(&edgeCount); err != nil {
		t.Fatalf("count edges: %v", err)
	}
	if edgeCount != 2 {
		t.Errorf("dep_edges count = %d, want 2 (app->flask, flask->werkzeug)", edgeCount)
	}
}

// [happy path dep ingest upserts by purl and soft-retires absent]
func TestIngestSoftRetiresAbsentPurl(t *testing.T) {
	st := openTestStore(t)
	s := New(st.DB)
	ctx := context.Background()

	if err := s.Ingest(ctx, readFixture(t, "sbom.cyclonedx.json")); err != nil {
		t.Fatalf("ingest 1: %v", err)
	}
	var firstLastSeen *string
	if err := st.DB.QueryRow(`SELECT last_seen FROM dependencies WHERE purl = ?`, purlFlask).
		Scan(&firstLastSeen); err != nil {
		t.Fatalf("read flask last_seen: %v", err)
	}

	if err := s.Ingest(ctx, readFixture(t, "sbom-noflask.json")); err != nil {
		t.Fatalf("ingest 2: %v", err)
	}

	// flask row survives (no delete) with retired=1.
	var retired int
	var name string
	var lastSeen *string
	if err := st.DB.QueryRow(`
		SELECT retired, name, last_seen FROM dependencies WHERE purl = ?`, purlFlask).
		Scan(&retired, &name, &lastSeen); err != nil {
		t.Fatalf("flask row deleted (want it to survive soft-retire): %v", err)
	}
	if retired != 1 {
		t.Errorf("flask retired = %d, want 1", retired)
	}
	if name != "flask" {
		t.Errorf("flask name = %q, want flask", name)
	}

	// app and werkzeug are present again -> retired cleared back to 0.
	for _, p := range []string{purlApp, purlWerkzeug} {
		var r int
		if err := st.DB.QueryRow(`SELECT retired FROM dependencies WHERE purl = ?`, p).Scan(&r); err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		if r != 0 {
			t.Errorf("%s retired = %d, want 0", p, r)
		}
	}

	// flask edges were rebuilt away; only app->werkzeug remains.
	var appFlask, appWerk int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM dep_edges WHERE from_purl = ? AND to_purl = ?`,
		purlApp, purlFlask).Scan(&appFlask); err != nil {
		t.Fatalf("count app->flask: %v", err)
	}
	if appFlask != 0 {
		t.Errorf("app->flask edge should be gone, found %d", appFlask)
	}
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM dep_edges WHERE from_purl = ? AND to_purl = ?`,
		purlApp, purlWerkzeug).Scan(&appWerk); err != nil {
		t.Fatalf("count app->werkzeug: %v", err)
	}
	if appWerk != 1 {
		t.Errorf("app->werkzeug edge should exist, found %d", appWerk)
	}
}

// [happy path dep_affected returns reverse dependency set]
func TestAffectedReturnsReverseDependencySet(t *testing.T) {
	st := openTestStore(t)
	s := New(st.DB)
	ctx := context.Background()

	if err := s.Ingest(ctx, readFixture(t, "sbom.cyclonedx.json")); err != nil {
		t.Fatalf("ingest: %v", err)
	}

	got, err := s.Affected(ctx, purlWerkzeug)
	if err != nil {
		t.Fatalf("affected: %v", err)
	}
	set := toSet(got)
	if !set[purlFlask] {
		t.Errorf("Affected(%s) missing %s: %v", purlWerkzeug, purlFlask, got)
	}
	if !set[purlApp] {
		t.Errorf("Affected(%s) missing transitive %s: %v", purlWerkzeug, purlApp, got)
	}
	if len(got) != 2 {
		t.Errorf("Affected(%s) = %v, want exactly {flask, app}", purlWerkzeug, got)
	}

	// Transitive set only: flask is a direct dependent, app a transitive one.
	got2, err := s.Affected(ctx, purlFlask)
	if err != nil {
		t.Fatalf("affected flask: %v", err)
	}
	set2 := toSet(got2)
	if !set2[purlApp] || len(got2) != 1 {
		t.Errorf("Affected(%s) = %v, want exactly {app}", purlFlask, got2)
	}
}

func TestGraph(t *testing.T) {
	st := openTestStore(t)
	s := New(st.DB)
	ctx := context.Background()

	if err := s.Ingest(ctx, readFixture(t, "sbom.graph.json")); err != nil {
		t.Fatalf("ingest: %v", err)
	}

	g, err := s.Graph(ctx)
	if err != nil {
		t.Fatalf("graph: %v", err)
	}
	if len(g.Nodes) != 4 {
		t.Fatalf("graph nodes = %d, want 4", len(g.Nodes))
	}
	nodeSet := toSet(g.Nodes)
	for _, p := range []string{purlApp, purlFlask, purlJinja2, purlWerkzeug} {
		if !nodeSet[p] {
			t.Errorf("graph nodes missing %s", p)
		}
	}
	if len(g.Edges) != 3 {
		t.Fatalf("graph edges = %d, want 3", len(g.Edges))
	}
	edgeSet := map[string]bool{}
	for _, e := range g.Edges {
		edgeSet[e.From+"->"+e.To] = true
	}
	for want := range map[string]bool{
		purlApp + "->" + purlFlask:      true,
		purlFlask + "->" + purlJinja2:   true,
		purlFlask + "->" + purlWerkzeug: true,
	} {
		if !edgeSet[want] {
			t.Errorf("graph edges missing %s (have %v)", want, g.Edges)
		}
	}
}

func TestList(t *testing.T) {
	st := openTestStore(t)
	s := New(st.DB)
	ctx := context.Background()

	if err := s.Ingest(ctx, readFixture(t, "sbom.cyclonedx.json")); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if err := s.Ingest(ctx, readFixture(t, "sbom-noflask.json")); err != nil {
		t.Fatalf("ingest noflask: %v", err)
	}

	active, err := s.List(ctx, false)
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	if len(active) != 2 {
		t.Errorf("list(active) = %d, want 2 (app, werkzeug)", len(active))
	}

	retired, err := s.List(ctx, true)
	if err != nil {
		t.Fatalf("list retired: %v", err)
	}
	if len(retired) != 1 {
		t.Fatalf("list(retired) = %d, want 1 (flask)", len(retired))
	}
	if retired[0].Purl != purlFlask {
		t.Errorf("list(retired) = %v, want flask", retired)
	}
}

func TestGet(t *testing.T) {
	st := openTestStore(t)
	s := New(st.DB)
	ctx := context.Background()

	if err := s.Ingest(ctx, readFixture(t, "sbom.cyclonedx.json")); err != nil {
		t.Fatalf("ingest: %v", err)
	}

	p, err := s.Get(ctx, purlFlask)
	if err != nil {
		t.Fatalf("get flask: %v", err)
	}
	if p.Purl != purlFlask || p.Name != "flask" || p.Version != "3.0.2" {
		t.Errorf("Get(flask) = %+v", p)
	}
	if p.Retired {
		t.Errorf("Get(flask) retired = true, want false")
	}

	if _, err := s.Get(ctx, "pkg:pypi/nope@9.9.9"); err == nil {
		t.Errorf("Get(missing) = nil error, want not-found error")
	}
}

func TestParseSBOMCycloneDX(t *testing.T) {
	pkgs, edges, err := ParseSBOM([]byte(readFixture(t, "sbom.cyclonedx.json")))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(pkgs) != 3 {
		t.Fatalf("pkgs = %d, want 3", len(pkgs))
	}
	byPurl := map[string]Pkg{}
	for _, p := range pkgs {
		byPurl[p.Purl] = p
	}
	flask := byPurl[purlFlask]
	if flask.Name != "flask" || flask.Version != "3.0.2" {
		t.Errorf("flask pkg = %+v", flask)
	}
	if flask.Ecosystem != "pypi" {
		t.Errorf("flask ecosystem = %q, want pypi", flask.Ecosystem)
	}
	if len(edges) != 2 {
		t.Fatalf("edges = %d, want 2: %v", len(edges), edges)
	}
	edgeSet := map[string]bool{}
	for _, e := range edges {
		edgeSet[e.From+"->"+e.To] = true
	}
	if !edgeSet[purlApp+"->"+purlFlask] || !edgeSet[purlFlask+"->"+purlWerkzeug] {
		t.Errorf("edges = %v", edges)
	}
}

func TestParseSBOMUnsupportedFormats(t *testing.T) {
	if _, _, err := ParseSBOM([]byte(`{"spdxVersion":"SPDX-2.3","name":"x"}`)); err == nil {
		t.Errorf("SPDX: want error, got nil")
	} else if !strings.Contains(err.Error(), "SPDX not yet supported") {
		t.Errorf("SPDX error = %q, want SPDX not yet supported", err.Error())
	}
	if _, _, err := ParseSBOM([]byte(`{"name":"x"}`)); err == nil {
		t.Errorf("unknown format: want error, got nil")
	}
	if _, _, err := ParseSBOM([]byte("   ")); err == nil {
		t.Errorf("empty: want error, got nil")
	}
}

func toSet(in []string) map[string]bool {
	out := make(map[string]bool, len(in))
	for _, s := range in {
		out[s] = true
	}
	return out
}
