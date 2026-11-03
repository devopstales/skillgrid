# Agent config block: the single CRITICAL_INSTRUCTION

Copyable block for `AGENTS.md` (or `CLAUDE.md`): the project's **one and only**
`<CRITICAL_INSTRUCTION>` — the behavioral nudge for Mnemonic **memory usage**,
**code-indexing usage**, and (when `ticketing.type: backlogmd`) the
**Backlog.md workflow**.

> **Only one `<CRITICAL_INSTRUCTION>` block may exist in the file. If multiple
> exist, only the LAST one is honored** — the others are dead weight. So every
> project-level critical nudge is a *section* of this single block, and no
> other region may emit a second `<CRITICAL_INSTRUCTION>` tag — including the
> `backlog` CLI's own injection (see *Single-block invariant* below).

Rendered when `mnemonic.enabled: true` in `.skillgrid/config.yaml`; the
`## Backlog.md Workflow` section is added when `ticketing.type: backlogmd`.
When mnemonic is disabled, omit the region entirely (never render an empty
block).

The nudge is behavioral: when to recall/save, the session protocol, the
orientation ladder, the known gotchas. Depth stays in the referenced files.
Never restate the full protocol here.

## The block

Copy exactly between the sentinels. Include the `## Backlog.md Workflow`
section only when `ticketing.type: backlogmd` (content: the current `backlog`
CLI nudge — refresh it from `backlog agents --update-instructions` output when
the CLI version changes). Do not add prose beyond it.

````markdown
<!-- mnemonic:start -->
<CRITICAL_INSTRUCTION>

## Mnemonic — Memory

This project uses Mnemonic persistent memory (per-project SQLite store). Memory
supplements the repo — when a memory and a committed artifact disagree, **the
committed artifact wins** (repo is the source of truth).

**Session protocol (every session):**

1. `mem_session_start` once, before first project work.
2. **Recall before you work:** `mem_context(limit: 5)` for recent session
   summaries; `mem_search(query: "<topic>")` for the topic at hand.
3. Previews are truncated (~300 chars): `mem_get_observation(id)` for the full
   body before relying on it as a constraint or acceptance criterion.
4. Before saying "done": `mem_session_summary` with all six sections (Goal /
   Instructions / Discoveries / Accomplished / Next Steps / Relevant Files),
   then `mem_session_end`.

**Save (`mem_save`) when:** a decision is made, a non-obvious finding or gotcha
is discovered, a convention is confirmed. `title` MUST equal `topic_key`;
`scope: "project"`; pass the active `session_id`. SDD artifacts use
`skillgrid/{YYYY-MM-DD-<topic>}/{briefing|blueprint|tasks|execution-progress|report|review-report}`
— upsert semantics (same key replaces in place).
**Do not save:** what is already committed — code, ADRs, specs. Git is the
source of truth; the knowledge bundle is the ADR record.

Full protocol: `skillgrid:mnemonic` skill. Save shape + session rules:
`~/.agents/skills/_shared/rules/mnemonic-memory.md`.

## Mnemonic — Code Index

The code index is maintained for this project (git root, per-project store).
Prefer it over tree-dumping; prefer index tools over `rg`/`grep` when exploring
unfamiliar areas (`rg`/`grep` remain right for exact-identifier lookups).

**Orientation ladder (this repo's mounted menu):**

1. `code_status` — freshness first (`fresh` | `lag` | `empty` | `unknown`). If
   not `fresh`, run `code_index` (incremental) before trusting any search or
   structural result. Check after clone, branch switch, or large edits.
2. `code_explore` — primary structural query: one call returns the relevant
   symbols' verbatim source, the call paths between them, and a blast-radius
   summary. Treat the source it returns as already read; do not re-verify with
   grep.
3. `code_search` (BM25) / `code_semantic_search` / `code_hybrid_search` — find
   where something is; keep result provenance (mem vs code, never silently fused).
4. `code_read` — exact slice, only for a path + line range already narrowed by
   the steps above. Never read a whole file speculatively.

**After a change:** `code_affected` answers "which tests do I run after this
change" (import + test edges, depth-capped).

Full convention (freshness semantics, intent router, CLI fallback):
`~/.agents/skills/_shared/rules/mnemonic-code-indexing.md`.

## Backlog.md Workflow

This project uses Backlog.md for task and project management.

**At the beginning of each conversation in this project, run `backlog instructions overview` before answering or taking action. Re-read it only if you have not read it yet in the current conversation.**

Use the overview to decide whether to search, read, create, or update Backlog tasks.

Before task lifecycle actions, read the matching detailed guide:
- `backlog instructions task-creation` before creating or splitting tasks
- `backlog instructions task-execution` before planning, changing status or assignee, adding a plan or implementation notes, or implementing task work
- `backlog instructions task-finalization` before checking acceptance criteria, writing final summaries, or moving tasks to terminal statuses

Use `backlog <command> --help` before running unfamiliar commands. Help shows options, fields, and examples.

Do not edit Backlog task, draft, document, decision, or milestone markdown files directly. Use the `backlog` CLI so metadata, relationships, and history stay consistent.

</CRITICAL_INSTRUCTION>
<!-- mnemonic:end -->
````

## Rendering rules

- **Placement:** after the `<!-- skillgrid:end -->` sentinel.
- **Single-block invariant:** the finished file must contain **exactly one**
  `<CRITICAL_INSTRUCTION>…</CRITICAL_INSTRUCTION>` — this block. The `backlog`
  CLI's own injection (`backlog agents --update-instructions`) appends its own
  `<!-- BACKLOG.MD GUIDELINES START/END -->` wrapper + version comment + a
  second tag at the end of the file. After running it: move its
  `## Backlog.md Workflow` content into this block as the last section, then
  delete the leftover wrapper, version comment, and second tag. If the block
  already carries a Backlog section, skip the injection and refresh that
  section from the CLI output instead. Never leave two tags in the file —
  with two, the last wins and this block dies.
- **Idempotent upsert:** search for `<!-- mnemonic:start -->`. **Found** →
  replace everything from that marker to `<!-- mnemonic:end -->` (inclusive)
  with the freshly rendered block. **Not found** → append the block at the
  placement position. Never append a second copy.
- **Menu adaptation:** the ladder names the **default narrow MCP menu**.
  Verify the session's mounted tool inventory before writing: if the project
  re-enables the full menu (`SKILLGRID_MCP_CODE_TOOLS`), extend ladder steps
  2–3 with the canonical ladder from
  `_shared/rules/mnemonic-code-indexing.md` (`code_map` /
  `code_search_symbols` / `code_outline` / `code_related`). Never name a tool
  that is not mounted — an agent sent to a missing tool is a defect.
- **Index state:** check the live state with `code_status` at render time and
  record the finding in the knowledge bundle, not in AGENTS.md. Ladder step 1
  is state-independent, so the wording holds for an empty or a fresh index.
- **One source of truth:** the block coexists with the preamble's
  `## Memory` / `## Code Indexing` reference lines and the sentinel's
  `### Memory` line. Those stay canonical per `_shared/agent-config/block.md`
  + `templates/agents-preamble.md`. The block is the prominent behavioral
  layer; it never restates the referenced protocol.
- **Region ownership:** onboarding owns the `skillgrid` sentinel region and
  this `mnemonic` region (including the folded-in Backlog section). The
  `BACKLOG.MD GUIDELINES` wrapper is the `backlog` CLI's — it exists only
  transiently, between its injection and the fold-in step.
