# Round 5 Verification Report — 2026-11-17-mnemonic-v2-observation-layer

User-directed change verification (narrow scope). The user instruction
("omp is a pi, so the same solution applies to it like pi") drove an
in-place revision of `blueprint.md`, `briefing.md`, and `acceptance.feature`:
omp's install shape was unified with pi's — folder + manifest
(`~/.omp/agent/extensions/mnemonic-obs/{mnemonic-obs.ts,package.json}`) with a
dual-key manifest (`pi` + `omp` extension keys) and one shared folder copyer,
replacing the earlier bare-file proposal. Scope: (a) delta-consistency sweep
of every omp install reference; (b) oh-my-pi claim check against
`/tmp/skillgrid-plugin-research/omp-local.md` + live host state; (c) spot-read
that no round-1–4-fixed line changed. Read-only toward the three spec files;
no git, no go test.

## 1. Delta-consistency sweep

Every omp install reference in the three files, classified:

| Site | Claim | Verdict |
|---|---|---|
| blueprint:160-167 (Install targets) | omp → `~/.omp/agent/extensions/mnemonic-obs/` — **same folder + manifest shape as pi**; discovery "covers direct `*.ts` files AND one-level subdirs with a manifest, and accepts the legacy `pi` manifest key — pinned 2026-11-17, user direction; the bare-file form remains the gryph/herdr precedent, not our choice" | ✓ folder + manifest |
| blueprint:154 (File Structure) | `plugins/obs/pi/{mnemonic-obs.ts,package.json}` — pi+omp source (repo-side single folder) | ✓ consistent with one-folder copyer |
| blueprint:439-444 (Task 4 adapter source) | `package.json`: `{"name":"@devopstales/pi-mnemonic-obs","pi":{"extensions":["./mnemonic-obs.ts"]},"omp":{"extensions":["./mnemonic-obs.ts"]}}` — "the ADR-0043 manifest shape for pi + the `omp` key for oh-my-pi, which also accepts the legacy `pi` key — one manifest serves both hosts" | ✓ dual-key manifest |
| blueprint:454-476 (Task 4 Step 1 tests) | install omp to fixture HOME → `~/.omp/agent/extensions/mnemonic-obs/{mnemonic-obs.ts,package.json}` exists ("the same folder + manifest shape as pi — omp auto-discovers one-level subdirs with a manifest"); `const AGENT = "omp"`, not `__AGENT__`; manifest record present (**kind `dir`**); re-install no-op; pi via `SKILL_PI_AGENT_DIR=<tmp>` → `<tmp>/extensions/mnemonic-obs/{mnemonic-obs.ts,package.json}` (**identical copyer, only the target dir differs**); uninstall removes exactly those (foreign `other-extension.ts` survives); detection omp → `~/.omp/agent` exists (unchanged) | ✓ folder + manifest, kind `dir`, detection unchanged |
| blueprint:478-479 (Task 4 Step 3 copyer) | "ONE folder copyer = read→write→hash for both omp and pi — same shape, only the target dir differs: omp `~/.omp/agent/extensions/`, pi `<pi-agent-dir>/extensions/`" | ✓ shared copyer |
| blueprint:491-493 (Task 4 Step 5) | live backstop: real install on this host (`obs install omp`), run this omp session's tool calls, `obs logs` → rows present | ✓ empirical settler (see §2) |
| briefing:131 (host table, omp row) | "**oh-my-pi** … the same pi ExtensionAPI; extension discovery = direct `*.ts` files **AND one-level subdirs with a manifest** (legacy `pi` manifest key accepted) \| `~/.omp/agent/extensions/mnemonic-obs/` — **same folder + manifest shape as pi** (user-pinned 2026-11-17: 'omp is a pi…'; the bare-file form remains the gryph/herdr precedent)" | ✓ folder + manifest |
| briefing:209-211 (Plugin sources) | "**omp** installs the same folder + manifest as pi (auto-discovered dir — omp is oh-my-pi, the same ExtensionAPI). One source, one shape, two target dirs." | ✓ folder + manifest |
| briefing:345-346 (Acceptance criterion 1) | "omp gets `~/.omp/agent/extensions/mnemonic-obs/` (same folder + manifest shape as pi — omp is oh-my-pi); `obs status` reports all 5 installed" | ✓ folder + manifest |
| acceptance:73-80 (`obs-install-omp-and-pi`) | `~/.omp/agent/extensions/mnemonic-obs/{mnemonic-obs.ts,package.json}` exists — "the SAME folder + manifest shape as pi (omp is oh-my-pi; its discovery covers one-level subdirs with a manifest)"; `const AGENT = "omp"` (never `__AGENT__`); package.json lists the extension under **BOTH the `pi` and `omp` keys**; manifest under the data dir records the file hashes; re-run idempotent; `obs install pi` creates `<dir>/extensions/mnemonic-obs/{…}` **via the identical copyer**; `obs uninstall omp` removes exactly the installed folder (a foreign extension survives) | ✓ folder + manifest + dual key |

**Non-delta omp references verified neutral** (ingest/CLI/evidence only, no
install-shape claim): blueprint:15-16 (build shape), 61-64 (T-1/T-2), 125
(spawn caps), 176 (envelope example), 188-190 (event mapping rows), 227
(Agent* set), 257 (normalize test), 366/375/392 (Task 3 fixtures/door check),
405 (host args), 423-432 (adapter API hooks), 653/661 (Task 9), 709 (Review
Summary history); briefing:26/31 (host list), 61-64 (gryph live evidence —
see below), 139 (architecture diagram), 203 (verb table), 223/228/238-240
(caps + R1), 331-337 (D6 evidence), 348 (criterion 2), 389 (build shape);
acceptance:14-17, 30 (ingest scenarios).

**Cross-consistency (exact on-disk shape):** the acceptance scenario
`obs-install-omp-and-pi`, briefing Acceptance criterion 1, and blueprint Task
4 Steps 1/3 all agree: `~/.omp/agent/extensions/mnemonic-obs/` containing
`mnemonic-obs.ts` (with `const AGENT = "omp"` after `__AGENT__`
substitution) + `package.json` (dual `pi`/`omp` extension keys), tracked in
`~/.skillgrid/mnemonic/obs-install.json` with content hashes and HostRecord
kind `dir` (the enum at blueprint:416-417, `file|dir|hooks-json`, carries
`dir`; no site left asserting kind `file` for omp), uninstall removes exactly
the folder. No contradictions.

**Gryph/herdr bare-file precedent — historical evidence only, as allowed:**
briefing:61-64 (gryph's own live install: `~/.omp/agent/extensions/gryph-hooks.ts`,
herdr's file, "auto-discovers `~/.omp/agent/extensions/*.ts`") and the
framing phrases "the bare-file form remains the gryph/herdr precedent, not
our choice" (blueprint:166-167) / "remains the gryph/herdr precedent"
(briefing:131). **No remaining bare-file claim for OUR omp install** in any
of the three files (full-text sweep of `bare`, `subdir`, `manifest`,
`mnemonic-obs` across all three files).

**Must-haves / threat matrix / detection:** no T-* names the omp install
path (T-2 is the live backstop; the path is pinned in Task 4 + acceptance —
consistent, no conflict). No threat-matrix row names the omp file; the
"Documentation-like paths" row ("installed adapters are `.ts` code files +
JSON config") remains true for the folder shape. Detection rule unchanged:
omp → `~/.omp/agent` exists (blueprint:466).

**SATISFIES cross-check:** all scenario names cited in Task 4
(`obs-install-omp-and-pi`, `obs-fail-open-missing-binary`,
`obs-live-omp-session-produces-events`) exist verbatim in acceptance.feature.

## 2. Oh-my-pi evidence check

**(a) Research notes** (`/tmp/skillgrid-plugin-research/omp-local.md`,
2026-11-17), §3a "Extension discovery (oh-my-pi `docs/extension-loading.md`…)":

> 1. `<cwd>/.omp/extensions/` and `~/.omp/agent/extensions/` (direct
> `*.ts`/`*.js` files + **one-level subdirs with index or manifest**).
> 2. Installed plugins under `~/.omp/plugins/node_modules` via
> `package.json` `omp.extensions` (legacy `pi.extensions`).
> …
> entry resolution `package.json#omp.extensions` → `index.ts` → `index.js`.

- **Subdir + manifest under `~/.omp/agent/extensions/`: SUPPORTED** — item 1
  names the user extensions dir explicitly, not just installed plugins.
  → **Not Major** (the Major trigger from the round-5 brief does not fire).
- **Legacy `pi` key:** documented for installed plugins (item 2) and in the
  plugin package.json layout (§3b: "legacy `pi` key ok"); the extensions-dir
  entry-resolution line names only `package.json#omp.extensions`. The spec's
  "legacy `pi` manifest key accepted" for the extensions-dir subdir is a
  slight over-claim of the notes' explicit wording — but **non-load-bearing**:
  the pinned manifest ships the `omp` key, which is the primary documented
  resolution path. → F-16 (sub-threshold minor, no fix required).

**(b) Live host state:** `~/.omp/agent/extensions/` currently holds exactly
`gryph-hooks.ts` + `herdr-omp-agent-state.ts` (two bare `.ts` files; no
subdirs, no manifests). Nothing on this host has yet empirically exercised
subdir+manifest discovery — **Task 4 Step 5 is what settles it empirically**
(real `obs install omp` → live omp session → `obs logs`). Execution nuance:
the research notes (§2, §3b) state new extension modules require a new
session / full restart (registration happens at load; spec has no restart
mention — grep confirms) — see F-17.

## 3. Round-1–4 fixed lines (spot-read, verbatim)

All intact — the delta touched only the omp install-shape text:

- Event mapping table (blueprint:186-197): pi,omp rows incl. the round-2
  F-1 fix `| pi, omp | session_shutdown (quit only) | session_end |` and the
  full cursor 20-hook rows — verbatim. ✓
- Exit-code contract (blueprint:199-201): "0 ok · 1 error (usage, parse,
  store, dialect, unknown event — always allow) · 2 reserved (block, phase 2
  — **never** a usage error)" — verbatim. ✓
- `Txn.Exec` (blueprint:295-297): Task 2 additive `Exec(ctx, query, args…)`
  on the `Txn` interface + "composes two raw statements inside one Txn" —
  verbatim. ✓
- `MatchFile` (blueprint:309-316): `**` segment semantics + no-`/` basename
  match + the four pinned test rows — verbatim. ✓
- Port attribution (briefing:89): v1 `skillgrid serve :7438` — verbatim. ✓
- Detection rules (blueprint:466), 30s/5s caps (blueprint:125; briefing:223),
  K6 envelope seam (briefing:119), pi/omp adapter unit test (blueprint:471-476),
  sessions derivations (Task 7 Produces), ADR-0058 citation (blueprint:289),
  ADR-0061 both sites (briefing:10, 295) — all verbatim. ✓

## 4. Findings

### F-16 — MINOR (sub-threshold) — "legacy `pi` manifest key accepted" over-claims the notes for the extensions-dir subdir

briefing:131 and blueprint:442-444 state the legacy `pi` manifest key is
accepted for the `~/.omp/agent/extensions/<subdir>/` manifest; the research
notes document legacy-pi-key acceptance for installed plugins (item 2) and
plugin package.json layouts (§3b), while the extensions-dir entry resolution
names only `package.json#omp.extensions`. Non-load-bearing: the pinned
dual-key manifest ships `omp.extensions` (the documented primary path), so
the install works either way. No fix required; the phrasing reads as
evidence-grounded as-is.

### F-17 — MINOR (sub-threshold, execution note) — Task 4 Step 5 / Task 9 backstop presumes the live session loaded the new extension

Task 4 Step 5 ("run this omp session's tool calls") and Task 9 Step 1
("omp = this session") assume the running omp session sees the freshly
installed extension. The research notes (§2: "new session or `/reload-plugins`
… may still require a full restart"; §3b: "extension modules/hooks/tools need
a full restart") say extensions load at session start. Pre-existing wording
(not introduced by this delta; it applied to the bare-file proposal too), but
the T-2 backstop only passes if the evidence session is started after install
(or restarted). Suggested fix (optional, at execution): Step 5 starts a new
omp session post-install.

## Verdict

**CLEAN** — 0 Blocker, 0 Major. The omp folder+manifest unification is
consistent across all three spec files (install targets, Task 4 tests/copyer/
manifest kind, host table, plugin sources, acceptance criterion 1, and the
`obs-install-omp-and-pi` scenario agree on the exact on-disk shape); the
oh-my-pi claims are supported by the 2026-11-17 research notes (subdir +
manifest discovery under `~/.omp/agent/extensions/` explicit in item 1), with
Task 4 Step 5 as the empirical settler on this host (no live subdir-manifest
example exists yet); no round-1–4-fixed line changed. Two sub-threshold
minors noted (F-16 legacy-pi-key wording, F-17 backstop session-restart
nuance), both non-blocking, neither introduced by the delta.
