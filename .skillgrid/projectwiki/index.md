# Project Index

Catalog of `.skillgrid/projectwiki/`. Every page gets a row here.

## Root

| Page | Summary |
|---|---|
| `ASSUMPTIONS.md` | Durable project understanding — VERIFIED / INFERRED / LOCKED / Open Questions |
| `index.md` | This catalog |
| `log.md` | Append-only timeline (newest last) |

## decisions/

| Page | Summary |
|---|---|
| `04-adr-0001-adopt-intent-driven-template-patterns.md` | Adopt the `.agents/` shared layout, the adversarial-authoring pattern, and the minimal grill-me skill from intent-driven-template (Not-adopted list: multi-CLI parity, spec-as-source, c4-diagrams, etc.) |
| `04-adr-0002-git-discipline-as-separate-skill.md` | Adopt a `git-discipline` skill as a separate first-class skill (phase-transition git state gates G1–G10), not folded into `_shared/work-unit-commits` |
| `04-adr-0003-per-project-sqlite-file.md` | mnemonic: one portable `.sqlite` per project under `~/.mnemonic/` (crash isolation, `cp`-able brain) |
| `04-adr-0004-vectors-in-file.md` | mnemonic: vectors inside the SQLite file — BLOB source of truth + vec0 (pure-Go, spike-gated) + brute-force floor; no external vector DB |
| `04-adr-0005-c-cgo-free-target-build.md` | mnemonic: C/CGo-free target build (pure-Go ONNX, spike-gated); CGo onnxruntime is the pre-decided fallback |
| `04-adr-0006-local-onnx-default-embedder.md` | mnemonic: local ONNX CPU model is the default embedder tier; API (Ollama-compatible) opt-in; hash-64 floor |
| `04-adr-0007-single-handler-two-adapters.md` | mnemonic: CLI + MCP over one shared Handler, byte-identical envelopes (K6 parity) |
| `04-adr-0008-memory-core-only-v1.md` | mnemonic: v1 = memory core only — code/web tools, serve, projectwiki, team_*, LLM curator, importer explicitly deferred |
| `04-adr-0009-wal-only-concurrency.md` | mnemonic: WAL + busy_timeout only — no leases, no write daemon in v1 |
| `04-adr-0010-multi-tool-binary.md` | mnemonic: `mnemonic <tool> <verb>` multi-tool binary, per-tool MCP servers, top-level init/doctor |
| `04-adr-0011-degradation-contract.md` | mnemonic: `degraded: true` never silent; floors never block save/search |
| `0013-pure-go-treesitter.md` | mnemonic code tool: tree-sitter in pure Go (`odvcencio/gotreesitter`), not a CGo binding — ADR-0005's C/CGo-free target holds for the whole binary |
| `0014-unified-embedding-space.md` | mnemonic code tool: one embedder model + dim per project — code and memory vectors comparable; no cross-domain fusion in v1 |

## invariants/

_(none yet)_

## architecture/

| Page | Summary |
|---|---|
| `mnemonic-memory-core.md` | mnemonic memory-core v1 (planned, draft): layering, package layout, data model, recall pipeline |

## gotchas/

| Page | Summary |
|---|---|
| `fts5-external-content-upsert.md` | FTS5 external-content triggers desync on the `topic_key` upsert/revision (UPDATE) path — cover INSERT/UPDATE/DELETE, regression-test it |

## entities/

| Page | Summary |
|---|---|
| `pi-worktree-plugin.md` | The `plugins/pi/pi-worktree` Pi plugin — scope, requirements R1–R13, build location |
| `mnemonic.md` | The new local-first memory binary in this repo (`mnemonic/`) — v1 scope, CLI shape, locked stack, design hub |

## concepts/

| Page | Summary |
|---|---|
| `scratch-checkpoints.md` | `refs/skillgrid/` turn-level snapshots: capture, dedup, restore, prune, atomic pair |
| `worktree-standard.md` | The `_shared/work-unit-commits` worktree contract: tiers, path template, merge defaults, removal safety |
| `in-file-vector-layer.md` | The "vector database" is three layers inside the SQLite file: BLOB source of truth + vec0 ANN + brute-force floor; dim-strict |
| `single-handler-two-adapters.md` | K6 seam: CLI + MCP over one shared Handler; byte-identical envelopes; parity-tested |
| `degradation-floor.md` | Every capability has a deterministic dependency-free floor; `degraded: true` never silent, never blocking |

## syntheses/

| Page | Summary |
|---|---|
| `mnemonic-core.md` | Design hub for the mnemonic-core change: read-order, provenance, shootdown risks, open spike-gated items |

## sources/

| Page | Summary |
|---|---|
| `skillgrid-mnemonic-v2.md` | Predecessor study: `../skillgrid/mnemonic` — inherited model/seams, cut scope creep |
| `engram.md` | Predecessor study: `../engram` — single-binary product shape; cloud/daemon/TUI dropped |

## preferences/

| Page | Summary |
|---|---|
| `build-preferences.md` | Programmer defaults: local-first offline, C/CGo-free target, local model over daemon, multi-tool CLI, file = portability unit |

## glossary/

| Page | Summary |
|---|---|
| `01-business-terms.md` | People + Work vocabulary (programmer builds, user runs) |
| `02-technical-terms.md` | mnemonic term cluster: observation, session, topic_key, scope, floor, degraded, RRF, vec0, K6 seam, trail |

## raw/

_(none yet)_
