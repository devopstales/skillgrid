package route

import (
	"context"
	"testing"
)

// TestExtractNetHTTP covers the stdlib nethttp extractor: Go 1.22
// "METHOD /path" bindings and Go 1.21 "/path" bindings (method ANY), with
// receiver-qualified and wrapped handler args stripped to the inner handler.
func TestExtractNetHTTP(t *testing.T) {
	cases := []struct {
		name     string
		src      string
		method   string
		pattern  string
		handler  string
		explicit bool
	}{
		{
			name:    "method binding",
			src:     "mux.HandleFunc(\"GET /health\", s.handleHealth)\n",
			method:  "GET",
			pattern: "/health",
			handler: "handleHealth",
		},
		{
			name:    "wrapped handler takes innermost call",
			src:     "s.mux.HandleFunc(\"POST /sessions\", s.requireWriteAuth(s.handleSessionCreate))\n",
			method:  "POST",
			pattern: "/sessions",
			handler: "handleSessionCreate",
		},
		{
			name:    "path-only binding defaults to ANY",
			src:     "mux.HandleFunc(\"/legacy\", handler)\n",
			method:  "ANY",
			pattern: "/legacy",
			handler: "handler",
		},
		{
			name:    "computed handler arg takes last identifier",
			src:     "mux.Handle(\"/static\", http.FileServer(root))\n",
			method:  "ANY",
			pattern: "/static",
			handler: "root",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Dispatch: .go files resolve as gin (listed first); the nethttp
			// routes are unioned into gin's result when gin finds none.
			fr := ExtractFile("server.go", []byte(tc.src))
			if fr.Framework != "gin" {
				t.Fatalf("dispatch framework = %q, want gin (.go dispatch)", fr.Framework)
			}
			if len(fr.Routes) != 1 {
				t.Fatalf("expected 1 route, got %d (src: %s)", len(fr.Routes), tc.src)
			}
			r := fr.Routes[0]
			if r.Framework != "nethttp" {
				t.Fatalf("route framework = %q, want nethttp", r.Framework)
			}
			if r.Method != tc.method {
				t.Errorf("method = %q, want %q", r.Method, tc.method)
			}
			if r.PathPattern != tc.pattern {
				t.Errorf("pattern = %q, want %q", r.PathPattern, tc.pattern)
			}
			if r.Handler != tc.handler {
				t.Errorf("handler = %q, want %q", r.Handler, tc.handler)
			}
			if !r.Explicit {
				t.Errorf("nethttp handler is a direct reference; expected Explicit=true")
			}
		})
	}
}

// TestExtractNetHTTPGinRegression locks the Go-file dispatch: gin and nethttp
// share the .go file type and gin is listed first, so any .go file resolves
// as gin in ExtractFile (src==nil OR a file with no gin binding). The
// nethttp extractor itself still works and its regex does not match gin's
// `r.Method(...)` shape.
func TestExtractNetHTTPGinRegression(t *testing.T) {
	src := "func healthHandler(c *gin.Context) {}\n\nr.GET(\"/x\", h)\n"
	fr := ExtractFile("main.go", []byte(src))
	if fr.Framework != "gin" {
		t.Fatalf("framework = %q, want gin (gin is listed before nethttp)", fr.Framework)
	}
	if len(fr.Routes) != 1 || fr.Routes[0].PathPattern != "/x" {
		t.Fatalf("gin route not extracted: %+v", fr.Routes)
	}
	// The nethttp extractor sees no routes in a gin-only file (its regex does
	// not match r.GET).
	if fr2 := extractNetHTTP("main.go", []byte(src)); len(fr2.Routes) != 0 {
		t.Errorf("extractNetHTTP should find no routes in a gin file, got %+v", fr2.Routes)
	}
}

// TestExtractNetHTTPNoRoutes covers a .go file with no bindings: the nethttp
// extractor yields zero routes, and the file-type dispatch reports gin
// (listed before nethttp for .go).
func TestExtractNetHTTPNoRoutes(t *testing.T) {
	src := "package main\n\nfunc helper() int { return 1 }\n"
	fr := ExtractFile("plain.go", []byte(src))
	if fr.Framework != "gin" {
		t.Fatalf("framework = %q, want gin (gin is listed before nethttp for .go)", fr.Framework)
	}
	if len(fr.Routes) != 0 || len(fr.Navigates) != 0 {
		t.Errorf("expected no routes/navigates, got %+v / %d", fr.Routes, len(fr.Navigates))
	}
	if fr2 := extractNetHTTP("plain.go", []byte(src)); len(fr2.Routes) != 0 {
		t.Errorf("extractNetHTTP should find no routes, got %+v", fr2.Routes)
	}
}

// TestNethttpHandlerStandalone pins the handler-stripping rule (last
// identifier in the arg wins, Go keywords skipped).
func TestNethttpHandlerStandalone(t *testing.T) {
	cases := []struct {
		arg  string
		want string
	}{
		{"handler", "handler"},
		{"s.handleHealth", "handleHealth"},
		{"s.requireWriteAuth(s.handleSessionCreate", "handleSessionCreate"},
		{"http.FileServer(root)", "root"},
		{"http.HandlerFunc(nil)", "nil"},
	}
	for _, tc := range cases {
		if got := nethttpHandler(tc.arg); got != tc.want {
			t.Errorf("nethttpHandler(%q) = %q, want %q", tc.arg, got, tc.want)
		}
	}
}

// fakeStore is a recording Store used to drive Run without a real index.
type fakeStore struct {
	fileID         int64
	storedRefs     []DroppedRef
	storedRefsFile int64
	storedRefsNow  string
	resolved       map[string]Resolution
}

func (f *fakeStore) StoreRouteNode(node RouteNode) (int64, error) {
	return int64(len(f.storedRefs) + 1), nil
}
func (f *fakeStore) StoreReferencesEdges(fileID int64, nodes []RouteNode, fileSymbolID int64) (int, error) {
	return len(nodes), nil
}
func (f *fakeStore) StoreNavigatesEdges(fileID int64, navs []NavigationNode) (int, error) {
	return len(navs), nil
}
func (f *fakeStore) StoreRouteDrops(fileID int64, dropped int) error { return nil }
func (f *fakeStore) StoreRouteMeta(fileID int64, symbolID int64, node RouteNode) error {
	return nil
}
func (f *fakeStore) FirstSymbolID(fileID int64) (int64, error) { return 0, nil }
func (f *fakeStore) ResolveHandler(fileID int64, name string) (int64, string, string, int) {
	if r, ok := f.resolved[name]; ok {
		return r.ID, r.UID, r.Confidence, r.Candidates
	}
	return 0, "", "", 0
}
func (f *fakeStore) StoreUnresolvedRefs(fileID int64, refs []DroppedRef, now string) error {
	f.storedRefsFile = fileID
	f.storedRefs = refs
	f.storedRefsNow = now
	return nil
}

// TestRunStoresUnresolvedRefs covers the Run hook: dropped references are
// persisted through StoreUnresolvedRefs with the file id and an RFC3339 now.
func TestRunStoresUnresolvedRefs(t *testing.T) {
	st := &fakeStore{fileID: 42, resolved: map[string]Resolution{}}
	src := "mux.HandleFunc(\"GET /missing\", s.handleMissing)\n"
	if _, err := Run(context.Background(), st, 42, "server.go", []byte(src), nil); err != nil {
		t.Fatalf("run: %v", err)
	}
	if st.storedRefsFile != 42 {
		t.Errorf("StoreUnresolvedRefs fileID = %d, want 42", st.storedRefsFile)
	}
	if len(st.storedRefs) != 1 || st.storedRefs[0].Name != "handleMissing" {
		t.Errorf("expected the dropped ref to be stored, got %+v", st.storedRefs)
	}
	if st.storedRefsNow == "" {
		t.Errorf("StoreUnresolvedRefs now must be non-empty (RFC3339)")
	}
}

// TestBuildNetHTTPDropsLogged covers the drop log: a Built with an
// unresolvable nethttp handler produces a DroppedRef with the correct
// name/line/candidates.
func TestBuildNetHTTPDropsLogged(t *testing.T) {
	src := "mux.HandleFunc(\"GET /missing\", s.handleMissing)\n"
	b := Build("server.go", []byte(src), fileSyms("other"), nothingIndex())
	if b.Dropped != 1 {
		t.Fatalf("expected 1 drop, got %d", b.Dropped)
	}
	if len(b.Drops) != 1 {
		t.Fatalf("expected 1 DroppedRef, got %d", len(b.Drops))
	}
	d := b.Drops[0]
	if d.Name != "handleMissing" || d.Kind != "route_handler" {
		t.Errorf("drop = %+v, want name handleMissing kind route_handler", d)
	}
	if d.Line != 1 {
		t.Errorf("drop line = %d, want 1", d.Line)
	}

	// A name with global candidates logs the count.
	idx := dummyIndex{globalCandidates: map[string]int{"handleAmbiguous": 2}}
	src2 := "mux.HandleFunc(\"GET /amb\", s.handleAmbiguous)\n"
	b2 := Build("server2.go", []byte(src2), fileSyms("other"), idx)
	if len(b2.Drops) != 1 || b2.Drops[0].Candidates != 2 {
		t.Errorf("expected drop with candidates=2, got %+v", b2.Drops)
	}
}
