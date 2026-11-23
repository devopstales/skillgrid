---
type: Synthesis
title: "mnemonic-core — design hub"
status: draft
sources:
  - ".skillgrid/change-specs/2026-11-23-mnemonic-core/brainstorm.md"
  - ".skillgrid/change-specs/2026-11-23-mnemonic-core/briefing.md"
---

# mnemonic-core — design hub

The hub for the [[entities/mnemonic]] design (change
`2026-11-23-mnemonic-core`, briefing status draft). A new local-first,
SQLite-based memory for AI agents in this repo (`skillgrid-skills/mnemonic/`):
one binary, one `.sqlite` per project, CLI + MCP over one Handler, vectors
inside the file, local ONNX as the default embedder, C/CGo-free target build.

## Read this first, in order

1. **The plan:** `.skillgrid/change-specs/2026-11-23-mnemonic-core/briefing.md`
   — requirements R1–R9, 9 locked ADRs, phasing (v0.1 core → v0.2 hybrid →
   v1.0), 9 success criteria, shootdown findings, open questions.
2. **The locked decisions:** ADR-0003…0011 in `decisions/` (per-project file,
   vectors-in-file, C/CGo-free target, local ONNX default, K6 seam,
   memory-core-only v1, WAL-only concurrency, multi-tool binary, degradation
   contract).
3. **The reusable design concepts:** [[concepts/in-file-vector-layer]] (BLOB +
   vec0 + brute force), [[concepts/single-handler-two-adapters]] (K6),
   [[concepts/degradation-floor]] (never silent, never blocking).
4. **The plan's shape:** [[architecture/mnemonic-memory-core]] (draft module
   map + data model + recall pipeline).

## Where the design came from

- **Data model + seams** inherited from [[sources/skillgrid-mnemonic-v2]] —
  observations/topic_key/FTS5/trails, dim-strict vectors, hash-64 floor, K6
  seam, project detection, exit codes.
- **Product shape** from [[sources/engram]] — single binary, agent-agnostic,
  doctor, per-project stores; cloud/daemon/TUI deliberately dropped.
- **Precedent for the floor discipline** is the installed
  `skillgrid:mnemonic` skill's degradation contract and this repo's
  projectwiki engine (same K6 seam).

## Known risks (from the briefing shootdown)

- Memory-only v1 leaves the installed `mnemonic` skill's code-ladder steps
  dangling → the skill flip ships with the deferred `code` tool, not v1.0.
- vec0 at 10k observations may be over-engineering → kept as a hedge for the
  future `code` tool; the floor makes a failed spike a non-event.
- Pure-Go ONNX is ~10× slower than the C core → per-save tolerable, bulk
  re-embed is a background job; spike owns the escape hatch.
- FTS5 external-content + topic_key upsert = trigger desync trap →
  [[gotchas/fts5-external-content-upsert]].

## Open (spike-gated, not decisions)

- Exact ONNX model + whether `unum-cloud/onnx` runs it (opset + latency).
- Go module path (assumed `github.com/devopstales/skillgrid-skills/mnemonic`).
- Model distribution (default: download-on-init + cache).
- Per-tool vs unified MCP servers for future tools (default: per-tool).

## Related

- Supersedes: none.
- Consumed by: [[entities/mnemonic]] (the durable entity view).
- Depends on: [[architecture/mnemonic-memory-core]],
  [[preferences/build-preferences]], [[glossary/02-technical-terms]].
