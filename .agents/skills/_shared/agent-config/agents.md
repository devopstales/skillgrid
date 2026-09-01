# AGENTS.md — agent config target

Cross-agent standard. Preferred **primary** target whenever `AGENTS.md` exists ([decision matrix](README.md#decision-matrix--which-file-gets-the-full-block)).

- File name: `AGENTS.md` (repo root).
- Render: the full block from [block.md](block.md), verbatim with placeholders filled.
- Placement: existing `<!-- skillgrid-sdd:start/end -->` sentinels → replace in place; none → append at end of file.
- No platform-specific note needed (this is the surface the other platforms point to).

If `CLAUDE.md` and/or `GEMINI.md` also exist in the repo, add the **secondary one-line pointer** (not the full block) to them — see [README](README.md#multi-platform-repos).
