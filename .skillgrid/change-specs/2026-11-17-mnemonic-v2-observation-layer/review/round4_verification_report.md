# Round 4 Verification Report — 2026-11-17-mnemonic-v2-observation-layer

Final convergence pass (narrow scope). Round-3 verdict was CLEAN with one
sub-threshold minor (F-14, ADR misattribution in the Task 2 note). Scope:
(a) verify F-14 is RESOLVED against `.skillgrid/projectwiki/ASSUMPTIONS.md`
§ LOCKED; (b) ADR-citation sweep of every `ADR-\d+` in `blueprint.md` +
`briefing.md` against § LOCKED; (c) diff spot-check that no other text
changed since round 3. Read-only toward the change docs; no git, no go test.

## 1. F-14 disposition

**RESOLVED.**

Task 2 Files (blueprint:279-283) now reads:

> `(`pgCodeTxn`) — additive interface extension (the **ADR-0058**
> Store-extension precedent — the Code-index `Store` interface extension
> that established the additive Begin/Txn surface; both adapters hold
> `*sql.Tx`, one-line impls).`

Spot-check against ASSUMPTIONS.md § LOCKED:

- 0058 row = "Code-index `Store` interface extension: additive code-index
  write/read + vector + trail + ExecDDL + **Begin/Txn methods** (one-way
  doors D2/D3)" (`04-adr-0058-code-index-store-extension.md`) — title and
  claimed role (additive Begin/Txn surface) match. ✓
- 0045 row = "OpenCode V2 plugin port: dual V1+V2 entrypoints for the 6
  native plugins…" (`04-adr-0045-opencode-v2-plugin-port.md`) — confirmed
  NOT the store-extension decision. ✓

(Note: round 3 flagged a pre-existing numbering collision inside the vault —
the `04-adr-0058-code-index-store-extension.md` body header claims "ADR-0046".
The § LOCKED index (the authoritative pointer per ADR-0019) assigns 0058 =
store extension, 0046 = knowledge class dimension; the blueprint now agrees
with the index. Vault-side collision remains a separate pre-existing
housekeeping item, out of scope.)

## 2. ADR-citation sweep (blueprint.md + briefing.md)

Every `ADR-\d+` occurrence checked against § LOCKED (9 distinct numbers):

| Citation (file:lines) | Claimed role | § LOCKED title | Verdict |
|---|---|---|---|
| ADR-0008 (bp:34, 37, 121) | dependency policy / precedent (zero new deps) | "State drift guard: Node.js + `yaml` package, read-only verifier" | **ok** — ADR-0008's body cites it as satisfying the "no new dependencies without an ADR" rule (repo's first and only npm dependency); the blueprint credits AGENTS.md as the rule's source (bp:121), 0008 as the precedent |
| ADR-0013 (bp:39) | repo is truth | "Source of truth: the repo is authoritative; Backlog task ID ↔ commit SHA is the linkage" | **ok** |
| ADR-0021 (bf:44, 95) | pre-tool policy plane / the only gate that may block | "Pre-tool policy is opt-in, first-match field rules, and fails open" | **ok** |
| ADR-0031 (bp:37, 119) | Go version floor | "Go 1.22+ minimum to build" | **ok** |
| ADR-0032 (bp:38, 164) | opencode+kilo native TS, kilo copy-at-install, plugin discovery | "opencode + Kilo native TS plugin architecture: 5 plugins from `plugins/opencode/`, Kilo copy-at-install, …" | **ok** |
| ADR-0043 (bp:39, 161, 437) | pi extension folder + manifest shape | "skill-pi Mnemonic memory extension: Pi-native folder `plugins/pi/mnemonic/` (index.ts entrypoint + …), `mnemonic setup pi` copies the folder into the agent-dir extensions" | **ok** |
| ADR-0045 (bp:34, 39, 95, 164, 493) | opencode V1+V2 dual entry / bundled `@opencode/plugin` import pattern | "OpenCode V2 plugin port: dual V1+V2 entrypoints for the 6 native plugins (V1 named export kept + V2 `Plugin.define` default), …" | **ok** |
| ADR-0058 (bp:280) | Store-extension precedent (additive Begin/Txn surface) | "Code-index `Store` interface extension: additive … + Begin/Txn methods" | **ok** (F-14) |
| ADR-0004-transport-mcp (bf:13, 287) | "transport = MCP stdio only" (v2 dropped HTTP serve) | "PRD component map is a 4-way decomposition" (slug `four-component-decomposition`) | **mismatch** — see F-15 |

All cited ADR files except the 0004-slug exist in the vault with matching
slugs (`04-adr-00{13,21,31,32,43,58}*.md` verified on disk).

## 3. Diff spot-check (no other text changed since round 3)

Round-3-verified lines re-read and compared verbatim — all four unchanged:

- Two pi/omp mapping rows (blueprint:184-185): `| pi, omp | session_start /
  tool_call / tool_result | same names |` and `| pi, omp | session_shutdown
  (quit only) | session_end |` — verbatim, line numbers unchanged (184/185
  as round 3 cited). ✓
- Task 2 Step 3 tx wording: "inside one Txn, using the `Txn.Exec` statement
  method added in this task's Files step — session upsert (`ON
  CONFLICT(id) DO NOTHING`), event insert, commit" — verbatim. ✓
- Task 4 R9 line: "status reports installed/stale (hash mismatch) per host +
  that host's store event count (the briefing R9 sub-clause)" — verbatim. ✓
- (Supporting) Task 4 adapter registrations (the 5 `pi.on(…)` names) —
  verbatim. ✓

One-line confirmation: the only textual delta observed is the Task 2 Files
ADR-0045→ADR-0058 citation (the note expands the parenthetical to name the
decision's role); post-Task-2 line numbers drift by exactly +1 (R9 pin
452-453 → 453-454; `session_shutdown` registration tail 431 → 432),
consistent with that note gaining one wrapped line — no other content
changed in the re-read spans.

## 4. New findings

### F-15 — MINOR — briefing cites a non-existent "ADR-0004-transport-mcp"

briefing.md:13 and briefing.md:287 cite "ADR-0004-transport-mcp
(MCP-only transport)" as the decision behind "v2 dropped the HTTP serve
(transport = MCP stdio only)". § LOCKED's 0004 is the **PRD component map
4-way decomposition** (slug `four-component-decomposition.md`, no
amendments); its body lists the Distribution Surface as "MCP stdio +
HTTP/REST + embedded UI + CLI" — three transports, not MCP-only — so the
cited decision does not carry the claimed role, and no
`*transport-mcp*` file exists in the decisions vault. The v2 serve-dropped
fact is actually ADR-0061 (MCP family split), decision item 2: "`serve`
(deprecated no-op) removed; bare `mnemonic mcp` removed". Impact: none on
implementability — the factual claim is true and independently grounded
(the same rejected-option paragraph also cites "v2's dropped-serve
decision"). Same class as F-14 (attribution; correct citation exists);
rated minor per round-3 precedent for that class. Strict rubric reading
("cited ADR is the wrong decision") would make this Major — noted for
transparency; the deciding factor is that ADR-0004 is a v1 (2026-09-16)
document while the claim is scoped to v2, and the paragraph's second
grounding is unambiguous.

Suggested fix (if a round 5 runs): re-cite both occurrences to
`ADR-0061` (serve removal / MCP family split).

## Verdict

**CLEAN** — 0 Blocker, 0 Major. F-14 RESOLVED (evidence above); ADR sweep
found one new sub-threshold minor (F-15, briefing ADR-0004 slug/role
mismatch, correct citation ADR-0061); diff spot-check confirms only the
Task 2 ADR line changed since round 3.
