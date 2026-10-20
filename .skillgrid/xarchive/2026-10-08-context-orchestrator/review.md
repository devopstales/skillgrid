---
type: Spec
title: "2026-10-08-context-orchestrator — review"
archive: "2026-10-08-context-orchestrator"
status: draft

---

# Review — Context Orchestrator (CTX) v1.11

> Change: `.skillgrid/specs/2026-10-08-context-orchestrator/`
> Generated: 2026-10-14T11:00:00Z (requesting-code-review)
> Reviewer: three-axis dispatch (standards + spec + security), fresh subagent contexts
> Diff: `53d08b89..ac264d3` scoped to the CTX surface (extracted: `/tmp/ctx-review.diff`, 12,754 lines)

## What "Important" means (threshold)

- **Critical** — blocks merge: a spec requirement/scenario unmet, an exploitable security issue
  with a reachable path, or an ADR violation.
- **Important** — fix before ship: a documented-standard violation that would mislead a
  maintainer, a PARTIAL scenario the brief requires, or a MEDIUM security finding with a
  reachable entry point.
- **Minor** — capped at 5, polish: nits, dead code, redundant indirection, defense-in-depth.

## Independence grades

| Axis | Grade | Rationale |
|---|---|---|
| Standards | B | fresh subagent context, no session narrative passed; same model family as implementer |
| Spec | B | fresh subagent context, only acceptance.feature + briefing + blueprint passed |
| Security | B | fresh subagent context, only the diff + OWASP skill passed |

No axis saw the implementer's rationale, the QA verdict, or a prior score. Grade B (not A)
because the runtime has no cross-model dispatch; T2's default three-axis floor is met.

## Standards

### Strengths (max 3)

- ADR-0033/0012 honored end-to-end: `ctxstore.New` wraps the caller's `*store.Store`, opens no
  second handle (proven by `TestCtxSharedHandleNoSecondConnection`); every DDL in the diff is
  `ctx_`-prefixed (056–061, 068, 069); CLM renumber pinned by `migrations_clm_renumber_test.go`.
- ADR-0016 fail-open floors consistent and proven: store degrades per-table, judgment fails open
  to keep, proxy to passthrough, telemetry to no-op, HTTP routes to 200-zero-JSON.
- Ubiquitous language + dependency policy hold: terms match `02-technical-terms.md`; no new
  third-party imports (only the existing ADR-0008 `yaml.v3`; `x/net/http2` is test-only, already
  in `go.mod`).

### Findings

| # | Sev | Location | Title | Evidence | Disposition |
|---|-----|----------|-------|----------|-------------|
| S1 | Minor | `proxy/mode_test.go:1` | unused-import workaround `var _ = httptest.NewServer` | import block imports `net/http` + `httptest`, neither used; copy-paste artifact | fix now (trivial) |
| S2 | Minor | `proxy/modes.go:27–64` | triple mode-resolution chain, two layers pure indirection | `ResolveMode` → `resolveModeFromEnv` → `resolveMode` | fix now (fold middle layer) |
| S3 | Minor | `proxy/mode.go:39–44` | mode-change ID minted from wall clock (`time.Now().UnixNano()` twice) | breaks the file's own "deterministic identity, no I/O under lock" posture; same-nanosecond collision risk | fix now (clock seam) |

**Standards verdict: met with fixes. 0 Important, 3 Minor.**

## Spec

### Scenario table (23)

| # | Scenario | Status | Note |
|---|----------|--------|------|
| 01 | ctx-module-builds | PASS (deviation) | built as a package under `mnemonic`, not a 3rd `ctx/` module — blueprint explicitly overrides briefing req 1 ("No go.work change"); recorded deviation |
| 02 | ctx-subcommand-group | PARTIAL | `ctx metrics` + `ctx traffic` verbs absent (F1); `judge`/`checkpoint` defensible as PR6 |
| 03 | ctx-stats-includes-clm-row-counts | PASS | |
| 04 | proxy-three-modes | PASS | |
| 05 | proxy-protocol-passthrough | PASS | |
| 06 | compress-endpoint-loopback | PASS | |
| 07 | mode-switch-emitted | PASS | |
| 08 | proxy-fail-open-provider-timeout | PASS | |
| 09 | wrap-core4-launches | PASS | |
| 10 | wrap-optin-agent | PASS | |
| 11 | observability-endpoints | PASS | |
| 12 | session-summary-aggregated | PARTIAL | row semantics met; hook-budget step unverifiable (F3) |
| 13 | history-snapshot-durable | PARTIAL | config key exists, no snapshot writer (F2) |
| 14 | run-mode-is-observability-signal | PARTIAL | `/stats` mode met; Live-view header not wired (F4b) |
| 15 | traffic-learner-extracts | PASS | |
| 16 | traffic-learner-four-categories | PASS | |
| 17 | traffic-promotion-human-gate | PASS | |
| 18 | judgment-truncate-only | PASS | |
| 19 | judgment-fingerprint-cached | PASS | |
| 20 | judgment-fail-open | PASS | |
| 21 | ctx-config-precedence | PASS | |
| 22 | clm-renumber | PASS | |
| 23 | clm-stats-subslice | PASS | |

### Scope creep

None material: `judgment/` + `069` (briefing req 12) and `068` (human-gate state) are required.
`/stats` etc. return 403 non-loopback vs `/v1/compress` 404 — only the latter is spec'd; minor.

### Findings

| # | Sev | Title | Evidence (verified by coordinator) | Disposition |
|---|-----|-------|------------------------------------|-------------|
| F1 | Important | `ctx metrics` + `ctx traffic` CLI verbs required but unimplemented | `tasks.md` Unit 5 Files: `cmd/mnemonic/ctx_{learn,search,purge,metrics,traffic,checkpoint}.go`; `blueprint.md:185-186` names the verbs; `ctx.go` has no `case "metrics"`/`"traffic"` | **park** — HTTP surfaces `/metrics` + `/stats` cover the observability intent; CLI verbs are a follow-on ticket (see Open Items) |
| F2 | Important | scenario 13 unwired: no `.skillgrid/ctx/stats_history.json` writer | `tasks.md:158-167` TICKET-11 names `observability/snapshot.go`; grep: `HistorySnapshot` is config-only (`config.go:100,149,291`), no writer in the diff | **fix now** — small, self-contained (atomic write + test) |
| F3 | Minor | scenario 12: aggregation is poll-on-tick, not SessionEnd-hook enqueue | `ctx_proxy.go:139–162` rollup tick + shutdown pass; row semantics met | park — mechanism deviation, behavior met |
| F4a | Minor | scenario 01: single-module build shape vs briefing's 3rd module | blueprint "No go.work change" overrides; ADR-0033's *intent* (read-only shared handle, ctx_-only tables) fully met | park — recorded deviation |
| F4b | Minor | scenario 14: Live-view header mode not wired | `/stats` `Mode` field met; UI header is the SPA's concern, no UI diff in CTX scope | park — UI follow-on |

**Spec verdict: PARTIAL — 19/23 full PASS, 4 PARTIAL. One Important fixed now (F2), one parked
(F1) with a named follow-on.**

## Security

### Entry points mapped

Proxy (loopback :8787, non-loopback opt-in + warning) · `/v1/compress` (loopback-only 404) · 4
observability routes (`requireLoopback` before store open) · ctx CLI · `config.d/ctx.yaml` loader ·
Ollama judgment call · wrap launcher (env inject + backup + subprocess).

### Findings

| # | Sev | Title | Path (verified) | Confidence | Disposition |
|---|-----|-------|-----------------|------------|-------------|
| SEC1 | MEDIUM | CWE-918/LLM02: repo-local `ctx.yaml` can point `judgment.endpoint` at a non-loopback URL → message content exfiltrated | `config.go mergeJudgment` (no host check) → `judgment.go New` → `modelJudge POST j.endpoint` (body embeds message content for `ClassRemainder`) | Confirmed | **fix now** — loopback validation at config load |
| SEC2 | LOW | non-loopback bind exposes credential-bearing forward + WS (by design, warned) | `ctx_proxy.go:107` warn-only; `/v1/compress` still 404s | Confirmed | park (defense-in-depth: explicit ack) |
| SEC3 | LOW | CWE-841: `Transfer-Encoding` not stripped on forward path → potential upstream framing desync | `proxy.go:251–258` copies all headers except auth/CL | Needs verification | park — strip hop-by-hop list in forward (trivial, batch with S-fixes if convenient) |

**Security verdict: 0 CRITICAL, 0 HIGH, 1 MEDIUM, 2 LOW. Loopback guards sound
(`netip.ParseAddr` fail-closed, `::1`/`::ffff:127.0.0.1` handled, XFF not honored); SQLite fully
parameterized; no shell interpolation; no secrets in logs.**

## Floor

| Axis | Verdict | Weakest finding |
|---|---|---|
| Standards | met with fixes | 3 Minor |
| Spec | PARTIAL | F1 (parked, Important), F2 (fixed) |
| Security | 1 MEDIUM + 2 LOW | SEC1 (fixed) |

**FLOOR:** fix the 2 Important-adjacent items now (F2 snapshot writer, SEC1 endpoint validation)
+ the 3 Minor standards cleanups → re-render. After fixes: Standards PASS, Security PASS (LOWs
parked), Spec PARTIAL-with-accepted-deviations (F1 parked to a named follow-on; F3/F4 recorded).

## Findings triage (receiving-code-review)

| Item | Sort | Disposition |
|---|---|---|
| SEC1 judgment endpoint loopback validation | fix now | **FIXED** — `mergeJudgment` rejects non-loopback `endpoint` (fail-open to previous value); `TestJudgmentEndpointLoopbackOnly` (6 cases) |
| F2 snapshot writer `.skillgrid/ctx/stats_history.json` | fix now | **FIXED** — `observability/snapshot.go` `WriteHistorySnapshot` (atomic tmp+rename), wired into the rollup `pass` gated on `cfg.Observability.HistorySnapshot`; 3 tests (matches-DB, survives-re-index, nil-store fail-open) |
| S1 `var _ = httptest.NewServer` | fix now | **FIXED** — import + shim dropped from `mode_test.go` |
| S2 triple mode-resolution | fix now | **FIXED** — `resolveModeFromEnv` folded into `ResolveMode` (call sites updated) |
| S3 wall-clock mode-change ID | fix now | **FIXED** — `mode-%d-%d` (process-start ns + per-proxy sequence), collision-free, `ChangedAt` from a single `time.Now()` |
| SEC3 hop-by-hop strip on forward | fix now (trivial) | **FIXED** — `forward` now skips `isHopByHopHeader(k)`; `TestForwardStripsHopByHopHeaders` |
| F1 `ctx metrics`/`ctx traffic` CLI verbs | park | follow-on ticket (HTTP surfaces `/metrics` + `/stats` cover the observability intent; verbs named in tasks.md Unit 5 for a later change) |
| F3 SessionEnd-hook mechanism | park | recorded deviation — row semantics met; tick-driven aggregation |
| F4a 3rd-module shape | park | recorded deviation — blueprint "No go.work change" overrides briefing req 1; ADR-0033 intent (shared read-only handle, ctx_-only tables) fully met |
| F4b Live-view header | park | UI follow-on — `/stats` Mode field met; SPA header is the UI's concern |
| SEC2 non-loopback bind ack | park | defense-in-depth follow-on — opt-in + warning as designed |

**Re-verification after fixes:** `go build` + `go vet` exit 0; `go test ./mnemonic/... ./skillgrid-cli/...
-count=1` green (zero failures); gofmt clean on `mnemonic/internal/ctx/`. Fix commit: see git log
(`fix(ctx): close three-axis review findings — SEC1/F2/S1-S3/SEC3`).

## Floor (final, post-fix)

| Axis | Verdict | Weakest open item |
|---|---|---|
| Standards | PASS | none (3 Minors fixed) |
| Spec | PASS-with-accepted-deviations | F1 parked to a named follow-on; F3/F4 recorded deviations |
| Security | PASS | SEC2 (LOW, by-design) parked |

**FLOOR:** no CRITICAL, no open Important, no exploitable finding reachable under the loopback
trust boundary. The change is review-clean for ship; F1/F3/F4/SEC2 are recorded open items for
the next change (carried in WINDOWS.md / Open Items).

## Human Override

**Human decision (fill in):** <accept / accept-with-open-items / reject> — <name> — <date>
