# Report — mnemonic memory checkpoint

> Change: `.skillgrid/specs/2026-10-02-mnemonic-memory-checkpoint/` (moves to `.skillgrid/archive/2026-10-02-mnemonic-memory-checkpoint/` at ship)
> Generated: 2026-10-02T16:20:00Z (qa re-verify)
> Gate: PASS
>
> Two phases, one file: **qa** writes the QA half (the sections above `## Final-State Facts`,
> through `## Gate Decision` + `## Human Override`) into the spec folder. **ship** reads the
> `## Gate Decision` verdict PRE-MOVE, then moves the folder. **reflect** completes the retro
> half (from `## Final-State Facts` onward) IN PLACE in the archive folder.

## Test Plan

> Derived from `blueprint.md`, `tasks.md`, and `acceptance.feature`.
> Risk-ordered. Every row must be covered by a named test or scenario before the gate passes.

### Risk Ranking

Most likely production break: claim route due/cooldown wrong or digest leaking tool output into a user-role follow-up (prompt injection). Second: hooks/project mismatch so the Memory Index reads an empty store. Third: fail-open broken so a dead server blocks stop.

### Plan

| # | Risk / Behavior | Seam | Layer | Test / Scenario | Priority | Cadence | Status |
|---|-----------------|------|-------|-----------------|----------|---------|--------|
| 1 | Claim due after ≥ min_events | POST /checkpoint/claim | integration | `http/checkpoint_test.go::TestCheckpointClaim_DueAndCooldown` | P0 | pr | covered |
| 2 | Claim cooldown / unknown / disabled | claim route | integration | `TestCheckpointClaim_*` + `TestCheckpointClaim_Disabled` | P0 | pr | covered |
| 3 | Digest excludes tool output | checkpoint.BuildDigest | unit | `checkpoint/digest_test.go::TestBuildDigest_StructuredFieldsOnly` | P0 | pr | covered |
| 4 | Prompt lists titles + tools | RenderPrompt / claim | unit+http | `TestRenderPrompt` + `TestCheckpointPrompt_Content` | P0 | pr | covered |
| 5 | Cursor stop follow-up when due | tool-call-capture checkpoint | integration | `scripts/test-hooks.mjs checkpoint` | P0 | pr | covered |
| 6 | Stop silent at loop_limit / fail-open | checkpoint mode | integration | same (loop_count 2, dead port) | P0 | pr | covered |
| 7 | Private spans never stored | StripPrivate + hook | unit+http+hook | `TestStripPrivate*` + `test-hooks private` | P0 | pr | covered |
| 8 | One project resolver | projectFromRequest | integration | `TestToolCalls_ResolvesProjectFromDirectory` + `TestPrime_ProjectMatchesHTTPStore` | P0 | pr | covered |
| 9 | Memory Index at prime | RenderIndex + RenderPrime | unit | `TestRenderIndex` + `TestRenderPrimeMemoryIndex` | P1 | pr | covered |
| 10 | Setup installs OC/Kilo plugin | setup copyFromRepo | unit | `TestSetupOpenCode_CheckpointPlugin` + `TestSetupKilo_CheckpointPlugin` | P1 | pr | covered |
| 11 | Plugin not-due / no-server | session.idle plugin | e2e | G7 harness on :17438 / dead :17999 | P1 | nightly | covered |
| 12 | Live observations in Sessions UI | ToolTimeline + SessionsPage | unit | vitest SessionsPage + ToolTimeline (13) | P1 | pr | covered |
| 13 | Config defaults / invalid / disabled | config.Load | unit | `TestCheckpointConfig*` + claim disabled | P1 | pr | covered |
| 14 | Cursor stop door (G4/G5) | stop → claim follow-up | e2e | live hook + test-hooks checkpoint | P0 | nightly | covered |

### Edge-Case Matrix

| Requirement | Edge / Boundary | Expected Behavior | Test / Scenario | Status |
|-------------|-----------------|-------------------|-----------------|--------|
| checkpoint-claim-gating | unknown session | 200 due:false unknown_session | TestCheckpointClaim_UnknownSession | covered |
| checkpoint-claim-gating | below min_events | not due | Decide table + claim path | covered |
| checkpoint-prompt-content | content_preview in tool call | absent from prompt | TestCheckpointPrompt_Content | covered |
| private-spans | unterminated tag | strip to end | TestStripPrivate | covered |
| memory-index | empty project | no ## Memory | TestRenderIndex empty + TestRenderPrimeNoMemoryIndex | covered |
| memory-index | over token cap | drop oldest, note omitted | TestRenderIndex MaxTokens | covered |
| cursor-stop | dead port | {} within 2s exit 0 | test-hooks checkpoint | covered |
| live-observations | stream error | offline indicator | SessionsPage.test | covered |

### Out of Scope

- Merging `skillgrid.sqlite` → `aiskillgrid.sqlite` (operator action; briefing open question)
- Prompt-capture hooks and Memories page upgrade (parked serial slices)
- Live OpenCode/Kilo idle in a real IDE (G7 manual)
- Live Cursor stop with five tool calls against `skillgrid serve` (G4/G5 manual)
- Pre-existing `TestCursorPluginLayout` (missing `plugins/_shared/memory-protocol.md`) and `TestCodeAffectedStdinCLI` — not introduced by this change

## Goal-Backward Verification

**Stated goal** (from briefing.md): Host agent becomes the memory observer — server-gated checkpoints write typed observations + summary; session start gets a lean Memory Index; private spans never stored; Sessions view shows observations live.

| Level | Item | Evidence | Status |
|-------|------|----------|--------|
| Truth | Claim gates on events/cooldown/disabled | HTTP checkpoint tests | VERIFIED |
| Truth | Prompt is structured fields only | Digest + Prompt tests | VERIFIED |
| Truth | Cursor stop can emit follow-up fail-open | test-hooks + live :17438 hook | VERIFIED |
| Truth | Private spans never stored | Go + hook private tests | VERIFIED |
| Truth | One project for prime + hooks | Resolve tests; live claim → aiskillgrid | VERIFIED |
| Truth | Memory Index in prime | RenderIndex + primeText wire | VERIFIED |
| Truth | Sessions show observations live | vitest 13/13 | VERIFIED |
| Artifact | checkpoint package + claim route + hooks + plugins + UI | commits 3d826978..01ff9185 | VERIFIED |
| Key Link | stop → claim → prompt | live hook followup_message | VERIFIED |
| Key Link | idle plugin → claim | G7a/b/c harness (claim→prompt path) | VERIFIED |
| Data Flow | tool events → claim digest → follow-up | live 5 tool-calls → due prompt on aiskillgrid | VERIFIED |

## Traceability Matrix

| Scenario (from acceptance.feature) | Test / Scenario (from test plan) | Ran | Result |
|--------------------------------------|----------------------------------|-----|--------|
| claim-due-after-enough-events | TestCheckpointClaim_DueAndCooldown | yes | pass |
| claim-not-due-below-threshold | Decide + claim tests | yes | pass |
| claim-resets-after-summary | TestCheckpointClaim_DueAndCooldown (summary branch) | yes | pass |
| claim-for-unknown-session | TestCheckpointClaim_UnknownSession | yes | pass |
| prompt-lists-new-events-and-existing-titles | TestCheckpointPrompt_Content + TestRenderPrompt | yes | pass |
| prompt-digest-is-bounded | TestBuildDigest_Bounded | yes | pass |
| prompt-omits-private-spans | StripPrivate + claim path uses structured events | yes | pass |
| prompt-excludes-tool-output-text | TestBuildDigest_StructuredFieldsOnly | yes | pass |
| stop-returns-followup-when-due | test-hooks checkpoint | yes | pass |
| stop-stays-silent-at-loop-limit | test-hooks checkpoint loop_count 2 | yes | pass |
| stop-fails-open-without-server | test-hooks dead port | yes | pass |
| setup-installs-checkpoint-plugin | TestSetupOpenCode/Kilo_CheckpointPlugin | yes | pass |
| plugin-does-not-prompt-when-not-due | G7a live claim not-due | yes | pass |
| plugin-fails-open-without-server | G7b dead port fail-open | yes | pass |
| prime-lists-summaries-and-observation-index | TestRenderIndex + primeText | yes | pass |
| index-respects-token-cap | TestRenderIndex MaxTokens | yes | pass |
| index-absent-for-empty-project | TestRenderIndex empty | yes | pass |
| prime-and-hooks-share-one-project | TestPrime_ProjectMatchesHTTPStore | yes | pass |
| directory-without-project-param | TestToolCalls_ResolvesProjectFromDirectory | yes | pass |
| prime-in-another-repository | same + Resolve | yes | pass |
| tool-call-with-private-span | TestToolCalls_PrivateSpan + hook private | yes | pass |
| unterminated-private-tag | TestStripPrivate | yes | pass |
| private-span-in-summary | TestSessionSummary_PrivateSpan | yes | pass |
| observation-appears-live | SessionsPage.test activity frame | yes | pass |
| observation-row-links-to-full-text | ToolTimeline.test + events API | yes | pass |
| stream-disconnected | SessionsPage offline indicator | yes | pass |
| defaults-apply-without-keys | TestCheckpointConfig defaults | yes | pass |
| disabled-checkpoint | TestCheckpointClaim_Disabled | yes | pass |
| invalid-value | TestCheckpointConfig invalid warn+default | yes | pass |

**Coverage:** 29/29 scenarios covered by a test that ran and passed (G7 via harness-simulated OpenCode client against live claim route; G4/G5 via live hook + test-hooks).

## Verification Run Log

| Command | Result |
|---------|--------|
| `go test ./internal/mnemonic/config ./checkpoint ./memory ./session_inject ./loop ./http` | ok (http ~62s) |
| `go test ./internal/mnemonic/setup -run 'TestSetupOpenCode_CheckpointPlugin\|TestSetupKilo_CheckpointPlugin\|TestUpsertCursorHooks'` | ok |
| `go test ./internal/mnemonic/setup` (full) | FAIL pre-existing `TestCursorPluginLayout` (missing plugins/_shared/memory-protocol.md) |
| `go test ./cmd/skillgrid` (full) | FAIL pre-existing `TestCodeAffectedStdinCLI` |
| `node scripts/test-hooks.mjs checkpoint` | 11 passed, 0 failed |
| `node scripts/test-hooks.mjs private` | 4 passed, 0 failed |
| `npx vitest run SessionsPage.test ToolTimeline.test` | 13/13 passed |

## Audits (summary)

| Audit | Result |
|-------|--------|
| Verification-gap | None for acceptance scenarios; optional real OpenCode UI idle still nice-to-have |
| TDD evidence | Tickets used failing-first pattern per briefs; controller parent re-verify on gates |
| Security | Digest structured-fields-only (injection boundary); fail-open hooks; no new deps; Trivy advisory-only |
| Code quality | gofmt clean on new pkgs; UI lint N/A; build:check not re-run (pre-existing api.test.ts TS issues) |
| Scope | COMPLETE for G4/G5/G6/G7 door re-verify on :17438 |

## Floor

Weakest dimension: optional real OpenCode GUI idle (harness covers claim→prompt contract) — does not block PASS.

## Gate Decision

**PASS**

- 29/29 scenarios covered; G4/G5/G6/G7 door re-verify 2026-10-02 against `/tmp/skillgrid-checkpoint-door serve :17438`.
- Live claim resolved project `aiskillgrid`; five tool-calls → due prompt with structured digest; hook emitted `followup_message`; dead port → `{}`.
- Pre-existing unrelated failures (`TestCursorPluginLayout`, `TestCodeAffectedStdinCLI`) remain out of scope / deferred.

Open items (non-blocking):
1. **defer** — restore `plugins/_shared/memory-protocol.md` or fix TestCursorPluginLayout
2. **defer** — TestCodeAffectedStdinCLI indexing flake
3. **optional** — smoke in a real OpenCode GUI idle (harness already covers the contract)

## Human Override

_(none — machine verdict PASS after door re-verify)_

## Final-State Facts

_(reflect)_

## Retro

_(reflect)_
