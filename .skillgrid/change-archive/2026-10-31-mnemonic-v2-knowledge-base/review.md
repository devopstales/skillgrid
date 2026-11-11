# Review — 2026-10-31-mnemonic-v2-knowledge-base

> Change: `.skillgrid/change-specs/2026-10-31-mnemonic-v2-knowledge-base/` (moves to `.skillgrid/change-archive/2026-10-31-mnemonic-v2-knowledge-base/` at ship)
> Generated: 2026-11-11 (requesting-code-review)
> Diff range: `96e6516d..708ccd47` (reviewed base `96e6516d..1576d788`; triage fix `708ccd47` appended)
> Rigor tier: T2 (per `_shared/planning/rigor-tiers.md`)
> Independence: B per axis — see `## Independence`
>
> The durable audit record of the code review. **requesting-code-review** writes this into
> the spec folder, next to `report.md` (which owns the QA gate). It is committed in the spec
> zone before code-zone work continues. **ship** archives it with the folder; **reflect**
> cites it for lineage. The three review axes are reported **side by side, never merged** —
> that separation is the point.

## What Important means

- **Critical** — must fix before merge. Security, data loss, broken functionality, a
  `SATISFIES` scenario whose RED evidence is missing or whose code path does not match the
  Given/When/Then. Blocks `ship`.
- **Important** — should fix before merge. Architecture drift from a term or an in-force
  ADR, a hard standard breach, a missing or partial requirement, a behavior the spec never
  asked for (scope creep). Unfixed Important issues do not proceed (per the skill's
  Red Flags).
- **Minor** — nice to have. Baseline smells, style, optimization, doc polish. These are the
  nits; they are capped and summarized, never one-per-line below the fold.

## Cap the nits

- **Max 5 Minor findings listed per axis.** Beyond that, collapse the rest into a count:
  "N further Minor items (style/smells) — see the raw reviewer report, not this file."
- A Minor finding that is actually a Critical dressed as polish gets re-labeled, not
  capped. Capping is for true nits, not for hiding a real issue behind a count.

## Do not report

- Anything tooling already enforces (lint, typecheck, format). If `gofmt` / the linter
  would catch it, it is not a finding here — it belongs to the QA gate's tooling pass.
- Informational context with no required action — put it in `FYI` at most, one line, and
  it does not count against any cap.

## Passes

### Standards

- **Worst issue (within this axis):** none Critical/Important — 10 Minor (4 fixed in `708ccd47`, 1 documented exception, 5 collapsed)
- **Findings:** 0 Critical / 0 Important / 10 Minor (capped to 5 listed)
- **Verdict:** met-with-fixes

### Spec

- **Worst issue (within this axis):** none — 27/27 acceptance scenarios PASS; residual gaps are 007 §2 scope omissions (SUGGESTION, W010), not requirement misses
- **Findings:** 0 Critical / 0 Important / 4 Minor (all listed; 1 fixed in `708ccd47`, 2 parked in W010, 1 already covered)
- **Verdict:** met-with-fixes

### Security

- **Worst issue (within this axis):** one Low — CWE-22 path-traversal oracle via vault-authored `code:` citation refs (fixed in `708ccd47` + `TestLintCodeCitationEscape`)
- **Findings:** 0 Critical / 0 High / 0 Medium / 1 Low / 0 Info
- **Verdict:** secure-with-fixes

## Findings

### Critical

- none

### Important

- none

### Minor

- [Standards] `lint.go` — dead `root` param ("reserved for parity") — **fixed in 708ccd47** (param dropped; all callsites migrated)
- [Standards] `lint.go` + `status.go` — orphan threshold hard-coded twice (Go `<= 1` and SQL literal) — **fixed in 708ccd47** (`orphanInDegreeMax` single definition; status SQL embeds it)
- [Standards] `lint.go` — `time.Now()` in the `past_stale_after` check (code-standards hidden-dependency rule) — **deferred as a named exception**: `stale_after` is a wall-clock horizon (007 §2); both test sides are pinned (2020-01-01 past / no future fixture); documented inline
- [Standards] `lint.go` — `_ = json.Unmarshal` of the aliases payload un-justified (never-swallow rule) — **fixed in 708ccd47** (justifying comment: engine-written payload, per-page degradation)
- [Standards] `lint.go` + `cmd/mnemonic/projectwiki.go` — raw severity strings crossing the engine→CLI boundary (`--strict` compared `== "error"`) — **fixed in 708ccd47** (`SeverityError`/`SeverityWarn`/`SeverityInfo` constants; CLI compares `projectwiki.SeverityError`)
- (5 further Minor items collapsed — test-helper naming (`writeVaultPage` for a repo file), `lintCodes` collapse masking double-emission, `goPackages` walk-error swallow (defensible, advisory check), post-`LIMIT` thin assessment, commit `c582984d` bundling four spec slices; plus 1 FYI: pre-existing `ORDER BY MAX(bm25)` tie-break absence in base `96e6516d`)
- [Spec] 007 §2 scope omissions — `stale_index` compares stored body_hash vs stored body, not live files (the handler pre-syncs the index, so lag is unobservable at lint time); `--fix` implements `missing_title` only (disjoint from §4's fix list); `memory:`/`trajectory:` citations not resolved; orphan citations not checked; ungrounded's unresolvable-anchors sub-clause not checked; coverage gap at package level — **parked in W010** (SUGGESTION; non-blocking for the 27-scenario oracle)
- [Spec] bootstrap second-run idempotency not executed by a test (code-verified: sentinel → early return) — **parked in W010**
- [Spec] `TestReindexRebuildsProjectWikiPages` asserted only scanned/changed, not the derived state — **fixed in 708ccd47** (now pins aliases, in_degree, body_hash — the W003 root fix)
- [Spec] cite-source twice → recorded once: covered at the MCP level by `TestProjectWikiCiteSourceIdempotent` (handler test pins the idempotent row) — **no action**

## Independence

| Axis | Grade | Rationale |
|------|-------|-----------|
| Standards | B | fresh subagent (reviewer), no shared session history, same model family as the implementer |
| Spec | B | fresh subagent (reviewer), no shared session history, same model family |
| Security | B | fresh subagent (security-reviewer), no shared session history, same model family |

**Reading the grade:** **A** = fresh subagent, no shared history (independent). **B** = fresh
context but same model family / toolchain as the implementer (independent, weaker). **C** =
inline self-review or the implementer's verdict leaked in — **diagnostic only**, cannot stand
as the independent evidence that `ship` or a merge requires.

## Triage (receiving-code-review)

- **Fixed by code in `708ccd47`** (6): CWE-22 escape guard + test; severity constants; orphan constant; dead `root` param; aliases justifying comment; reindex derived-state test strengthening.
- **Deferred / parked:** 007 scope omissions + bootstrap second-run test → W010 (SUGGESTION); `time.Now()` → named inline exception; 5 collapsed Standards minors → raw reviewer report (no change planned).
- **Verification after fix:** full mnemonic suite green (`go test ./...`, exit 0), `go build` + `go vet` clean, gofmt clean.

## Verdict

- **Standards:** met-with-fixes
- **Spec:** met-with-fixes
- **Security:** secure-with-fixes
- **Floor (decides):** met-with-fixes
- **Worst issue (across all three axes):** the Low CWE-22 path-traversal oracle (fixed)
- **Unfixed Important count (must be 0 to proceed):** 0
