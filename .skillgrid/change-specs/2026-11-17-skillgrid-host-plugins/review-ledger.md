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
