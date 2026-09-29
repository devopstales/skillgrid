# Design: llmwiki-okf-wiki — OKF v0.2 `.wiki/` Compiler (Pillar 1)

> **STATUS:** `draft` (2026-09-29)
> **Spec:** `.skillgrid/specs/2026-09-29-llmwiki-okf-wiki/spec.md`

## Hypothesis

A pure-Go compiler that reads `.skillgrid/` + `web_cache` + `raw/` and emits an OKF v0.2 / Obsidian / llmwiki-triple-conformant `.wiki/` is **deterministic and dependency-free** (markdown + YAML frontmatter only, both already in the Go stdlib/stack), and **no-churn** on unchanged sources.

- **Right when:** `go test ./...` passes and two consecutive `wiki compile` runs over a fixture produce a byte-identical `.wiki/` (empty `git diff`).
- **Wrong when:** the emitter needs a third-party YAML/markdown dep (violates the no-new-deps lock), OR a no-op recompile rewrites `generated.at` (churn), OR Obsidian/llmwiki can't resolve the emitted `[[wikilinks]]`/`source_path`.

## Thinnest MVP + door check

**Thinnest slice (lands the capability end-to-end):** OKF v0.2 emitter + conformance gate + ASSUMPTIONS.md ADR adapter + `state.yaml` State adapter + `wiki compile` CLI + `index.md`/`log.md`/`AGENTS.md` + determinism test. That alone produces a valid `.wiki/` a human can open in Obsidian.

**Door check (before expanding adapters):** run the thinnest slice against the real repo. If ADRs + State render correctly in Obsidian with working `[[ADR-0006]]` links, continue with the remaining adapters (Terms, Specs, Spikes, Constraints, Architecture, web_cache, raw). If the emitter fights the YAML frontmatter or links don't resolve, stop and fix the emitter before adding sources.

## Architecture

```
cmd/skillgrid/wiki.go                (cobra: wiki compile | wiki lint)
        │  openMemService pattern (service.New + ResolveProject)
        ▼
internal/mnemonic/wiki/
  concept.go       Concept, SourceRef, Verifier, Edge types; NewConcept helpers
  emitter.go       Emit(concept)→string ; stable YAML frontmatter writer (hand-rolled, no dep)
  conform.go       Conform(page)→[]error  (OKF v0.2 gate)
  skillgrid.go     pillar-1 adapter: ParseSkillgrid(dir)→[]Concept  (ASSUMPTIONS/terms/state/specs/spikes/constraints/architecture)
  webcache.go      ParseWebCache(rows)→[]Concept  (Finding; cited→wiki/, uncited→research/)
  raw.go           ParseRaw(root)→[]Concept  (Finding/Source; never writes raw/)
  slug.go          Slugify(title)→string ; TypeDir(type)→string  (adr/, concepts/, entities/, research/)
  compile.go       Compile(CompileInput)→CompileResult  (assemble, sort, content-hash gate, write .wiki/)
  index.go         RenderIndex(concepts)→string ; AppendLog ; RenderAgents()→string
  wiki_test.go     + per-file _test.go + compile_e2e_test.go (fixture .skillgrid/ + seeded web_cache)
```

**Dependencies of `internal/mnemonic/wiki`:** stdlib only (`os`, `path/filepath`, `strings`, `time`, `bytes`, `hash/fnv` or `crypto/sha256`). It consumes `*webcache.Service` rows via the `service` facade in the CLI (the package itself takes a plain `[]WebRow` slice so it stays pure + testable — the CLI does the `.Web().Get/Search` and hands it rows).

**Why a hand-rolled YAML frontmatter writer:** the only new deps lock is "no new deps without an ADR." We already emit YAML-shaped text (frontmatter) and the field set is fixed and small (`type`, `title`, `description`, `resource`, `tags`, `sources`, `generated`, `verified`, `status`, `stale_after`, `source_path`). A ~120-line deterministic writer (sorted keys, no nulls, absolute-UTC timestamps) avoids pulling a YAML emitter and keeps output byte-stable. Reading user `raw/` frontmatter (R6) MAY use a minimal `---`-block line parser (key: value + `key:` lists) — no full YAML parser needed; unknown keys are preserved by re-emitting the raw block.

## Output layout

```
.wiki/
├── AGENTS.md            ← schema layer (type taxonomy, slug rule, lint rule) — regenerated
├── raw/                 ← USER-DROPPED only; compiler NEVER writes here
│   └── …(unchanged)…
└── wiki/
    ├── adr/             type: ADR
    ├── concepts/        type: Term, Constraint
    ├── entities/        type: Spec, Spike, State, Architecture, Finding (cited)
    ├── research/        type: Finding (uncited web_cache, status: draft)
    ├── index.md         ← catalog grouped by type (llmwiki reads for search)
    └── log.md           ← append-only compile history (one line per content-changing compile)
```

`TypeDir` map: `ADR→adr`, `Term|Constraint→concepts`, `Spec|Spike|State|Architecture|Finding→entities` (cited findings), uncited web findings→`research`. Slug = `Slugify(title)` (lowercase, alnum+`-`, trim). Concept ID = file path minus `.md` (OKF §2).

## Edge model (Pillar 1 = EXTRACTED only)

`Edge{From, To string, Kind string, Confidence "EXTRACTED"}`. Sources of edges:
- ADR→ADR via `Supersedes`/`Amends` in the in-force table / ADR body.
- State→Spec via `state.yaml.pipeline.current_change` → matching `specs/<change>/`.
- Finding→Spec/ADR via a page citing the web `url` in its `sources`.
- Term→Term via explicit "see [[…]]" in the glossary body.
Edges are rendered as `[[slug]]` in the relevant page body AND recorded so `index.md`/`log.md` can reference them. **No LLM pass in Pillar 1** — `--no-llm` is accepted and ignored (follow-up adds INFERRED edges + `wiki_inferred_edges` persistence).

### Link syntax: `[[wikilinks]]` (deliberate extension, not OKF-canonical)

Karpathy's pattern and the whole community (nashsu/llm_wiki, dsiu/okf-llm-wiki) use Obsidian-style `[[wikilinks]]`; the **OKF SPEC §6 standard uses plain markdown links** (`/tables/customers.md` or relative). We emit `[[slug]]` / `[[slug|alias]]` because `.wiki/` must open natively in Obsidian and microsoft/llmwiki — both traverse `[[wikilinks]]` — and `source_path` is the backlink target. This is a documented **producer profile/extension** of OKF (exactly what dsiu does), permitted by §4.1 (producers MAY add to the interoperable surface). **Consequence:** `wiki lint` MUST NOT require markdown links and MUST NOT fail a page for using `[[wikilinks]]`; the conformance gate checks the reserved-frontmatter surface only, never link syntax. External URLs (not bundle concepts) still use standard markdown links.

## Determinism (the no-churn key)

- Stable sort: concepts by `(TypeDir, Slug)`; edges by `(From, To, Kind)`.
- `generated.at`: the compiler keeps a **prior-manifest** (`.wiki/.wiki-manifest.json`, gitignored) mapping concept-id → content-hash + `generated.at`. On recompile, if a concept's content-hash is unchanged, reuse its stored `generated.at`; otherwise set to the compile clock. The manifest itself is NOT under `wiki/` and is not part of the linted bundle.
- Content-hash gate: a page file is rewritten only if its new bytes ≠ current bytes (sha256). `raw/` is never hashed-for-write (only read).
- Clock: `CompileInput.Now time.Time` (injected; default `time.Now().UTC()`). Tests pin `Now`.
- `log.md`: append a line only when the set of content-hashes changed vs the prior manifest (no-op compile → no log line, no file touch).

## Threat matrix

| # | Threat (trust boundary) | Applicable? | Mitigation / covering scenario |
|---|---|---|---|
| T1 | Malicious/odd `.skillgrid/` markdown (untrusted input to the parser) | Yes — parsers read arbitrary markdown. | Tolerant parsing; never `exec`, never unmarshal arbitrary YAML into structs with side effects. Scenario 4.5 (malformed section → warning, not panic). |
| T2 | `raw/` files with frontmatter injection (user-dropped → could carry a `source_path` pointing outside the bundle) | Yes — user-dropped markdown is semi-trusted. | `source_path` is validated to be a bundle-relative path under `raw/` (reject absolute / `..` traversal). Scenario 6.1. |
| T3 | Path traversal / write escape (compiler writes only `.wiki/wiki/` + `AGENTS.md`, never `raw/`, never outside `.wiki/`) | Yes — file I/O. | All write paths computed under the `--out` root; `filepath.Clean` + prefix check before write. Scenario 6.3, 6.4. |
| T4 | Non-deterministic output (clock, map iteration) → git churn | Yes — correctness of the no-churn guarantee. | Injected clock; sorted iteration (no range-over-map for output). Scenario 8.1, 8.2, 8.3. |
| T5 | Large `web_cache` (thousands of rows) → slow/bloat compile | Low — project-scoped, TTL-bounded. | Read only fresh + 4 sources; `research/` caps per-source listing in `index.md` (full rows still emitted). No hard block (advisory). |
| T6 | Stop-and-ask: does the compiler ever write outside the repo root? | Assumption. | Default `--out` = `.wiki` at the project root (resolved from CWD). `--out` is validated to be within the project root. No network, no exec. |

**Stop-and-ask security assumptions:** the compiler is **local, read-mostly, offline**. Its only side effect is writing files under `.wiki/` (and the gitignored `.wiki/.wiki-manifest.json`). It reads `web_cache` from the local SQLite (no network) and `.skillgrid/` + `raw/` from the filesystem. No secrets are read or emitted; `sources[].author` is an actor string, not a credential.

## Rollback

`wiki compile` is additive (new package + new command + a `.wiki/` dir). Rollback = remove the `.wiki/` dir (git `git checkout -- .` or `rm -rf .wiki`), `git revert` the two commits (spec zone, then code zone). No migration, no schema change to existing Mnemonic tables (Pillar 1 adds no store schema — the prior-manifest is a gitignored file). Follow-up INFERRED-edge persistence (R in a later change) WILL add a `wiki_inferred_edges` table.

## Success criteria (measurable)

1. `go test ./...` green (new package ≥ ~80% of its own lines; full suite green).
2. No-op recompile → `git diff --exit-code .wiki` returns 0 (byte-identical).
3. `skillgrid wiki compile` on the real repo produces `.wiki/` that (a) passes `skillgrid wiki lint`, (b) opens in Obsidian with resolvable `[[links]]`, (c) is accepted by microsoft/llmwiki (`.wiki/{raw,wiki,index.md,log.md,AGENTS.md}` present).
4. Zero new entries in `go.mod` (`go mod tidy` is a no-op).
