# Review — 2026-10-03-local-ollama-models

> Change: `.skillgrid/specs/2026-10-03-local-ollama-models/` (moves to `.skillgrid/archive/2026-10-03-local-ollama-models/` at ship)
> Generated: 2026-10-06T11:52:55Z (requesting-code-review)
> Diff range: `5ebd23bf..6f32fab` (pre-review); post-fix commits add the review remediation
> Rigor tier: T2 (Beta) — three-axis pass is the default review at T2
> Independence: Grade B per axis — fresh subagent contexts, same toolchain family as the implementer (no leak of session narrative or prior verdict)
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
  asked for (scope creep). Unfixed Important issues do not proceed.
- **Minor** — nice to have. Baseline smells, style, optimization, doc polish. Capped and summarized.

## Cap the nits

- Max 5 Minor findings listed per axis; beyond that, collapse into a count.
- A Minor finding that is actually a Critical dressed as polish gets re-labeled, not capped.

## Do not report

- Anything tooling (gofmt / go vet) already enforces.
- Informational context with no required action — `FYI` at most, one line.

## Passes

### Standards

- **Worst issue (within this axis):** dead branch in `chatModel` (both arms returned `localLLMModel`)
- **Findings:** 0 Critical / 0 Important / 3 Minor
- **Verdict:** met (two of three Minor fixed in-pass; one logged)

### Spec

- **Worst issue (within this axis):** none — all 10 scenarios PASS, TDD evidence authentic
- **Findings:** 0 Critical / 0 Important / 2 Suggestion (advisory, pre-existing)
- **Verdict:** met

### Security

- **Worst issue (within this axis):** unbounded embed vector decode from `/api/embed` (CWE-400) — Low
- **Findings:** 0 Critical / 0 High / 0 Medium / 1 Low
- **Verdict:** secure (one Low, defense-in-depth, logged)

## Findings

### Critical

- none (all three axes)

### Important

- none (all three axes)

### Minor / Suggestion / Low

- [standards] `provider.go:359-362` `chatModel` — **fixed in-pass.** Dead branch: `if role == roleChat { return localLLMModel }` then `return localLLMModel` — both arms identical (Speculative Generality). Collapsed to a single `return localLLMModel`. Verified against the code: the loop returns the catalog entry for any role; the fallback was only ever `localLLMModel`.
- [standards] `provider.go:223-237` `ollamaVersion` HTTP seam — **fixed in-pass.** The seam returns the server-reported bare `body.Version` (e.g. `"0.36.0"`) but every floor test stubbed the seam to a bare string, so the real handler→`parseOllamaVersion` path was never exercised. Added `TestOllamaVersionSeamThroughParse` driving the real `/api/version` handler through `parseOllamaVersion`/`ollamaVersionAtLeast` (above/at/below floor + unreachable→empty). Verified: `parseOllamaVersion` treats a bare version as a single field and returns it correctly.
- [standards] `provider.go` `chatSmokeOK` / `ollamaEmbedDimension` — **logged.** `json.Marshal` error ignored on fixed input maps. Marshal of a constant-shape map cannot realistically fail; acceptable for a non-fatal probe. No action.
- [spec] `e6280d34` (W1 fix) bundles impl + tests + spec + report in one commit — **logged.** Weakens TDD-evidence granularity for the W1 fix (RED not separable from GREEN). Standard mode (`testing.tdd: false`) so not a gate finding; consistent with the sibling W015 pattern. No action.
- [spec] `state.yaml:10` malformed YAML (nested mapping in compact mapping) — **logged, pre-existing** (WINDOWS W017). Breaks the state-drift check; not introduced by this change. Advisory.
- [security] `provider.go:465-471` `ollamaEmbedDimension` — **logged, defense-in-depth.** No bound on the `[]float32` decoded from `/api/embed`; a hostile/misconfigured local Ollama can allocate arbitrary RAM (CWE-400, ASVS V1.2.1) and write a bogus dimension. Low: URL is fixed `localhost:11434` (not env-configurable in production, so no true SSRF), daemon is local and operator-installed, one-shot install-time probe, 3s client timeout bounds time. Same latent class already exists in the pre-existing runtime embedder (`embedder/ollama.go:96`). Optional fix: wrap `resp.Body` in `io.LimitReader` and/or sanity-check `dim` against a plausible range.

## Independence

| Axis | Grade | Rationale |
|------|-------|-----------|
| Standards | B | fresh subagent context, same toolchain family as the implementer; no session narrative or prior verdict leaked |
| Spec | B | fresh subagent context, same toolchain family; saw only diff + acceptance.feature/blueprint/briefing |
| Security | B | fresh subagent context, same toolchain family; saw only diff + owasp-security skill + references |

**Reading the grade:** **A** = fresh subagent, no shared history (independent). **B** = fresh
context but same model family / toolchain as the implementer (independent, weaker). **C** =
inline self-review or the implementer's verdict leaked in — **diagnostic only**. All three
axes are Grade B: fresh contexts, but same toolchain family, so they are independent evidence
weaker than a cross-model Grade A. No axis is Grade C.

## Verdict

- **Standards:** met (2 Minor fixed in-pass, 1 logged)
- **Spec:** met
- **Security:** secure (1 Low logged)
- **Floor (decides):** met
- **Worst issue (across all three axes):** unbounded embed vector decode (Low, defense-in-depth) — install-time probe against a fixed localhost endpoint
- **Unfixed Important count (must be 0 to proceed):** 0
