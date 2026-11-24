# Mnemonic Pi/omp Code Index Plugin Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship `@devopstales/mnemonic-code-pi` — a pi/omp extension that (1) automatically appends compact code-graph context (callers/callees/tests/contracts/linked memories, ≤1.5 KiB) to grep/bash/read results, and (2) exposes the `mnemonic code` subsystem as native, capability-gated `code_*` tools, `/codeindex` commands, status line, and 4 workflow skills — backed by three new Go verbs (`code call`, `code call --schema`, `code augment`) on the Phase 2A surface.

**Architecture:** Go side: a single `internal/code/tools` registry (name, draft-07 schema, readonly/defer/admin flags, per-tool timeout, run func) that both the Phase 2A MCP server and the new `code call` verb iterate — one source of truth; `code call --schema` dumps it; `code augment` composes store queries (edges, test refs, `observation_code_links`) into a budgeted text block. TS side: `plugins/pi/pi-mnemonic-code/` — a spawn-based `Transport` (`mnemonic code call` / `code augment`, JSON envelope, ≤4 in flight, fail-open), a stateless-module `Controller` (binary probe, index-state cache, session dedup caches, drop/error counters), capability-gated tool registration driven entirely by the `--schema` dump, a ported pi-gitnexus pattern extractor, and a `tool_result` hook that appends augment blocks.

**Tech Stack:** Go 1.26 (`CGO_ENABLED=0`), `modernc.org/sqlite`, `github.com/mark3labs/mcp-go` (existing), Node ≥23.6 (native type stripping) + `node:test`, `typebox` (regular dependency), `@earendil-works/pi-tui` (optional peer, lazy), pi extension API (`pi.registerTool`, `pi.on('session_start'|'before_agent_start'|'tool_result'|'session_shutdown')`, `pi.registerCommand`, `pi.registerFlag`).

**Spec:** `docs/superpowers/specs/2026-11-24-mnemonic-pi-omp-codeindex-design.md`
**Prerequisite:** Phase 2A plan (`docs/superpowers/plans/2026-11-24-mnemonic-phase2a-core-code-graph.md`) green — the Go tasks build on its `mcp`, `cli`, `engine`, `store`, `api`.

## Hypothesis

**Claim:** A pi/omp agent whose grep/read results carry appended graph context, backed by 6 always-on `code_*` tools and 5 on-demand, will resolve code-location questions with fewer tool calls than a grep-only workflow, and the plugin never throws into the host.
**Right condition:** On the fixture repo, a grep for an indexed symbol returns results + a correct `[CodeIndex: …]` block in the same tool result; `code_explore`/`code_affected` answer from the index without file reads; missing binary / corrupt DB / malformed JSON never produce a host-level hook error (doctor shows counts instead).
**Wrong condition:** The host cannot load the package, `tool_result` content cannot be modified on omp (and no fallback lands), or augment output is wrong (names/lines that don't exist in the fixture) or floods context (>8 KiB per result).
**Thinnest MVP:** Stub-binary plugin in omp: one probe tool + one augment append after grep (Tasks 1–2 of this plan cover the door check with the real Go contract stubbed).
**Door check:** Task 1 (omp compatibility spike). If the plugin can't load on omp 18.16 or `tool_result` results aren't mutable with a workable fallback, STOP — the user decides fallback (hidden-message append) or omp-specific packaging before Task 2 onward.

## Global Constraints

- Go: module path `mnemonic`; `CGO_ENABLED=0 go build ./...` and `CGO_ENABLED=0 go test -race ./...` must pass; files `<600` LOC; SQL only in `internal/code/store`; `os.Getwd()` forbidden in `internal/code/*` (paths confine to `code_index_meta.active_root`).
- Envelope contract is identical to the memory plugin (spec §3 of `2026-11-24-mnemonic-pi-omp-plugin-design.md`): stdout `{"ok":true,"result":…}` / `{"ok":false,"error":{"code","message","retryable"}}`; exit codes per `apperr` (2 `VALIDATION_ERROR` … 9 `PATH_OUTSIDE_ROOT`, 10 `INDEX_BUSY`, 11 `LSP_UNAVAILABLE`); `augment` uses exit `3` for "skip silently" (no index / no match).
- Plugin: zero dependencies beyond `typebox`; `pi-tui` optional (lazy import, notify fallback); `node:child_process` only, no shell (`shell: false`), args arrays; Node ≥23.6 type stripping for `src/*.ts` and `test/*.ts`.
- Fail-open invariant: no hook path may throw into the host; every transport failure maps to a `Controller` state change + doctor counter. Tools return readable error text, never reject.
- Augment limits: ≤3 patterns/result, 5 s per pattern, 1.5 KiB per pattern, 8 KiB combined append; session dedup (`augmented`/`empty` caches) cleared on session start and after a successful `code_index`.
- Subagents: session hooks and augment skip subagents (`ctx.agent.kind !== "main"` guard via host adapter).
- Tests: `node --test test/` (type stripping); banned: source splicing and `new Function` sandboxes. Go tests: real `gotreesitter` fixtures per Phase 2A compliance rule (mocks only for fault injection).
- Commits: conventional (`feat(code):`, `feat(plugin):`, `test:`, `docs:`); one commit per task step group as listed.
- Plan/spec amendments below are applied in Task 1 and Task 2 steps respectively.

## Spec/plan amendments made by this plan

1. Design spec §1: "2A-only binaries get 9 tools" → **11** (6 direct + 3 deferred + 2 admin, matching Phase 2A's MCP surface).
2. Design spec §4: "direct = always in the prompt (7 tools)" → **6** (the table lists six: `code_explore`, `code_read`, `code_impact`, `code_affected`, `code_status`, `code_index`).
3. Design spec §2/§5: `typebox` is a **regular dependency** (not a lazy peer) — matches the shipped v0 precedent (`plugins/pi/mnemonic/api.js` static import) and removes a load-order hazard; `pi-tui` stays optional/lazy. v0.1 `/codeindex` menu is the **text/notify menu**; the pi-tui component is v0.2.
4. Phase 2A plan Task 10: CLI list gains `call` and `augment` verbs; `mcp/server.go` tool definitions move into a shared `internal/code/tools` registry; `api.Service` gains `Augment`; `cli.IO` gains `In io.Reader`.
5. Phase 2 spec §6.3 CLI reference: add `mnemonic code call <tool> [--timeout 5s]`, `mnemonic code call --schema`, `mnemonic code augment <pattern> [--max-bytes 1536]`.

## Must-Haves (Goal-Backward Verification)

**Truths:**
1. `mnemonic code call code_status` (stdin `{}`) against an indexed fixture repo prints an `ok:true` envelope with `freshness`/counts; against a fresh data dir prints `freshness:"not_indexed"`-equivalent. (`backstop` — needs built binary + fixture index.)
2. `mnemonic code call --schema` lists exactly the tools the MCP `all` profile lists (11 in 2A) with matching draft-07 schemas.
3. `mnemonic code augment ParseURL` on the fixture prints a ≤1536-byte block whose caller/callee names and `file:line` pairs all exist in the fixture; unknown pattern → exit 3, empty stdout. (`backstop`.)
4. In omp, a grep result for an indexed symbol carries an appended `[CodeIndex: …]` block in the **same tool result** the agent sees. (`backstop` — e2e, Task 10.)
5. With the binary missing, every hook is a no-op and every tool returns a readable `OFFLINE` error; the host log contains zero uncaught hook errors. (`backstop`.)
6. A binary with a 2A-only schema dump yields exactly 11 registered plugin tools; a 2D-complete dump yields 16; unknown future tools register with pass-through schema (no hard-coded list).
7. Repeated greps of the same symbol append the block at most once per session; after `/codeindex index` succeeds the symbol can be augmented again.

**Artifacts:**
- `plugins/pi/pi-mnemonic-code/` — `package.json`, `tsconfig.json`, `src/{index,host,transport,controller,augment,schemas,tools,commands,instructions}.ts`, `test/*.test.ts`, `skills/codeindex-*/SKILL.md` (4), `README.md`.
- Go: `internal/code/tools/registry.go`(+test), `internal/code/cli/call.go`(+test), `internal/code/engine/augment.go`(+test), `internal/code/store/graph_augment.go`(queries) — plus modifications named in Task 2/3.
- `docs/superpowers/specs/omp-compat.md` — spike record (shared with the memory plugin's Task 0).

**Key links:**
- Plugin `tools.ts` registration is driven by the live `--schema` dump (contract test asserts key-set equality, Task 10).
- Hook append text == Go `augment` stdout verbatim (e2e diff, Task 10).
- `code call` and the MCP server iterate the **same** `tools.Registry()` (test: counts + schemas equal, Task 2).

**One-way-door decisions:**
- `> ⚠ one-way:` (Task 2) CLI verb names `code call` / `code call --schema` / `code augment` and the envelope are a **cross-plugin contract** (the memory plugin defines the same shape for `memory call`); additive today, expensive to rename once both plugins ship.
- `> ⚠ one-way:` (Task 9) npm package name `@devopstales/mnemonic-code-pi` and its `code_*` tool names are published contract; the v0 `@devopstales/pi-mnemonic` `code_*` tools are removed in the same window (no duplicate names coexisting).

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| Subprocess spawn of `mnemonic` | Applicable — binary path from env/PATH could be attacker-influenced in hostile envs | `MNEMONIC_BIN` → merged PATH → fixed `~/.skillgrid/mnemonic/bin/mnemonic`; `spawn(bin, args, {shell:false})`; no stdin interpolation beyond JSON; hard timeouts (5 s default, 120 s `code_index`) | `transport.test.ts`: stub binary on PATH records argv; hostile `MNEMONIC_BIN` with spaces fails cleanly to `OFFLINE`, never a shell command |
| Untrusted JSON on stdout (envelope) | Applicable — binary output is parsed untrusted | Strict envelope parse (`ok` discriminator); malformed → `BAD_ENVELOPE` error result, hook returns original content | `transport.test.ts`: stub prints garbage/ truncated JSON → `call` resolves `{ok:false,error.code:"BAD_ENVELOPE"}`, no throw |
| Untrusted augment content in agent context | Applicable — symbol names/paths are repo content injected after every grep | Byte budgets (1.5 KiB/8 KiB), labeled `[CodeIndex: …]` delimiter block, no host-side interpretation of block content, session dedup bounds repetition | `augment.test.ts` (Go): 40-caller fixture → block ≤ `--max-bytes`; TS hook test: append never exceeds 8 KiB |
| Path arguments in `code_read`/`code_neighbors` | Applicable — agent-supplied paths | Server is authoritative (`PATH_OUTSIDE_ROOT` confinement from Phase 2A); plugin passes paths through, maps exit 9 to readable error | Go test (reuse 2A suite): `/etc/passwd` → exit 9; TS: envelope error text returned, no crash |
| Config file write (`~/.pi|omp/mnemonic-code.json`) | Applicable — settings persisted to user home | Single JSON file, atomic write (tmp+rename), write errors ignored (fail-open), no secrets stored | `controller.test.ts` (via commands test): unwritable dir → settings save no-ops, plugin functional |
| Prompt/PR automation, routing, executable-file classification | N/A — the plugin spawns one known binary and appends text; no PR/route/executable classification logic | — | — |

## File Structure

```text
# Go (mnemonic module — extends Phase 2A)
internal/code/
├── tools/{registry.go,registry_test.go}       # Create: single tool source of truth (moved from mcp/server.go)
├── cli/{call.go,call_test.go}                 # Create: `code call` + `--schema` + `augment` verbs
├── engine/{augment.go,augment_test.go}        # Create: pattern → budgeted context block
├── store/graph_augment.go                     # Create: Callers/Callees/TestRefs rows (+ queries test)
├── mcp/server.go                              # Modify: iterate tools.Registry() (counts/flags unchanged)
├── api/{types.go,service.go}                  # Modify: Augment(ctx, pattern, maxBytes) on Service
└── cli/cli.go                                 # Modify: dispatch `call`/`augment`; IO gains In io.Reader

# TS (plugins/pi/pi-mnemonic-code/)
plugins/pi/pi-mnemonic-code/
├── package.json                               # @devopstales/mnemonic-code-pi, pi.extensions+skills
├── tsconfig.json
├── src/
│   ├── index.ts                               # register(pi): controller + tools + hooks + commands (<200 LOC)
│   ├── host.ts                                # pi/omp normalisation, shell PATH, binary resolution
│   ├── transport.ts                           # spawnTransport: call/augment/schemas, ≤4 in flight
│   ├── controller.ts                          # state machine: probe, index-state, dedup, counters
│   ├── augment.ts                             # pattern extraction (ported from pi-gitnexus)
│   ├── schemas.ts                             # draft-07 → TypeBox converter + per-tool metadata
│   ├── tools.ts                               # capability-gated registration
│   ├── commands.ts                            # /codeindex dispatch, settings, doctor
│   └── instructions.ts                        # CODE_INDEX_INSTRUCTIONS
├── test/
│   ├── transport.test.ts  controller.test.ts  augment.test.ts
│   ├── tools.test.ts      index.test.ts       contract.test.ts
├── skills/codeindex-{exploring,debugging,pr-review,refactoring}/SKILL.md
└── README.md
```

---

### Task 1: OMP Compatibility Spike (Door Check) + Package Scaffold

**Files:**
- Create: `plugins/pi/pi-mnemonic-code/package.json`, `tsconfig.json`, `src/index.ts` (probe version), `src/host.ts` (minimal: `isSubagent`, `isUserPrompt`, `isRealShutdown`, `detectHost`)
- Create: `test/spike.probe.md` (manual spike runbook, kept with the package)
- Create: `docs/superpowers/specs/omp-compat.md` (shared with memory plugin Task 0; create if absent)
- Create: `test/fixtures/probe-repo/` (3 Go files: `main.go` calling `ParseURL` from `url.go`, `url_test.go`)

**Interfaces:**
- Produces: `host.ts` API used by every later task:
  ```ts
  export interface Host {
    name: "pi" | "omp" | "unknown";
    configDir(): string;              // ~/.omp or ~/.pi (by detection) + "/mnemonic-code.json"
    isSubagent(ctx: unknown): boolean;
    isUserPrompt(event: any): boolean;
    isRealShutdown(event: any): boolean;
  }
  ```
- Records (in `omp-compat.md`): raw event payloads for `session_start`, `before_agent_start`, `tool_result`, `tool_execution_end`, `session_shutdown`; whether `tool_result` handler return `{content:[…]}` mutates the agent-visible result; whether `exposure:"deferred"` tools are prompt-hidden yet reachable via `tool_search`; whether `@earendil-works/pi-tui` resolves; whether plain draft-07 objects work as `parameters` or a TypeBox instance is required.

- [ ] **Step 1: Scaffold the package**

`package.json`:

```json
{
  "name": "@devopstales/mnemonic-code-pi",
  "version": "0.1.0",
  "type": "module",
  "license": "MIT",
  "description": "Mnemonic code index for pi/omp: auto-augmented search results + native capability-gated code_* tools, /codeindex commands, workflow skills.",
  "keywords": ["pi-package", "pi", "pi-extension", "omp", "code-index", "call-graph", "agent", "mnemonic"],
  "engines": { "node": ">=23.6" },
  "files": ["src", "skills", "README.md"],
  "dependencies": { "typebox": "^1.0.0" },
  "peerDependencies": {
    "@earendil-works/pi-coding-agent": ">=0.74",
    "@earendil-works/pi-tui": ">=0.74"
  },
  "peerDependenciesMeta": { "@earendil-works/pi-tui": { "optional": true } },
  "scripts": { "test": "node --test test/" },
  "publishConfig": { "access": "public" },
  "pi": { "extensions": ["./src/index.ts"], "skills": ["./skills"] }
}
```

`tsconfig.json`: `{ "compilerOptions": { "target": "es2022", "module": "nodenext", "moduleResolution": "nodenext", "strict": true, "skipLibCheck": true, "noEmit": true, "types": ["node"] }, "include": ["src", "test"] }`.

`src/host.ts` (dual-host duck typing, per the shipped v0 memory plugin's verified comments):

```ts
import { spawn } from "node:child_process";
import { homedir } from "node:os";
import { join, delimiter } from "node:path";

export interface Host {
  name: "pi" | "omp" | "unknown";
  configDir(): string;
  isSubagent(ctx: unknown): boolean;
  isUserPrompt(event: any): boolean;
  isRealShutdown(event: any): boolean;
}

export function detectHost(): Host {
  // omp is a pi fork; the loaded plugin file path distinguishes installs
  // (~/.omp/plugins vs ~/.pi/extensions). Env override wins for tests.
  const forced = process.env.MNEMONIC_CODE_HOST;
  let name: Host["name"] = "unknown";
  if (forced === "pi" || forced === "omp") name = forced;
  else if (typeof process.env.PI_PLUGIN_PATH === "string" && process.env.PI_PLUGIN_PATH.includes(`${homedir()}/.omp`)) name = "omp";
  else if (name === "unknown" && process.versions?.node) name = "pi"; // default until proven omp
  const dir = name === "omp" ? ".omp" : ".pi";
  return {
    name,
    configDir: () => join(homedir(), dir, "mnemonic-code.json"),
    // oh-my-pi (omp): ctx.agent.kind identifies subagents; absent => main.
    isSubagent: (ctx) => {
      const kind = (ctx as any)?.agent?.kind;
      return kind !== undefined && kind !== "main";
    },
    // omp: before_agent_start carries reason "prompt" (user) | "revive" (parked subagent).
    // pi: event.messageType === "user".
    isUserPrompt: (event) => event?.reason === "prompt" || event?.messageType === "user",
    // omp: SessionShutdownEvent has no reason (process exit = real). pi: reason "quit" only.
    isRealShutdown: (event) => event?.reason === undefined || event?.reason === "quit",
  };
}

/** Resolve PATH via the login shell (nvm/fnm/volta), merged over the agent PATH. Port of pi-gitnexus resolveShellPath/mergePaths. */
export function resolveShellPath(): Promise<string> {
  if (process.platform === "win32") return Promise.resolve(process.env.PATH ?? "");
  const shell = process.env.SHELL ?? "/bin/sh";
  return new Promise((resolve) => {
    let out = "";
    let done = false;
    const finish = (v: string) => { if (!done) { done = true; clearTimeout(timer); resolve(v); } };
    const timer = setTimeout(() => { child.kill(); finish(process.env.PATH ?? ""); }, 3000);
    const child = spawn(shell, ["-lc", 'printf %s "$PATH"'], { stdio: ["ignore", "pipe", "ignore"] });
    child.stdout!.on("data", (d) => { out += d.toString(); });
    child.on("close", () => {
      const seen = new Set<string>();
      const merged = [...(process.env.PATH ?? "").split(delimiter), ...out.trim().split(delimiter)]
        .filter((d) => { if (!d || seen.has(d)) return false; seen.add(d); return true; });
      finish(merged.join(delimiter));
    });
    child.on("error", () => finish(process.env.PATH ?? ""));
  });
}

/** MNEMONIC_BIN -> PATH (merged) -> ~/.skillgrid/mnemonic/bin/mnemonic */
export function resolveBinary(pathEnv: string): string | null {
  const env = process.env.MNEMONIC_BIN;
  if (env && env.trim()) return env.trim();
  const { existsSync } = require("node:fs") as typeof import("node:fs");
  for (const dir of pathEnv.split(delimiter)) {
    if (!dir) continue;
    const p = join(dir, "mnemonic");
    if (existsSync(p)) return p;
  }
  const fixed = join(homedir(), ".skillgrid/mnemonic/bin/mnemonic");
  return existsSync(fixed) ? fixed : null;
}
```

(If `detectHost` proves wrong against the recorded payloads in Step 3, fix it there and re-run the probe — the recorded payloads are the contract.)

- [ ] **Step 2: Write the probe `src/index.ts`**

```ts
import { appendFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { detectHost } from "./host.js";

const LOG = join(tmpdir(), `mnemonic-code-probe-${process.pid}.jsonl`);
const host = detectHost();

function log(kind: string, payload: unknown): void {
  try { appendFileSync(LOG, JSON.stringify({ kind, host: host.name, payload }) + "\n"); } catch { /* fail-open */ }
}

export default function register(pi: any): void {
  pi.registerTool({
    name: "probe_omp",
    label: "Probe",
    description: "Spike probe: returns PING.",
    parameters: { type: "object", properties: {} }, // plain draft-07 on purpose (spike question)
    exposure: "direct",
    async execute() { return { content: [{ type: "text", text: "PING" }], details: undefined }; },
  });
  pi.registerTool({
    name: "probe_deferred",
    label: "Probe Deferred",
    description: "Spike probe: deferred exposure.",
    parameters: { type: "object", properties: {} },
    exposure: "deferred",
    async execute() { return { content: [{ type: "text", text: "DEFERRED-PING" }], details: undefined }; },
  });
  pi.on("session_start", (_e: any, ctx: any) => { log("session_start", { cwd: ctx?.cwd, agent: ctx?.agent }); });
  pi.on("before_agent_start", (e: any, ctx: any) => { log("before_agent_start", { reason: e?.reason, messageType: e?.messageType, systemPromptLen: e?.systemPrompt?.length }); });
  pi.on("tool_result", (e: any, ctx: any) => {
    log("tool_result", { toolName: e?.toolName, contentLen: e?.content?.length });
    if (e?.toolName === "grep") {
      return { content: [...(e.content ?? []), { type: "text", text: "\n---\n[PROBE-AUGMENT]\n---" }] };
    }
  });
  pi.on("tool_execution_end", (e: any) => { log("tool_execution_end", { toolName: e?.toolName }); });
  pi.on("session_shutdown", (e: any) => { log("session_shutdown", { reason: e?.reason }); });
  try { ctxNotify(pi, `probe loaded (host=${host.name}, log=${LOG})`); } catch { /* */ }
  function ctxNotify(pi: any, msg: string) { pi.getUI?.()?.notify?.(msg, "info"); }
}
```

- [ ] **Step 3: Run the spike on omp 18.16 (and pi if installed) — record in `docs/superpowers/specs/omp-compat.md`**

1. `cd plugins/pi/pi-mnemonic-code && npm install` (peers resolve from the omp install when dropped in).
2. Drop the folder into `~/.omp/plugins/pi-mnemonic-code` (per memory spec: plain Pi packages load on omp).
3. In a scratch clone of `test/fixtures/probe-repo`, run omp one-shot: prompt "Use the grep tool to find ParseURL, then call probe_omp and probe_deferred."
4. Read the probe JSONL + the omp session log; answer, verbatim in `omp-compat.md`:
   - [ ] Package loaded on omp? (notify present, tools registered)
   - [ ] `tool_result` fired for `grep`? Did the appended `[PROBE-AUGMENT]` line appear in the agent-visible result (check the session transcript, not just the log)?
   - [ ] `probe_deferred` absent from the initial tool list but returned by `tool_search`?
   - [ ] Plain draft-07 `parameters` object accepted (model could call `probe_omp` with no args)?
   - [ ] `@earendil-works/pi-tui` import resolves in the extension process?
   - [ ] Raw payloads for all five events recorded (trimmed to shape, no secrets).
5. **Door check verdict (STOP gate):** if (a) the package doesn't load, (b) `tool_result` content is not mutable and `tool_execution_end` cannot substitute, or (c) neither plain schema nor TypeBox works — stop, report to the user with the recorded payloads, and await decision (fallback append mechanism or omp-specific package) before Task 2.

- [ ] **Step 4: Run unit test for host duck-typing**

`test/host.test.ts` (full):

```ts
import test from "node:test";
import assert from "node:assert/strict";

test("host event duck-typing", async () => {
  const { detectHost } = await import("../src/host.js");
  process.env.MNEMONIC_CODE_HOST = "omp";
  const h = detectHost();
  assert.equal(h.isUserPrompt({ reason: "prompt" }), true);
  assert.equal(h.isUserPrompt({ reason: "revive" }), false);
  assert.equal(h.isUserPrompt({ messageType: "user" }), true);
  assert.equal(h.isSubagent({ agent: { kind: "subagent" } }), true);
  assert.equal(h.isSubagent({ agent: { kind: "main" } }), false);
  assert.equal(h.isSubagent({}), false);
  assert.equal(h.isRealShutdown({}), true);            // omp: no reason = real
  assert.equal(h.isRealShutdown({ reason: "reload" }), false);
  delete process.env.MNEMONIC_CODE_HOST;
  const pi = detectHost();
  assert.equal(pi.isUserPrompt({ messageType: "user" }), true);
  assert.equal(pi.isUserPrompt({}), false);
  assert.equal(pi.isRealShutdown({ reason: "quit" }), true);
  assert.equal(pi.isRealShutdown({}), false);           // pi: missing reason = not the quit path
});
```

Run: `node --test test/host.test.ts` → PASS.

- [ ] **Step 5: Commit**

```bash
git add plugins/pi/pi-mnemonic-code docs/superpowers/specs/omp-compat.md
git commit -m "test(plugin): omp compatibility spike + package scaffold for mnemonic-code-pi"
```

---

### Task 2: Go Tool Registry + `code call` + `code call --schema`

> ⚠ one-way: CLI verb names `code call` / `code call --schema` and the JSON envelope are a cross-plugin contract shared with the memory plugin (`memory call`); additive now, expensive to rename after both plugins ship. Confirm before executing.

**Files:**
- Create: `internal/code/tools/registry.go`, `internal/code/tools/registry_test.go`
- Create: `internal/code/cli/call.go`, `internal/code/cli/call_test.go`
- Modify: `internal/code/mcp/server.go` (iterate `tools.Registry()`), `internal/code/cli/cli.go` (dispatch `call`; `IO` gains `In io.Reader`), `cmd/mnemonic/main.go` (wiring unchanged if `cli.Run` handles it)
- Modify: `docs/superpowers/plans/2026-11-24-mnemonic-phase2a-core-code-graph.md` (Task 10 interface list), `docs/superpowers/specs/2026-11-24-mnemonic-phase2-code-indexing-design.md` (§6.3), `docs/superpowers/specs/2026-11-24-mnemonic-pi-omp-codeindex-design.md` (amendments 1–3)

**Interfaces:**
- Consumes: Phase 2A `api.Service`, `mcp.NewServer`, `cli.Run`, `apperr`.
- Produces:
  ```go
  // internal/code/tools
  type Tool struct {
      Name        string
      Description string
      Schema      json.RawMessage // draft-07, additionalProperties:false
      ReadOnly    bool
      Defer       bool
      Admin       bool
      Timeout     time.Duration   // default for `code call`; 0 => 5s; code_index => 120s
      Run         func(ctx context.Context, svc api.Service, args json.RawMessage) (any, error)
  }
  func Registry() []Tool            // the 11 Phase 2A tools; later slices append here
  func Lookup(name string) (Tool, bool)
  ```
  ```go
  // cli: mnemonic code call <tool> [--timeout 5s] | mnemonic code call --schema
  // stdin: JSON args object. stdout: {"ok":true,"result":…} | {"ok":false,"error":{"code","message","retryable"}}
  // exit: 0 ok; apperr code otherwise (2 validation … 11 lsp-unavailable). Unknown tool => exit 2.
  ```

- [ ] **Step 1: Apply plan/spec amendments** (edit the three doc files per the Amendments list above; one-line edits, no renumbering).

- [ ] **Step 2: Write failing registry + call tests**

`internal/code/tools/registry_test.go`:

```go
package tools

import (
	"encoding/json"
	"testing"
)

func TestRegistryCoversPhase2ASurfaceExactly(t *testing.T) {
	tools := Registry()
	want := map[string]bool{
		"code_explore": false, "code_read": false, "code_impact": false, "code_affected": false,
		"code_status": false, "code_index": false,
		"code_search": false, "code_neighbors": false, "code_rename": false,
		"code_watch": false, "code_backfill_embeddings": false,
	}
	if len(tools) != len(want) {
		t.Fatalf("registry size = %d, want %d", len(tools), len(want))
	}
	for _, tl := range tools {
		ok, exists := want[tl.Name]
		if !exists || ok {
			t.Fatalf("unexpected or duplicate tool %q", tl.Name)
		}
		want[tl.Name] = true
		if len(tl.Schema) == 0 {
			t.Fatalf("tool %s has empty schema", tl.Name)
		}
		var s map[string]any
		if err := json.Unmarshal(tl.Schema, &s); err != nil {
			t.Fatalf("tool %s schema not JSON: %v", tl.Name, err)
		}
		if s["additionalProperties"] != false {
			t.Fatalf("tool %s schema must set additionalProperties:false", tl.Name)
		}
	}
	if tl, ok := Lookup("code_index"); !ok || tl.Timeout <= 0 {
		t.Fatalf("code_index must declare a long default timeout, got %v", tl)
	}
	if _, ok := Lookup("code_nope"); ok {
		t.Fatal("Lookup must miss for unknown names")
	}
}
```

`internal/code/cli/call_test.go`:

```go
package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestCallSchemaMatchesRegistry(t *testing.T) {
	var out, errb bytes.Buffer
	code := Run(context.Background(), []string{"call", "--schema"}, IO{Out: &out, Err: &errb, In: strings.NewReader("")}, nil)
	if code != 0 {
		t.Fatalf("call --schema exit = %d, err=%s", code, errb.String())
	}
	var env struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("schema output not JSON: %v (%s)", err, out.String())
	}
	if len(env.Tools) != 11 {
		t.Fatalf("schema tool count = %d, want 11", len(env.Tools))
	}
}

func TestCallUnknownToolReturnsEnvelopeExit2(t *testing.T) {
	var out, errb bytes.Buffer
	code := Run(context.Background(), []string{"call", "code_nope"}, IO{Out: &out, Err: &errb, In: strings.NewReader("{}")}, nil)
	if code != 2 {
		t.Fatalf("unknown tool exit = %d, want 2", code)
	}
	if !strings.Contains(out.String(), `"ok":false`) || !strings.Contains(out.String(), `"code"`) {
		t.Fatalf("missing error envelope: %s", out.String())
	}
}

func TestCallCodeStatusHappyPath(t *testing.T) {
	var out, errb bytes.Buffer
	code := Run(context.Background(), []string{"call", "code_status"}, IO{Out: &out, Err: &errb, In: strings.NewReader(`{}`)}, newFakeStatusService("fresh"))
	if code != 0 {
		t.Fatalf("call code_status exit = %d, err=%s", code, errb.String())
	}
	if !strings.Contains(out.String(), `"ok":true`) || !strings.Contains(out.String(), `"freshness":"fresh"`) {
		t.Fatalf("bad envelope: %s", out.String())
	}
}
```

(`newFakeStatusService`/`newFakeWatchService` already exist from Phase 2A `cli_test.go`; add `freshness` to the fake's status DTO if absent.)

- [ ] **Step 3: Run tests to verify they fail**

Run: `CGO_ENABLED=0 go test ./internal/code/tools/... ./internal/code/cli/... -v`
Expected: FAIL (`undefined: Registry`, `undefined: In` field, etc.).

- [ ] **Step 4: Implement `tools/registry.go` by moving the 11 tool definitions out of `mcp/server.go`**

Each Phase 2A MCP tool definition becomes a `Tool` entry: `Schema` = the exact draft-07 JSON the server used; `Run` decodes args into the existing request DTO and calls the same `api.Service` method the MCP path used; `Defer`/`Admin` mirror the `WithDeferLoading` flags; `code_index.Timeout = 120 * time.Second`; read tools `ReadOnly: true`. `mcp/server.go` becomes: iterate `Registry()`, filter `Admin` unless profile is `admin|all`, wrap `Run` in the mcp-go handler, attach `WithDeferLoading(t.Defer)`. Phase 2A's `TestPhase2AMCPToolSurfaceHasZeroStubsAndCorrectDeferFlags` must pass unchanged (9 agent / 11 all).

`internal/code/cli/call.go` (full):

```go
package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"time"

	"mnemonic/internal/code/tools"
	"mnemonic/internal/platform/apperr"
)

type callEnvelope struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *callError      `json:"error,omitempty"`
}

type callError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

func writeCallError(io IO, err error) int {
	code := apperr.Code(err)         // "VALIDATION_ERROR" | "NOT_FOUND" | "INDEX_BUSY" | …
	exit := apperr.ExitCode(err)     // 2..11
	env := callEnvelope{OK: false, Error: &callError{
		Code: code, Message: err.Error(), Retryable: code == "INDEX_BUSY",
	}}
	b, _ := json.Marshal(env)
	fmt.Fprintln(io.Out, string(b))
	if exit == 0 {
		exit = 1
	}
	return exit
}

func runCall(ctx context.Context, args []string, io IO, svc api.Service) int {
	if len(args) > 0 && args[0] == "--schema" {
		type schemaOut struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			Schema      json.RawMessage `json:"schema"`
			ReadOnly    bool            `json:"readonly"`
			Defer       bool            `json:"defer"`
			Admin       bool            `json:"admin"`
		}
		out := struct {
			Tools []schemaOut `json:"tools"`
		}{}
		for _, t := range tools.Registry() {
			out.Tools = append(out.Tools, schemaOut{t.Name, t.Description, t.Schema, t.ReadOnly, t.Defer, t.Admin})
		}
		b, _ := json.Marshal(out)
		fmt.Fprintln(io.Out, string(b))
		return 0
	}
	fs := flag.NewFlagSet("call", flag.ContinueOnError)
	fs.SetOutput(io.Err)
	fs.Parse(args[1:])
	if fs.NArg() != 1 {
		fmt.Fprintln(io.Err, "usage: mnemonic code call <tool> [--timeout 5s] | mnemonic code call --schema")
		return 2
	}
	t, ok := tools.Lookup(fs.Arg(0))
	if !ok {
		return writeCallError(io, apperr.New(apperr.ErrValidation, fmt.Sprintf("unknown code tool %q", fs.Arg(0))))
	}
	timeout := t.Timeout
	if d := fs.Duration("timeout"); *d > 0 {
		timeout = *d
	} else if timeout <= 0 {
		timeout = 5 * time.Second
	}
	raw, err := io.ReadAll(io.In)
	if err != nil {
		return writeCallError(io, apperr.New(apperr.ErrValidation, "reading args: "+err.Error()))
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	res, err := t.Run(cctx, svc, raw)
	if err != nil {
		return writeCallError(io, err)
	}
	b, merr := json.Marshal(res)
	if merr != nil {
		return writeCallError(io, apperr.New(apperr.ErrValidation, "encoding result: "+merr.Error()))
	}
	fmt.Fprintf(io.Out, `{"ok":true,"result":%s}`+"\n", b)
	return 0
}
```

(`flag.Duration` needs `--timeout` declared: add `fs.Duration("timeout", 0, "")` before `Parse`.) Dispatch `call` in `cli.Run` alongside the Phase 2A verbs; extend `cli.IO` with `In io.Reader` (existing call sites pass `strings.NewReader("")`).

- [ ] **Step 5: Run full build + race suite**

Run: `CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go test -race ./internal/code/... -v`
Expected: PASS, including Phase 2A's MCP surface test unchanged.

- [ ] **Step 6: Commit**

```bash
git add internal/code cmd/mnemonic docs/superpowers
git commit -m "feat(code): add shared tool registry and mnemonic code call/--schema verbs"
```

---

### Task 3: Go `code augment` Verb (Budgeted Context Block)

**Files:**
- Create: `internal/code/engine/augment.go`, `internal/code/engine/augment_test.go`
- Create: `internal/code/store/graph_augment.go` (+ tests appended to `store_test.go`)
- Modify: `internal/code/api/service.go` (`Augment` on `Service`), `internal/code/cli/call.go` or new `cli/augment.go` (verb dispatch)

**Interfaces:**
- Consumes: `store.Meta`, `store.Neighbors` (Phase 2A `graph_io.go`), `store.ObservationCodeLinks` (Phase 2A `search_io.go`), CSR-free (pure SQL — augment must stay <50 ms and must not take the index writer lock).
- Produces:
  ```go
  // api.Service addition
  Augment(ctx context.Context, pattern string, maxBytes int) (string, error)
  // sentinel errors (exported from engine): ErrAugmentNoIndex, ErrAugmentNoMatch
  // CLI: mnemonic code augment <pattern> [--max-bytes 1536]
  //   stdout: context block; exit 0. exit 3: no index or no match (no stdout). apperr code otherwise.
  ```
  Block format (deterministic; the TS hook appends it verbatim):
  ```
  callers (2/5):
    HandleUpload   internal/handler/upload.go:88   (EXTRACTED)
    HandleBackup   internal/handler/backup.go:41   (INFERRED)
  callees (3):
    parseQuery     internal/net/url.go:210         (EXTRACTED)
    normalizeHost  internal/net/url.go:240         (EXTRACTED)
    validatePort   internal/net/url.go:301         (SYNTHESIZED)
  tests: TestParseURL (internal/net/url_test.go:12)
  memories: [decision] upload auth via signed URLs (uid a1b2c3)
  weakest hop: INFERRED
  ```

- [ ] **Step 1: Write failing store tests** (`store/graph_augment.go` queries)

Append to `internal/code/store/store_test.go`:

```go
func TestCallersCalleesTestRefsRows(t *testing.T) {
	d := openTestRepoDB(t)
	ctx := context.Background()
	// Seed 4 symbols across 2 files (one a _test.go):
	//   url.go:      ParseURL (exported), parseQuery (internal)
	//   upload.go:   HandleUpload (calls ParseURL, calls parseQuery)
	//   url_test.go: TestParseURL (calls ParseURL)
	// (reuse Phase 2A's FileBundle/UpsertFileBundle + edge insert helpers)
	callers, err := Callers(ctx, d, symID(t, "u-parseurl"), 5)
	if err != nil || len(callers) != 2 {
		t.Fatalf("Callers = %v err=%v, want 2", callers, err)
	}
	if callers[0].Name != "HandleUpload" || callers[0].FilePath != "internal/handler/upload.go" {
		t.Fatalf("top caller wrong: %+v", callers[0])
	}
	callees, err := Callees(ctx, d, symID(t, "u-parseurl"), 5)
	if err != nil || len(callees) != 0 { // ParseURL calls nothing in fixture
		t.Fatalf("Callees = %v err=%v, want 0", callees, err)
	}
	tests, err := TestRefs(ctx, d, symID(t, "u-parseurl"), 3)
	if err != nil || len(tests) != 1 || tests[0].Name != "TestParseURL" {
		t.Fatalf("TestRefs = %v err=%v", tests, err)
	}
}
```

- [ ] **Step 2: Run to verify fail**

Run: `CGO_ENABLED=0 go test ./internal/code/store/... -run TestCallersCallees -v`
Expected: FAIL (`undefined: Callers`).

- [ ] **Step 3: Implement store queries**

`internal/code/store/graph_augment.go`:

```go
package store

import (
	"context"
	"database/sql"

	"mnemonic/internal/platform/sqlitepool"
)

type NeighborRow struct {
	Name       string
	FilePath   string
	Line       int
	Confidence string
	PageRank   float64
}

func queryNeighbors(ctx context.Context, d *sqlitepool.DB, symID int64, direction, kinds string, limit int) ([]NeighborRow, error) {
	// direction "in": WHERE e.to_symbol_id = ? ; "out": WHERE e.from_symbol_id = ?
	q := `SELECT s.name, f.path, IFNULL(e.line, 0), e.confidence, s.pagerank
	      FROM code_edges e
	      JOIN code_symbols s ON s.id = e.from_symbol_id
	      JOIN code_files f ON f.id = e.file_id
	      WHERE e.to_symbol_id = ? AND e.kind IN (kinds)
	      ORDER BY s.pagerank DESC, s.id LIMIT ?`
	// (swap join/where for direction == "out": s.id = e.to_symbol_id)
	rows, err := d.R.QueryContext(ctx, q, symID, limit)
	// scan into []NeighborRow
}

func Callers(ctx context.Context, d *sqlitepool.DB, symID int64, limit int) ([]NeighborRow, error) {
	return queryNeighbors(ctx, d, symID, "in", "'calls','references'", limit)
}
func Callees(ctx context.Context, d *sqlitepool.DB, symID int64, limit int) ([]NeighborRow, error) {
	return queryNeighbors(ctx, d, symID, "out", "'calls','imports','instantiates'", limit)
}
func TestRefs(ctx context.Context, d *sqlitepool.DB, symID int64, limit int) ([]NeighborRow, error) {
	// callers restricted to code_files.is_test = 1
}
```

- [ ] **Step 4: Write failing engine tests**

`internal/code/engine/augment_test.go`:

```go
package engine

import (
	"context"
	"strings"
	"testing"
)

func TestAugmentBudgetAndContent(t *testing.T) {
	e := newTestLocalWithFixture(t) // Phase 2A helper: index the polyglot fixture repo
	ctx := context.Background()

	out, err := e.Augment(ctx, "ParseURL", 1536)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) > 1536 {
		t.Fatalf("block %d bytes > 1536", len(out))
	}
	for _, want := range []string{"callers", "HandleUpload", "internal/handler/upload.go:88", "weakest hop"} {
		if !strings.Contains(out, want) {
			t.Fatalf("block missing %q:\n%s", want, out)
		}
	}
	// Every emitted file:line must exist in the fixture (no phantom references).
	for _, line := range strings.Split(out, "\n") {
		if i := strings.Index(line, ".go:"); i > 0 {
			path := line[:i+3]
			if _, err := os.Stat(filepath.Join(fixtureRoot(t), path[:i-2])); err != nil {
				t.Fatalf("phantom path in block: %s", path)
			}
		}
	}

	if _, err := e.Augment(ctx, "NoSuchSymbolXYZ", 1536); err != ErrAugmentNoMatch {
		t.Fatalf("unknown pattern err = %v, want ErrAugmentNoMatch", err)
	}
	empty := newTestLocalEmpty(t)
	if _, err := empty.Augment(ctx, "ParseURL", 1536); err != ErrAugmentNoIndex {
		t.Fatalf("empty index err = %v, want ErrAugmentNoIndex", err)
	}
}
```

- [ ] **Step 5: Implement `engine/augment.go`**

```go
package engine

var (
	ErrAugmentNoIndex = errors.New("code index empty")
	ErrAugmentNoMatch = errors.New("no matching symbol")
)

const augmentDefaultMaxBytes = 1536

func (l *Local) Augment(ctx context.Context, pattern string, maxBytes int) (string, error) {
	if maxBytes <= 0 {
		maxBytes = augmentDefaultMaxBytes
	}
	meta, err := l.store.Meta(ctx)
	if err != nil {
		return "", err
	}
	if meta.FilesCount == 0 {
		return "", ErrAugmentNoIndex
	}
	syms, err := l.resolveAugmentSymbols(ctx, pattern) // exact name first (top by pagerank, ≤2), then FTS/segment top-3, dedup, total ≤2
	if err != nil {
		return "", err
	}
	if len(syms) == 0 {
		return "", ErrAugmentNoMatch
	}
	var b strings.Builder
	weakest := rankMax // LSP_RESOLVED > EXTRACTED > SYNTHESIZED > INFERRED > AMBIGUOUS
	for i, s := range syms {
		if len(syms) > 1 {
			fmt.Fprintf(&b, "%s:\n", s.Name)
		}
		callers, err := l.store.Callers(ctx, s.ID, 5)
		if err != nil {
			return "", err
		}
		if len(callers) > 0 {
			fmt.Fprintf(&b, "callers (%d/5):\n", len(callers))
			for _, c := range callers {
				fmt.Fprintf(&b, "  %-20s %s:%d  (%s)\n", c.Name, c.FilePath, c.Line, c.Confidence)
				weakest = minRank(weakest, rankOf(c.Confidence))
			}
		}
		callees, err := l.store.Callees(ctx, s.ID, 5)
		// same rendering under "callees (n):"
		tests, err := l.store.TestRefs(ctx, s.ID, 3)
		if len(tests) > 0 {
			// "tests: Name (path:line)" comma-joined, ≤3
		}
		links, err := l.store.ObservationCodeLinks(ctx, s.UID, s.FilePath, 3)
		if len(links) > 0 {
			// "memories: [type] title (uid)" comma-joined, ≤3
		}
		if i < len(syms)-1 {
			b.WriteString("\n")
		}
	}
	b.WriteString("weakest hop: " + rankName(weakest))
	out := b.String()
	if len(out) > maxBytes {
		out = truncateAtLineBoundary(out, maxBytes) // drop callees lines first, then hard-cut at '\n'
	}
	return out, nil
}
```

`resolveAugmentSymbols`: `SELECT id, name, uid, file_id FROM code_symbols WHERE name = ? AND kind NOT IN ('file','section') ORDER BY pagerank DESC LIMIT 2`; if 0 rows, reuse the Phase 2A search lexical legs (FTS bm25 + `code_name_segments`) top-3, dedup by uid, cap 2. (Both queries are read-only; no CSR load.)

- [ ] **Step 6: Wire the CLI verb**

In `cli.Run` dispatch: `augment` → parse `--max-bytes` (default 1536), call `svc.Augment`, map `ErrAugmentNoIndex`/`ErrAugmentNoMatch` → return 3 with empty stdout, other errors → `writeCallError` style envelope on stdout + apperr exit, success → `fmt.Fprintln(io.Out, out)`.

Add a `call_test.go`-style test: `TestAugmentVerbExitCodes` (fixture service: match → 0 + block; no match → 3; corrupt meta → apperr code).

- [ ] **Step 7: Run full build + race suite**

Run: `CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go test -race ./internal/code/... -v`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/code
git commit -m "feat(code): add mnemonic code augment verb with budgeted graph context"
```

---

### Task 4: Plugin Core — `transport.ts`, `controller.ts`

**Files:**
- Create: `plugins/pi/pi-mnemonic-code/src/transport.ts`
- Create: `plugins/pi/pi-mnemonic-code/src/controller.ts`
- Create: `plugins/pi/pi-mnemonic-code/test/transport.test.ts`, `test/controller.test.ts`
- Create: `test/fixtures/stub-mnemonic/` (stub binary scripts: `ok`, `err`, `hang`, `garbage`, `schema-2a`, `schema-2d`, `augment-ok`, `augment-empty`)

**Interfaces:**
- Produces (consumed by Tasks 5–8):
  ```ts
  // transport.ts
  export interface CodeError { code: string; message: string; retryable: boolean }
  export interface CodeResult<T = unknown> { ok: boolean; result?: T; error?: CodeError }
  export interface ToolSchema { name: string; description: string; schema: object; readonly: boolean; defer: boolean; admin: boolean }
  export interface Transport {
    call<T = unknown>(tool: string, args: Record<string, unknown>, opts?: { timeoutMs?: number; signal?: AbortSignal }): Promise<CodeResult<T>>;
    augment(pattern: string, opts?: { timeoutMs?: number; signal?: AbortSignal }): Promise<string | null>;
    schemas(): Promise<ToolSchema[]>;
    binaryPath(): string | null;
  }
  export function spawnTransport(bin: string | null): Transport
  // controller.ts
  export interface IndexState {
    freshness: string; files: number; symbols: number; unresolved: number;
    watch: boolean; capabilities: Record<string, boolean>;
  }
  export class Controller {
    constructor(opts: { transport: Transport; host: Host; env: Record<string, string | undefined>; now?: () => number })
    indexState: IndexState | null
    augmentOn: boolean
    readonly offline: boolean; readonly ready: boolean; readonly stale: boolean; readonly indexing: boolean
    async onSessionStart(cwd: string): Promise<void>
    onSessionEnd(): void
    shouldAugment(): boolean
    async refreshIndexState(cwd: string): Promise<void>
    async augmentPatterns(cwd: string, patterns: string[]): Promise<string | null>
    markIndexing(run: boolean): void
    noteError(msg: string): void
    noteDrop(): void
    doctor(): Record<string, unknown>
  }
  ```

- [ ] **Step 1: Write stub binaries** (each a `#!/usr/bin/env node` script in `test/fixtures/stub-mnemonic/`, `chmod +x`)

```js
// ok.js — echo a canned envelope for code_status
const tool = process.argv[2]; // "code" argv[1], actual: argv = [node, script, "code", "call", tool, ...]
process.stdin.resume().on("data", () => {});
console.log('{"ok":true,"result":{"freshness":"fresh","files_indexed":120,"symbols_count":840,"unresolved":3,"watch_active":true,"capabilities":{"core":true,"vectors":true}}}');
```
Variants: `err.js` (exit 5, `{"ok":false,"error":{"code":"NOT_FOUND","message":"no index","retryable":false}}`), `hang.js` (sleep 60 s), `garbage.js` (print `not json` exit 0), `schema-2a.js` (11-entry `{"tools":[…]}` dump with minimal valid draft-07 schemas, `code_index` flagged), `schema-2d.js` (16 entries: adds `code_topology`, `code_contracts`, `code_pdg`, `code_taint`, `code_lsp_sync`), `augment-ok.js` (prints a 2-line callers block for any pattern), `augment-empty.js` (exit 3, no stdout). A `stub-mnemonic` wrapper script dispatches on `argv[2..]` (`code call <tool>` / `code augment <p>` / `code call --schema`) via an env var `STUB_MODE=ok|err|hang|garbage|schema-2a|schema-2d|augment-ok|augment-empty`.

- [ ] **Step 2: Write failing transport tests**

`test/transport.test.ts` (full core; the suite covers every Threat Matrix row for envelope/subprocess):

```ts
import test from "node:test";
import assert from "node:assert/strict";
import { spawnTransport, type Transport } from "../src/transport.js";

const STUB = new URL("./fixtures/stub-mnemonic/stub-mnemonic", import.meta.url).pathname;
const mk = (mode: string): Transport => spawnTransport(STUB);
process.env.STUB_MODE = "ok";

test("call parses ok envelope", async () => {
  process.env.STUB_MODE = "ok";
  const r = await mk("ok").call("code_status", {});
  assert.equal(r.ok, true);
  assert.equal((r.result as any).freshness, "fresh");
});

test("call maps error envelope + exit code", async () => {
  process.env.STUB_MODE = "err";
  const r = await mk("err").call("code_status", {});
  assert.equal(r.ok, false);
  assert.equal(r.error?.code, "NOT_FOUND");
});

test("malformed stdout -> BAD_ENVELOPE, no throw", async () => {
  process.env.STUB_MODE = "garbage";
  const r = await mk("garbage").call("code_status", {});
  assert.equal(r.ok, false);
  assert.equal(r.error?.code, "BAD_ENVELOPE");
});

test("timeout kills child -> OFFLINE/timeout error", async () => {
  process.env.STUB_MODE = "hang";
  const r = await mk("hang").call("code_status", {}, { timeoutMs: 200 });
  assert.equal(r.ok, false);
  assert.match(r.error!.code, /TIMEOUT|OFFLINE/);
});

test("missing binary -> OFFLINE without spawn", async () => {
  const r = await spawnTransport(null).call("code_status", {});
  assert.equal(r.ok, false);
  assert.equal(r.error?.code, "OFFLINE");
});

test("schemas caches and parses both dumps", async () => {
  process.env.STUB_MODE = "schema-2a";
  const t = mk("schema-2a");
  const a = await t.schemas();
  assert.equal(a.length, 11);
  assert.equal((await t.schemas()).length, 11); // cached
  process.env.STUB_MODE = "schema-2d";
  assert.equal((await spawnTransport(STUB).schemas()).length, 16);
});

test("augment returns stdout or null on exit 3", async () => {
  process.env.STUB_MODE = "augment-ok";
  assert.ok(await mk("augment-ok").augment("ParseURL"));
  process.env.STUB_MODE = "augment-empty";
  assert.equal(await mk("augment-empty").augment("ParseURL"), null);
});

test("concurrency bounded to 4", async () => {
  process.env.STUB_MODE = "hang";
  const t = mk("hang");
  const start = Date.now();
  await Promise.all(Array.from({ length: 8 }, () => t.call("code_status", {}, { timeoutMs: 1500 })));
  assert.ok(Date.now() - start >= 500, "8 calls at 4-wide should take >= 2 batches");
});
```

- [ ] **Step 3: Run to verify fail, then implement `src/transport.ts`**

Full implementation (the plan text is the implementation spec — port verbatim, adjusting only to make Step 2 pass):

```ts
import { spawn } from "node:child_process";

export interface CodeError { code: string; message: string; retryable: boolean }
export interface CodeResult<T = unknown> { ok: boolean; result?: T; error?: CodeError }
export interface ToolSchema { name: string; description: string; schema: object; readonly: boolean; defer: boolean; admin: boolean }
export interface CallOpts { timeoutMs?: number; signal?: AbortSignal }
export interface Transport {
  call<T = unknown>(tool: string, args: Record<string, unknown>, opts?: CallOpts): Promise<CodeResult<T>>;
  augment(pattern: string, opts?: CallOpts): Promise<string | null>;
  schemas(): Promise<ToolSchema[]>;
  binaryPath(): string | null;
}

const MAX_INFLIGHT = 4;
const OFFLINE: CodeResult = { ok: false, error: { code: "OFFLINE", message: "mnemonic binary not found", retryable: false } };
const BAD: CodeError = { code: "BAD_ENVELOPE", message: "malformed mnemonic envelope", retryable: false };

export function spawnTransport(bin: string | null): Transport {
  let inflight = 0;
  const waiters: Array<() => void> = [];
  let schemaCache: ToolSchema[] | null = null;

  const acquire = (): Promise<void> => new Promise((res) => {
    if (inflight < MAX_INFLIGHT) { inflight++; res(); } else { waiters.push(() => { inflight++; res(); }); }
  });
  const release = (): void => { inflight--; waiters.shift()?.(); };

  const run = (argv: string[], stdin: string, timeoutMs: number, signal?: AbortSignal) =>
    new Promise<{ code: number; stdout: string; stderr: string }>((resolve) => {
      if (!bin) { resolve({ code: -1, stdout: "", stderr: "mnemonic binary not found" }); return; }
      const child = spawn(bin, argv, { stdio: ["pipe", "pipe", "pipe"] }); // shell: false by default
      let stdout = "";
      let stderr = "";
      let settled = false;
      const finish = (code: number) => { if (!settled) { settled = true; clearTimeout(timer); resolve({ code, stdout, stderr }); } };
      const timer = setTimeout(() => { child.kill("SIGKILL"); finish(-2); }, timeoutMs);
      signal?.addEventListener("abort", () => child.kill("SIGTERM"), { once: true });
      child.stdout.on("data", (d) => { stdout += d.toString(); });
      child.stderr.on("data", (d) => { stderr += d.toString(); });
      child.on("error", (e) => finish(-1));
      child.on("close", (code) => finish(code ?? -1));
      child.stdin.end(stdin);
    });

  const dur = (ms: number): string => (ms < 1000 ? `${ms}ms` : `${Math.round(ms / 1000)}s`);

  return {
    binaryPath: () => bin,
    async call<T>(tool: string, args: Record<string, unknown>, opts?: CallOpts): Promise<CodeResult<T>> {
      await acquire();
      try {
        const timeoutMs = opts?.timeoutMs ?? 5000;
        const r = await run(["code", "call", tool, "--timeout", dur(timeoutMs)], JSON.stringify(args ?? {}), timeoutMs + 500, opts?.signal);
        const env = parseEnvelope<T>(r.stdout);
        if (env) return env;
        if (r.code === -1) return { ok: false, error: { code: "OFFLINE", message: r.stderr, retryable: false } };
        if (r.code === -2) return { ok: false, error: { code: "TIMEOUT", message: `mnemonic code call ${tool} exceeded ${timeoutMs}ms`, retryable: true } };
        return { ok: false, error: { code: `EXIT_${r.code}`, message: r.stderr.trim().slice(0, 500) || `mnemonic code call ${tool} exited ${r.code}`, retryable: r.code === 10 } };
      } finally {
        release();
      }
    },
    async augment(pattern: string, opts?: CallOpts): Promise<string | null> {
      if (!bin) return null;
      await acquire();
      try {
        const r = await run(["code", "augment", pattern, "--max-bytes", "1536"], "", opts?.timeoutMs ?? 5000, opts?.signal);
        if (r.code !== 0) return null;
        const out = r.stdout.trim();
        return out ? out : null;
      } finally {
        release();
      }
    },
    async schemas(): Promise<ToolSchema[]> {
      if (schemaCache) return schemaCache;
      if (!bin) return [];
      const r = await run(["code", "call", "--schema"], "", 5000);
      if (r.code !== 0) return [];
      try {
        const j = JSON.parse(r.stdout) as { tools?: ToolSchema[] };
        schemaCache = Array.isArray(j.tools) ? j.tools : [];
      } catch {
        schemaCache = [];
      }
      return schemaCache;
    },
  };
}

function parseEnvelope<T>(stdout: string): CodeResult<T> | null {
  const s = stdout.trim();
  if (!s.startsWith("{")) return null;
  try {
    const j = JSON.parse(s) as { ok?: unknown; result?: T; error?: CodeError };
    if (j && typeof j === "object" && typeof j.ok === "boolean") {
      if (j.ok) return { ok: true, result: j.result };
      if (j.error) return { ok: false, error: j.error };
    }
    return { ok: false, error: { ...BAD } };
  } catch {
    return { ok: false, error: { ...BAD } };
  }
}
```

Run: `node --test test/transport.test.ts` → PASS.

- [ ] **Step 4: Write failing controller tests**

`test/controller.test.ts` (core scenarios; full file in the implementation):

```ts
import test from "node:test";
import assert from "node:assert/strict";
import { Controller } from "../src/controller.js";
import { spawnTransport } from "../src/transport.js";
import { detectHost } from "../src/host.js";

const host = detectHost();
const mk = (mode: string) => new Controller({
  transport: spawnTransport(new URL("./fixtures/stub-mnemonic/stub-mnemonic", import.meta.url).pathname),
  host, env: process.env as Record<string, string | undefined>,
});

test("session start caches index state; ready on fresh", async () => {
  process.env.STUB_MODE = "ok";
  const c = mk("ok");
  await c.onSessionStart("/tmp");
  assert.equal(c.ready, true);
  assert.equal(c.indexState?.files, 120);
  assert.equal(c.offline, false);
});

test("augmentPatterns dedups per session and caches empties", async () => {
  process.env.STUB_MODE = "augment-ok";
  const c = mk("augment-ok");
  await c.onSessionStart("/tmp");
  const first = await c.augmentPatterns("/tmp", ["ParseURL"]);
  assert.ok(first && first.includes("callers"));
  assert.equal(await c.augmentPatterns("/tmp", ["ParseURL"]), null); // deduped
  process.env.STUB_MODE = "augment-empty";
  assert.equal(await c.augmentPatterns("/tmp", ["parseQuery"]), null);
  assert.equal(await c.augmentPatterns("/tmp", ["parseQuery"]), null); // empty-cached
});

test("stale notice fires once", async () => {
  process.env.STUB_MODE = "ok-stale"; // variant: freshness "stale"
  const c = mk("ok-stale");
  await c.onSessionStart("/tmp");
  process.env.STUB_MODE = "augment-ok";
  const a = await c.augmentPatterns("/tmp", ["ParseURL"]);
  assert.ok(a!.startsWith("index stale"));
  const b = await c.augmentPatterns("/tmp", ["parseQuery"]);
  assert.ok(!b!.startsWith("index stale"));
});

test("offline: shouldAugment false, no throws", async () => {
  const c = new Controller({ transport: spawnTransport(null), host, env: {} });
  await c.onSessionStart("/tmp");
  assert.equal(c.offline, true);
  assert.equal(c.shouldAugment(), false);
  assert.equal(await c.augmentPatterns("/tmp", ["ParseURL"]), null);
});

test("paused during code_index run", async () => {
  process.env.STUB_MODE = "ok";
  const c = mk("ok");
  await c.onSessionStart("/tmp");
  c.markIndexing(true);
  assert.equal(c.shouldAugment(), false);
  c.markIndexing(false);
  assert.equal(c.shouldAugment(), true);
});
```

- [ ] **Step 5: Implement `src/controller.ts`**

State fields (no module globals): `indexState: IndexState | null`, `augmentOn = true`, private `augmented = new Set<string>()`, `empty = new Set<string>()`, `indexing = false`, `staleNoticeSent = false`, `dropCount = 0`, `errors: string[]` (ring, cap 5), `sessionCwd = ""`.

Methods:
- `onSessionStart(cwd)`: clear both dedup sets, `staleNoticeSent = false`, `indexing = false`, `sessionCwd = cwd`, `await refreshIndexState(cwd)`.
- `refreshIndexState(cwd)`: `call("code_status", {}, {timeoutMs: 5000})`; on `!r.ok` → `indexState = null`, `noteError(r.error.message)`; on ok map fields defensively (`freshness`, `files_indexed ?? files ?? 0`, `symbols_count ?? symbols ?? 0`, `unresolved ?? 0`, `watch_active ?? false`, `capabilities ?? {}`).
- `shouldAugment()`: `augmentOn && !offline && ready && !indexing` where `ready = freshness === "fresh" || "stale"`.
- `augmentPatterns(cwd, patterns)`: filter (`trim`, len ≥3, not in `augmented`/`empty` lowercase), `slice(0, 3)`; `Promise.all` over `transport.augment(p)`; update caches by result; join non-null blocks with `"\n"`; prepend `index stale — run /codeindex index for a full refresh\n` once when `stale && !staleNoticeSent`; return null when nothing. Combined-block cap enforced by the hook (Task 7), not here.
- `markIndexing(run)`, `onSessionEnd()` (`indexing = false`), `noteError` (push + shift past 5), `noteDrop`, `doctor()` → `{ binary: transport.binaryPath(), indexState, augmentOn, drops: dropCount, errors, deduped: augmented.size, empty: empty.size }`.

Run: `node --test test/controller.test.ts` → PASS.

- [ ] **Step 6: Commit**

```bash
git add plugins/pi/pi-mnemonic-code
git commit -m "feat(plugin): add spawn transport and fail-open controller for mnemonic-code-pi"
```

---

### Task 5: Capability-Gated Tool Registration (`schemas.ts`, `tools.ts`)

**Files:**
- Create: `plugins/pi/pi-mnemonic-code/src/schemas.ts`, `src/tools.ts`, `src/instructions.ts`
- Create: `plugins/pi/pi-mnemonic-code/test/tools.test.ts`
- Reuse: `test/fixtures/stub-mnemonic/stub-mnemonic` with `schema-2a`/`schema-2d` modes (Task 4)

**Interfaces:**
- Consumes: `Transport.schemas()`, `Transport.call()`, `Controller` (for `noteError` + `markIndexing`), host (spike verdict from Task 1: plain draft-07 vs TypeBox requirement).
- Produces:
  ```ts
  // schemas.ts
  export function draft07ToTypeBox(s: object): unknown   // returns TypeBox TObject when typebox loads, else the raw object
  export const TOOL_META: Record<string, { timeoutMs: number; exposure: "direct" | "deferred"; descriptionSuffix?: string }>
  // tools.ts
  export async function registerTools(pi: any, c: Controller): Promise<number> // returns count registered
  ```

- [ ] **Step 1: Write failing registration tests**

`test/tools.test.ts` (core; full file in implementation):

```ts
import test from "node:test";
import assert from "node:assert/strict";
import { registerTools, TOOL_META } from "../src/tools.js";
import { Controller } from "../src/controller.js";
import { spawnTransport } from "../src/transport.js";
import { detectHost } from "../src/host.js";

const STUB = new URL("./fixtures/stub-mnemonic/stub-mnemonic", import.meta.url).pathname;
const mkPi = () => {
  const tools: any[] = [];
  return { pi: { registerTool: (t: any) => tools.push(t) }, tools };
};

test("2A schema dump registers exactly 11 tools with correct exposures", async () => {
  process.env.STUB_MODE = "schema-2a";
  const c = new Controller({ transport: spawnTransport(STUB), host: detectHost(), env: process.env as any });
  const { pi, tools } = mkPi();
  const n = await registerTools(pi, c);
  assert.equal(n, 11);
  const direct = tools.filter(t => t.exposure === "direct").map(t => t.name).sort();
  assert.deepEqual(direct, ["code_affected", "code_explore", "code_index", "code_impact", "code_read", "code_status"]);
  const deferred = tools.filter(t => t.exposure === "deferred").map(t => t.name).sort();
  assert.deepEqual(deferred, ["code_backfill_embeddings", "code_neighbors", "code_rename", "code_search", "code_watch"]);
  const explore = tools.find(t => t.name === "code_explore");
  assert.ok(explore.parameters, "must carry a usable parameter schema");
});

test("2D dump registers 16 tools; unknown future tool passes through as deferred", async () => {
  process.env.STUB_MODE = "schema-2d";
  const c = new Controller({ transport: spawnTransport(STUB), host: detectHost(), env: process.env as any });
  const { pi, tools } = mkPi();
  assert.equal(await registerTools(pi, c), 16);
  assert.ok(tools.some(t => t.name === "code_topology"));
  // schema-2d stub includes a 17th entry "code_future" not in TOOL_META:
  assert.equal(await registerTools(mkPi().pi, new Controller({ transport: spawnTransport(STUB), host: detectHost(), env: process.env as any })), 17);
  const fut = tools.find(t => t.name === "code_future");
  assert.equal(fut?.exposure, "deferred");
});

test("no schema (offline) registers zero tools, resolves cleanly", async () => {
  const c = new Controller({ transport: spawnTransport(null), host: detectHost(), env: {} });
  const { pi, tools } = mkPi();
  assert.equal(await registerTools(pi, c), 0);
  assert.equal(tools.length, 0);
});

test("code_index tool uses 120s timeout and markIndexing around the call", async () => {
  process.env.STUB_MODE = "ok";
  const c = new Controller({ transport: spawnTransport(STUB), host: detectHost(), env: process.env as any });
  const { pi, tools } = mkPi();
  await registerTools(pi, c);
  const idx = tools.find(t => t.name === "code_index");
  assert.equal(TOOL_META.code_index.timeoutMs, 120000);
  const res = await idx.execute("id1", { full: true }, new AbortController().signal, () => {}, { cwd: "/tmp" });
  assert.ok(res.content[0].text.includes('"ok":true'));
  assert.equal(c.indexing, false); // released after
});
```

- [ ] **Step 2: Run to verify fail, then implement**

`src/schemas.ts`:

```ts
import { createRequire } from "node:module";
const require = createRequire(import.meta.url);
let TB: any = null;
try { TB = require("typebox").Type; } catch { TB = null; } // optional at runtime; plain objects otherwise

/** Convert a draft-07 schema object to a TypeBox object when available (spike Task 1 pins whether this is required). */
export function draft07ToTypeBox(s: object): unknown {
  if (!TB) return s;
  if (s.type === "object") {
    const props: Record<string, unknown> = {};
    for (const [k, v] of Object.entries((s.properties ?? {}) as Record<string, object>)) {
      props[k] = convert(v);
    }
    return TB.Object(props, { additionalProperties: false });
  }
  return convert(s);
  function convert(v: object): unknown {
    if (typeof v !== "object" || v === null) return v;
    const t = v as Record<string, any>;
    switch (t.type) {
      case "string": return t.minLength ? TB.String({ minLength: t.minLength, maxLength: t.maxLength }) : TB.String();
      case "integer": case "number":
        return t.type === "integer"
          ? TB.Integer({ minimum: t.minimum, maximum: t.maximum })
          : TB.Number({ minimum: t.minimum, maximum: t.maximum });
      case "boolean": return TB.Boolean();
      case "array": return TB.Array(convert(t.items ?? { type: "string" }));
      case "object": return draft07ToTypeBox(v);
      default:
        if (t.enum) return TB.Union(t.enum.map((e: string) => TB.Literal(e)));
        return TB.Any();
    }
  }
}
```

`src/tools.ts` — per-tool metadata (server owns schemas/descriptions; the plugin adds timeouts, exposure, guidance suffix, and the `project` passthrough when the server schema already declares it):

```ts
export const TOOL_META: Record<string, { timeoutMs: number; exposure: "direct" | "deferred"; descriptionSuffix?: string }> = {
  code_explore:  { timeoutMs: 15000,  exposure: "direct", descriptionSuffix: " Call FIRST before reading files in unknown areas; returns budgeted source + call trails + linked memories." },
  code_read:     { timeoutMs: 5000,   exposure: "direct", descriptionSuffix: " Use after code_search/code_explore narrows the location." },
  code_impact:   { timeoutMs: 5000,   exposure: "direct", descriptionSuffix: " Run before editing a symbol: will_break (hop 1) vs likely_affected." },
  code_affected: { timeoutMs: 10000,  exposure: "direct", descriptionSuffix: " Map git changes (working/staged/base_ref) to impacted callers, tests and flows — the PR-review entry point." },
  code_status:   { timeoutMs: 5000,   exposure: "direct", descriptionSuffix: " Check freshness before searching; stale → run code_index." },
  code_index:    { timeoutMs: 120000, exposure: "direct", descriptionSuffix: " Incremental or full sync; user-initiated only (never auto-run)." },
  code_search:   { timeoutMs: 5000,   exposure: "deferred" },
  code_neighbors:{ timeoutMs: 5000,   exposure: "deferred" },
  code_rename:   { timeoutMs: 10000,  exposure: "deferred", descriptionSuffix: " TWO-STEP: dry_run (default) → review diff → apply with returned plan_token." },
  code_topology: { timeoutMs: 10000,  exposure: "deferred", descriptionSuffix: " Architecture questions start here (overview/communities/god_nodes/flows/cycles/path)." },
  code_contracts:{ timeoutMs: 5000,   exposure: "deferred" },
  code_pdg:      { timeoutMs: 5000,   exposure: "deferred" },
  code_taint:    { timeoutMs: 10000,  exposure: "deferred", descriptionSuffix: " Source→sink traces — security and injection bugs." },
  code_watch:            { timeoutMs: 5000, exposure: "deferred" },
  code_backfill_embeddings: { timeoutMs: 30000, exposure: "deferred" },
  code_lsp_sync: { timeoutMs: 120000, exposure: "deferred" },
};
// Admin tools (code_watch, code_backfill_embeddings, code_lsp_sync) register with defaultActive: false when the host supports it.
const ADMIN = new Set(["code_watch", "code_backfill_embeddings", "code_lsp_sync"]);

export async function registerTools(pi: any, c: Controller): Promise<number> {
  const schemas = await c.transport.schemas();
  let n = 0;
  for (const s of schemas) {
    const meta = TOOL_META[s.name] ?? { timeoutMs: 5000, exposure: "deferred" as const };
    const params = draft07ToTypeBox(s.schema);
    pi.registerTool({
      name: s.name,
      label: s.name,
      description: s.description + (meta.descriptionSuffix ?? ""),
      parameters: params,
      exposure: meta.exposure,
      ...(ADMIN.has(s.name) ? { defaultActive: false } : {}),
      async execute(_id: string, args: Record<string, unknown>, signal: AbortSignal, _onUpdate: unknown, ctx: { cwd?: string }) {
        const a: Record<string, unknown> = { ...(args ?? {}) };
        if (typeof (s.schema as any)?.properties?.project === "object" && a.project == null && ctx?.cwd) a.project = ctx.cwd;
        if (s.name === "code_index") c.markIndexing(true);
        let r;
        try {
          r = await c.transport.call(s.name, a, { timeoutMs: meta.timeoutMs, signal });
        } finally {
          if (s.name === "code_index") { c.markIndexing(false); if (r?.ok) void c.refreshIndexState(ctx?.cwd ?? c.sessionCwd); }
        }
        if (!r.ok) {
          c.noteError(`${s.name}: ${r.error?.code} ${r.error?.message}`);
          return { content: [{ type: "text", text: `[${s.name}] ${r.error?.code}: ${r.error?.message}` }], details: undefined };
        }
        return { content: [{ type: "text", text: JSON.stringify(r.result, null, 2) }], details: undefined };
      },
    });
    n++;
  }
  return n;
}
```

`src/instructions.ts`:

```ts
export const CODE_INDEX_INSTRUCTIONS = `
[CodeIndex active] Graph context is appended to grep/read results.
- Unknown area? Call code_explore FIRST — it returns budgeted source + call trails.
- Before editing: code_impact (or code_affected for git changes); before merging: code_affected.
- code_rename is two-step: dry_run → plan_token → apply.
- Confidence tiers: trust LSP_RESOLVED/EXTRACTED; question INFERRED/AMBIGUOUS.
- Stale index: run /codeindex index.`;
```

- [ ] **Step 3: Run `node --test test/tools.test.ts` → PASS, then full `node --test test/`.**

- [ ] **Step 4: Commit**

```bash
git add plugins/pi/pi-mnemonic-code
git commit -m "feat(plugin): capability-gated code_* tool registration from schema dump"
```

---

### Task 6: Pattern Extraction Port (`augment.ts`)

Ported from `tintinweb/pi-gitnexus` `src/gitnexus.ts` (MIT) — functions `extractLiteralFromRegex`, `tokenizeBashCmd`, `extractPattern`, `extractFilePatternsFromContent`, `CODE_EXTENSIONS`. Keep the port verbatim where behavior is proven; add `glob` tool support; drop `read_many` unless the Task 1 spike shows the host has it (pi v2/omp read is single-path).

**Files:**
- Create: `plugins/pi/pi-mnemonic-code/src/augment.ts`
- Create: `plugins/pi/pi-mnemonic-code/test/augment.test.ts`

**Interfaces:**
- Produces:
  ```ts
  export const CODE_EXTENSIONS: Set<string>
  export function extractPattern(toolName: string, input: Record<string, unknown>): string | null
  export function extractFilePatternsFromContent(content: { type: string; text?: string }[], limit?: number): string[]
  /** Primary + secondary candidates for a tool_result event. */
  export function collectCandidates(toolName: string, input: Record<string, unknown>, content: { type: string; text?: string }[]): string[]
  ```

- [ ] **Step 1: Write the test corpus first** (cases derived from the gitnexus extractor semantics + host tool shapes):

`test/augment.test.ts` (full):

```ts
import test from "node:test";
import assert from "node:assert/strict";
import { extractPattern, extractFilePatternsFromContent, collectCandidates, CODE_EXTENSIONS } from "../src/augment.js";

test("grep: longest identifier literal from regex", () => {
  assert.equal(extractPattern("grep", { pattern: "validateUser|checkAuth" }), "validateUser");
  assert.equal(extractPattern("grep", { pattern: "foo_[a-z]+" }), null); // no literal >=3 chars? "foo" is 3 chars -> actually "foo"
  assert.equal(extractPattern("grep", { pattern: '"exactPhrase"' }), "exactPhrase");
  assert.equal(extractPattern("grep", { pattern: "a.b" }), null); // "a","b" too short
});
```

(When porting, compute the exact expected values from the reference implementation and pin them — the two `//`-annotated lines above are the ones to verify against the reference before pinning; every other case in the file is unambiguous.)

Additional required cases:
- `bash` `grep -rn "parseRequestURL" src/` → `parseRequestURL`; `rg --type go HandleUpload . | head` → `HandleUpload` (pipe-safe: head's arg is not taken); `cat internal/net/url.go` → `url`; `find . -name "upload_test.go"` → `upload_test`; `wc -l foo bar.go` → `bar`.
- `read` `internal/net/url.go` → `url`; `read` `README.md` → null (not in CODE_EXTENSIONS).
- `glob` `**/upload*.go` → `upload`.
- `extractFilePatternsFromContent([{type:"text",text:"internal/handler/upload.go:88:HandleUpload\ninternal/net/url.go:210:parseQuery\nother.txt:1:x"}], 2)` → `["upload", "url"]`.
- `collectCandidates("grep", {pattern:"parseRequestURL"}, [grep result lines])` → primary first, secondary after, deduped, no dupes.
- Quoting: `bash` `grep "a b" file.ts` → null (space breaks identifier); `grep 'validateToken' x.go` → `validateToken`.
- Length guard: everything <3 chars → null.

- [ ] **Step 2: Run to verify fail; implement `src/augment.ts` by porting**

Port `extractLiteralFromRegex` (split on `[\\^$.*+?()[\]{}|]+`, strip quotes, require ≥3 chars + `/^[a-zA-Z_]\w*$/`, longest wins), `tokenizeBashCmd` (quote-aware, `|` boundary tokens at `|`/`&&`/`||`/`;`), `extractPattern` (per-tool dispatch: grep → literal; find/glob → basename minus ext/glob chars; bash → tokenizer scan for grep/rg pattern, cat/head/tail/less/wc filename with CODE_EXTENSIONS check, find -name/-iname value; read → basename with extension check; add `glob`), `extractFilePatternsFromContent` (line regex `^([^\n:]+\.\w+):\d+:`, basename minus ext, ≥3 chars, `limit` default 2). `CODE_EXTENSIONS` = gitnexus set (`.sol .ts .tsx .js .jsx .py .go .rs .java .kt .swift .c .cpp .h .hpp .cs .rb .php .vy .fe .huff .md .mdx`).

`collectCandidates`:

```ts
export function collectCandidates(toolName: string, input: Record<string, unknown>, content: { type: string; text?: string }[]): string[] {
  const primary = extractPattern(toolName, input);
  const secondary = (toolName === "grep" || toolName === "bash")
    ? extractFilePatternsFromContent(content, 2)
    : [];
  const out: string[] = [];
  for (const p of [primary, ...secondary]) {
    if (p && !out.some((x) => x.toLowerCase() === p.toLowerCase())) out.push(p);
  }
  return out;
}
```

- [ ] **Step 3: Run `node --test test/augment.test.ts` → PASS (all corpus cases).**

- [ ] **Step 4: Commit**

```bash
git add plugins/pi/pi-mnemonic-code
git commit -m "feat(plugin): port proven pattern extraction from pi-gitnexus"
```

---

### Task 7: Auto-Augment Hook (`tool_result`)

**Files:**
- Modify: `plugins/pi/pi-mnemonic-code/src/index.ts` (hook wiring — index.ts is created in this task if not already scaffolded beyond the probe; final form below)
- Create: `plugins/pi/pi-mnemonic-code/test/index.test.ts`

**Interfaces:**
- Consumes: `Controller.shouldAugment/augmentPatterns`, `collectCandidates`, host `isSubagent`.
- Produces: the `tool_result` handler; combined-append cap `MAX_APPEND_BYTES = 8192`; append shape:
  ```
  \n\n---\n[CodeIndex: <pattern1>, <pattern2>]\n<block(s)>\n---
  ```
  (single pattern → `[CodeIndex: <pattern>]`; multi-block body joined by `\n`.)

- [ ] **Step 1: Write failing hook tests**

`test/index.test.ts` (full core):

```ts
import test from "node:test";
import assert from "node:assert/strict";
import { createExtension } from "../src/index.js"; // factory export: (opts: {controller?, host?}) => register function — keeps index.ts testable
import { Controller } from "../src/controller.js";
import { spawnTransport } from "../src/transport.js";
import { detectHost } from "../src/host.js";

const STUB = new URL("./fixtures/stub-mnemonic/stub-mnemonic", import.meta.url).pathname;
const mk = () => {
  const controller = new Controller({ transport: spawnTransport(STUB), host: detectHost(), env: process.env as any });
  const handlers: Record<string, ((e: any, ctx: any) => any)[]> = {};
  const pi = {
    registerTool: () => {}, registerCommand: () => {}, registerFlag: () => {},
    on: (name: string, h: any) => { (handlers[name] ??= []).push(h); },
  };
  const register = createExtension({ controller });
  register(pi as any);
  const fire = (name: string, e: any, ctx: any = { cwd: "/tmp", agent: { kind: "main" } }) =>
    Promise.all((handlers[name] ?? []).map((h) => h(e, ctx)));
  return { controller, pi, fire };
};

test("grep result gets appended augment block", async () => {
  process.env.STUB_MODE = "ok";
  const { controller, fire } = mk();
  await controller.onSessionStart("/tmp");
  process.env.STUB_MODE = "augment-ok";
  const e = { toolName: "grep", input: { pattern: "parseRequestURL" }, content: [{ type: "text", text: "src/x.go:10:parseRequestURL" }] };
  const out = await fire("tool_result", e);
  const changed = out.find(Boolean);
  assert.ok(changed, "handler must return a modified result");
  const text = changed.content.map((c: any) => c.text ?? "").join("");
  assert.ok(text.includes("[CodeIndex: parseRequestURL]"));
  assert.ok(text.includes("---"));
});

test("no index -> unchanged", async () => {
  process.env.STUB_MODE = "err";
  const { controller, fire } = mk();
  await controller.onSessionStart("/tmp"); // indexState null
  const e = { toolName: "grep", input: { pattern: "parseRequestURL" }, content: [{ type: "text", text: "x" }] };
  assert.equal((await fire("tool_result", e)).find(Boolean), undefined);
});

test("subagent session skips augment", async () => {
  process.env.STUB_MODE = "ok";
  const { controller, fire } = mk();
  await controller.onSessionStart("/tmp");
  process.env.STUB_MODE = "augment-ok";
  const e = { toolName: "grep", input: { pattern: "parseRequestURL" }, content: [{ type: "text", text: "x" }] };
  assert.equal((await fire("tool_result", e, { cwd: "/tmp", agent: { kind: "subagent" } })).find(Boolean), undefined);
});

test("append capped at 8 KiB", async () => {
  process.env.STUB_MODE = "ok";
  const { controller, fire } = mk();
  await controller.onSessionStart("/tmp");
  process.env.STUB_MODE = "augment-big"; // stub variant: 1.5 KiB block per pattern
  const e = { toolName: "bash", input: { command: "grep -rn one a; grep -rn two b; grep -rn three c" }, content: [{ type: "text", text: "" }] };
  const out = (await fire("tool_result", e)).find(Boolean);
  const added = out.content.map((c: any) => c.text ?? "").join("").length - 0;
  assert.ok(added <= 8192 + 256, "append must respect the combined cap");
});

test("toggle off disables hook (tools unaffected)", async () => {
  process.env.STUB_MODE = "ok";
  const { controller, fire } = mk();
  await controller.onSessionStart("/tmp");
  process.env.STUB_MODE = "augment-ok";
  const e = { toolName: "grep", input: { pattern: "parseRequestURL" }, content: [{ type: "text", text: "x" }] };
  assert.ok((await fire("tool_result", e)).find(Boolean));
  controller.augmentOn = false;
  assert.equal((await fire("tool_result", e)).find(Boolean), undefined);
});
```

- [ ] **Step 2: Implement the hook in `src/index.ts`** (final module form):

```ts
import { Controller } from "./controller.js";
import { spawnTransport } from "./transport.js";
import { detectHost, resolveBinary, resolveShellPath, type Host } from "./host.js";
import { collectCandidates } from "./augment.js";
import { registerTools } from "./tools.js";
import { registerCommands } from "./commands.js";
import { CODE_INDEX_INSTRUCTIONS } from "./instructions.js";

const AUGMENT_TOOLS = new Set(["grep", "bash", "read", "glob", "find"]);
const MAX_APPEND_BYTES = 8 * 1024;

export interface ExtensionOpts { controller?: Controller; host?: Host }

export function createExtension(opts: ExtensionOpts = {}) {
  const host = opts.host ?? detectHost();
  return function register(pi: any): void {
    const env = process.env as Record<string, string | undefined>;
    // Binary resolution is async (shell PATH) — resolved in onSessionStart; controller starts offline.
    const controller = opts.controller ?? new Controller({ transport: spawnTransport(null), host, env });

    void (async () => {
      const path = await resolveShellPath();
      const bin = resolveBinary(path);
      (controller as { transport: ReturnType<typeof spawnTransport> }).transport = bin ? spawnTransport(bin) : controller.transport;
    })();

    pi.on("session_start", async (_e: any, ctx: any) => {
      if (host.isSubagent(ctx)) return;
      try {
        const path = await resolveShellPath();
        const bin = resolveBinary(path);
        controller.transport = spawnTransport(bin);
        await controller.onSessionStart(ctx?.cwd ?? process.cwd());
        if (controller.offline) notify(pi, ctx, "CodeIndex: mnemonic binary not found — run `mnemonic doctor` or set MNEMONIC_BIN.", "warning");
        else if (!controller.indexState) notify(pi, ctx, "CodeIndex: no index in this project — run /codeindex index.", "info");
        else if (controller.stale) notify(pi, ctx, "CodeIndex: index stale — /codeindex index for a full refresh.", "info");
        else notify(pi, ctx, `CodeIndex active: ${controller.indexState.files} files, ${controller.indexState.symbols} symbols.`, "info");
        setStatus(pi, ctx);
      } catch (err) {
        controller.noteError(`session_start: ${String(err)}`); // fail-open
      }
    });

    pi.on("before_agent_start", async (e: any, ctx: any) => {
      if (host.isSubagent(ctx) || !host.isUserPrompt(e)) return;
      if (!controller.ready || env.MNEMONIC_CODE_PI_INJECT === "0") return;
      if (typeof e?.systemPrompt !== "string") return;
      return { systemPrompt: e.systemPrompt + CODE_INDEX_INSTRUCTIONS };
    });

    pi.on("tool_result", async (e: any, ctx: any) => {
      try {
        if (!controller.shouldAugment() || host.isSubagent(ctx)) return;
        if (!AUGMENT_TOOLS.has(e?.toolName)) return;
        if (!Array.isArray(e?.content)) return;
        const candidates = collectCandidates(e.toolName, e.input ?? {}, e.content);
        if (!candidates.length) return;
        const body = await controller.augmentPatterns(ctx?.cwd ?? process.cwd(), candidates);
        if (!body) return;
        const label = `[CodeIndex: ${candidates.slice(0, 3).join(", ")}]`;
        let append = `\n\n---\n${label}\n${body}\n---`;
        if (Buffer.byteLength(append, "utf8") > MAX_APPEND_BYTES) {
          append = append.slice(0, MAX_APPEND_BYTES) + "\n… (truncated)";
        }
        return { content: [...e.content, { type: "text", text: append }] };
      } catch (err) {
        controller.noteError(`tool_result: ${String(err)}`);
        return; // fail-open: never throw into the host
      }
    });

    pi.on("session_shutdown", async (e: any, ctx: any) => {
      try {
        if (!host.isRealShutdown(e) || host.isSubagent(ctx)) return;
        controller.onSessionEnd();
      } catch { /* fail-open */ }
    });

    void registerTools(pi, controller).then((n) => {
      controller.noteError(n === 0 ? "tools: none registered (offline or empty schema)" : "");
    });
    registerCommands(pi, controller, host);
  };
}

function notify(pi: any, ctx: any, msg: string, level: "info" | "warning" | "error"): void {
  try { ctx?.ui?.notify?.(msg, level); } catch { /* fail-open */ }
}
function setStatus(pi: any, ctx: any): void {
  try {
    const c = /* controller from closure */ null; // set by registerCommands status-line wiring (Task 8)
  } catch { /* */ }
}

// Default export so pi loads it directly; createExtension is the testable factory.
import def from "./index-default.js"; // (or inline: export default createExtension()({}); see step 3 note)
```

Step 3 note (default export shape): pi calls the module's **default export** with `pi`. Export `default createExtension()` directly at the bottom of `index.ts` (a module-level instance), and let `createExtension` accept injected deps for tests — do **not** import the module into itself; drop the `index-default.js` line and write `export default createExtension();` after the function. `setStatus` is implemented properly in Task 8 (it needs the controller; keep a module-closure `let activeController` assigned inside `register`, and have Task 8's status code read it).

- [ ] **Step 3: Run `node --test test/index.test.ts` → PASS; then full `node --test test/`.**

- [ ] **Step 4: Commit**

```bash
git add plugins/pi/pi-mnemonic-code
git commit -m "feat(plugin): auto-augment tool_result hook with dedup, budgets, fail-open"
```
