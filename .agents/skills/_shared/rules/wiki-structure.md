# Wiki Structure (`.wiki/` bundle layout + conformance rules)

Single source of truth for the `.wiki/` output tree produced by `skillgrid wiki compile`. If a skill, adapter, or emitter disagrees with this file, **this file wins**.

## What it is

A **project-scoped, compiler-generated** knowledge bundle that is simultaneously:
- **OKF v0.2** conformant (valid frontmatter, valid `type` values, absolute-UTC timestamps)
- **Obsidian**-native (resolves `[[wikilinks]]`, opens as a vault)
- **llmwiki**-shaped (`.wiki/{raw,wiki,index.md,log.md,AGENTS.md}` layout)

The compiler (`internal/mnemonic/wiki/`) is a **pure function** of (`.skillgrid/` contents, fresh `web_cache` rows, `raw/` contents, compile clock). It is read-only over all inputs and write-only into `.wiki/`.

## Directory layout

```
.wiki/
├── AGENTS.md            ← schema layer (type taxonomy, slug rule, lint rule) — regenerated every compile
├── raw/                 ← USER-DROPPED only; compiler NEVER writes here
│   └── …(unchanged)…
├── wiki/
│   ├── adr/             type: ADR
│   ├── concepts/        type: Term, Constraint
│   ├── entities/        type: Spec, Spike, State, Architecture, Finding (cited)
│   ├── research/        type: Finding (uncited web_cache, status: draft)
│   ├── index.md         ← catalog grouped by type (llmwiki reads for progressive disclosure)
│   └── log.md           ← append-only compile history (one line per content-changing compile)
└── .wiki-manifest.json  ← gitignored; content-hash → generated.at map (determinism anchor)
```

`TypeDir` mapping: `ADR→adr`, `Term|Constraint→concepts`, `Spec|Spike|State|Architecture|Finding→entities` (cited findings), uncited web findings→`research`.

## Page format

Every page is a UTF-8 markdown document with a YAML frontmatter block delimited by `---`:

```yaml
---
type: ADR                      # REQUIRED — the only always-required OKF key
title: "Vector search: …"
tags: [adr]
generated:
  by: skillgrid-wiki
  at: 2026-09-29T19:02:08Z     # absolute UTC ISO 8601
status: stable                 # draft | stable | deprecated
source_path: .skillgrid/ASSUMPTIONS.md   # producer extension (OKF §4.1)
---

<markdown body>
```

### Frontmatter rules

| Key | Required | Type | Notes |
|-----|----------|------|-------|
| `type` | **yes** | string | One of the taxonomy below |
| `title` | yes | string | Human-readable, quoted if it contains `:` |
| `tags` | no | `[string]` | Omit (not `null`) when absent |
| `generated.by` | yes | string | Producer agent (always `skillgrid-wiki`) |
| `generated.at` | yes | datetime | Absolute UTC ISO 8601 (`…T…Z`) |
| `status` | no | string | `draft` \| `stable` \| `deprecated` |
| `source_path` | no | string | Bundle-relative path (backlink target) |
| `sources` | no | `[object]` | Each entry MUST have `resource` (non-empty) |
| `sources[].resource` | **yes** (per entry) | string | URL or path |
| `sources[].author` | no | string | Actor identifier |
| `sources[].last_modified` | no | datetime | Absolute UTC ISO 8601 |
| `verified` | no | `[object]` | `{ by: <actor>, at: <datetime> }` |
| `stale_after` | no | datetime | Absolute UTC ISO 8601 (not a relative TTL) |
| `description` | no | string | ≤ 200 chars |
| `resource` | no | string | Primary resource URL |

**Absent optional keys MUST NOT be serialized as `null`** (OKF §5 — absence is meaningful).

### Type taxonomy

| Type | Dir | Source | Status |
|------|-----|--------|--------|
| `ADR` | `adr/` | `ASSUMPTIONS.md` `### ADR-NNNN` | in-force→`stable`, superseded→`deprecated` |
| `Constraint` | `concepts/` | `ASSUMPTIONS.md` `### Locked constraints` | always `stable`, no `stale_after` |
| `Term` | `concepts/` | `artifacts/01-business-terms.md`, `02-technical-terms.md` | `stable` |
| `State` | `entities/` | `state.yaml` | `draft` |
| `Spec` | `entities/` | `specs/<change>/` (one per change dir) | from `briefing.md` STATUS line |
| `Spike` | `entities/` | `spikes/<NNN-name>/` (one per spike dir) | from verdict (VALIDATED/INVALIDATED/PARTIAL) |
| `Architecture` | `entities/` | `ARCHITECTURE.md` (existence-gated) | `stable` |
| `Finding` (cited) | `entities/` | `web_cache` (fresh, cited by another page) | `stable` |
| `Finding` (uncited) | `research/` | `web_cache` (fresh, not cited) | `draft` |
| `Source` | `entities/` | `raw/**/*.md` without OKF frontmatter | `draft` |
| `Finding` (raw) | `entities/` | `raw/**/*.md` with OKF frontmatter `type: Finding` | from frontmatter |

## Link syntax

- **Internal links** use Obsidian `[[wikilinks]]` — `[[adr-0006]]`, `[[term-name]]`. This is a **documented producer extension** of OKF (permitted by §4.1). `wiki lint` MUST NOT require markdown links and MUST NOT fail a page for using `[[wikilinks]]`.
- **External URLs** (not bundle concepts) use standard markdown links: `[title](https://…)`.
- ADR Supersedes/Amends relationships render as `[[adr-NNNN]]` in **both** endpoints' bodies (bidirectional).

## Determinism rules

1. **Stable sort:** concepts by `(TypeDir, Slug)`; edges by `(From, To, Kind)`. Never by map iteration or insertion order.
2. **Content-hash gate:** a page file is rewritten only if its new sha256 ≠ current sha256. The hash computation **strips both `generated.at` and `stale_after`** from the content before hashing (time-derived fields must not cause churn).
3. **`generated.at` preservation:** if a concept's content-hash is unchanged, reuse the prior `generated.at` from `.wiki-manifest.json`. Only changed concepts get the new compile clock.
4. **No-op recompile:** two consecutive `wiki compile` runs over unchanged sources produce a **byte-identical** `.wiki/` (empty `git diff`).
5. **`log.md`:** append a line only when the set of content-hashes changed vs the prior manifest. No-op compile → no log line, no file touch.
6. **Injected clock:** `CompileInput.Now time.Time` (tests pin it; default `time.Now().UTC()`).

## Source adapters

| Adapter | Input | Output concepts | Tolerant? |
|---------|-------|-----------------|-----------|
| `ParseAssumptions` | `.skillgrid/ASSUMPTIONS.md` | ADR + Constraint + edges (Supersedes/Amends) | missing section → no concepts, no error |
| `ParseState` | `.skillgrid/state.yaml` | State (1) + warnings (unknown keys) | missing file → nil, no error |
| `ParseTerms` | `artifacts/01/02-*-terms.md` | Term + edges (`[[…]]` cross-refs) | missing file → no concepts, no error |
| `ParseSpecs` | `.skillgrid/specs/<change>/` | Spec (1 per change dir) | empty dir → no concepts, no error |
| `ParseSpikes` | `.skillgrid/spikes/<NNN-name>/` | Spike (1 per spike dir) | empty dir → no concepts, no error |
| `ParseArchitecture` | `.skillgrid/ARCHITECTURE.md` | Architecture (1) | missing file → nil, no error |
| `ParseWebCache` | `[]WebRow` (from CLI) | Finding (cited→entities, uncited→research) | empty slice → no concepts, no error |
| `ParseRaw` | `.skillgrid/raw/**/*.md` | Finding / Source | missing dir → no concepts, no error |

**The wiki package is pure** — no `database/sql`, no `os.Open` on the SQLite DB. The CLI (`cmd/skillgrid/wiki.go`) does the DB read (`loadFreshWebRows`) and passes `[]WebRow` to the pure adapter.

## Conformance gate (`wiki lint`)

`wiki.Conform(page)` checks, at minimum:
1. `type` is present and non-empty
2. Every `sources[]` entry has a non-empty `resource`
3. Every timestamp parses as absolute UTC ISO 8601
4. `status` (if present) is one of `draft|stable|deprecated`

`wiki lint` walks `.wiki/wiki/**/*.md`, runs `Conform` on each, prints errors, exits non-zero on any failure. A clean bundle exits 0.

## Write boundaries

- **Writes:** `.wiki/wiki/**/*.md`, `.wiki/wiki/index.md`, `.wiki/wiki/log.md`, `.wiki/AGENTS.md`, `.wiki/.wiki-manifest.json`
- **Never writes:** `.wiki/raw/` (user-dropped, read-only), anything outside `.wiki/`
- **Reads:** `.skillgrid/` (all pillar-1 sources), `raw/` (recursive walk), `~/.skillgrid/mnemonic/web_cache.db` (via CLI)
- **No network, no exec, no secrets in output.**

## CLI

```
skillgrid wiki compile [--project DIR] [--dir DIR] [--out DIR] [--no-llm] [--json]
skillgrid wiki lint
skillgrid wiki          # no subcommand → print usage, exit 0
```

- `compile`: reads all sources, assembles concepts, writes `.wiki/`. Flags: `--project` (default `.`), `--dir` (default `.skillgrid`), `--out` (default `.wiki`), `--no-llm` (no-op in Pillar 1), `--json` (machine-readable result).
- `lint`: walks `.wiki/wiki/**/*.md`, runs conformance gate, non-zero exit on failure.
- Project + data dir resolution reuses the existing `openMemService` pattern.
