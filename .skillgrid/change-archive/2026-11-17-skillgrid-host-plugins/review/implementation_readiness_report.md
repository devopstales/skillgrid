# Stage 9 — Implementation Readiness (Round 1)

**Reviewers:** fresh `flint` subagents × 3 (implementation-readiness, codebase-details, style) + 2 re-runs (implementation-readiness, style) — same harness deviation as stages 5/7/8 (`reviewer` backend 429; `flint` fallback)
**Personas:** `.agents/skills/_shared/elephant-goldfish/{implementation-readiness,codebase-details,style}.md`
**Frozen candidate reviewed:** blueprint.md blob c2a256b0 (post-stages-5/7/8 revision, 490 lines)
**Revised candidate after fold-in:** blueprint.md blob b12cd7d1 (496 lines)

Isolation attestation (all five dispatches): fresh context, read-only, no
project instructions, no network/MCP — four-part contract re-asserted verbatim
in each dispatch; the re-runs added a grep-grounding rule (cite only
grep-verified target text) after the first-pass impl-readiness/style reports
cited content absent from the target.

## Findings (reviewer outputs, condensed verbatim)

### codebase-details (28m17s; thorough — verified citations against the working tree)

| ID | Severity | Scope | Finding |
|---|---|---|---|
| F-S9-CD-1 | Major | §Per-host opencode row / Task 4 | V2 mechanism grounding misattributed: blueprint anchored the durable `<context key>` injection to "repo precedent: plan.ts:75 synthetic-on-session.created", but plan.ts:75 (pi-opencode.md:119-122) performs `ctx.session.synthetic` (message injection) — a different API than the chosen `ctx.session.instructions.entry.put` (documented as the durable-instruction API at pi-opencode.md:127-128). The live plugin `~/.config/opencode/plugins/skillgrid-context.ts:126-165` documents a verified v2.0.28 hook surface (`ctx.session.hook("context")`) not cited. |
| F-S9-CD-2 | Minor | Task 6 prerequisite | rename file list inaccurate vs the actual working-tree diff (test files + new `ompplugins.go` missing) |
| F-S9-CD-3 | Minor | Task 2 / File Structure | `hooks/session-start.test.mjs` cited without a Create entry (labeled `Test:`) |
| F-S9-CD-4 | Minor | header block (lines 22/24/26/32/38) | dangling `§2`/`§3`/`§6` references left over from the stage-7 restructure (numbered sections no longer exist) |
| F-S9-CD-5 | Minor | pi row / Task 3 | `pi-opencode.md §80, §87-88, §192, §189` read as section refs but the research file has no such sections — they are line anchors, ambiguously formatted |
| F-S9-CD-6 | Minor | Task 8 / File Structure | `docs/user-guide/11-plugins.md` (exists, documents the router plugin for this change) absent from the docs task; its per-host table is stale vs the revised design (pi + omp rows) |
| F-S9-CD-7 | Minor | Task 6 | "idempotent when unchanged" implies content comparison; `copyFile` (hooktree.go:196-213) unconditionally overwrites |
| F-S9-CD-8 | Minor | Task 1 risk | cursor hook runs via user-level `~/.cursor/hooks.json` too (setup/cursor.go:63-73: "whether or not the plugin is enabled") — the risk framed plugin-only |
| F-S9-CD-9 | Minor | pi row / Task 3 | main-agent guard `ctx.agent.kind` documented for the omp ExtensionAPI (omp-local.md:44-46), not plainly for pi — citation gap |
| F-S9-CD-10 | Minor | Task 7 | "pi: accept the project-trust prompt" reads as an install framing despite D3 |

### implementation-readiness (first pass, 34.4s — ungrounded)

| ID | Severity | Scope | Finding |
|---|---|---|---|
| F-S9-IR-1 | Major | Task 1 | "Files list omits the installer wiring that consumes the new plugin manifest (`openRuntime`/`storeFor` in main.go)" |
| F-S9-IR-2 | Minor | Task 2 | "RED tests described but not with concrete file paths or assertion shapes" |
| F-S9-IR-3 | Nit | Task 4 | "Consumes/Produces table uses `openRuntime`/`storeFor` without naming the owning module" |

### style (first pass, 56.4s — ungrounded)

| ID | Severity | Scope | Finding |
|---|---|---|---|
| F-S9-ST-1 | Minor | "lines 4–5" | dash-splice + meta-instruction "in the persona brief itself" (quoted the dispatch prompt) |
| F-S9-ST-2 | Minor | "line 8" | first-person narration "I am using the skillgrid:writing-blueprints skill…" (quoted a prior transcript, not the blueprint) |
| F-S9-ST-3 | Major | "line 12" | "Budget check done — writing the blueprint now…" (quoted a prior transcript, not the blueprint) |

### implementation-readiness re-run (37.5s — not run)

Unfilled template output (`### F-S9R-IR-1 — <Severity>`, `## Verdict: REVISE (N findings)`) — no usable findings. Per the dispatch protocol (retry once), the implementation-readiness persona is recorded **not run**; its stage-9 verdict is capped at "not verified", not "clean".

### style re-run (13m36s; grounded)

| ID | Severity | Scope | Finding |
|---|---|---|---|
| F-S9R-ST-1 | Nit | line 49 (Hypothesis door check) | em-dash casual clause punctuation ("stop — re-verify") — RULE-B |
| F-S9R-ST-2 | Nit | line 22 (Architecture) | semicolons as casual clause separators — RULE-B |
| F-S9R-ST-3 | Minor | line 45 (Hypothesis claim) | ~46-word sentence, nested parentheticals before the predicate — RULE-12 |
| F-S9R-ST-4 | Nit | line 36 heading | "notes" lowercase where title case requires "Notes" — RULE-G |

## Resolutions (main-agent verified + folded into b12cd7d1)

- **F-S9-CD-1 → fixed (Major):** opencode row + Task 4 Produces now attribute each citation to what it supports — the durable-instruction API (pi-opencode.md:127-128), the `session.created` event timing (pi-opencode.md:119-122; plan.ts:75 is the synthetic-message precedent for the event, not for the API), and the live v2.0.28-verified hook surface (skillgrid-context.ts:126-165, `ctx.session.hook("context")` — a per-request concern, noted as the verified surface, not the chosen shape). The API choice itself (`instructions.entry.put`) stands: it is the documented durable-instruction path matching the Must-Have's `<context key>` observable.
- **F-S9-CD-2 → fixed:** Task 6 prerequisite list corrected against `git diff --name-only` + untracked: `install/{agents,config,hookbridge,hookbridge_test,install,install_test,lsp,lsp_test}.go`, `setup/{pi,pi_test,setup}.go`, new untracked `install/{ompplugins,ompplugins_test}.go`.
- **F-S9-CD-3 → fixed (Nit):** new test files are now explicit `Create:` entries (Tasks 2/3/4/5); the existing `install_test.go` is a `Modify:` (Task 6).
- **F-S9-CD-4 → fixed:** all dangling `§N` references re-pointed to named sections / Task 6 (header block lines 22, 24, 26, 32, 38).
- **F-S9-CD-5 → fixed:** research citations are now explicit line anchors ("lines 80, 87-88, 192", "line 189" — content verified at those lines).
- **F-S9-CD-6 → fixed:** `docs/user-guide/11-plugins.md` added to Task 8 (Files + Step 3 + Step 4 verify) and File Structure, with the specific stale rows named (pi "shared Mnemonic seam", omp "extension under ~/.omp/agent/extensions/") for alignment with the Per-host mechanism table.
- **F-S9-CD-7 → fixed:** Task 6 now cites hooktree.go:196-213 and states "unconditional overwrite; re-runs with unchanged content leave the file unchanged".
- **F-S9-CD-8 → fixed:** Task 1 risk now names both run paths (plugin or installer user-level `~/.cursor/hooks.json`, setup/cursor.go:63-73) and narrows the AGENTS.md fallback to machines with neither.
- **F-S9-CD-9 → fixed:** the `ctx.agent.kind` guard is cited to the pi-based ExtensionAPI documentation (omp-local.md:44-46: `agent (kind: "main" | "sub")`) in the pi row + Task 3 Step 3; the stub test + live smoke pin the exact shape.
- **F-S9-CD-10 → rejected (verified):** Task 7 Step 1's `--agents omp,opencode,kilo,cursor` already excludes pi per D3; "pi: accept the project-trust prompt if shown" is a smoke-setup instruction for the committed project extension (which is how it activates), not an install claim. The text matches the D3 design as written.
- **F-S9-IR-1/2/3 → rejected (verified):** `openRuntime`/`storeFor` are absent from the blueprint (`grep -n` = 0 hits — they belong to the separate in-flight memory-plugins change); Task 1 creates no plugin manifest (installer wiring is Task 6, which lists its files); Task 2's RED spec is concrete at Step 1 (file `hooks/session-start.test.mjs`, `spawn` with `cwd:` override, three named assertions). The 34.4s first pass reviewed the wrong context.
- **F-S9-ST-1/2/3 (first pass) → rejected (verified):** the quoted "lines 4–5/8/12" text is the dispatch prompt and a prior session transcript — none of it is in the blueprint at those lines.
- **F-S9R-ST-1 → fixed:** door check "stop — re-verify" → "stop and re-verify".
- **F-S9R-ST-2 → fixed:** line 22 restructured to three sentences (periods; also fixes F-S9-CD-4's `§3`).
- **F-S9R-ST-3 → fixed:** the Hypothesis claim split into short sentences (Clause (a)/Clause (b) kept explicit per the stage-7 F-3 fix).
- **F-S9R-ST-4 → rejected (verified):** the blueprint template (`templates/blueprint.md`: "## Terms", "## Must-Haves…", "## Review Summary") and the repo's own precedent (`2026-11-17-mnemonic-v2-host-memory-plugins/blueprint.md`: "## Session protocol", "## Tool mapping") use sentence case for these headings — RULE-G title case is overridden by template + repo convention.

## Stage-9 rollup

20 findings raised (10 codebase-details, 3 impl-readiness, 3 style, 4 style re-run) + 1 not-run persona (impl-readiness, both passes unusable — 34.4s ungrounded, 37.5s template output). **12 fixed, 8 rejected with evidence.** Coverage note: the actionability lens is partially carried by the stage-7 structural critic (F-S7-1 task-substrate fix) + this stage's citation verification; the dedicated impl-readiness persona is recorded not verified (harness/model variance on the local 40B — the two thorough runs, 28m/13m, grounded; the two fast runs, 34s/37s, did not read the target).

## Verdict: REVISE → folded (12 fixes applied); candidate b12cd7d1
