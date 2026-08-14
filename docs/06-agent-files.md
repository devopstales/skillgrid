# Agent instruction files

On **project**-scope `aiskillgrid install`, generate (or refresh) shared agent instruction files at the project root so clients that read them pick up Skillgrid conventions.

## Files

| File | Typical consumers |
|------|-------------------|
| `AGENT.md` | Generic / multi-agent instruction entry (portable) |
| `CLAUDE.md` | Claude Code and compatible tools |
| `GEMINI.md` | Gemini CLI and compatible tools |

Templates live in the hub under `packs/instructions/` (e.g. [`skillgrid-block.md`](../packs/instructions/skillgrid-block.md)). Content should point agents at Skillgrid-installed skills, agreed tools (Engram, GitNexus), and the composition rules in [05-skills.md](05-skills.md):

- **Plans / specs / tasks:** Superpowers only (`docs/superpowers/`, `.superpowers/`). Do not use OpenSpec or Backlog.md.
- Superpowers owns the process spine (TDD/debug); mattpocock owns domain grilling (`grill-me` / `grill-with-docs`); Engram `memory-protocol` when Engram is wired.
- Do not run Engram `sdd-flow` or `gentle-ai install` as a second planning system.

## Behavior

1. Run during **project** install (after skills/MCP steps, or as part of scaffold).
2. If a file is **missing**: create from template.
3. If a file **exists**: merge Skillgrid-managed sections (marked block) without wiping user content — same spirit as MCP merge; one-time `.bak` on first Skillgrid write to that file.
4. **Global** scope does not write these into random repos; only project scope (cwd / `--` project dir).

## Implementation status

Agreed, not implemented yet — see [TODO.md](TODO.md).
