# Task 15 — Provenance chain metadata (review)

Change: `014-mnemonic-performance` · Step 15 (`15-provenance-tracking`) · FRESH reviewer

## Verdict: `PASS WITH WARNINGS`

The core contract is correct: the JSON roundtrip is clean, the migration is purely
additive, and immutability is enforced structurally (by omission, stronger than
COALESCE). The warnings are (1) an out-of-scope behavior change bundled into this
diff, (2) the 15.2 warning-logging hook that was deferred, and (3) an indentation
regression in the CLI dispatcher. None blocks the provenance feature itself.

## Findings

### [MINOR] `SearchOwnerScoped` TTL filter removed — out-of-scope behavior change
`skillgrid-cli/internal/mnemonic/memory/service.go:629` (diff hunk at service.go @@602)

The step-15 diff deletes a WHERE clause from the owner-scoped search path that is
unrelated to provenance:

```diff
 		FROM observations o
 		INNER JOIN observations_fts ON observations_fts.rowid = o.id
 		WHERE observations_fts MATCH ? AND o.deleted_at IS NULL AND o.project = ?`+scopeClause+`
-		  AND (o.expires_at IS NULL OR o.expires_at = '' OR strftime('%s', o.expires_at) > strftime('%s', 'now'))
 		  AND `+clause+`
```

That soft-exclude filter (added in step 04 TTL) was the reason `SearchOwnerScoped`
skipped expired rows. The sibling `SearchWithScope` still HAS it (service.go:571),
so the two search paths now disagree on whether expired rows are returned. The
implementer's report makes no mention of this, so it is either an accidental
drive-by or an intentional change that was not flagged. If intentional, it belongs
in its own task with a rationale + test; if accidental, restore the line.

- What's wrong: an owner-scoped `mem search` now returns soft-expired observations
  that `mem search` (default scope) filters out. Asymmetric search semantics.
- Fix: restore the line, or (if the intent was to honor `retrieval_usage`/visibility
  filtering instead) document it in the report and add a test pinning the new
  behavior.

### [MINOR] 15.2 "log a warning if a different provenance is attempted" not implemented
`skillgrid-cli/internal/mnemonic/memory/service.go:399` (topic-key upsert) + `timeline_update.go:103` (Update)

The brief (15.2.c) asks to "log a warning if a different provenance is attempted."
There is no such log at either update path. The upsert (service.go:399-408) and
`Update()` (timeline_update.go:103) both silently drop `in.Provenance`. The report
(Concerns §1) acknowledges this was deferred to the curation producer (step 05 LLM
extraction). That is defensible — no production call site sets `SaveInput.Provenance`
yet — but the test comment at `provenance_immutable_test.go:13` says "a warning is
logged," which is not true. The contract (original preserved) is enforced; only the
observable warning is missing.

- What's wrong: the test's doc-comment overstates what is verified ("a warning is
  logged" — nothing is logged).
- Fix: either (a) add a `log`/`t.Log`-style warning in the upsert when
  `in.Provenance != nil` and the existing row already has a non-NULL provenance, or
  (b) correct the comment to "the attempt is a no-op on the column (no warning yet;
  the producer logs it)."

### [NIT] Indentation regression in the mem CLI dispatcher
`skillgrid-cli/cmd/skillgrid/mem.go:89-92`

The `relations` and new `provenance` cases use two-space indents while the rest of
the switch uses tabs:

```go
	case "graph":
		runMemGraph(svc, projID, limit)
 	case "relations":
 		runMemRelations(svc, projID, pos, minConf)
 	case "provenance":
 		runMemProvenance(svc, projID, pos)
	case "expire":
```

`gofmt` is the source of truth; this compiles (verified: `go build` exit 0) but will
be rewritten by a gofmt pass. The same two-space pattern appears in the usage string
hunk. Not a behavior issue, but it is noise that will show up in the next gofmt run.

- Fix: run `gofmt -w` on `cmd/skillgrid/mem.go`.

### [NIT] `provenanceEqual` swallows marshal errors
`skillgrid-cli/internal/mnemonic/memory/provenance_immutable_test.go:140`

```go
func provenanceEqual(a, b *Provenance) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ab) == string(bb)
}
```

A failed marshal returns `""` for both, which compares equal — a false positive. In
practice `MarshalJSON` here never fails (it only marshals strings/slices), so this
is latent, not active. Not worth a fix; noted for completeness.

### [OK — not a finding] The 4th commit (`5f34108`) is a real fix, not cleanup
`skillgrid-cli/internal/mnemonic/memory/service.go:664,684` + `obsSelectCols:1764`

`obsSelectCols` now ends with `provenance` (service.go:1769) and `scanObservations`
scans it (service.go:1774-1776). Before 5f34108, `Get()`/`List` (the two callers of
`obsSelectCols` at service.go:664 and 684) SELECTed 23 columns but `scanObservations`
scanned 24 — a hard "sql: expected 24 destination arguments" arity error. So 5f34108
fixed a genuine bug the initial commit introduced. I confirmed ALL scanObservations-fed
SELECTs now match: `SearchWithScope` (service.go:567), `SearchOwnerScoped`
(service.go:626), `Get`/`List` via `obsSelectCols` (service.go:664/684),
`governance` admin list (governance.go:405,451), `CrossLinkQuery` (crosslink.go:30),
and the raw-SELECT test path (improve_test.go:330). Arity is consistent everywhere.

Note: `dream_rollback.go` (`ObservationSnapshot`) intentionally does NOT mirror
`provenance` — its SELECT (dream_rollback.go:52-56) is an explicit column list, not
`obsSelectCols`, so its arity is unchanged and self-consistent. Restoring an
observation after a failed dream therefore leaves provenance at its pre-distill
value (unaffected by the upsert's `ON CONFLICT DO UPDATE`, which lists no provenance
column). Correct, and immutability holds on the rollback path too.

## Evidence checked

1. **JSON roundtrip (15.1)** — VERIFIED. `Provenance` (provenance.go:19-24) has the
   four fields with the four snake_case tags. `MarshalJSON` (provenance.go:28-45)
   normalizes nil `SourceFiles` → `[]string{}` so the stored JSON always emits
   `source_files: []`, never `null`. `TestProvenanceChainStorage`
   (provenance_test.go) asserts all four fields round-trip (provenance_test.go:86-97)
   AND independently unmarshals the raw column into `map[string]any` and checks all
   four keys are present (provenance_test.go:61-75). No-provenance save reads back
   nil (provenance_test.go:99-135). Consistent: nil is normalized to `[]` on write
   and unmarshals back to a non-nil empty slice on read — there is no `null`
   anywhere in the stored form.

2. **Immutability (15.2 — CRITICAL)** — VERIFIED, enforced by omission.
   - The topic-key upsert (`Save()`, service.go:399-408) sets `source, session_id,
     type, title, content, normalized_hash, revision_count, last_seen_at,
     updated_at` — `provenance` is NOT in the SET clause. A re-save with a different
     (or absent) provenance leaves the stored chain intact.
   - `Update()` (timeline_update.go:25-115) builds its SET clause from `UpdateInput`
     (`{Title, Content, Type, TopicKey, Scope}` — timeline_update.go:14-20) only;
     `provenance` is not a field of `UpdateInput`, so the mem_update path cannot
     change or establish it.
   - I grepped every `UPDATE observations` in the package: none writes `provenance`.
     The only `ON CONFLICT` upsert that touches an observation is
     `dream_rollback.go:179` (restore), which lists no provenance column. So there is
     no other write path that can clobber the chain.
   - `TestProvenanceImmutability` (provenance_immutable_test.go) covers: (1a) upsert
     without provenance → preserved; (1b) upsert with a DIFFERENT provenance →
     original preserved byte-for-byte (json compare at line 102-114); (2) `Update()`
     leaves it intact; (3) a chain-less obs stays NULL after an update. The test
     genuinely exercises both real update paths (the topic-key upsert AND the
     mem_update path), not just one.
   - Caveat: the test scopes to the topic-key upsert (same row) and deliberately
     does NOT cover the 24h hash-dedup INSERT-new-row branch (report Concerns §3);
     a fresh row legitimately starts a fresh chain, which is per-row-immutability by
     construction. That is a reasonable scoping decision, documented, and not a gap
     in the "immutable once set" guarantee for a given row.

3. **Migration (15.1)** — VERIFIED. `027_provenance.sql` is a single
   `ALTER TABLE observations ADD COLUMN provenance TEXT;` — purely additive, nullable
   (TEXT defaults to NULL), no backfill. Pre-027 rows read back NULL. No existing
   schema touched.

4. **Save() population** — VERIFIED, but note it is NOT best-effort.
   `SaveInput.Provenance *Provenance` (service.go:213) is serialized in `Save()`
   (service.go:437-444) and passed as the last INSERT param (service.go:453). A
   marshal failure is a HARD error (`return 0, fmt.Errorf("serialize provenance: %w")`)
   — the brief's checklist asks "is it best-effort?"; the implementer chose the
   opposite (fail-loud), which is fine and arguably better, but is a deliberate
   deviation worth noting. `Observation.Provenance *Provenance`
   (service.go:280, `json:"provenance,omitempty"`) is parsed on read via
   `parseProvenanceColumn` (provenance.go:50-59): NULL/empty → nil; unparseable JSON
   → hard error (so a corrupt column is never silently an empty chain). `Get()`
   (service.go:707-712) and `scanObservations` (service.go:1774-1776) both decode it.

5. **CLI (15.3)** — VERIFIED. `runMemProvenance` (mem.go:598-649) handles:
   missing id → exit 2 (line 605); non-integer id → exit 2 (line 610); id not found
   (checked via `ObsTitle`, line 624) → exit 1 "observation N not found in this
   project" (line 625); no provenance → exit 0, prints the structured
   `{"observation_id": N, "provenance": "no provenance recorded..."}` (line 633);
   chain present → exit 0, pretty-printed structured JSON with the four snake_case
   labels (line 640). `TestMemProvenanceCLI` (mem_provenance_test.go) asserts all four
   chain values, all four labels, the no-provenance message, and the missing-id
   error. Note: the CLI prints *pretty-printed structured JSON*, not a
   whitespace-delimited "chain" string — the brief said "formatted as a readable
   chain (not raw JSON)." Pretty JSON with the four labels satisfies "readable, not
   raw single-line JSON," though it is not a literal `session_id → curate_command →
   source_files → llm_reasoning` chain rendering. Acceptable interpretation.

6. **Additive-ness / tool contract** — VERIFIED. `Observation` gains one
   `omitempty` pointer field (service.go:280), so all `mem_*`/`code_*`/`web_*` output
   shapes are unchanged when provenance is NULL (field absent). `SaveInput` gains one
   optional pointer field (service.go:213). No MCP tool signature changed. The
   default no-provenance path is unchanged (NULL column, field omitted). The one
   non-additive delta is the `SearchOwnerScoped` TTL-filter removal (Finding #1),
   which is a search-behavior change, not a shape change.

7. **TDD discipline** — VERIFIED. Three test files: (a) `provenance_test.go` — JSON
   roundtrip, all 4 fields + raw-column key check + no-provenance nil; (b)
   `provenance_immutable_test.go` — immutability across upsert-without, upsert-with-
   different, `Update()`, and chain-less-stays-NULL; (c) `mem_provenance_test.go` —
   the CLI (all four elements, no-provenance message, missing-id error). All three
   named tests exist and are the RED/GREEN evidence the report cites.

8. **4th commit (`5f34108`)** — VERIFIED as a real fix (see Finding [OK] above). It
   added `provenance` to `obsSelectCols` so the 24-column `scanObservations` no longer
   hits an arity mismatch in `Get()`/`List`. Not just cleanup.

## Approved to merge: **yes**

The provenance feature is correct and well-tested. Before merging, address Finding #1
(the `SearchOwnerScoped` TTL-filter removal — confirm intent; restore or document +
test) and #2 (deferred warning log — at minimum fix the over-stated test comment).
Findings #3/#4 are nits. None of the findings affect the immutability guarantee,
which is enforced structurally and holds on every write path including the dream
rollback.
