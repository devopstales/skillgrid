# @devopstales/omp-skillgrid-guards

P0 enforcement guards for the skillgrid pipeline, run as an **omp hook**
(https://omp.sh/docs/hooks) on the agent's `write` / `edit` tool calls.

The skillgrid pipeline's state machine is the filesystem
(`_shared/rules/sdd-structure.md`): artifact order, STATUS banner ownership,
and canonical names are stated as prose MUSTs that the agent self-polices.
This plugin turns the three most-repeated, fully deterministic of those rules
into machine checks that fire at the moment of violation instead of at audit
time.

## The three guards

| Guard | Rule (source) | Check | On violation |
|---|---|---|---|
| **Spec-zone file set** (`src/fileset.ts`) | canonical names are load-bearing — `resume` and phase derivation key off exact filenames (sdd-structure § Directory Structure; the spec-zone-only HARD-GATEs) | a new file under `change-specs/<topic>/` must be a canonical artifact (`briefing.md`, `blueprint.md`, `acceptance.feature`, `tasks.md`, `adr.md`, `findings.md`, `ledger.md`, `report.md`, `review.md`, `*.council.md`) or live under `shared/`, `modules/`, `sketches/`, `evaluations/`; `change-archive/` is immutable after the ship move | block |
| **STATUS banner ownership** (`src/banner.ts`) | "the agent may write every value except `approved` … an agent edit after approval reverts the banner to `revised`" (sdd-structure § STATUS Banner) | on write/edit of the four banner files: banner present + valid value; fresh file starts at its initial value; the agent never writes `approved`; an edit of an `approved` doc must revert to `revised` | block |
| **Strict artifact order** (`src/artifact-order.ts`) | "artifact creation order is permanent and sequential … a skill MUST NOT run unless every preceding skill has completed" (sdd-structure § Phase Order) + the `user-gate-slice` gate | `blueprint.md` ← briefing `approved`; `acceptance.feature` ← blueprint `approved`/`assumed`/`accepted`; `tasks.md` ← acceptance `approved`; `report.md` ← execution signal (checked ticket or scratch ledger); `review.md` ← `report.md`; code-zone writes on an executing change ← cleared `user-gate-slice` gate. Fast-track waivers (`## Fast-Track Waiver` block in the briefing) relax exactly what they skip | block |

## Merge and failure semantics

- The first `block` wins; its `reason` becomes the tool-call error text the
  agent sees. Non-blocking `context` notes are delivered as
  `additionalContext` (e.g. "editing a pre-existing non-canonical file").
- **Fail-open**: missing `.skillgrid/`, unparseable `state.yaml`/`config.yaml`,
  ambiguous multi-active state, or any internal error → no decision. The
  runner is fail-closed on a throwing `tool_call` handler, so the factory
  catches everything. A guard bug degrades to a no-op; it never bricks the
  session.

## Coverage (v1)

- Tools: `write` and `edit` only. Agent writes via `bash` (heredoc, `sed -i`)
  are out of scope — the git-hook layer (`~/.skillgrid/hooks/precommit-zone-guard.js`
  et al.) still applies at commit time.
- The **user's** hand-edit bypasses the hook by design (they edit in their own
  editor, which the tool hooks do not see) — that is exactly the ownership
  model: the user's line-3 edit *is* the approval record.
- `edit` content is simulated (string replace of `old_string`); fuzzy
  whitespace matching the tool may apply beyond a literal match is not
  simulated (the guard fails open when it cannot determine the result).

## Wiring

Ambient project discovery loads `<cwd>/.omp/hooks/pre/*.ts`. This project
wires the factory through the thin shim:

```
.omp/hooks/pre/skillgrid-guards.ts   →  plugins/omp/skillgrid-guards/hooks/skillgrid-guards.ts
```

The shim re-exports the factory; the plugin directory is the single source of
truth (tests, backstop, and the logic live here). To activate in another
skillgrid project: copy `plugins/omp/skillgrid-guards/` + the shim (or point
an omp extension path at `hooks/skillgrid-guards.ts`).

## Test

```
cd plugins/omp/skillgrid-guards
bun install
bun test          # 5 suites: banner, fileset, artifact-order, resolve-change, guards (integration + adapter)
```

## Self-validation backstop

Per the repo's self-validation rule (skillgrid SKILL.md § Verification), this
change's own checkers were run against the repo's live spec zone — negative
half first, positive half after:

```
bun scripts/backstop.mjs [project-root]   # defaults to cwd
```

It exercises the guards against the newest active change's spec folder: a
deliberately broken write (invented filename) must be blocked; canonical
writes must pass. Exit 0 = both halves proven.

## Files

```
src/types.ts            minimal structural hook types (no @oh-my-pi import)
src/resolve-change.ts   project root + active change resolution (fail-open)
src/banner.ts           line-3 STATUS banner parse + ownership check
src/artifact-order.ts   strict-sequence predecessor check + execution gate
src/fileset.ts          canonical spec-zone name check
src/guards.ts           combined entrypoint (merge: first block wins)
hooks/skillgrid-guards.ts  omp hook factory (pi.on("tool_call", …))
tests/*.test.ts         bun test suites (tests sit beside, per craft rule)
scripts/backstop.mjs    self-validation backstop (negative + positive half)
```
