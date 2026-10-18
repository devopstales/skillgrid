package memory

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// seedWebRow inserts a web_cache row (+ FTS via trigger) into the fixture store
// and returns its id. Used by the fused-recall tests (TICKET-06).
func seedWebRow(t *testing.T, fx *ownerFixture, project, source, title, content, url string, fetchedAt, expiresAt string) int64 {
	t.Helper()
	res, err := fx.st.DB.Exec(`
		INSERT INTO web_cache (project, source, cache_key, url, title, query, content, content_hash, fetched_at, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		project, source, fmt.Sprintf("%s-%s", source, title), url, title, "kumquat", content, "h", fetchedAt, expiresAt, fetchedAt)
	if err != nil {
		t.Fatalf("insert web_cache: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("web_cache last insert id: %v", err)
	}
	return id
}

// findWebHit returns the fused hit for a web_cache row (Corpus "web_cache").
func findWebHit(hits []SearchHit, webID int64) (SearchHit, bool) {
	for _, h := range hits {
		if h.Corpus == "web_cache" && h.WebCacheID == webID {
			return h, true
		}
	}
	return SearchHit{}, false
}

// TestMemSearch_FusedWebCache (TICKET-06): mem_search now fuses unexpired
// web_cache rows into the observation recall. An unexpired web row sharing a
// term surfaces labeled (corpus=web_cache + source/url/fetched_at/expires_at +
// retrieve pointer) and resolvable via web_cache_get; an expired row sharing
// the same term is absent; observations-only behavior is otherwise unchanged.
func TestMemSearch_FusedWebCache(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "")
	fx := newOwnerFixture(t, "fuse-web")
	ctx := context.Background()
	// Deterministic: the fused web hit must keep its fused position, not ride
	// the observation decay sort (ADR-0047).
	fx.svc.SetDecay(DecayConfig{Enabled: false})

	obsID, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "decision",
		Title: "kumquat decision", Content: "kumquat decision notes", Owner: fx.ownerA,
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	future := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	past := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	fetched := time.Now().UTC().Add(-time.Minute).Format(time.RFC3339)
	webID := seedWebRow(t, fx, "fuse-web", "fetch", "kumquat api", "kumquat api docs", "https://example.com/kumquat", fetched, future)
	expiredID := seedWebRow(t, fx, "fuse-web", "fetch", "kumquat stale", "kumquat stale page", "https://example.com/kumquat-stale", fetched, past)

	hits, err := fx.svc.SearchOwnerScopedBlend(ctx, fx.ownerA, "", "kumquat", "any", "", 10, Vector{})
	if err != nil {
		t.Fatalf("blend: %v", err)
	}

	// Observation hit present and unchanged in shape.
	if len(hits) == 0 || hits[0].Observation.ID != obsID {
		t.Fatalf("observation hit missing or reordered: %+v", hits)
	}
	if hits[0].Corpus != "observations" {
		t.Fatalf("observation hit corpus = %q, want observations", hits[0].Corpus)
	}

	// Unexpired web hit present + labeled.
	wh, ok := findWebHit(hits, webID)
	if !ok {
		t.Fatalf("unexpired web hit %d missing: %+v", webID, hits)
	}
	if wh.WebSource != "fetch" || wh.WebURL != "https://example.com/kumquat" ||
		wh.WebFetchedAt != fetched || wh.WebExpiresAt != future {
		t.Fatalf("web hit labels = %+v", wh)
	}
	if !strings.Contains(wh.Retrieve, "web_cache_get id="+fmt.Sprint(webID)) {
		t.Fatalf("web hit retrieve pointer = %q, want web_cache_get id=%d", wh.Retrieve, webID)
	}

	// Expired web row absent.
	if _, leaked := findWebHit(hits, expiredID); leaked {
		t.Fatalf("expired web row %d leaked: %+v", expiredID, hits)
	}

	// The retrieve pointer actually resolves: web_cache_get semantics are a
	// project-scoped SELECT by id (the MCP tool wraps this same query).
	var gotTitle, gotURL, gotExpires, gotFetched string
	err = fx.st.DB.QueryRow(`
		SELECT title, url, expires_at, fetched_at FROM web_cache WHERE id = ? AND project = ?`,
		webID, "fuse-web").Scan(&gotTitle, &gotURL, &gotExpires, &gotFetched)
	if err != nil {
		t.Fatalf("retrieve pointer does not resolve: %v", err)
	}
	if gotTitle != "kumquat api" || gotURL != "https://example.com/kumquat" ||
		gotExpires != future || gotFetched != fetched {
		t.Fatalf("retrieve pointer resolved to wrong row: %q %q %q %q", gotTitle, gotURL, gotExpires, gotFetched)
	}
}

// TestMemSearch_NoWebCache (TICKET-06): a project with no web_cache rows keeps
// the exact observations-only behavior (the no-regression contract).
func TestMemSearch_NoWebCache(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "")
	fx := newOwnerFixture(t, "fuse-none")
	ctx := context.Background()

	id, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "decision",
		Title: "durian stand", Content: "durian stand notes", Owner: fx.ownerA,
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	hits, err := fx.svc.SearchOwnerScopedBlend(ctx, fx.ownerA, "", "durian", "any", "", 10, Vector{})
	if err != nil {
		t.Fatalf("blend: %v", err)
	}
	if len(hits) != 1 || hits[0].Observation.ID != id || hits[0].Corpus != "observations" {
		t.Fatalf("no-web-cache behavior changed: %+v", hits)
	}
	if hits[0].MatchedVia != "keyword" {
		t.Fatalf("matched_via = %q, want keyword", hits[0].MatchedVia)
	}
}

// TestMemSearch_ExpiredExcluded (TICKET-06): when the ONLY web_cache rows
// sharing the term are expired, none of them surface.
func TestMemSearch_ExpiredExcluded(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "")
	fx := newOwnerFixture(t, "fuse-exp")
	ctx := context.Background()

	obsID, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "decision",
		Title: "longan note", Content: "longan note", Owner: fx.ownerA,
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	past := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	fetched := time.Now().UTC().Add(-time.Hour * 2).Format(time.RFC3339)
	seedWebRow(t, fx, "fuse-exp", "fetch", "longan one", "longan one", "https://example.com/l1", fetched, past)
	seedWebRow(t, fx, "fuse-exp", "fetch", "longan two", "longan two", "https://example.com/l2", fetched, past)

	hits, err := fx.svc.SearchOwnerScopedBlend(ctx, fx.ownerA, "", "longan", "any", "", 10, Vector{})
	if err != nil {
		t.Fatalf("blend: %v", err)
	}
	if len(hits) != 1 || hits[0].Observation.ID != obsID {
		t.Fatalf("expired web rows surfaced: %+v", hits)
	}
}

func TestOwnerScopedBlendKeywordFloor(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "")
	fx := newOwnerFixture(t, "blend-kw")
	ctx := context.Background()
	id, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "decision",
		Title: "banana stand", Content: "banana stand notes", Owner: fx.ownerA,
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	hits, err := fx.svc.SearchOwnerScopedBlend(ctx, fx.ownerA, "", "banana", "any", "", 10, Vector{})
	if err != nil {
		t.Fatalf("blend: %v", err)
	}
	if len(hits) != 1 || hits[0].Observation.ID != id {
		t.Fatalf("hits = %+v", hits)
	}
	if hits[0].MatchedVia != "keyword" {
		t.Fatalf("matched_via = %q", hits[0].MatchedVia)
	}
	if hits[0].Signals.Vector != 0 || hits[0].Signals.Entity != 0 {
		t.Fatalf("signals = %+v", hits[0].Signals)
	}
	if hits[0].Signals.Keyword <= 0 || hits[0].Signals.Keyword > 1 {
		t.Fatalf("keyword signal out of range: %v", hits[0].Signals.Keyword)
	}
}

func TestOwnerScopedBlendHidesPrivate(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "")
	fx := newOwnerFixture(t, "blend-priv")
	ctx := context.Background()
	if _, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "decision",
		Title: "secret banana", Content: "secret banana", Owner: fx.ownerA,
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	hits, err := fx.svc.SearchOwnerScopedBlend(ctx, fx.ownerB, "", "banana", "any", "", 10, Vector{})
	if err != nil {
		t.Fatalf("blend: %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("private row leaked: %+v", hits)
	}
}

func TestOwnerScopedBlendHybrid(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "1")
	fx := newOwnerFixture(t, "blend-hyb")
	ctx := context.Background()
	lexID, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "decision",
		Title: "lexical banana", Content: "lexical banana", Owner: fx.ownerA,
	})
	if err != nil {
		t.Fatalf("save lex: %v", err)
	}
	semID, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "decision",
		Title: "unrelated topic", Content: "unrelated topic", Owner: fx.ownerA,
	})
	if err != nil {
		t.Fatalf("save sem: %v", err)
	}
	if err := fx.svc.SetEmbedding(ctx, lexID, EncodeVector(Vector{Data: []float32{0, 1}}), "test"); err != nil {
		t.Fatalf("embed lex: %v", err)
	}
	if err := fx.svc.SetEmbedding(ctx, semID, EncodeVector(Vector{Data: []float32{1, 0}}), "test"); err != nil {
		t.Fatalf("embed sem: %v", err)
	}
	hits, err := fx.svc.SearchOwnerScopedBlend(ctx, fx.ownerA, "", "banana", "any", "", 10, Vector{Data: []float32{1, 0}})
	if err != nil {
		t.Fatalf("blend: %v", err)
	}
	via := map[int64]string{}
	for _, h := range hits {
		via[h.Observation.ID] = h.MatchedVia
		for _, v := range []float64{h.Signals.Keyword, h.Signals.Vector, h.Signals.Recency, h.Signals.Entity, h.Signals.Decay, h.Signals.Importance} {
			if v < 0 || v > 1 {
				t.Fatalf("signal out of range on %d: %+v", h.Observation.ID, h.Signals)
			}
		}
	}
	if via[lexID] != "hybrid" {
		t.Fatalf("lexical hit matched_via = %q, want hybrid", via[lexID])
	}
	if via[semID] != "vector" {
		t.Fatalf("semantic hit matched_via = %q, want vector", via[semID])
	}
}

// TestDecayDisabledKeepsBM25 is TICKET-03: with decay explicitly off, blend
// order matches SearchOwnerScoped (BM25) on an equivalent pair. The hot row
// would outrank the cold row if decay ran; it must not.
func TestDecayDisabledKeepsBM25(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "")
	ctx := context.Background()
	seen := time.Now().UTC().Add(-40 * 24 * time.Hour).Format(time.RFC3339)

	seed := func(t *testing.T, project string) (fx *ownerFixture, coldID, hotID int64) {
		t.Helper()
		fx = newOwnerFixture(t, project)
		var err error
		coldID, err = fx.svc.Save(ctx, SaveInput{
			SessionID: fx.sessionID, Type: "decision",
			Title: "banana banana banana", Content: "banana banana banana", Owner: fx.ownerA,
		})
		if err != nil {
			t.Fatalf("save cold: %v", err)
		}
		hotID, err = fx.svc.Save(ctx, SaveInput{
			SessionID: fx.sessionID, Type: "decision",
			Title: "banana", Content: "banana", Owner: fx.ownerA,
		})
		if err != nil {
			t.Fatalf("save hot: %v", err)
		}
		if _, err := fx.svc.DB().Exec(`UPDATE observations SET retrieval_usage = 0, last_seen_at = ? WHERE id = ?`, seen, coldID); err != nil {
			t.Fatalf("stamp cold: %v", err)
		}
		if _, err := fx.svc.DB().Exec(`UPDATE observations SET retrieval_usage = 20, last_seen_at = ? WHERE id = ?`, seen, hotID); err != nil {
			t.Fatalf("stamp hot: %v", err)
		}
		return fx, coldID, hotID
	}

	plain, _, _ := seed(t, "decay-off-bm25")
	raw, err := plain.svc.SearchOwnerScoped(ctx, plain.ownerA, "", "banana", "any", "", 10)
	if err != nil {
		t.Fatalf("bm25: %v", err)
	}
	if len(raw) != 2 {
		t.Fatalf("bm25 hits = %d, want 2", len(raw))
	}

	fx, coldID, hotID := seed(t, "decay-off-blend")
	fx.svc.SetDecay(DecayConfig{Enabled: false})
	hits, err := fx.svc.SearchOwnerScopedBlend(ctx, fx.ownerA, "", "banana", "any", "", 10, Vector{})
	if err != nil {
		t.Fatalf("blend: %v", err)
	}
	if len(hits) != len(raw) {
		t.Fatalf("blend hits = %d, bm25 hits = %d", len(hits), len(raw))
	}
	for i := range raw {
		if hits[i].Observation.Title != raw[i].Title {
			t.Fatalf("decay reordered hit %d: blend %q, bm25 %q", i, hits[i].Observation.Title, raw[i].Title)
		}
	}
	if hits[0].Observation.ID != coldID {
		t.Fatalf("first hit = %d, want cold %d (hot %d must stay second)", hits[0].Observation.ID, coldID, hotID)
	}
}

// TestOwnerScopedBlendEmbedderError is the caller contract for an embedder
// failure: mem_search drops the error and passes an empty query vector.
// Embedding stays enabled so a non-empty vector would have taken the hybrid
// leg; the empty vector must still return hits as keyword.
func TestOwnerScopedBlendEmbedderError(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "1")
	fx := newOwnerFixture(t, "blend-emb-err")
	ctx := context.Background()
	id, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "decision",
		Title: "banana stand", Content: "banana stand notes", Owner: fx.ownerA,
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	hits, err := fx.svc.SearchOwnerScopedBlend(ctx, fx.ownerA, "", "banana", "any", "", 10, Vector{})
	if err != nil {
		t.Fatalf("blend: %v", err)
	}
	if len(hits) != 1 || hits[0].Observation.ID != id {
		t.Fatalf("hits = %+v", hits)
	}
	if hits[0].MatchedVia != "keyword" {
		t.Fatalf("matched_via = %q", hits[0].MatchedVia)
	}
	if hits[0].Signals.Vector != 0 {
		t.Fatalf("signals.vector = %v", hits[0].Signals.Vector)
	}
}

// blendVecSave saves an embedded observation and returns its id.
func blendVecSave(t *testing.T, fx *ownerFixture, title, content, scope string, vec []float32) int64 {
	t.Helper()
	ctx := context.Background()
	id, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "decision",
		Title: title, Content: content, Owner: fx.ownerA, Scope: scope,
	})
	if err != nil {
		t.Fatalf("save %q: %v", title, err)
	}
	if err := fx.svc.SetEmbedding(ctx, id, EncodeVector(Vector{Data: vec}), "test"); err != nil {
		t.Fatalf("embed %q: %v", title, err)
	}
	return id
}

func blendHitIDs(hits []SearchHit) map[int64]string {
	out := map[int64]string{}
	for _, h := range hits {
		out[h.Observation.ID] = h.MatchedVia
	}
	return out
}

// TestBlendVectorLegDropsSuperseded: Get does not filter invalid_at, so the
// vector leg must. A superseded row (invalid_at in the past) with an embedding
// is absent while a live row with the same keywords still returns.
func TestBlendVectorLegDropsSuperseded(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "1")
	fx := newOwnerFixture(t, "blend-vec-superseded")
	ctx := context.Background()
	oldID := blendVecSave(t, fx, "papaya orchard old", "papaya orchard old notes", "", []float32{1, 0})
	liveID := blendVecSave(t, fx, "papaya orchard live", "papaya orchard live notes", "", []float32{1, 0})

	past := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	if _, err := fx.svc.DB().Exec(`UPDATE observations SET invalid_at = ?, status = ? WHERE id = ?`, past, StatusSuperseded, oldID); err != nil {
		t.Fatalf("supersede: %v", err)
	}

	hits, err := fx.svc.SearchOwnerScopedBlend(ctx, fx.ownerA, "", "papaya", "any", "", 10, Vector{Data: []float32{1, 0}})
	if err != nil {
		t.Fatalf("blend: %v", err)
	}
	got := blendHitIDs(hits)
	if _, leaked := got[oldID]; leaked {
		t.Fatalf("superseded observation %d leaked via vector leg: %+v", oldID, got)
	}
	if via, ok := got[liveID]; !ok || via != "hybrid" {
		t.Fatalf("live observation %d missing or wrong matched_via %q: %+v", liveID, via, got)
	}
}

// TestBlendVectorLegDropsExpired: a vector-only hit with expires_at in the
// past is dropped; an unexpired embedded row with no keyword match still
// returns as matched_via=vector.
func TestBlendVectorLegDropsExpired(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "1")
	fx := newOwnerFixture(t, "blend-vec-expired")
	ctx := context.Background()
	expiredID := blendVecSave(t, fx, "stale kiwi", "stale kiwi notes", "", []float32{1, 0})
	liveID := blendVecSave(t, fx, "fresh mango", "fresh mango notes", "", []float32{1, 0})
	anchorID := blendVecSave(t, fx, "anchor guava", "anchor guava notes", "", []float32{0, 1})

	past := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	if err := fx.svc.SetExpiresAt(ctx, expiredID, past); err != nil {
		t.Fatalf("expire: %v", err)
	}

	hits, err := fx.svc.SearchOwnerScopedBlend(ctx, fx.ownerA, "", "guava", "any", "", 10, Vector{Data: []float32{1, 0}})
	if err != nil {
		t.Fatalf("blend: %v", err)
	}
	got := blendHitIDs(hits)
	if _, leaked := got[expiredID]; leaked {
		t.Fatalf("expired observation %d leaked via vector leg: %+v", expiredID, got)
	}
	if got[liveID] != "vector" {
		t.Fatalf("live vector-only observation %d matched_via = %q, want vector: %+v", liveID, got[liveID], got)
	}
	if _, ok := got[anchorID]; !ok {
		t.Fatalf("keyword anchor %d missing: %+v", anchorID, got)
	}
}

// TestBlendVectorLegHonorsScope: with a non-empty request scope, a vector-only
// hit from a different scope is dropped; an empty scope keeps both.
func TestBlendVectorLegHonorsScope(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "1")
	fx := newOwnerFixture(t, "blend-vec-scope")
	ctx := context.Background()
	projID := blendVecSave(t, fx, "scoped lychee", "scoped lychee notes", "project", []float32{1, 0})
	persID := blendVecSave(t, fx, "personal plum", "personal plum notes", "personal", []float32{1, 0})
	anchorID := blendVecSave(t, fx, "anchor guava", "anchor guava notes", "project", []float32{0, 1})

	hits, err := fx.svc.SearchOwnerScopedBlend(ctx, fx.ownerA, "", "guava", "any", "project", 10, Vector{Data: []float32{1, 0}})
	if err != nil {
		t.Fatalf("blend scoped: %v", err)
	}
	got := blendHitIDs(hits)
	if _, leaked := got[persID]; leaked {
		t.Fatalf("out-of-scope observation %d leaked via vector leg: %+v", persID, got)
	}
	if _, ok := got[projID]; !ok {
		t.Fatalf("in-scope vector-only observation %d missing: %+v", projID, got)
	}
	if _, ok := got[anchorID]; !ok {
		t.Fatalf("anchor %d missing: %+v", anchorID, got)
	}

	hits, err = fx.svc.SearchOwnerScopedBlend(ctx, fx.ownerA, "", "guava", "any", "", 10, Vector{Data: []float32{1, 0}})
	if err != nil {
		t.Fatalf("blend unscoped: %v", err)
	}
	if _, ok := blendHitIDs(hits)[persID]; !ok {
		t.Fatalf("empty scope should keep personal observation %d: %+v", persID, blendHitIDs(hits))
	}
}

// TestBlendDecayRanksHighUsageAboveCold: with decay on, equal-BM25 rows are
// reordered so the high-usage row beats the cold row last seen 40 days ago.
// A decay-off baseline on an identical fixture keeps the cold row first, so
// the reorder is observed rather than assumed.
func TestBlendDecayRanksHighUsageAboveCold(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "")
	ctx := context.Background()
	seen := time.Now().UTC().Add(-40 * 24 * time.Hour).Format(time.RFC3339)

	seed := func(project string) (fx *ownerFixture, coldID, hotID int64) {
		fx = newOwnerFixture(t, project)
		var err error
		coldID, err = fx.svc.Save(ctx, SaveInput{
			SessionID: fx.sessionID, Type: "decision",
			Title: "banana alpha", Content: "banana alpha", Owner: fx.ownerA,
		})
		if err != nil {
			t.Fatalf("save cold: %v", err)
		}
		hotID, err = fx.svc.Save(ctx, SaveInput{
			SessionID: fx.sessionID, Type: "decision",
			Title: "banana bravo", Content: "banana bravo", Owner: fx.ownerA,
		})
		if err != nil {
			t.Fatalf("save hot: %v", err)
		}
		if _, err := fx.svc.DB().Exec(`UPDATE observations SET retrieval_usage = 0, last_seen_at = ? WHERE id = ?`, seen, coldID); err != nil {
			t.Fatalf("stamp cold: %v", err)
		}
		if _, err := fx.svc.DB().Exec(`UPDATE observations SET retrieval_usage = 20, last_seen_at = ? WHERE id = ?`, seen, hotID); err != nil {
			t.Fatalf("stamp hot: %v", err)
		}
		return fx, coldID, hotID
	}

	off, offCold, _ := seed("decay-rank-off")
	off.svc.SetDecay(DecayConfig{Enabled: false})
	hits, err := off.svc.SearchOwnerScopedBlend(ctx, off.ownerA, "", "banana", "any", "", 10, Vector{})
	if err != nil {
		t.Fatalf("blend off: %v", err)
	}
	if len(hits) != 2 || hits[0].Observation.ID != offCold {
		t.Fatalf("decay off baseline: want cold %d first, got %+v", offCold, blendHitIDs(hits))
	}

	on, _, onHot := seed("decay-rank-on")
	on.svc.SetDecay(DefaultDecayConfig())
	hits, err = on.svc.SearchOwnerScopedBlend(ctx, on.ownerA, "", "banana", "any", "", 10, Vector{})
	if err != nil {
		t.Fatalf("blend on: %v", err)
	}
	if len(hits) != 2 {
		t.Fatalf("decay on: hits = %d, want 2", len(hits))
	}
	if hits[0].Observation.ID != onHot {
		t.Fatalf("decay on: first hit = %d, want high-usage %d", hits[0].Observation.ID, onHot)
	}
}
