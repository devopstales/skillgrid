# Review Ledger — 2026-11-17-skillgrid-host-plugins

Resolution record per `_shared/craft/review-ledger.md`. Normalization
boundary: reviewer vocabulary (Blocker/Major/Nit) → CRITICAL/WARNING/
SUGGESTION happens only in the round entries below.

## Round 1 — 2026-11-18

**Frozen candidate:** base 3f033a21 → head add39cfa, 1 file, hash 887750e5ca0677eda32919e04d505c338331fbb8 (blueprint.md)
**Reviewer:** Goldfish review, Consolidated preset (fresh `reviewer` subagents; 5 stages)
**Independence grade:** A (fresh-context spawn, read-only, blank context, no network/MCP)
**Budget:** round 1/5 (tier T2)

### Stages

| Stage | Persona(s) | Report | Verdict |
|---|---|---|---|
| 5 | zoom-out | `review/zoom_out_report.md` | REVISE (5) → folded |
| 6 | comprehension | `review/comprehension_report.md` | REVISE (6) → folded |
| 7 | design-critic + structural + style | `review/critic_report.md` | REVISE (11) → folded |
| 8 | zoom-out (fresh) | `review/intermediate_zoom_out_report.md` | REVISE (5) → folded |
| 9 | implementation-readiness + codebase-details + style (+2 re-runs) | `review/implementation_readiness_report.md` | REVISE (20 raised; 12 fixed, 8 rejected) → folded; impl-readiness persona NOT VERIFIED (both passes unusable) |

**Harness deviation (Round 1):** the `reviewer` agent's backend
(`google-antigravity/gemini-4-hydrogen-sonar`) returned 429
RESOURCE_EXHAUSTED after 10 retries on both the initial spawn and the
one retry. Per the dispatch-protocol fallback (`fresh reviewer subagents
(fallback general)`), the persona was re-run on the local `flint` agent
under the identical dispatch contract (same persona brief, reading list,
four-part isolation, output schema). Independence grade A holds:
fresh-context spawn, read-only, blank context (no project instructions),
no network/MCP. The 429 was a backend-quota failure, not a persona or
findings issue.

### Findings

Severity normalized at this boundary (Blocker/Major → CRITICAL, Minor → WARNING, Nit → SUGGESTION). File:Line = the blueprint section at fix time (line numbers shifted across folds). Fix Commit `809adfd2` = the round-1 fix commit (all fixes landed there). Lineage = the stage report.

| ID | Severity | Location | Finding (short) | Disposition | Fix Commit | Lineage |
|---|---|---|---|---|---|---|
| F-S5-1 | CRITICAL | header Tech Stack / Per-host | repo moved (d5148778 tree deletions + WT `skill-pi`→`omp` rename); plan stale | FIXED | 809adfd2 | zoom_out_report |
| F-S5-2 | CRITICAL | Per-host omp row / Design contract | shipped `session-start.js` seam unevaluated; proposed omp extension redundant | FIXED | 809adfd2 | zoom_out_report |
| F-S5-3 | CRITICAL | Per-host pi/omp rows | row split conflated one host (both `~/.omp/agent/extensions`) | FIXED | 809adfd2 | zoom_out_report |
| F-S5-4 | WARNING | Must-Haves R4 / Task 2 | fail-open not anchored to the in-repo precedent | FIXED | 809adfd2 | zoom_out_report |
| F-S5-5 | SUGGESTION | Hypothesis / Design contract | context cost vs shipped full-content hook unweighed | FIXED | 809adfd2 | zoom_out_report |
| F-C6-1 | CRITICAL | Must-Haves | goal-backward method + Truths relationship never stated | FIXED | 809adfd2 | comprehension_report |
| F-C6-2 | CRITICAL | Terms + header | self-containment claim vs unsettled terms / external cross-refs | PARTIAL (cross-refs = normal blueprint behavior) | 809adfd2 | comprehension_report |
| F-C6-3 | WARNING | Tech Stack vs Architecture | injector count 4 vs 4 pointer rows unreconciled | FIXED | 809adfd2 | comprehension_report |
| F-C6-4 | WARNING | Must-Haves / Task 1 | acceptance scoped R1–R4 while surface runs R1–R6 | FIXED | 809adfd2 | comprehension_report |
| F-C6-5 | WARNING | Task 7 smoke | "router read before first action" not operationalized | FIXED | 809adfd2 | comprehension_report |
| F-C6-6 | SUGGESTION | header / §2 / Threat Matrix | over-long lines in critical sections | PARTIAL (table rows dense by construction — declined) | 809adfd2 | comprehension_report |
| F-1 | CRITICAL | Per-host omp / exception / R3,R4 | no evaluation record for the shipped hook; guard unplaced on the full-content path | FIXED | 809adfd2 | critic_report |
| F-2 | CRITICAL | Task 6 (installer) | installer design conditional on the in-flight ADR-0067 rework | FIXED | 809adfd2 | critic_report |
| F-3 | WARNING | Hypothesis / Task 7 | either/or smoke oracle; hypothesis's two clauses collapsed | FIXED | 809adfd2 | critic_report |
| F-4 | WARNING | Classification / Global Constraints | L3 not operationalized for TS/JS/bash artifacts | FIXED | 809adfd2 | critic_report |
| F-5 | SUGGESTION | Task 5 risk | kilo risk conflates skill-list visibility with injection | FIXED | 809adfd2 | critic_report |
| F-S7-1 | CRITICAL | whole document | template conformance: no Files/Interfaces/SATISFIES/Steps substrate, no Review Summary | FIXED (tail restructured) | 809adfd2 | critic_report |
| F-S7-2 | WARNING | Findings / Decision notes | cross-refs not resolvable from document | PARTIAL (spec-zone convention — files exist) | 809adfd2 | critic_report |
| F-S7-3 | WARNING | Tasks preamble | wave granularity substituted for task ordering (root: F-S7-1) | FIXED | 809adfd2 | critic_report |
| F-S7-4 | SUGGESTION | header / Design contract | placeholder path syntax not literal | FIXED | 809adfd2 | critic_report |
| F-STYLE-1 | WARNING | Must-Haves intro | contraction "doesn't" (RULE-I) | FIXED | 809adfd2 | critic_report |
| F-STYLE-2 | SUGGESTION | header lines | em-dash/semicolon density (RULE-B) | PARTIAL (declined kernel: established register in table cells / Per-host table) | 809adfd2 | critic_report |
| F-S8-1 | CRITICAL | Task 6 / Design contract | bootstrap-distribution step had no pinned installer counterpart | FIXED (`syncRouterBootstrap` seam pinned) | 809adfd2 | intermediate_zoom_out_report |
| F-S8-2 | CRITICAL | Task 4 / File Structure / opencode row | tree re-creation premise unreconciled with the stale installer contract | FIXED (Reconciliation paragraph) | 809adfd2 | intermediate_zoom_out_report |
| F-S8-3 | WARNING | Design contract / Task 5 | Kilo pointer-vs-full-body deviation unjustified | FIXED (deviation rationale added) | 809adfd2 | intermediate_zoom_out_report |
| F-S8-4 | WARNING | R3 / Task 1 (cursor) | cursor subagent skip unspecified (no subagent marker in stdin) | FIXED (structural guard + live check) | 809adfd2 | intermediate_zoom_out_report |
| F-S8-5 | WARNING | Threat Matrix / copyFile | exec-bit row vs `copyFile` 0644 drift | FIXED (row re-anchored to installCursorHookScripts) | 809adfd2 | intermediate_zoom_out_report |
| F-S9-CD-1 | CRITICAL | Per-host opencode row / Task 4 | V2 grounding misattributed (plan.ts:75 synthetic vs the `instructions.entry.put` API) | FIXED (citations attributed per API) | 809adfd2 | implementation_readiness_report |
| F-S9-CD-2 | WARNING | Task 6 prerequisite | rename file list inaccurate vs actual diff | FIXED | 809adfd2 | implementation_readiness_report |
| F-S9-CD-3 | WARNING | Task 2 / File Structure | test file cited without a Create entry | FIXED (Create labels, Tasks 2–5) | 809adfd2 | implementation_readiness_report |
| F-S9-CD-4 | WARNING | header block | dangling §2/§3/§6 refs post-restructure | FIXED (re-pointed to named sections) | 809adfd2 | implementation_readiness_report |
| F-S9-CD-5 | WARNING | Per-host pi row / Task 3 | line anchors misformatted as § sections | FIXED ("lines 80, 87-88, 192" etc.) | 809adfd2 | implementation_readiness_report |
| F-S9-CD-6 | WARNING | Task 8 / File Structure | `11-plugins.md` absent; its per-host table stale (pi/omp rows) | FIXED (Task 8 + File Structure) | 809adfd2 | implementation_readiness_report |
| F-S9-CD-7 | WARNING | Task 6 | "idempotent when unchanged" implied content comparison | FIXED (copyFile semantics stated) | 809adfd2 | implementation_readiness_report |
| F-S9-CD-8 | WARNING | Task 1 risk | cursor hook also runs via user-level `~/.cursor/hooks.json` | FIXED (both run paths named) | 809adfd2 | implementation_readiness_report |
| F-S9-CD-9 | WARNING | Per-host pi row / Task 3 | `ctx.agent.kind` documented for the omp surface, not plainly for pi | FIXED (omp-local.md:44-46 citation) | 809adfd2 | implementation_readiness_report |
| F-S9-CD-10 | WARNING | Task 7 | trust-prompt note reads as install framing | REJECTED (`--agents` already excludes pi per D3; the note is a smoke-setup instruction for the committed extension) | n/a | implementation_readiness_report |
| F-S9-IR-1 | CRITICAL | Task 1 | "installer wiring `openRuntime`/`storeFor` omitted" | REJECTED (identifiers absent from the blueprint — grep-verified; they belong to the separate memory-plugins change; installer wiring is Task 6, which lists its files) | n/a | implementation_readiness_report |
| F-S9-IR-2 | WARNING | Task 2 | "RED tests lack concrete paths/assertions" | REJECTED (Step 1 names the file, the spawn harness, and three assertions) | n/a | implementation_readiness_report |
| F-S9-IR-3 | SUGGESTION | Task 4 | "Consumes/Produces uses `openRuntime`/`storeFor`" | REJECTED (grep-verified absent) | n/a | implementation_readiness_report |
| F-S9-ST-1 | WARNING | "lines 4–5" | dash-splice + meta-instruction | REJECTED (quoted the dispatch prompt, not the blueprint) | n/a | implementation_readiness_report |
| F-S9-ST-2 | WARNING | "line 8" | first-person narration | REJECTED (quoted a prior transcript, not the blueprint) | n/a | implementation_readiness_report |
| F-S9-ST-3 | CRITICAL | "line 12" | em-dash/semicolon pile-up ("Budget check done…") | REJECTED (quoted a prior transcript, not the blueprint) | n/a | implementation_readiness_report |
| F-S9R-ST-1 | SUGGESTION | line 49 (Hypothesis door check) | em-dash casual clause punctuation (RULE-B) | FIXED | 809adfd2 | implementation_readiness_report |
| F-S9R-ST-2 | SUGGESTION | line 22 (Architecture) | semicolon clause separators (RULE-B) | FIXED | 809adfd2 | implementation_readiness_report |
| F-S9R-ST-3 | WARNING | line 45 (Hypothesis claim) | ~46-word nested-parenthetical sentence (RULE-12) | FIXED (split; Clause (a)/(b) kept explicit per F-3) | 809adfd2 | implementation_readiness_report |
| F-S9R-ST-4 | SUGGESTION | line 36 heading | "notes" lowercase vs title case (RULE-G) | REJECTED (template + repo precedent use sentence case: "## Session protocol", "## Tool mapping") | n/a | implementation_readiness_report |

Totals: 47 raised — 13 CRITICAL, 25 WARNING, 9 SUGGESTION. Dispositions: 35 FIXED + 4 PARTIAL + 8 REJECTED.

### Dispositions

- **FIXED (35):** all folded into the revised candidate b12cd7d1 (commit 809adfd2).
- **PARTIAL (4):** F-C6-2 (cross-reference list is normal blueprint behavior — ADRs/review artifacts live in the repo), F-C6-6 (table rows dense by construction), F-S7-2 (spec-zone convention), F-STYLE-2 (declined kernel: the em-dash register is established in table cells / the Per-host table; flagged header lines were re-pointed). Each partial's remainder is recorded with its rationale in the stage report's Resolutions.
- **REJECTED (8):** each verified against the target before disposition — F-S9-CD-10 (text matches the D3 design), F-S9-IR-1/2/3 + F-S9-ST-1/2/3 (reviewer hallucination / misattribution: quoted text absent from the blueprint at the cited lines; grep + line-check evidence in the stage report), F-S9R-ST-4 (template + repo precedent override the persona's title-case rule).
- **NOT RUN (1 persona):** Stage-9 implementation-readiness — first pass ungrounded (34.4s; cited identifiers absent from the target), re-run returned an unfilled template (37.5s). Per the dispatch protocol (retry once) it is recorded not run; its verdict is capped at "not verified", not "clean". Compensation: the actionability lens is partially carried by the S7 structural critic (F-S7-1 task substrate) + S9 codebase-details (all citations verified against the working tree).

### Round Verdict

**PASS — revised candidate b12cd7d1 (496 lines).** All 47 findings resolved (35 fixed, 4 partial with recorded remainder, 8 rejected with evidence); no CRITICAL open. One persona not verified (recorded, capped). Consolidated cross-stage dedup + traceability matrix: `review/consolidated_review_report.md`. Next: the R13 user-review gate (blueprint approval).

## Round 2 — 2026-11-22 (code review r1)

**Frozen candidate:** base d3026233 → head 9146b1f3, 43 files, +4,456 −514
**Reviewer:** code-review skill — core five (Standards, Spec, Security, Edge-case, Verification-gap) + Red-team after consolidation; fresh subagents, Blindness Contract
**Independence grade:** A (fresh context, read-only, pinned range only, working-tree WIP ignored)
**Budget:** round 2/5 (tier T2)

Full audit record: `review.md` (same directory). Findings below are the normalized ledger view (reviewer vocabulary → CRITICAL/WARNING/SUGGESTION).

### Findings

| ID | SEVERITY | LOCATION | FINDING | DISPOSITION | FIX COMMIT | SOURCE |
|---|---|---|---|---|---|---|
| CR2-1 | CRITICAL | `.omp/extensions/skillgrid-router-omp.ts:62-71` | no idempotency guard; per-turn re-push of full router on live systemPrompt array | FIXED (INTRO marker guard + case) | 3a12b7d1 | edge-case + standards (2-lens) |
| CR2-2 | CRITICAL | `plugins/kilo/skillgrid-router.ts:70-72` | `__dirname/../..` dead in installed layout; dead checkout leg + bogus skills push | FIXED (project-root derivation + mirror fallback + 2 cases) | 3a12b7d1 | spec + security + standards + verification-gap (4-lens) |
| CR2-3 | CRITICAL | `hooks/cursor-session-start.sh:81` | R3 unreadable-config: `-f` only → injected (empirically verified pre-fix) | FIXED (`-f && -r`; isFile+R_OK on all 5 walks) | 3a12b7d1 | spec (coordinator empirical) |
| CR2-4 | CRITICAL | `plugins/opencode/skillgrid-router.ts:186-203` | ≥2.2 path (a): `setup()` never settles (unbounded for-await); untested default path | FIXED (background IIFE listener) | 3a12b7d1 | red-team |
| CR2-5 | WARNING | pi/omp/kilo resolvers | whitespace-only installed asset shadows checkout leg | FIXED (trim-guard) | 3a12b7d1 | edge-case |
| CR2-6 | WARNING | `hooks/session-start.js:110` | `process.cwd()` instead of harness payload cwd | FIXED (payload preferred + case) | 3a12b7d1 | edge-case + security |
| CR2-7 | WARNING | `plugins/opencode/skillgrid-router.ts` | plan agent filtered as subagent; Plugin-import fail-closed; messages-branch dedup; stream-iteration throw | FIXED (4 sub-fixes) | 3a12b7d1 | edge-case |
| CR2-8 | WARNING | `report.md:64,72` | R1/data-flow claim vs evidence mismatch (T7 recorded no edit) | FIXED (lines corrected) | 9b8c3409 | red-team + verification-gap |
| CR2-9 | WARNING | `docs/user-guide/04-hooks.md` | absent `.claude/settings.json` claim; omp "superseded" overstated | FIXED (wiring + wording) | 3a12b7d1 | standards + red-team |
| CR2-10 | WARNING | `hooks/session-start.js:61-74` | config.yaml as directory passes detection | FIXED (isFile, all 5 walks) | 3a12b7d1 | edge-case |
| CR2-11 | SUGGESTION | `plugins/kilo/skillgrid-router.ts` | dead code (frontmatter/normalizePath/envConfigDir/configDir) | FIXED (deleted) | 3a12b7d1 | standards + edge-case |
| CR2-12 | SUGGESTION | `plugins/kilo/skillgrid-router.ts:115` | module-level bootstrap cache stale across cases | FIXED (cwd-keyed + seam reset) | 3a12b7d1 | verification-gap |
| CR2-13 | SUGGESTION | `plugins/opencode/test/skillgrid-router.test.ts:71` | `REPO_ROOT = process.cwd()` cwd-dependent | FIXED (file-derived) | 3a12b7d1 | standards + edge-case |
| CR2-P1 | SUGGESTION | `README.md:42` | `--agents …,omp` is post-rename key set; head key is `skill-pi` | PARKED — rename commit reconciles | n/a | red-team |
| CR2-P2 | SUGGESTION | opencode ≥2.2 path (a) | live-unverified on target version (env-limited) | PARKED — held-out live run ≥2.2 | n/a | verification-gap |
| CR2-P3 | SUGGESTION | `hooks/session-start.js` output contract | no live host-consumption observation (env-limited) | PARKED — held-out live session | n/a | verification-gap |
| CR2-P4 | SUGGESTION | all 5 resolvers | installed-vs-checkout precedence (both present, different) untested | PARKED — QA W016 adjudication (SUGGESTION) | n/a | verification-gap (QA W016) |
| CR2-P5 | SUGGESTION | `installOmpHookBridge` | omp SessionStart hook dormant via dispatch-order quirk; double-inject risk if it changes | PARKED — drop hook entry if dispatch changes (doc states this) | n/a | red-team |
| CR2-P6 | SUGGESTION | `plugins/opencode/skillgrid-router.ts:40` | `MAX_WALK_UP = 12` vs unbounded walks (R3 divergence >12 levels) | PARKED — align or document | n/a | standards |
| CR2-P7 | SUGGESTION | `setup/opencode.go` + `install/agents.go` | skip-tolerance stats repoRoot only; retired plugins never dropped | PARKED — ADR-0067 rework / retirement pass | n/a | edge-case |
| CR2-P8 | SUGGESTION | kilo transform hooks | dual-channel guards independent (both can inject) | PARKED — env-limited runtime check | n/a | standards |
| CR2-P9 | SUGGESTION | `router_distribution_test.go` `skill-pi` branch | stale vs in-flight omp rename (user WIP) | PARKED — rename commit | n/a | standards |
| CR2-P10 | SUGGESTION | `hooks/cursor-session-start.sh:78-84` | relative payload cwd walk (cursor passes absolute) | PARKED — env-limited | n/a | edge-case |
| CR2-P11 | SUGGESTION | `hooks/session-start.js:78-88` | stdin without EOF awaits until harness timeout | PARKED — harness contract | n/a | edge-case |
| CR2-P12 | SUGGESTION | `session-start.test.mjs` unreadable-config case | chmod-000 flaky as root | PARKED — skip-with-note if it bites | n/a | edge-case |
| CR2-P13 | SUGGESTION | `review/smoke-evidence.md` | duplicate TICKET-02/03 sections | PARKED — next evidence append | n/a | verification-gap |

### Dispositions

- **FIXED (13):** CR2-1..13, all in the round-2 code-zone fix commit (+ spec-zone CR2-8 in the review.md/report.md commit). 10 new regression cases added (walk-up ×4, kilo installed-layout, omp static-fallback, session-start router-missing, session-start payload-cwd); all five affected suites green; cursor R3 scenario empirically re-verified.
- **PARKED (13):** CR2-P1..P13 — each with a named follow-up owner in `review.md` § Parked.
- **REJECTED (8, verified at location):** Security's 11 excluded candidates (no exploit path, its report), SpecLens kilo walk-up "divergence" (residual covered by the CR2-2 mirror fallback), the omp pinned-head divergence call (observable contract met; mechanism is not the authority), Edge-case opencode V1 "no mechanism" (path (b) context-hook is the 1.x mechanism; residual = CR2-P2 env-limited verification), plus three Edge-case items subsumed by CR2-2/CR2-7 fixes (kilo `_bootstrapCache` trap → CR2-12; kilo config-hook guard → CR2-2 sub-fix; opencode test cwd → CR2-13).

### Round Verdict

**PASS** — 34 distinct findings resolved (13 fixed, 13 parked with owners, 8 rejected with evidence); no CRITICAL open. Fix-loop verification: five suites green + empirical cursor R3 re-check. Next: ship.
