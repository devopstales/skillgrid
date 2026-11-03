# Task 11 — Portable Export — Review

Reviewer: fresh, no prior context. Scope: step-11 diff (4 files). Tests re-run locally (all 3 pass).

## Verdict

**PASS WITH WARNINGS**

## Findings

### Major

**M1. Streaming is a lie — `json.Encoder.Encode` buffers the whole value, so OOM protection is illusory**
File: `skillgrid-cli/internal/mnemonic/memory/export.go:270-274`
```go
func WriteBundle(_ context.Context, bundle *ExportBundle, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(bundle)
}
```
`json.Encoder.Encode` is **not** incremental: it marshals the entire bundle into an internal `*bytes.Buffer` before writing the single result to `w` (encoding/json: "Encode appends the JSON encoding of v to the internal buffer and copies the buffer contents to the destination stream"). Two consequences, both contradicted by the comments at export.go:252-257, 266-269 ("writes each top-level value incrementally… the whole bundle is never materialized as a single []byte (the encoder flushes per-Encode)") and the mem.go:55-62 CLI comment:
- Peak memory ≈ bundle in memory + a complete JSON copy of it → exactly the "marshal first, write after" pipeline the brief wanted avoided.
- `w` receives **nothing until the end**. For stdout that means the user sees zero output for the entire export duration, and on `--file` failure mid-encode (disk full) the partial file is inconsistent.
The bundle-in-memory part (ExportProject returns `*ExportBundle`) is an acceptable, documented compromise per the brief's parenthetical — but the report and code comments claim more than is true.
Fix (any one): (a) correct the comments to say the *bundle* is in memory and the encoder adds a full-sized JSON buffer; (b) implement true streaming: write `{"format":...,"observations":[` then `enc.Encode(obs)` per record, `], "graph_edges": [` … — trivial with a shared encoder and gives real incremental output; (c) at minimum stream per-section (Encode each top-level slice separately).

### Minor

**m1. `--skip-embeddings` is not "no embedding fields" — it emits `"embeddings": []`, and the CLI test is tautological**
Files: `skillgrid-cli/cmd/skillgrid/mem.go:88-93`, `mem_export_test.go:224-232`
`bundle.Embeddings = bundle.Embeddings[:0]` reuses the backing array, so the JSON contains `"embeddings": []` (the field is not `omitempty` on `ExportBundle`, by design — see brief check 5). That is *data*-correct (no vectors), but the brief wording "verify the output JSON has no embedding fields" is not met, and the test only proves it:
```go
if strings.Contains(out, `"embeddings": [`) && !strings.Contains(out, `"embeddings": []`) {
```
`"embeddings": []` contains the substring `"embeddings": [`, so the guard only fires when a populated array is present. It would pass unchanged if the flag were a no-op that left `"embeddings": [...]` populated — no wait, it would fail in that case, but it would also pass if the field were emitted as `null` or the slice left non-nil-empty in any shape. The per-observation check (`"vector"` absent) is sound. Suggested tightening: unmarshal `out` and assert `len(b.Embeddings) == 0` and every `obs.Embeddings == nil`.
Fix: unmarshal-based assertions (as above); decide and document whether `--skip-embeddings` should omit the key (needs a pointer or a second struct type).

**m2. Import is not idempotent when the target project store pre-exists with data**
File: `skillgrid-cli/internal/mnemonic/memory/export.go:307-351` (files/symbols creation)
The doc comment promises "idempotent re-import" at the observation level, which is true via `INSERT OR IGNORE` on explicit ids (observations) and the composite unique (edges). But the support tables are keyed differently on import:
- Files: `INSERT OR IGNORE ... path 'imported/%d'` — the bundle does not export the source file's path, so the imported file is always a synthetic row. On a re-import into the same store this hits the existing `imported/N` row (idempotent, fine). But if the destination store already indexes real code (its own files/symbols with colliding integer ids), `INSERT OR IGNORE` on explicit `id` skips, and the imported edges then point at the destination's *pre-existing* symbol of that id — silently resolving to the wrong symbol. Roundtrip-into-fresh-store (the documented design) is safe; cross-import into a live store is a documented-elsewhere assumption, not a defect — but it deserves one line in the doc comment: "import is lossless only into a fresh/empty store; into a populated store, symbol ids are reused best-effort and may alias existing symbols."
Fix: add the caveat sentence to `ImportProject`'s doc comment (code behavior is acceptable as documented).

**m3. `duplicate_count`, `prompt_id`, and observation-level `embedding`/`embedding_model`/`embedding_created_at` are exported but not re-imported**
Files: `export.go:113,123` (scanned, `dups` and `promptID` never used); `008_obs_lifecycle.sql:54-56` (observation has its own `embedding BLOB` columns from a prior migration)
The observation SELECT scans `o.prompt_id` and `o.duplicate_count` and then discards both; a roundtrip through export/import zeroes `duplicate_count` and `prompt_id` on the imported copy, and the observation-level `embedding` BLOB (step 08's per-observation embedding path, distinct from the symbol bridge) is neither exported nor imported. The roundtrip test does not notice because `Save` leaves all of these at defaults. For a "portable" artifact this is silent data loss.
Fix: export `prompt_id`, `duplicate_count` in metadata (they are int; the metadata map is `map[string]string`, so stringify like `revision_count`) and restore them on import; either export the observation `embedding` BLOB as base64 metadata or document the omission as a v1 limitation in `ExportBundle`'s comment.

**m4. Dangling `graph_ref` after import is silently possible**
Files: `export.go:368-401` (observations inserted before/with symbols), doc 280-291
`graph_ref` is exported (✓) and restored (✓, test asserts it). But symbols are only recreated if they are referenced by *edges or embeddings*. An observation whose `graph_ref` symbol appears in neither gets an `INSERT OR IGNORE` observation pointing at a non-existent symbol id. `observations.graph_ref` has no FK (022_triple_store.sql:23 is a bare `INTEGER`), so the insert succeeds and the reference dangles. Not a crash, but the "portable" contract should say: graph_ref may be dangling if the referenced symbol has no edges/embeddings.
Fix: add that symbol id to `neededSymbols` (cheap: iterate `bundle.Observations` and union `GraphRef`) so the referenced symbol is always materialized — this makes graph_ref preservation *lossless* and removes the caveat entirely.

**m5. Embeddings export pass is skipped when no observation has a `graph_ref`**
File: `export.go:166` — `if len(obsByRef) > 0 {`
The symbol-embeddings query is gated on the existence of at least one observation↔symbol binding. A project with a healthy codeindex (embeddings, edges) but no observation referencing any symbol exports `bundle.Embeddings == nil` — the `embeddings` section is empty even though the store has vectors. The roundtrip test doesn't catch it because its fixture links an observation. This looks like an accidental coupling of "attach per-observation vectors" with "populate the top-level section" (the separate-pass refactor, per the report).
Fix: always run the embeddings pass; keep the per-observation attachment inside it.

### Nits

**n1.** `ExportRecord` doc comment (export.go:31-34) claims "The embeddings map is keyed by symbol id" — the actual key is the literal string `"vector"` (export.go:187-189). Misleading; fix the comment.

**n2.** `obsByRef[graphRef.Int64] = len(bundle.Observations) - 1` (export.go:153): if two observations share a `graph_ref`, the per-observation attachment wins to the *last* one only. Fine (best-effort is documented) — but "last wins" is worth a half-line in the comment.

**n3.** `atoi64orZero` (export.go:457-470) silently returns 0 for negatives/overflow; fine for the current fields, but `strconv.Atoi` + fallback is 2 lines and less surprising.

**n4.** `mem_export_test.go` seed: `seedExportProject` inserts an edge with `to_id = symID` (self-edge) — works, but a distinct target symbol would have exercised the `to_id`-omitted and present paths better.

**n5.** Brief interface drift (not a defect in code): the brief specifies `ExportProject(ctx, projectID)` / `ImportProject(ctx, projectID, reader)`; the implementation is `ExportProject(ctx)` / `ImportProject(ctx, r)` with the project bound to the `*Service` (see `New(st, projectID)`, service.go:271). This matches the existing per-project service pattern in the codebase and the CLI's `svc.Open(projID)` flow, so it is the *better* shape — noting it so the spec is updated, not the code.

**n6.** Report's JSON sample (report.md:78) shows `"confidence": "EXTRACTED"` as a string — correct; the schema is primitive-only as required.

## Evidence checked (the 9 checks)

1. **JSON schema portability** — PASS. Envelope `{"format":"cogx-v1","project","exported_at","observations","graph_edges","embeddings"}`; primitives only (int64→number, []byte→base64 string, timestamps RFC3339 strings). **Version field present**: `format: "cogx-v1"` (export.go:81) — detectable schema evolution. Field names clear snake_case. Verified by running `TestMemExportCLI` and the structure test; sample output shape matches the report's sample.
2. **Base64 roundtrip** — PASS, stronger than required. Encode: `base64.StdEncoding.EncodeToString(blob)` (export.go:182); decode: `base64.StdEncoding.DecodeString(em.Vector)` (export.go:434). The roundtrip test verifies the **decoded bytes are byte-identical** to the source BLOB (`if string(blob) != string(srcBlob)`, export_test.go:1125-1127), not merely non-empty; the structure test additionally checks `len(raw) == 4*dim` (export_test.go:972). Empty/nil BLOB: not explicitly tested, but `EncodeToString(nil)` → `""` and `DecodeString("")` → `[]byte{}` roundtrip cleanly, so the empty case is safe by construction (no special-casing needed, none present).
3. **Import idempotency** — PARTIAL. Documented choice: observations & edges & files & sessions & symbols = `INSERT OR IGNORE`; embeddings = `INSERT OR REPLACE` (latest wins, mirrors codeindex). Re-importing the same bundle into the same store is a no-op for observations/edges (explicit ids + composite unique) and stable for the rest. See m2 for the caveat about pre-existing non-empty stores.
4. **Streaming** — **FAIL on the letter, acceptable on the spirit if the claims are fixed** (M1). No `json.Marshal → []byte` in the CLI path (that part is true — it uses `json.NewEncoder(w)`), but `Encoder.Encode` still materializes the full JSON internally before writing, so peak memory is *worse* than marshal-then-write (both copies coexist) and output is all-or-nothing. The brief's parenthetical ("if the encoder writes incrementally to the file/stdout, that's acceptable") is satisfied by the *shape* of the code but not by the *semantics* the comments claim.
5. **--skip-embeddings** — PARTIAL. Top-level section emptied (`"embeddings": []`) and per-observation `embeddings` map nilled (omitted via `omitempty`) — data-correct, field-shape not "omitted". Test is weak/tautological (m1).
6. **Edge temporal fields** — PASS. `valid_from`/`valid_to` exported (export.go:224-243) and imported (export.go:420-428); roundtrip test asserts `valid_to` survives for the windowed edge (export_test.go:1099-1106). No data loss.
7. **graph_ref** — PASS with caveat. Exported per observation (export.go:151-154), restored on import (export.go:383-397), test asserts the bound id survives (export_test.go:1131-1138). Dangling-reference case not handled → m4.
8. **Additive-ness / tool contract** — PASS. Diff touches only `mem.go` (CLI switch + flags + usage text), `mem_export_test.go`, `export.go`, `export_test.go`. No MCP tool registrations, no `Save`/`Search` signature changes, no changes to existing tables (read-only SELECTs on export; standard INSERTs on import). `mem_*`/`code_*`/`web_*` contracts untouched.
9. **TDD discipline** — PASS (structure), GAP (counts). The three named tests exist and match RED→GREEN per the report; re-ran locally: `TestExportProjectStructure` PASS, `TestExportImportRoundtrip` PASS (5 obs/3 edges/2 emb, ids+content asserted), `TestMemExportCLI` PASS (stdout/file/skip-embeddings). However, the brief's 11.3.a item "verify streaming for large stores (no OOM)" has **no corresponding assertion** — the CLI test checks content only, and the "streaming" property is untestable against the current implementation anyway (see M1). Counts verified in the roundtrip test (5/3/2 exact).

## Approved to merge

**Yes — with M1 (fix or down-sell the streaming claims in comments/report) and m1 (tighten the skip-embeddings test) addressed before archive.** m2/m4 are one-line doc/code changes worth doing in the same fix round; m3 is a genuine portability gap to track (either fix or record as a v1 limitation in the change's known-issues).
