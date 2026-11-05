---
name: kb
description: Operate the project's knowledge-base engine (vault, index, query, validate, lint, status, cite, eval) through the SDD lifecycle.
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: openwiki:openwiki-integration + repocodex-agent-playbook
---

# Knowledge Base (`kb`)

Manage the project's durable knowledge base: an OKF bundle at `.skillgrid/` plus the Mnemonic retrieval index over it, served by the `kb` engine (`mnemonic-v2`).

## Modes
- **Read:** retrieve decisions/invariants/findings when they bear on the task; stop once grounded.
- **Maintain:** author/revise concepts inside the SDD lifecycle (briefing → blueprint → tasks → gate → execution).

## Commands
- `kb bootstrap` — scaffold vault if absent; no-KB detection.
- `kb index` / `kb reindex` — full/incremental derived FTS rebuild.
- `kb query` — stage-1 ranked links + snippets; stage-2 `kb_read` hydrates.
- `kb validate` — pin-check (fresh/stale/unresolved) + claims.
- `kb lint` — deterministic health + safe fix.
- `kb status` — index freshness + run state.
- `kb cite-source` — external-source citation + section granularity.
- `kb eval` — retrieval regression net.

## Rules
- Concepts live in `decisions/`/`invariants/`/`workflows/`/`glossary/`; one concept per why.
- Same-edit rule: concept + `index.md` + `log.md` + `reverse-index.md` in one commit.
- Decisions go through `skillgrid:architectural-decision-records`.
