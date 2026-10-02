# Report — mnemonic memory checkpoint

> Change: `.skillgrid/specs/2026-10-02-mnemonic-memory-checkpoint/` (moves to `.skillgrid/archive/2026-10-02-mnemonic-memory-checkpoint/` at ship)
> Generated: 2026-10-02T16:10:00Z (qa)
> Gate: CONCERNS
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
| 11 | Plugin not-due / no-server | session.idle plugin | e2e | G7 manual | P1 | nightly | pending |
| 12 | Live observations in Sessions UI | ToolTimeline + SessionsPage | unit | vitest SessionsPage + ToolTimeline (13) | P1 | pr | covered |
| 13 | Config defaults / invalid / disabled | config.Load | unit | `TestCheckpointConfig*` + claim disabled | P1 | pr | covered |
| 14 | Cursor IDE end-to-end door (G4/G5) | stop → claim → mem_save | e2e | acceptance G4/G5 | P0 | nightly | pending |

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
| Truth | Cursor stop can emit follow-up fail-open | test-hooks checkpoint 11/0 | VERIFIED (script); IDE door PENDING |
| Truth | Private spans never stored | Go + hook private tests | VERIFIED |
| Truth | One project for prime + hooks | Resolve tests | VERIFIED |
| Truth | Memory Index in prime | RenderIndex + primeText wire | VERIFIED (unit); live prime against aiskillgrid PENDING |
| Truth | Sessions show observations live | vitest 13/13 | VERIFIED (mocked SSE); live browser PENDING |
| Artifact | checkpoint package + claim route + hooks + plugins + UI | commits 3d826978..01ff9185 | VERIFIED |
| Key Link | stop → claim → prompt | test-hooks + claim HTTP | VERIFIED |
| Key Link | idle plugin → claim | setup installs file; runtime G7 | PRESENT_BEHAVIOR_UNVERIFIED |
| Data Flow | tool events → claim digest → mem_save → events.observations | HTTP feed + claim tests | VERIFIED (partial; no single end-to-end process test) |

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
| plugin-does-not-prompt-when-not-due | G7 manual | no | pending |
| plugin-fails-open-without-server | G7 manual | no | pending |
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

**Coverage:** 27/29 scenarios covered by a test that ran and passed. **2 pending:** plugin-does-not-prompt-when-not-due, plugin-fails-open-without-server (G7 manual). Script-level Cursor stop is covered; live IDE G4/G5 EVIDENCE still unmarked in acceptance.feature.

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
| Verification-gap | 2 manual plugin scenarios pending; live IDE door deferred |
| TDD evidence | Tickets used failing-first pattern per briefs; controller parent re-verify on gates |
| Security | Digest structured-fields-only (injection boundary); fail-open hooks; no new deps; Trivy advisory-only |
| Code quality | gofmt clean on new pkgs; UI lint N/A; build:check not re-run (pre-existing api.test.ts TS issues) |
| Scope | COMPLETE for automated package gates; UNSCOPED for G7/G4/G5 live IDE |

## Floor

Weakest dimension: **pending manual G7 plugin scenarios** → caps verdict at **CONCERNS** (non-answer on e2e plugin path). Not UNREADABLE; named re-run: OpenCode idle with serve up/down.

## Gate Decision

**CONCERNS**

- No CRITICAL findings; all automated P0 claim/privacy/stop/resolver gates green.
- 2 acceptance scenarios pending (plugin not-due / plugin no-server) — G7 manual.
- Live Cursor G4/G5 EVIDENCE unmarked (script contract covered by test-hooks).
- Pre-existing unrelated test failures outside this change's Owns.

Open items (triage):
1. **human look** — G7 OpenCode/Kilo idle with serve up (not due / dead server)
2. **human look** — G4/G5 Cursor stop door in this repo
3. **defer** — restore `plugins/_shared/memory-protocol.md` or fix TestCursorPluginLayout (out of scope)
4. **defer** — TestCodeAffectedStdinCLI indexing flake (out of scope)

## Human Override

_(none — machine verdict CONCERNS)_

## Final-State Facts

_(reflect)_

## Retro

_(reflect)_
