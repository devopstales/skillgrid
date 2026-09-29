---
name: codebase-inspection
description: "Measure a codebase with pygount: LOC, language breakdown, file counts, and code-vs-comment ratios. Use when sizing a repo, enrolling a brownfield project, or answering how big something is."
license: MIT
metadata:
  author: Hermes Agent (ported; see ADR-0015)
  version: "1.0"
  part-of: skillgrid
---

# Codebase Inspection

**Announce at start:** "I'm using the skillgrid:codebase-inspection skill to measure this codebase."

## Overview

Answers "how big is this repo and what is it made of" with measured numbers — lines of code, language breakdown, file counts, code-vs-comment ratios — using `pygount`. This is the quantitative complement to reading code: run it first when enrolling a brownfield project (`skillgrid:onboarding` Step 1.5), before slicing work on unfamiliar code, or whenever someone asks about codebase size or composition.

Ported from the Hermes `codebase-inspection` skill under MIT (see ADR-0015). Behavioral delta from upstream: scope selection prefers the repo's own ignore files where noted; findings feed skillgrid enrollment (`existing` capabilities) instead of a chat answer.

## When to Use

- User asks for a LOC count, language breakdown, or code-vs-comment ratios
- Enrolling an existing codebase (`skillgrid:onboarding` Step 1.5) — measure before planning on top of reality
- Sizing unfamiliar code before `skillgrid:slicing` (line counts calibrate the 400-line budget risk)
- Comparing composition before/after a large change

**When NOT to use:** For duplication measurement — use `skillgrid:jscpd` / `skillgrid:dry-refactoring`. For symbol-level orientation (callers, blast radius) — use `skillgrid:mnemonic` code index. For a quick single-file count — `wc -l` is faster than installing anything.

## Prerequisites

`pygount` on PATH (pip package; approved by ADR-0015, installed on demand, never vendored):

```bash
pip install --break-system-packages pygount 2>/dev/null || pip install pygount
```

If installation fails (locked-down machine), fall back to `tokei`/`scc` if present, else `wc -l` per language via `git ls-files`, and say which fallback produced the numbers.

## The Process

### 1. Basic summary (most common)

```bash
pygount --format=summary \
  --folders-to-skip=".git,node_modules,venv,.venv,__pycache__,.cache,dist,build,.next,.tox,.eggs,*.egg-info" \
  .
```

**Always pass `--folders-to-skip`.** Without it pygount crawls dependency trees and may take minutes or hang. Adjust per stack:

```bash
# Python projects
--folders-to-skip=".git,venv,.venv,__pycache__,.cache,dist,build,.tox,.eggs,.mypy_cache"

# JavaScript/TypeScript projects
--folders-to-skip=".git,node_modules,dist,build,.next,.cache,.turbo,coverage"

# Go projects
--folders-to-skip=".git,vendor,dist,build,bin"

# General catch-all
--folders-to-skip=".git,node_modules,venv,.venv,__pycache__,.cache,dist,build,.next,.tox,vendor,third_party"
```

### 2. Narrow or deepen

```bash
# One language only
pygount --suffix=py --format=summary .

# Largest files first (find where the complexity lives)
pygount --folders-to-skip=".git,node_modules,venv" . | sort -t$'\t' -k1 -nr | head -20

# Machine-readable (feed enrollment notes or a report)
pygount --format=json .
```

### 3. Report

Record, don't just chat: total LOC, top 3 languages with file counts, code-vs-comment ratio, the 5 largest files, and which tool produced the numbers (pygount or fallback). When enrolling, file this under the `existing` capabilities record so `skillgrid:writing-blueprints` plans against measured reality.

## Interpreting results

Summary columns: **Language**, **Files**, **Code** (executable/declarative lines), **Comment**, **%**. Pseudo-languages: `__empty__`, `__binary__`, `__generated__` (heuristic), `__duplicate__`, `__unknown__`. Known quirks: Markdown counts as 0 code (all comment — expected); JSON counts conservatively (use `wc -l` for JSON); for very large monorepos prefer `--suffix` over full scans.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "I can tell the size by looking at the tree" | Directory listings don't give LOC, ratios, or language share. Measure — one command. |
| "Skip the exclusions, it's a small repo" | `node_modules` or `vendor` hides in small repos too. Always pass `--folders-to-skip`. |
| "The exact number doesn't matter" | For enrollment and slicing calibration it does — the 400-line budget risk reads differently at 5K vs 500K LOC. |
| "I'll measure after planning" | Measurement calibrates the plan. Measure first, slice second. |

## Red Flags

- Running pygount with no `--folders-to-skip` (hang risk on dependency trees)
- Quoting Markdown "code lines" as code (pygount counts them as comments by design)
- Using these numbers as quality judgments (LOC measures size, not health — pair with `skillgrid:jscpd` for that)
- Installing pygount into a project venv instead of user scope (it's a measuring tool, not a project dependency)

## Verification

- [ ] Numbers came from a stated tool (pygount or named fallback), not estimates
- [ ] `--folders-to-skip` covered the repo's dependency/build dirs (no `node_modules`/`vendor` rows in the table)
- [ ] Report holds total LOC, top-3 languages, code-vs-comment ratio, 5 largest files
- [ ] When enrolling: figures recorded in the `existing` capabilities record, not just chat
