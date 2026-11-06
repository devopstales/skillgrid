## Task 05 — handoff-watchdog — Report

### 05.1 — Usage-signal decision (documented in `watchdog.go`)

**Chosen: caller-supplied fraction (`usage`, 0.0–1.0), NOT an internal token estimate.**

`Check` takes the current context-usage as a fraction supplied by the caller —
a client context-`%` / token-ratio. The `Check` signature is
`Check(ctx, db, projectID, projectRoot, usage float64, b Bundle)`. The interface
"takes a fraction either way" (per the brief), so the caller can report a
context percentage or a token ratio and the watchdog treats both identically.

Rationale (also in the `watchdog.go` comment):
- **Hermetic / dependency-free.** The fraction is an *input*, not computed from
  a model — so the watchdog never needs a tokenizer and tests drive it directly.
- **Single code path.** A `%` and a token-ratio both normalize to [0,1], so one
  comparison against the threshold covers both signals.
- **No new surface.** It does not add an MCP tool or a Fact Memory store — it
  just gates the same `Relay.Handoff`.

### Off by default / never always-on

- `EnvWatchdog = "SKILLGRID_HANDOFF_WATCHDOG"` (enable gate) and
  `EnvWatchdogThreshold = "SKILLGRID_HANDOFF_WATCHDOG_THRESHOLD"` (fraction).
- `Check` first evaluates the gate; when it is unset / falsy (`""`, `0`, `off`,
  `false`, `no`) it returns immediately — a **no-op**. No row, no bundle, no
  error.
- Therefore the default state (no env, no flag) is a no-op and the watchdog
  **never auto-hands-off** unless explicitly enabled **and** at/past threshold.
  It is never always-on.

### 05.2 — RED → GREEN (enabled + past threshold → SAME Handoff path)

**RED (05.2.b):** `watchdog_test.go` written; `go test -run 'Watchdog'` failed
at build — `undefined: EnvWatchdog`, `EnvWatchdogThreshold`, `Check`
(`[build failed]`).

**GREEN (05.2.d):** after implementing `watchdog.go`, `go test -run 'Watchdog'`
→ all PASS.

The "same Handoff path" is **proven, not assumed**: `TestWatchdog` asserts the
`Check` result reports `HandedOff == true` **and** that a `session_handoffs` row
was written (count == 1) **and** that the 3 `.skillgrid/.cleave/` files are on
disk (each path `os.Stat` succeeds). This is exactly what `Relay.Handoff`
produces for an operator handoff — the watchdog literally calls
`relay.Handoff(ctx, db, projectID, "", projectRoot, b)`, not a re-implementation.

### 05.3 — Disabled / default / below-threshold → no-op (no row, no bundle)

- `TestWatchdogDisabled`: gate unset (`""`), usage 0.99 → no-op. Asserts
  `HandedOff == false`, **0 handoff rows**, and **no** `NEXT_PROMPT.md` in the
  cleave dir (`os.IsNotExist`).
- `TestWatchdogDefault`: both env vars cleared (true default state), usage
  0.999 (far past any sensible threshold) → no-op. Proves it is **never
  always-on**. Asserts 0 rows + no cleave bundle.
- `TestWatchdogBelow`: gate enabled, threshold 0.8, usage 0.5 (< 0.8) → no-op.
  Asserts 0 rows + no cleave bundle.

Each asserts **no handoff row AND no cleave bundle** (not merely "no error").

### 05.4 — Invalid config fails closed (clear error, no auto-handoff)

`TestWatchdogInvalidConfig`: gate enabled, `SKILLGRID_HANDOFF_WATCHDOG_THRESHOLD
= "not-a-number"`. `Check` returns a **clear config error** naming the threshold
("…invalid SKILLGRID_HANDOFF_WATCHDOG_THRESHOLD \"not-a-number\"; must be a
number in [0,1]"), `HandedOff == false`, and asserts **0 rows + no cleave
bundle** (no auto-handoff). An enabled-but-empty threshold also fails closed.

### Test suite output

Focused (RED→GREEN):
```
go test ./skillgrid-cli/internal/mnemonic/relay/ -run 'Watchdog' -count=1 -v
--- PASS: TestWatchdog (0.11s)
--- PASS: TestWatchdogDisabled (0.10s)
--- PASS: TestWatchdogDefault (0.13s)
--- PASS: TestWatchdogBelow (0.10s)
--- PASS: TestWatchdogInvalidConfig (0.10s)
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/relay  1.308s
```

Full relevant suite:
```
go test ./internal/mnemonic/relay/ ./internal/mnemonic/mcp/ -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/relay   5.874s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp    42.507s
```

005/008/010/011/013 baselines (still pass):
```
go test ./internal/mnemonic/memory/... ./internal/mnemonic/route/... \
  ./internal/mnemonic/affected/... ./internal/mnemonic/community/... \
  ./internal/mnemonic/pdg/... ./internal/mnemonic/codeindex/... -count=1
ok  .../internal/mnemonic/memory          12.199s
ok  .../internal/mnemonic/memory/layer     6.966s
ok  .../internal/mnemonic/route            3.804s
ok  .../internal/mnemonic/affected        12.737s
ok  .../internal/mnemonic/community        4.852s
ok  .../internal/mnemonic/pdg              4.994s
ok  .../internal/mnemonic/codeindex       17.714s
```

### Commits

- `d529769` — feat(mnemonic): optional session handoff watchdog

### Global constraints held

- Off by default; never always-on. ✔ (default state is a no-op)
- Disabled / below threshold → no-op (never auto-handoff). ✔
- Invalid config → fail closed (clear error, no auto-handoff). ✔
- Session Relay only — no Fact Memory, no separate MCP tool, no always-on
  watchdog. ✔
- Triggers the SAME `Relay.Handoff` path (calls `relay.Handoff`, not a
  re-implementation). ✔

### Concerns

- **Env var names:** `SKILLGRID_HANDOFF_WATCHDOG` (enable gate) +
  `SKILLGRID_HANDOFF_WATCHDOG_THRESHOLD` (fraction 0.0–1.0). The brief named the
  gate explicitly; the threshold name is my addition following the repo's
  `MNEMONIC_*` env convention (upper-snake, namespaced by product).
- **Wiring:** the watchdog is a **library** — `relay.Check` is the call site the
  CLI (`cmd/skillgrid/session.go`) or MCP (`internal/mnemonic/mcp/tools_session_*.go`)
  can invoke later. It is NOT yet wired into a live call path in this step (the
  brief scopes it to `watchdog.go` + `watchdog_test.go`; no new MCP tool is added).
  A future step can call `relay.Check` at session-turn boundaries. The fraction
  (`usage`) is caller-supplied, so the caller is the one that knows the
  context-`%`/token-ratio at check time.
