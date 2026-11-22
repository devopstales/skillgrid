# Code Review — 2026-11-17-skillgrid-host-plugins

**Change:** `.skillgrid/change-specs/2026-11-17-skillgrid-host-plugins` (archives to `.skillgrid/change-archive/` at ship)
**Generated:** 2026-11-22 (code-review, round 1)
**Diff range:** `d30262338b04905f6f5eca0bc5b7a3ce17d2b12a`..`9146b1f3fc1c2ff71939c1b108080a2ae4af1b34` (43 files, +4,456 −514)
**Reviewer tier:** T2
**Independence grade:** A — six fresh-context subagents (Standards, Spec, Security, Edge-case, Verification-gap, Red-team) with the Blindness Contract (no session history, no prior verdicts, no "why this way" narrative; pinned range only, working-tree WIP ignored)
**Status:** PASS (after round-1 fix loop)

## Scope and method

Core five lenses fanned out over the pinned diff (Blindness Contract), red team dispatched after the five reports consolidated (hunting cross-cutting / integration-boundary misses, not re-reporting). Coordinator verified every finding at its location before disposition. Skipped with reason: Performance (one-shot UI-less installer + five small injectors; no hot-path data access), Accessibility (no UI surface), Data access (no SQL in this change).

Raw findings: 68 (Standards 10, Spec 8, Security 1+11 excluded, Edge-case 22, Verification-gap 9, Red-team 4+3 verified-not-new) → 34 distinct after dedup. Disposition: 13 fixed in-loop, 13 parked (deferred with named follow-ups), 8 rejected as false positives / non-issues with evidence.

## Findings — round 1

### Important (all FIXED in the round-1 fix commit)

| ID | Location | Finding | Disposition |
|---|---|---|---|
| CR1-1 | `.omp/extensions/skillgrid-router-omp.ts:62-71` | No idempotency guard: `before_agent_start` fires per turn against the live systemPrompt array (ADR-0044); the full router re-pushed every turn (Edge-case high + Standards, 2-lens) | FIXED — INTRO marker guard + regression case |
| CR1-2 | `plugins/kilo/skillgrid-router.ts:70-72` | `repoRoot = __dirname/../..` mis-derives in the installed layout (`~/.config/kilo/plugin/` → `~/.config`): dead checkout-asset leg + bogus `~/.config/.agents/skills` push (Spec + Security + Standards + Verification-gap, 4-lens; unit tests never modeled the installed layout) | FIXED — project-root-derived asset/skills paths (mirror fallback), guarded config hook, dead code removed, installed-layout + walk-up test cases |
| CR1-3 | `hooks/cursor-session-start.sh:81` | R3 unreadable-config scenario: `[ -f ]`-only — a chmod-000 config still injected (Spec critical-candidate; verified empirically pre-fix: unreadable → injected) | FIXED — `&& [ -r ]`; same isFile+R_OK hardening applied to the pi/omp/kilo/opencode project walks + `hooks/session-start.js` (R3 SHALL consistency, all 5 hosts) |
| CR1-4 | `plugins/opencode/skillgrid-router.ts:186-203` | opencode ≥2.2 (current major) path (a): `setup()` for-awaits an unbounded event stream — the setup Promise may never settle (hang / deferred plugin load); every live run used 2.0.28 path (b) (Red-team F1) | FIXED — background IIFE listener; `setup()` resolves immediately |
| CR1-5 | pi/omp/kilo resolvers | Whitespace-only installed asset shadows the checkout leg (Edge-case; opencode already trim-guarded) | FIXED — trim-guarded all three resolvers |
| CR1-6 | `hooks/session-start.js:110` | Detection used `process.cwd()`, ignoring the harness payload `cwd` (Edge-case + Security) | FIXED — payload cwd preferred (fallback `process.cwd()`) + regression case |
| CR1-7 | `plugins/opencode/skillgrid-router.ts` | plan-mode primary agent filtered as subagent; `@opencode/plugin` resolving without a `Plugin` export fail-closed the module load; messages branch re-unshifted on reused arrays; one stream-iteration throw killed the loop (Edge-case set) | FIXED — `plan` allowed as main; import guard; messages-branch dedup; per-iteration resilience |
| CR1-8 | `report.md:64,72` | R1 claim-vs-evidence mismatch: marker edit attributed to the TICKET-07 opencode run, which recorded no edit; the only edit (TICKET-01 Oracle 4) ran pre-installed-copy (Red-team F2 + Verification-gap) | FIXED — both lines corrected to the evidence |
| CR1-9 | `docs/user-guide/04-hooks.md` | Claimed `.claude/settings.json` wiring that does not exist in the repo (Standards); omp "superseded" claim overstated the dormant-hook state (Red-team F4) | FIXED — wiring corrected to the actual hook-bridge facts; omp wording made precise with the named follow-up |
| CR1-10 | `hooks/session-start.js:61-74` | `.skillgrid/config.yaml` as a directory passed `existsSync`+`R_OK` (Edge-case) | FIXED — `statSync().isFile()` in all five walks |

### Minor (fixed along with the round)

| ID | Finding | Disposition |
|---|---|---|
| CR1-11 | kilo dead code: `extractAndStripFrontmatter`, `normalizePath`, `envConfigDir`/`configDir` (Standards + Edge-case) | FIXED — deleted |
| CR1-12 | kilo `_bootstrapCache` module-level staleness across test cases (Verification-gap) | FIXED — cache keyed by session cwd; `__resetBootstrapCacheForTests` clears the key |
| CR1-13 | opencode test `REPO_ROOT = process.cwd()` — suite cwd-dependent (Standards + Edge-case) | FIXED — derived from the test file location |

### Parked (deferred, logged — no in-scope fix)

| ID | Finding | Follow-up owner |
|---|---|---|
| CR1-P1 | `README.md:42` documents `--agents …,omp` — post-rename key set; at the pinned head the key is `skill-pi` (unknown-key failure after the npm batch until the rename lands) (Red-team F3) | the in-flight `skill-pi`→`omp` rename commit must reconcile the README key set |
| CR1-P2 | opencode ≥2.2 path (a) live-unverified (no ≥2.2 runtime on the machine; stub tests self-consistent) (Verification-gap) | held-out: live `opencode run` ≥2.2 with proxy-captured request asserting the `<context key="skillgrid-router">` block |
| CR1-P3 | `session-start.js` rewritten output contract has no live host-consumption observation (headless omp does not surface hook context) (Verification-gap) | held-out: live omp / Claude Code session with the extension disabled |
| CR1-P4 | installed-vs-checkout precedence (both sources present, different content) untested — QA W016, adjudicated SUGGESTION | temp-HOME marker test (cursor/kilo suites) when the layout stabilizes |
| CR1-P5 | omp dual mechanism: hook bridge still writes the SessionStart hook for omp; dormant under the current dispatch order (probe-verified TICKET-07); a dispatch-order change double-injects the 14KB router (Red-team F4) | drop the SessionStart entry from `installOmpHookBridge` if omp dispatch ever changes (doc now states this) |
| CR1-P6 | opencode `MAX_WALK_UP = 12` vs unbounded walks elsewhere — R3 divergence for projects >12 levels deep (Standards) | align or document the bound |
| CR1-P7 | installer skip-tolerance stats repoRoot only (staged-fallback edge); retired plugin files + config entries never dropped (Edge-case) | ADR-0067 tree rework or an explicit retirement pass |
| CR1-P8 | kilo dual-channel guards independent — both channels can inject (Standards) | env-limited runtime check |
| CR1-P9 | `case "skill-pi"` test branch stale against the in-flight omp rename (user WIP) (Standards) | the rename commit |
| CR1-P10 | cursor walk on a relative payload cwd (cursor passes absolute; T1 already bounded the walk) (Edge-case) | env-limited |
| CR1-P11 | `session-start.js` stdin without EOF awaits until harness timeout (harness closes the pipe by contract) (Edge-case) | harness-contract dependent |
| CR1-P12 | chmod-000 test flaky when run as root (Edge-case, env-dependent) | skip-with-note if it bites |
| CR1-P13 | `review/smoke-evidence.md` duplicate TICKET-02/03 sections (evidence hygiene) | next evidence append |

### Rejected (false positives / non-issues, verified at location)

Security's 11 excluded candidates (no exploit path; documented in its report), SpecLens's kilo "divergence from opencode walk-up" (residual after the CR1-2 fix — the mirror fallback covers installed machines), the omp divergence call on the pinned head (spec-faithful: observable contract met, mechanism not the authority), Edge-case's opencode V1 claim ("no bootstrap mechanism for 1.x" — path (b) context-hook fallback is exactly that mechanism; the residual is env-limited verification, logged as CR1-P2).

## Verification (evidence)

- All five affected suites green post-fix: pi `ALL TESTS PASS`, omp `6 checks passed`, kilo `PASS: all kilo router plugin cases green`, opencode `15 checks passed`, session-start `pass 6, fail 0`.
- 10 new regression cases: walk-up (pi, omp, kilo, opencode), kilo installed-layout (plugin loaded from a temp `~/.config/kilo/plugin/`), omp static-fallback, session-start router-missing + payload-cwd.
- Cursor R3 scenario re-verified empirically post-fix: chmod-000 config → 0 bootstrap mentions; readable config from a subdir → bootstrap present.
- Go sources untouched this round (user WIP in flight); no installer behavior change.

## Verdict

**PASS** — round-1 fix loop applied; 34 distinct findings resolved (13 fixed, 13 parked with named follow-ups, 8 rejected with evidence); no CRITICAL open. Next: ship.
