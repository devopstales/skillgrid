# Stage 6 — Comprehension Test (Round 1)

**Reviewer:** `flint` (fresh spawn; `reviewer` backend 429 — same harness deviation as Stage 5)
**Persona:** `.agents/skills/_shared/elephant-goldfish/comprehension.md`
**Target:** blueprint.md (post-stage-5 state) + briefing.md + 4 research files

Isolation attestation: all four enforced (fresh context, read-only, no
project instructions, no network/MCP)

## Findings (reviewer output, verbatim)

| ID | Severity | Scope | Description (condensed) |
|---|---|---|---|
| F-C6-1 | Major | §Must-Haves | "goal-backward verification" method + the Must-Haves/Truths relationship never stated — section not paraphrasable without guessing |
| F-C6-2 | Major | §Terms + header block | self-containment claim ("None beyond briefing.md") while terms used before settled (router, injector, backstop, RED, T2/L3, tracer thread) and cross-refs outside reading list (ADRs, shared rules, review artifacts, F-S5-*) |
| F-C6-3 | Minor | §Tech Stack vs §Architecture | "three host plugins + one pi project extension" (4) does not reconcile with the four pointer injectors (pi ext + opencode + kilo + cursor) |
| F-C6-4 | Minor | §5 | heading scopes acceptance to R1–R4 but the must-have surface runs R1–R6; R6 never addressed in the plan |
| F-C6-5 | Minor | §5 live smoke | "router read before first action" not operationalized into a concrete observable |
| F-C6-6 | Nit | lines 17/81/138 | critical sections as single over-long lines burden a line-based reader |

## Resolutions (all PROPOSE-FIX, applied this stage)

- **F-C6-1 → fixed:** one-paragraph method note under the Must-Haves heading (derived backward from R1–R6; three categories; `backstop` tag semantics — source: writing-blueprints skill Must-Haves section).
- **F-C6-2 → fixed (partial):** Terms section now (a) states the document's own vocabulary precisely (forward-reference from header block to §1/§2) and (b) lists the SDD/process vocabulary with its cited source instead of claiming "none beyond briefing.md". The reviewer's cross-reference list (ADRs, review artifacts) is normal blueprint behavior — the ADRs live in the repo and the review artifacts are this round's own outputs; not a defect.
- **F-C6-3 → fixed:** Tech Stack now names the three TS host injectors explicitly (pi project extension + opencode + kilo modules), 1:1 with the §2 pointer rows; cursor is the Bash row.
- **F-C6-4 → fixed:** §5 heading now "(acceptance R1–R6, per host)"; new item 1 = static asset content check covering R1 + R6 (the `{load_hint}` variants table incl. tool-mapping hints).
- **F-C6-5 → fixed:** live smoke now has concrete observables — positive: first assistant turn announces `Using skillgrid:using-skillgrid to …` or first tool call reads the router; negative: transcript/JSONL shows no load/announce/bootstrap in non-skillgrid cwd.
- **F-C6-6 → fixed (partial):** Architecture paragraph split into a bullet structure. The §2/Threat-Matrix long lines are table rows — dense by construction; not split (declined kernel, recorded here).
