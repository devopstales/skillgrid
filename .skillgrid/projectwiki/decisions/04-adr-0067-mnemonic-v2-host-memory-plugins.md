---
type: TechnicalDecision
title: "mnemonic-v2 5-host memory plugins: unified host-MCP transport, adapter-owns session ID, skillgrid-cli installer, one-shot v1 data import"
status: "accepted"
supersedes: none
date: 2026-11-17
---

# Mnemonic-v2 5-host memory plugins: architecture

## Context and Problem Statement

The change `2026-11-17-mnemonic-v2-host-memory-plugins` gives five coding-agent
hosts (pi, omp, opencode, kilo, cursor) the mnemonic-v2 memory surface
(22 agent-profile `mem_*` tools + prompt capture + session registration +
compaction recovery). The engine already exposes everything over MCP stdio
(`mnemonic memory mcp --profile agent`); what was missing was the host-facing
plugin layer and a clean v1 cutover.

The anchored hypothesis (user-confirmed) was a literal port of the engram
reference architecture: for pi/omp, a per-session spawned `mnemonic memory
mcp` stdio child behind `pi.registerTool` with 22 static typebox schemas, a
hand-rolled ~150-line TS MCP wire client, and a Go-generated 278-line runtime
contract projection checked at spawn (mismatch → fail-closed for the
session). A six-lens Goldfish shootdown (2026-11-17) attacked this and the
rest of the design; two blockers and several design-shape majors were
user-dispositioned. The decisions below are the outcome. They are hard to
reverse (installer shape, host config surfaces, and the capture protocol are
all user-visible and cross-change load-bearing — the queued obs and
host-plugins changes both migrate onto them) and they trade against each
other, so they clear the ADR bar as one architecture decision.

## Considered Options

- **A. Literal engram port (spawned child + registerTool + wire client + contract projection)** — proven in engram; maximum control (deterministic fail-closed spawn, cwd pinning, explicit lifecycle). Rejected by the user at the shootdown gate: the projection/self-heal machinery targets a risk profile (an independently released HTTP server) this local-binary design does not have; omp's live `~/.omp/agent/mcp.json` already runs a host-spawned MCP child and pi has a built-in `mcp.json` mechanism; the port would add a second MCP wire-protocol implementation to the repo plus a Go↔TS schema-duplication seam for 22 tools.
- **B. Unified host-MCP transport (chosen)** — every host gets a `mnemonic-memory` MCP config entry; pi/omp and opencode/kilo additionally get thin hooks-only extensions (shared TS source per pair) for capture, session registration, compaction recovery, protocol host-shell, and a session-start liveness probe; cursor gets hooks + an alwaysApply rule. One transport for five hosts; the heaviest engram machinery (child lifecycle, self-heal, projection, wire client) is dropped.
- **C. Hybrid (host-MCP for tools + a spawned child only for the contract gate)** — the gate would only check what it spawns, not what the host runs; more moving parts for a weaker guarantee. Rejected.

Secondary decisions (same gate): **session identity** — adapter-owns (engram pattern) vs protocol-embedded static IDs vs dynamic per-session IDs; **installer home** — new `mnemonic/internal/hosts` package vs extending skillgrid-cli vs command-local; **v1 data continuity** — Go `migrate` verb vs accepting the fork vs one-shot script.

## Decision Outcome

Chosen option: **B**, plus the secondary decisions below.

1. **Unified host-MCP transport.** All five hosts dispatch memory tools
   through a host-owned MCP entry: key `mnemonic-memory`, command
   `[<abs-bin>, "memory", "mcp", "--profile", "agent"]` (absolute binary
   path in every entry — the GUI-PATH case must not be green-but-dead).
   pi/omp and opencode/kilo get a thin hooks-only extension each (one shared
   TS source per pair; node builtins only; no MCP client, no HTTP). The
   pi/omp extension adds a **session-start liveness probe** (binary resolves
   → MCP `initialize` via transient spawn → one store read via
   `memory search --limit 1`); on failure it injects an actionable warning
   (cause + exact repair command) and records it to a per-project scratch
   file surfaced by `status`. This probe replaces engram's spawn-time
   contract projection: with a host-owned child there is nothing to
   fail-close against, and the host's own "server broken / tools absent"
   behavior covers the rest.
2. **Adapter-owns session identity.** The adapter (extension/plugin/hook) is
   the single authority on `session_id`: pi/omp inject it, opencode/kilo
   rewrite it in `tool.execute.before` (with the `<key>_` prefix
   normalizer), cursor pins the stable per-host ID `cursor` in the rule
   (hooks cannot rewrite MCP args). This matters because `mem_save`
   auto-registers unknown session IDs — a wrong guess silently forks the
   session instead of erroring. Known simplification: concurrent cursor
   sessions in one project share the `cursor` session row.
3. **Installer home: extend skillgrid-cli.** The memory family joins
   `skillgrid-cli/internal/install/` (`skillgrid install|uninstall|status
   --family memory`), which already resolves all five hosts' config paths
   and merges JSON(C) preserving foreign keys. The manifest is keyed
   **(host, family)** (a second family — obs — writes the same
   `~/.cursor/hooks.json`); tracked-set-only: foreign files/entries are
   never adopted, never clobbered, never deleted. The merge/manifest
   machinery is an exported package boundary so the queued obs change
   migrates onto it instead of carrying a second installer.
4. **One-shot v1 data import script (no Go verb).** The v1 store for a
   project (`<v1-id>.sqlite`, e.g. `aiskillgrid.sqlite`) and the v2 store
   (`<repo-name>.sqlite`) are different files with incompatible schemas
   (v1 `observations` = VIEW over satellites; v2 = TABLE); the engine
   resolves the repo to the v2 name, so v1-era observations would be
   silently lost to recall. `scripts/mnemonic-v1-import.sh` (sqlite3 CLI,
   idempotent, `--dry-run`) bridges them per project. This holds the
   user-locked "zero Go engine changes except `memory save-prompt`" — a
   `migrate` verb was considered and rejected at the gate to keep that
   lock intact.
5. **Zero-Go lock held.** The only new Go surface is the
   `mnemonic memory save-prompt` CLI verb (per-turn prompt capture; stdin
   = plain text or host hook JSON; idempotent under re-fire). Recorded
   exceptions: a ≤2,048-rune guard test on the MCP instructions text (and a
   text trim if it fails), nothing else.

### Implementation

- `plugins/memory/pi-omp/` + `plugins/memory/opencode-kilo/` — shared TS
  sources (kilo byte-identical copy; no `__AGENT__` substitution — it has
  no v2 consumer).
- `skillgrid-cli/internal/install/` — memory family + exported
  Install/Uninstall/Status + Manifest boundary; manifest at
  `~/.skillgrid/mnemonic/memory-install.json`.
- `mnemonic/cmd/mnemonic` — `save-prompt` verb on the existing
  prompt-store path.
- `scripts/mnemonic-v1-import.sh`; v1 cutover inventory per the briefing
  (R8).

### Consequences

- Good, because one transport serves five hosts: no second MCP wire-protocol
  implementation, no per-session child lifecycle, no hand-mirrored schemas —
  the deferred code-index and project-wiki families each add one config
  entry per host, not a per-host code change.
- Good, because the single Go-authored protocol source (MCP initialize
  instructions) with host-shell-only deltas removes the four-copy drift the
  engram reference already exhibits (its pi/opencode/kilo copies diverge in
  the installed tree).
- Good, because the liveness probe + scratch record + `status` answer
  "is memory working in this host right now, and why did it die" — the
  failure mode the shootdown found un-triageable.
- Bad, because fail-closed becomes fail-open: a broken/absent MCP server
  means missing tools, not a blocked session. Mitigation: the probe warning
  names the cause and repair; `status` is the triage command. Accepted at
  the gate (engram's fail-closed bought nothing this design needs — its
  child is the same binary the host would spawn anyway).
- Bad, because cursor sessions in one project share a session row.
  Accepted (per-project stores keep projects apart; the dynamic-ID
  alternative depends on a hook the queued host-plugins change provides).
- Bad, because the v1 import is a script, not a repeatable engine command —
  a machine that skips it loses v1 recall silently. Mitigation: the script
  is documented and the cutover task runs it for this repo; promoting it to
  a verb is a later, cheap change.

### Revisit criteria

Revisit this ADR (write a superseding ADR) when **any** of:
1. pi or omp is verified not to spawn `mcp.json` servers at the session
   cwd — that host falls back to a per-session spawned child (deviation
   recorded in the blueprint at execution).
2. The code-index or project-wiki change lands and its per-host wiring
   shows the thin extensions need a family abstraction — generalize the
   extension before a third family, not after.
3. A host is found to surface MCP server `instructions` unreliably and the
   protocol starts drifting again — re-home the single source (e.g. a
   generated per-host file under installer control).
4. v1 data continuity is needed on machines where the script was never run
   — promote `mnemonic-v1-import` to a `memory migrate` verb.
