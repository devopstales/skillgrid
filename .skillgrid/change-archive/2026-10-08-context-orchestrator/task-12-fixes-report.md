# T12 (TASK-100) — Review Fixes (REWORK) — Report

Status: **DONE** — all 3 fixes applied, all 4 gates pass, committed.

Commit: **f92e97f5** — `fix(ctx): T12 review fixes — pathPrefix dir-boundary, cross-rule first-match-wins, sha256 identity`

Staged only:
- `mnemonic/internal/ctx/traffic/learner.go`
- `mnemonic/internal/ctx/traffic/learner_test.go`

(`mnemonic/mnemonic`, `evict.go`, and all unrelated shared-repo files were NOT staged.)

---

## Fix 1 (CRITICAL C1) — pathPrefix dir-boundary guard

File: `mnemonic/internal/ctx/traffic/learner.go`

- Loop (was ~lines 291-305): removed the `i == len(targets[0])` full-string
  acceptance; now only path-boundary candidates (`i > 0 && targets[0][i-1] == '/'`)
  are considered. New block at learner.go:302-318.
- Final guard (was line 307): replaced
  `if bestPrefix == "" { return nil }` with
  `if bestPrefix == "" || !strings.HasSuffix(bestPrefix, "/") { return nil }`
  (learner.go:325-327). Full-filename degenerate matches are rejected while real
  shared directory prefixes like `src/` still resolve.
- Evidence-collection loop left as-is (already filters by
  `strings.HasPrefix(e.Target, bestPrefix)`).

Tests added (`learner_test.go`):
- `TestExtract_PathPrefix_FileStringPrefixIsNotDirectory` (learner_test.go:104-124):
  NEGATIVE — file_read targets `["src.go", "src.go.bak"]` (2 distinct, MinTimes=2
  via a dedicated rule set) → `Extract` returns **0** candidates.
- `TestExtract_PathPrefix_DirectoryPrefixYieldsCandidate` (learner_test.go:126-148):
  POSITIVE — targets `["src/a.go", "src/b.go", "src/c.go"]` → exactly **1**
  candidate, architecture, memory text references prefix `src/`.
- Verified `TestExtract_NoMatchingRules` still passes (unchanged).

## Fix 2 (CRITICAL C2) — cross-rule first-match-wins (event consumption)

File: `mnemonic/internal/ctx/traffic/learner.go`

- `Extract` (learner.go:84-127): added a `consumed map[int]bool` (keyed by event
  Seq) persisting across the rule loop. Per candidate:
  1. below `MinConfidence` → skipped **without** consuming (matches "a memory
     below min_confidence is not stored");
  2. any evidence Seq already consumed → skipped (first matching rule already
     won);
  3. otherwise emitted, and all its evidence Seqs are marked consumed.
  Rules are applied in file order, so an earlier rule's emission prevents a
  later rule from reusing the same events.
- New helper `eventsConsumed` (learner.go:130-138): partial overlap (ANY shared
  Seq) = consumed = skip, for determinism.
- `Extract` doc comment now states the cross-rule consumption contract
  explicitly, including the MinConfidence scoping decision (learner.go:62-70).

Tests added (`learner_test.go`):
- `TestExtract_CrossRuleFirstMatchWins` (learner_test.go:150-173): two identical
  `messageRepetition` rules (rule[0] and rule[1]) over the same 2 repeated
  user_messages → total candidates == **1** (the first rule's count), not double.
  Asserts the contract directly without depending on shipped rules.yaml.
- `TestExtract_CrossRuleBelowFloorDoesNotConsume` (learner_test.go:175-197):
  first rule at conf 0.4 below a 0.6 floor is dropped and must NOT consume; the
  later 0.8 rule still emits (asserts the floor scoping).

## Fix 3 (IMPORTANT I2) — SHA-1 → SHA-256 (identity only)

File: `mnemonic/internal/ctx/traffic/learner.go`

- Import switched from `crypto/sha1` to `crypto/sha256` (learner.go:4).
- `memoryID` (learner.go:385-393): now
  `h := sha256.Sum256([]byte(sessionID + "\x00" + ruleID + "\x00" + key)); return hex.EncodeToString(h[:20])`
  — first 20 bytes → 40-hex output (same shape as before). Doc comment updated to
  "sha256 truncated to 20 bytes; deterministic identity, not security".
- Mechanical: no tests hardcode the hex; determinism/idempotency tests
  (`TestExtract_DeterministicIDs`, `TestExtract_RerunIdempotentViaStore`) re-pass,
  confirming identity still stable and dedupe still works.

---

## Gates (all PASS)

1. `go build ./...` (workdir `mnemonic/`) → ok (no output)
2. `go test ./internal/ctx/traffic/... -count=1` →
   `ok github.com/devopstales/skillgrid/mnemonic/internal/ctx/traffic 3.955s`
   (all 30 tests PASS, including the 4 new ones; `TestExtract_NoMatchingRules` PASS)
3. `go test ./internal/ctx/store/... -count=1` →
   `ok github.com/devopstales/skillgrid/mnemonic/internal/ctx/store 3.322s`
4. `go vet ./internal/ctx/...` → clean (no output; sha1 deprecation warning gone)

## New/changed tests (summary)

New (4), all in `mnemonic/internal/ctx/traffic/learner_test.go`:
- `TestExtract_PathPrefix_FileStringPrefixIsNotDirectory` (C1 negative)
- `TestExtract_PathPrefix_DirectoryPrefixYieldsCandidate` (C1 positive)
- `TestExtract_CrossRuleFirstMatchWins` (C2 contract)
- `TestExtract_CrossRuleBelowFloorDoesNotConsume` (C2 floor scoping)

Changed: none (existing tests untouched and re-passing).

## Unfixed findings

None. All three review findings (C1, C2, I2) are fixed. `evict.go` left
unchanged as instructed. No new dependencies added.
