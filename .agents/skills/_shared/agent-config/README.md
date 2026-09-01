# Agent config templates (shared)

`block.md` holds the canonical `## Agent skills` payload; this family decides *which root config file* to write it into and *how many copies* are allowed. The payload lives in ONE place ([block.md](block.md)) so AGENTS.md, CLAUDE.md, and GEMINI.md never drift.

## Targets

| target file | convention | notes |
|---|---|---|
| [AGENTS.md](agents.md) | cross-agent standard (open standard) | preferred primary when present |
| [CLAUDE.md](claude.md) | Claude Code surface | preferred if the project is Claude-first and has no AGENTS.md |
| [GEMINI.md](gemini.md) | Google Gemini CLI surface | preferred if the project is Gemini-first and has no AGENTS.md |

## Decision matrix — which file gets the full block

Pick exactly one **primary** target (the one that receives the full block from [block.md](block.md)):

1. `AGENTS.md` exists → **AGENTS.md**.
2. else `CLAUDE.md` exists → **CLAUDE.md**.
3. else `GEMINI.md` exists → **GEMINI.md**.
4. else none exist → **ask the user** which to create (sdd-init Hard Rule: never silently pick a platform). Default suggestion to the user: `AGENTS.md` (cross-agent).

## Multi-platform repos

When more than one of the three files exists, write the **full block to the primary** (step 1–3) and a **one-line pointer** to the full block's location in the other files, so every platform agent sees a consistent instruction and only one copy can drift:

```markdown
<!-- skillgrid-sdd:start (secondary) -->
SDD config lives here: `AGENTS.md` → `## Agent skills`. Do not duplicate that block.
<!-- skillgrid-sdd:end (secondary) -->
```

## Shared rules (all targets)

- Idempotent upsert with `<!-- skillgrid-sdd:start/end -->` sentinels — defined in [block.md](block.md#idempotent-upsert-required). Never append a second block.
- Never create a second root config file when one already exists (sdd-init Hard Rule).
- Keep the block tight — it loads in context in Claude Code / Gemini CLI on every run. No prose beyond [block.md](block.md).
- The block is plain markdown; identical across AGENTS.md / CLAUDE.md / GEMINI.md. Only the file name and the optional one-line platform note differ.
