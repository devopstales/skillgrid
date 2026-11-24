# Mnemonic Pi / omp Memory Plugin — Design

Status: approved design (brainstorming). Next step: implementation plan via `superpowers:writing-plans`.
Related: `2026-11-24-mnemonic-sqlite-memory-design.md` (core memory), plans in `docs/superpowers/plans/`.

## 1. Goals and decisions

- Give Pi and omp (a Pi fork) agents Mnemonic memory as **native tools** (`pi.registerTool`), not MCP.
- Transport: spawn the `mnemonic` CLI (`mnemonic memory call <tool>`), JSON in/out, **fail-open**, 5 s default timeout.
- Packaging: ship **`@devopstales/mnemonic-memory-pi`** only. Create `@devopstales/mnemonic-memory-omp` **only if** the Pi package is not enough on omp (decided by the Task 0 spike).
- Behaviours in v0.1 (all selected): session lifecycle, native tools, protocol injection, per-prompt capture, compaction recovery, passive capture, replay-safe saves, cross-project saves, status UI + doctor.
- Reference only (do not copy): `~/git/ai-test/engram/plugin/pi` (3.4k-line monolith, HTTP, source-splicing tests, per-request runtime-contract gate) and `skillgrid-v2/plugins/memory/pi-omp` (v0, old CLI verbs).

Evidence omp can run Pi packages: omp 18.16.0 loads plain Pi packages (`pi.extensions` manifest, peerDeps `@earendil-works/pi-coding-agent|pi-tui`, `typebox`), e.g. `pi-lens`, `rpiv-todo` in `~/.omp/plugins`. Only `omp-designer` ships a separate `omp` manifest. Known deltas: omp `before_agent_start` has `reason: "prompt"|"revive"`, `session_shutdown` has no reason, subagents identified via `ctx.agent.kind`.

## 2. Package layout

Location: `plugins/pi/pi-mnemonic-memory/` (if an omp package is ever needed: `plugins/omp/omp-mnemonic-memory/`, a thin entry reusing the Pi controller plus an `omp.extensions` manifest).

```text
plugins/pi/pi-mnemonic-memory/
├── package.json                 # pi.extensions ["./src/index.ts"]; zero runtime deps;
│                                #   optional peerDeps: pi-coding-agent, pi-tui, typebox
├── src/
│   ├── index.ts                 # default export register(pi): builds controller, registers tools+hooks (<150 LOC)
│   ├── host.ts                  # HostAdapter: only file aware of pi vs omp event differences
│   ├── controller.ts            # createController({transport, host, env, clock}); no module globals
│   ├── transport.ts             # Transport interface + spawnTransport
│   ├── redact.ts                # <private>…</private> -> [REDACTED], recursive over args
│   ├── instructions.ts          # MEMORY_INSTRUCTIONS
│   ├── tools/{schemas,register,chrome}.ts
│   └── hooks/{session,prompt,protocol,compaction,passive}.ts
└── test/                        # node:test, injected fake Transport/host
```

- `host.ts` normalises: prompt event -> `{kind: "user"|"synthetic"|"revive", prompt}`; real shutdown vs reload; `isSubagent(ctx)`; `hostName`.
- `typebox` / `pi-tui` are imported lazily and only in `tools/register.ts` and `tools/chrome.ts`. If missing: plain JSON-schema parameters and default rendering. The plugin must never fail to load.

## 3. Go-side contract (`mnemonic memory call`)

```text
mnemonic memory call <tool_name> [--timeout 5s]
  stdin : JSON object of tool arguments (same schema as the MCP tool)
  stdout: {"ok":true,"result":{...}}
  failure: {"ok":false,"error":{"code":"CONFLICT","message":"...","retryable":false}} + exit code 2-8
mnemonic memory call --schema      # dumps all tool schemas (JSON), used by the plugin and contract test
```

- A single Go tool registry (`internal/tools`: name, JSON schema, handler, tier) feeds MCP (Plan 3), the daemon (Plan 4) and `memory call`. `memory call` and `--schema` are added to **Plan 2**.
- Binary resolution: `MNEMONIC_BIN` -> `mnemonic` on `PATH` -> `~/.skillgrid/mnemonic/bin/mnemonic`. Missing binary = offline mode (tools return a clear error, hooks no-op).
- Concurrency: at most 4 spawns in flight. Hook calls are fire-and-forget and bounded; tool calls are awaited.
- Capability gating: the plugin registers only tools present in the `--schema` dump, so v0.1 works with Plan 1 + `call` and gains tools as later plans land.

```ts
interface Transport {
  call<T>(tool: string, args: object, opts?: {timeoutMs?: number; signal?: AbortSignal}): Promise<Result<T>>;
  schemas(): Promise<ToolSchema[]>; // cached per process
}
```

Future: a daemon-HTTP transport can implement the same interface.

## 4. Hook behaviours

Rule for all hooks: **fail open**. A failed call never throws into the host; it flips the status to `offline` and logs the reason for `/mnemonic doctor`.

| Hook | Behaviour |
|---|---|
| `session_start` | Skip subagents. Mint `<host>-<ts36>-<hex>` id. Project: `MNEMONIC_PROJECT` > git toplevel basename > cwd basename. Call `mem_session_start {id, project, directory}` (idempotent by id; reload/resume create no duplicates). |
| `session_shutdown` | End only on a real exit (adapter decides); no-op on reload. `mem_session_end` bounded to 2 s; also end satellite sessions. |
| `before_agent_start` (user prompts only) | Capture: redact, skip if <=10 chars / synthetic / subagent, fire-and-forget `mem_save_prompt` (server dedups by session+hash). Protocol injection: if any `mem_*` tool active and `MNEMONIC_PI_INJECT != 0`, append `MEMORY_INSTRUCTIONS` to the system prompt; include the one-shot recovery notice if pending. |
| `session_compact` | Send the compaction summary to `mem_session_summary` automatically, then `mem_context`, and queue a recovery notice injected once as a hidden message (`display:false`) next turn. |
| `tool_execution_end` | Passive capture via `mem_capture_passive`, `MNEMONIC_PI_PASSIVE=off|task|all`, **default `all`**. Guards: skip `mem_*` results (no feedback loop), skip results <200 chars, dedup by content hash per session, redact, cap 32 KiB, bounded queue of 8 (overflow dropped and counted in doctor), fire-and-forget. |
| `mem_save` tool | Plugin attaches a fresh `operation_id` (UUID); on timeout/transport error retries once with the same id; result flagged `replayed` (server `save_operations` ledger). |
| Cross-project | Tools accept optional `project` and `cwd`; a different project lazily opens a satellite session `<sid>@<project>`, cached per project. |

- Redaction: `<private>…</private>` is redacted in every string argument client-side; Go redacts again (defence in depth).
- Status: `ctx.ui.setStatus("mnemonic", "mem · <project> ✓|offline")` when a UI exists. `/mnemonic doctor` shows binary path/version, schema count, current project, last 5 errors, drop counter; falls back to a `mnemonic_doctor` tool if the host has no command API.
- Tool tiers. Direct: `mem_save`, `mem_search`, `mem_context`, `mem_get_observation`, `mem_session_summary`, `mem_update`, `mem_suggest_topic_key`, `mem_current_project`, `mem_save_prompt`, `mem_capture_passive`. `exposure:"deferred"`: `mem_lookup`, `mem_pin`, `mem_unpin`, `mem_judge`, `mem_judge_many`, `mem_compare`, `mem_consolidate`, `mem_list_projects`, `mem_search_prompts`. Compact one-line result rendering when `pi-tui` is available.

## 5. Packaging and install

- npm `@devopstales/mnemonic-memory-pi`, `files: [src, README.md]`, `pi.extensions: ["./src/index.ts"]`, published with provenance.
- Pi: `pi install npm:@devopstales/mnemonic-memory-pi`. omp: `omp plugin install @devopstales/mnemonic-memory-pi` (or drop in `~/.omp/plugins`).
- `mnemonic setup pi|omp` (Plan 5) will automate install + `call --schema` check; out of scope here.

## 6. omp compatibility spike (Task 0, gate)

1. Install the built package into omp 18.16.0 with a probe tool.
2. Log raw payloads of `session_start`, `before_agent_start`, `session_shutdown`, `session_compact`, `tool_execution_end`.
3. Verify `@earendil-works/pi-tui` / `typebox` imports resolve, and that `exposure:"deferred"`, `appendSystemPrompt` and hidden custom messages work.
4. Record results in `docs/.../omp-compat.md`.

Outcome: only `host.ts`-absorbable differences -> no omp package. Real failure in tool registration/rendering/packaging -> create `plugins/omp/omp-mnemonic-memory/` (thin entry + `omp.extensions`).

## 7. Testing

- Unit (`node --test`, type stripping): `redact`, controller state machine, transport (stub script that prints canned JSON / fails / hangs), host adapter with pi and omp fixtures, tool schemas.
- Contract: TypeBox schemas vs `mnemonic memory call --schema` snapshot against the real Go binary in CI (skipped if absent).
- Fail-open: missing binary, timeout, malformed JSON must never reject a hook.
- E2E smoke (manual, documented): Pi and omp with a temp `MNEMONIC_HOME`: save, search, compact; verify with `mnemonic memory search`.
- Banned: source splicing and `new Function` sandboxes.

## 8. Rollout order

1. Task 0: omp spike.
2. Go `memory call` + tool registry (added to Plan 2).
3. Plugin core: transport, redact, controller, host.
4. Tools.
5. Hooks (session, prompt/protocol).
6. Compaction recovery.
7. Passive capture.
8. Status + doctor.
9. Packaging and docs.

## 9. Risks / open items

- omp import resolution and API deltas are unverified until the spike.
- Passive capture default `all` is noisy; mitigated by guards and `MNEMONIC_PI_PASSIVE`.
- Plugin v0.1 depends on `memory call` (Plan 2) shipping; until then only a stubbed transport can be tested.
