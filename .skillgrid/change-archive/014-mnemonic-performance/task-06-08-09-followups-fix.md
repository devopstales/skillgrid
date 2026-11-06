# Archive-Gate Follow-ups Fix — `014-mnemonic-performance` (F1/F2/F3)

**Scope:** the three "fix before archive" findings from the step 06 / 08 / 09 reviews.
All fixes are in `memory` / `service` / `embedder`. Each fix is a targeted change + a test.
No schema changes. HEAD stays on `feat/v2-structure`.

| Finding | File(s) | Commit |
|---------|---------|--------|
| F1 (step 06 M1) | `service/service.go` (+ `service/resolve_embedder_test.go`) | `1e263ee` |
| F2 (step 08 M1) | `memory/retrieve.go` (+ `memory/improve_test.go`) | `ea2d893` |
| F3 (step 09 M1) | `memory/session_promotion.go` (+ `memory/session_promotion_test.go`) | `788d7be` |

---

## F1 — `resolveEmbedder` migrated to `BuildFromConfig` (step 06 M1)

**Problem:** `resolveEmbedder` (the runtime embedder selection path, used at
`service.go:1716/1744/1841/1917`) was NOT updated in step 06. It still used the old
`switch cfg.Embedder.Provider` (external / off / default→`*Onnx`), so a config with
`provider: ollama` or `provider: local` was silently mapped to `*Onnx` at every call site,
while `embedder.Default()` (which step 06 updated to use `BuildFromConfig`) returned the
correct type. The two selection paths disagreed.

**Fix:** delegate `resolveEmbedder` to `embedder.BuildFromConfig` — the single
config-driven selector that already routes `ollama`/`local`/`onnx`/`external`/`off` and
degrades a provider that fails to load to the Null Adapter (06.4). Both paths now agree.

Quoted change (`service/service.go:1936`):

```go
// resolveEmbedder builds the process embedder from config by delegating to
// embedder.BuildFromConfig (014 step 06 M1). BuildFromConfig is the single
// config-driven selector that also routes the ollama and local providers, so
// the runtime selection path (this function) and the process default
// (embedder.Default) agree: a config provider of ollama/local resolves the
// matching embedder here too, instead of silently degrading to *Onnx.
//
// BuildFromConfig degrades a provider that fails to load to the Null Adapter
// (06.4) — a non-nil embedder with a zero-length vector — and returns Null for
// off/""/unknown providers. Callers keep their nil checks (a nil embedder
// still means the FTS+signals floor), and the nil-or-zero-length gate already
// skips the vector leg for both nil and Null, so the off path degrades to the
// FTS+signals floor exactly as before.
func resolveEmbedder(configRoot string) embedder.Embedder {
	cfg := config.Load(configRoot)
	return embedder.BuildFromConfig(embedder.EmbedderConfig{
		Provider:  cfg.Embedder.Provider,
		Dimension: cfg.Embedder.Dimension,
		Indexing:  toAsym(cfg.Embedder.Indexing),
		Query:     toAsym(cfg.Embedder.Query),
		BaseURL:   cfg.Embedder.BaseURL,
		Model:     cfg.Embedder.Model,
		APIKey:    cfg.Embedder.APIKey,
		ModelDir:  cfg.Embedder.ModelDir,
	})
}
```

(Replaces the prior `switch cfg.Embedder.Provider { case "external" / case "off","" / default: *Onnx }`.)

**Test** (`service/resolve_embedder_test.go`, 3 tests; one quoted — the ollama route):

```go
// TestResolveEmbedderRoutesOllama covers 014 step 06 M1: resolveEmbedder (the
// runtime selection path) must route to the ollama embedder, agreeing with
// embedder.Default (which selects via embedder.BuildFromConfig). Pre-fix it
// silently mapped provider:ollama to *Onnx (the default branch), so the two
// selection paths disagreed. With a live mock server, BuildFromConfig's eager
// Verify probe succeeds and the *Ollama embedder is returned.
func TestResolveEmbedderRoutesOllama(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"nomic-embed-code","embedding":[0.1,0.2,0.3]}`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	if err := osMkdirConfigD(dir); err != nil {
		t.Fatalf("mkdir config.d: %v", err)
	}
	body := "mnemonic:\n  embedder:\n    provider: ollama\n    base_url: " + srv.URL + "\n    model: nomic-embed-code\n    dimension: 3\n"
	if err := osWriteConfigYAML(dir, body); err != nil {
		t.Fatalf("write indexing.yaml: %v", err)
	}

	e := resolveEmbedder(dir)
	if e == nil {
		t.Fatalf("provider=ollama: resolveEmbedder returned nil, want *Ollama")
	}
	if _, ok := e.(*embedder.Ollama); !ok {
		t.Fatalf("provider=ollama: resolveEmbedder → %T, want *Ollama", e)
	}
}
```

`TestResolveEmbedderRoutesLocal` (provider:local + model file present → `*LocalONNX` or Null
degrade, never `*Onnx`) and `TestResolveEmbedderOffIsNull` (provider:off → Null, never `*Onnx`)
complete the coverage.

**RED→GREEN evidence:** The F1 code + test were already present in the working tree (uncommitted)
when I started; the test file had a compile error — `osWriteFile(modelDir, "test-model.onnx", …)`
called with 3 args where the helper takes 2. I fixed the call to
`osWriteFile(filepath.Join(modelDir, "test-model.onnx"), …)`.

- RED (pre-compfix): `go test ./internal/mnemonic/service/ -run 'TestResolveEmbedder'` →
  `build failed: too many arguments in call to osWriteFile`.
- GREEN (post-compfix): `--- PASS: TestResolveEmbedderRoutesOllama / TestResolveEmbedderRoutesLocal /
  TestResolveEmbedderOffIsNull` (the local case logs the expected
  `warning: embedder provider "local" failed to load: … dlopen(libonnxruntime.dylib …) (degrading to null)`
  and passes via the Null degrade).

---

## F2 — `improve()` wired into the fact-mode leg (step 08 M1)

**Problem:** `improve()` was only called from `SearchOwnerScoped`. The fact-mode leg
(`SearchOwnerScopedRetrieveFTS` → `rrfFallbackOwnerScopedFTS` → `BlendedSearch`) returned its
ranked slice without passing through `improve()`, so the `SetImprove` wired in
`service/retrieval.go` had **zero effect** on the ordering `mem_context`/`mem_timeline`
(fact mode) return — a dead hook with a comment claiming otherwise.

**Fix:** in `rrfFallbackOwnerScopedFTS`, pass the full pre-truncate `visible` slice through
`s.improve(...)` before `layerHitsFromObservations` (`memory/retrieve.go:253`). The re-rank
applies to the full slice (the leg fetches `limit*3`), and is a no-op when improve is disabled
(the default) or unwired, preserving byte-identity.

Quoted change (`memory/retrieve.go`):

```go
		if ok {
			visible = append(visible, o)
		}
	}
	// Self-improvement feedback loop (014 step 08 M1): the fact-mode leg must
	// re-rank through the same improve() hook SearchOwnerScoped uses, so the
	// opt-in is consistent across every search surface. It applies to the full
	// pre-truncate slice (the leg fetched limit*3), and is a no-op when improve
	// is disabled (the default) or unwired (nil improveCfg → returns input
	// unchanged), so the byte-identity property holds.
	return layerHitsFromObservations(s.improve(ctx, visible), limit), nil
}
```

**Test** (`memory/improve_test.go`):

```go
// TestImproveWiredIntoFactModeLeg covers 014 step 08 M1: the fact-mode search
// leg (SearchOwnerScopedRetrieveFTS → rrfFallbackOwnerScopedFTS → BlendedSearch)
// must pass its ranked slice through improve(), agreeing with the
// SearchOwnerScoped wiring. Pre-fix the leg returned the raw BlendedSearch
// order and the SetImprove hook had zero effect on fact-mode ordering.
func TestImproveWiredIntoFactModeLeg(t *testing.T) {
	fx := newImproveFixture(t, "improve-factleg")
	ctx := context.Background()

	coldID := seedImproveObs(t, fx, "factcold cold asset", "factcold factcold factcold filler body one", fx.ownerA, 0, 1*time.Hour)
	hotID := seedImproveObs(t, fx, "fakthot hot asset", "fakthot filler body two", fx.ownerA, 100, 1*time.Hour)
	midID := seedImproveObs(t, fx, "factmid mid asset", "factmid filler body three", fx.ownerA, 10, 1*time.Hour)

	raw, err := rawSQLSearch(ctx, fx, "factcold fakthot factmid")
	if err != nil {
		t.Fatalf("raw SQL search: %v", err)
	}
	if len(raw) != 3 {
		t.Fatalf("expected 3 raw hits, got %d", len(raw))
	}
	rawIDs := ids(raw)
	rawUsage := usageOrders(raw)
	if slices.Equal(rawUsage, []int{100, 10, 0}) {
		t.Fatalf("test setup broken: raw order is already usage-desc (%v); ...", rawUsage)
	}

	hits, err := fx.svc.SearchOwnerScopedRetrieveFTS(ctx, fx.ownerA, "fact", "factcold fakthot factmid", "any", 10)
	if err != nil {
		t.Fatalf("fact-mode search: %v", err)
	}
	if len(hits) != 3 {
		t.Fatalf("expected 3 fact-mode hits, got %d", len(hits))
	}
	gotIDs := make([]int64, len(hits))
	for i, h := range hits {
		gotIDs[i] = h.TargetID
	}
	wantIDs := []int64{hotID, midID, coldID}
	if !slices.Equal(gotIDs, wantIDs) {
		t.Fatalf("fact-mode leg must re-rank by retrieval usage desc via improve(): raw %v, got %v, want %v",
			rawIDs, gotIDs, wantIDs)
	}
	if slices.Equal(gotIDs, rawIDs) {
		t.Fatalf("fact-mode leg returned the raw SQL order; improve() was not applied: %v", gotIDs)
	}
}
```

**RED→GREEN evidence:** RED established structurally — before the wiring the leg returned
`layerHitsFromObservations(visible, limit)` (the raw `BlendedSearch`/FTS order), so the leg's
order equals `rawIDs` (the no-improve SQL order) and the `slices.Equal(gotIDs, rawIDs)` /
`wantIDs` assertions would fail. GREEN:

```
=== RUN   TestImproveWiredIntoFactModeLeg
--- PASS: TestImproveWiredIntoFactModeLeg (2.44s)
```
(run under `-race` as part of the memory suite below).

**Note (context):** step 13.3 later replaced `SearchOwnerScoped`'s own `improve()` call with
`importanceRerank` (importance_score instead of binary retrieval_usage). `improve()` remains
the opt-in gate (`mnemonic.improve.enabled`) and is the re-rank the step-08 brief specified; the
fact-mode leg now calls it, exactly as the finding prescribed ("pass the slice through
`improve()` (if it's set)"). `importanceRerank` is NOT applied in the fact-mode leg (out of scope
for this finding).

---

## F3 — session-keyed promotion dedup (step 09 M1)

**Problem:** `upsertPromotedObservation` deduped on
`WHERE project = ? AND type = 'session_log' AND title = ? AND content = ?`. The session summary
can CHANGE between two `SessionEnd` calls, so the `content = ?` clause missed and a **second**
node observation was inserted (1 symbol, 2 node observations, 2 FTS rows).

**Fix:** key the dedup on the session-stable `(project, type='session_log', title)` (the title
encodes the session id: `[session <id>] Session summary`), and `UPDATE` the `content` in place so
the FTS update trigger reindexes the node with the latest summary. Promotion is now idempotent
per session regardless of content.

Quoted new query + update (`memory/session_promotion.go`):

```go
	var existingID int64
	err := db.QueryRowContext(ctx, `
		SELECT id FROM observations
		WHERE project = ? AND type = 'session_log' AND title = ?
		  AND deleted_at IS NULL
		ORDER BY id LIMIT 1`,
		s.projectID, title,
	).Scan(&existingID)
	if err == nil {
		if _, err := db.ExecContext(ctx, `
			UPDATE observations SET content = ?, last_seen_at = ?, updated_at = ? WHERE id = ?`,
			content, now, now, existingID); err != nil {
			return 0, fmt.Errorf("touch promoted observation: %w", err)
		}
		return existingID, nil
	}
```

(Replaces `WHERE … AND title = ? AND content = ?` + `UPDATE … SET last_seen_at = ?, updated_at = ?`.)

**Why title-keyed (not `session_id`-keyed):** the review's suggested query
`WHERE session_id = ? AND type = 'session_promotion'` names a type that does not exist (the node
is `type='session_log'`, with `source='session_promotion'`), and the `type='session_log'` +
`session_id` pair could in principle match a hand-saved session-log observation. The title is the
dedicated, session-stable node marker (it already appears in `sessionObservationIDs`' exclusion
filter) and the review explicitly sanctioned it ("or `AND title = ?` without the `content = ?`
clause — and `UPDATE … SET content = ?` in place"). `ORDER BY id LIMIT 1` resolves to the node
node on a re-trigger.

**Test** (`memory/session_promotion_test.go`) — the previously-untested changed-summary case:

```go
// TestSessionEndIdempotentPromotionChangedSummary covers 014 step 09 M1:
// re-triggering SessionEnd for the same session with a CHANGED summary must
// reuse the existing node observation (updating its content in place) rather
// than inserting a second one. Pre-fix the dedup was content-keyed, so a
// changed summary missed the dedup and created a second node.
func TestSessionEndIdempotentPromotionChangedSummary(t *testing.T) {
	fx := newFixture(t, "mem-promo")
	ctx := context.Background()
	db := fx.st.DB

	if _, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessID, Type: "decision",
		Title: "Chose atomic rotation", Content: "refresh tokens must rotate atomically",
	}); err != nil {
		t.Fatalf("save A: %v", err)
	}
	if _, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessID, Type: "discovery",
		Title: "Found the race", Content: "the refresh path raced on concurrent swaps",
	}); err != nil {
		t.Fatalf("save B: %v", err)
	}

	// 1) Promote with summary X.
	if err := fx.svc.PromoteSession(ctx, fx.sessID, promotionSummary); err != nil {
		t.Fatalf("first end (summary A): %v", err)
	}
	obsID := promotionObservationID(t, db, fx.sessID)
	if obsID == 0 {
		t.Fatalf("expected a promoted node after the first end")
	}
	var contentA string
	if err := db.QueryRow(`SELECT content FROM observations WHERE id = ?`, obsID).Scan(&contentA); err != nil {
		t.Fatalf("read node content A: %v", err)
	}
	if contentA != promotionSummary {
		t.Fatalf("node content must be the first summary, got %q", contentA)
	}

	// 2) Promote AGAIN with a DIFFERENT summary Y (also clears the threshold).
	if err := fx.svc.PromoteSession(ctx, fx.sessID, promotionSummaryChanged); err != nil {
		t.Fatalf("second end (summary B): %v", err)
	}

	// 3) Exactly ONE node observation must exist.
	var nodeCount int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM observations
		 WHERE project = 'mem-promo' AND type = 'session_log'
		   AND title = ?`,
		"[session "+fx.sessID+"] Session summary",
	).Scan(&nodeCount); err != nil {
		t.Fatalf("count node observations: %v", err)
	}
	if nodeCount != 1 {
		t.Fatalf("changed-summary re-trigger must not duplicate the node observation, got %d", nodeCount)
	}

	// 4) The existing node is reused (same id) and its content is refreshed.
	var contentB string
	if err := db.QueryRow(`SELECT content FROM observations WHERE id = ?`, obsID).Scan(&contentB); err != nil {
		t.Fatalf("read node content B: %v", err)
	}
	if contentB != promotionSummaryChanged {
		t.Fatalf("node content must be refreshed to the new summary, got %q", contentB)
	}
	if got := promotionObservationID(t, db, fx.sessID); got != obsID {
		t.Fatalf("re-trigger must reuse the existing node observation (got %d, want %d)", got, obsID)
	}

	// 5) Still exactly 1 graph symbol.
	if n := promotionNodeCount(t, db, fx.sessID); n != 1 {
		t.Fatalf("changed-summary re-trigger must keep exactly 1 graph symbol, got %d", n)
	}
}
```

`promotionSummaryChanged` is a second ≥100-rune, ≥1-`## ` summary distinct from `promotionSummary`.

**RED→GREEN evidence** (stashed the F3 production change, kept the new test):

- RED (content-keyed dedup):
  ```
  [mnemonic] session s1: promoted to graph node 3
  [mnemonic] session s1: promoted to graph node 4
      session_promotion_test.go:398: changed-summary re-trigger must not duplicate the node observation, got 2
  --- FAIL: TestSessionEndIdempotentPromotionChangedSummary (0.18s)
  ```
  (a second node, id 4, is created — exactly the review's failure mode.)
- GREEN (session-keyed dedup): `--- PASS: TestSessionEndIdempotentPromotionChangedSummary` (0.89s,
  also under `-race` in the memory suite below).

---

## Full relevant suite + output

```
$ go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1 -race
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory	261.166s

$ go test ./skillgrid-cli/internal/mnemonic/service/ -count=1 -race
--- FAIL: TestSearchObservationsAllParallelManyStores (163.15s)
    parallel_search_many_test.go:69: parallel search (60 stores) too slow: 2.336048466s (ceiling 780ms, waves=4 NumCPU=16 latency=15ms)
FAIL
FAIL	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service	506.600s

$ go test ./skillgrid-cli/internal/mnemonic/embedder/ -count=1
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/embedder	1.197s
```

**Service-suite caveat (pre-existing, environmental):** the only service failure is
`TestSearchObservationsAllParallelManyStores` — a wall-clock ceiling test (asserts 60 parallel
stores finish in ≤780ms). It is **unrelated to F1/F2/F3** (F1's `resolveEmbedder` change has no
search path; F2/F3 are in the `memory` package, not `service`). Confirmed pre-existing by
stashing all my changes and re-running on the clean baseline:

```
=== BASELINE (no F1/F2/F3) ===
--- FAIL: TestSearchObservationsAllParallelManyStores (197.12s)
    parallel_search_many_test.go:69: parallel search (60 stores) too slow: 2.294841875s (ceiling 780ms, waves=4 NumCPU=16 latency=15ms)
```

It fails identically with and without my changes (host under load / `-race` overhead). My
`resolveEmbedder` tests pass in isolation:

```
$ go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestResolveEmbedder' -count=1
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service	2.471s
```

## Baselines

```
$ go test ./skillgrid-cli/internal/mnemonic/store/... ./skillgrid-cli/internal/mnemonic/codeindex/... \
          ./skillgrid-cli/internal/mnemonic/route/... ./skillgrid-cli/internal/mnemonic/affected/... \
          ./skillgrid-cli/internal/mnemonic/community/... ./skillgrid-cli/internal/mnemonic/pdg/... -count=1
ok  	.../store		8.805s
ok  	.../codeindex	25.388s
ok  	.../route		1.945s
ok  	.../affected	15.217s
ok  	.../community	4.975s
ok  	.../pdg		6.959s
```

## Commits (on `feat/v2-structure`)

```
1e263ee fix(mnemonic): migrate resolveEmbedder to BuildFromConfig (014 step 06 M1)
ea2d893 fix(mnemonic): wire improve() into the fact-mode search leg (014 step 08 M1)
788d7be fix(mnemonic): key session-promotion dedup on session, not content (014 step 09 M1)
```

`git diff --stat 1e263ee^..788d7be` — 6 files, all mine:
```
skillgrid-cli/internal/mnemonic/memory/improve_test.go            |  63 +++++++++++
skillgrid-cli/internal/mnemonic/memory/retrieve.go                |   8 +-
skillgrid-cli/internal/mnemonic/memory/session_promotion.go       |  28 +++--
skillgrid-cli/internal/mnemonic/memory/session_promotion_test.go  |  99 ++++++++++
skillgrid-cli/internal/mnemonic/service/resolve_embedder_test.go  | 119 +++++++++++
skillgrid-cli/internal/mnemonic/service/service.go                |  45 +++----
```

No `git add -A`; pre-existing working-tree drift (`http/ui/*`, `http/*_test.go`, `ui/fonts/`,
`.agents/skills/sdd-tasks/SKILL.md`, `docs/plan/*`) left untouched.

## Concerns

1. **Service wall-clock test flakiness** (pre-existing, not mine): `TestSearchObservationsAllParallelManyStores`
   asserts a 780ms ceiling for 60 parallel stores and fails at ~2.3s under `-race` on this host, with or
   without these changes. Recommend a per-NumCPU or looser ceiling; not an archive blocker for these 3 fixes.
2. **F2 uses `improve()`, not `importanceRerank`:** the finding named `improve()` explicitly, and that is
   what the leg now calls. Step 13.3's `importanceRerank` is a distinct (later) re-rank used only by
   `SearchOwnerScoped`; the fact-mode leg does not apply it. If the intent was to make the fact-mode leg
   match step 13.3's importance re-rank as well, that is a separate follow-up.
3. **F3 dedup key = title, not `session_id`:** chosen because the node type is `session_log` (not
   `session_promotion`) and the title is the dedicated session-stable node marker (already used in
   `sessionObservationIDs`' exclusion). Behaviorally identical to a `session_id`-keyed query for the
   node; the content is refreshed in place and the FTS update trigger reindexes.
