# CLAUDE.md — agent config target

Claude Code's root instruction file. Primary only when `CLAUDE.md` exists and `AGENTS.md` does not ([decision matrix](README.md#decision-matrix--which-file-gets-the-full-block)).

- File name: `CLAUDE.md` (repo root).
- Render: the full block from [block.md](block.md), verbatim with placeholders filled.
- Placement: existing sentinels → replace in place; none → append at end of file.
- Claude Code loads `CLAUDE.md` on every session, so keep the block tight (already the case in [block.md](block.md)).

When `AGENTS.md` is the repo-wide standard alongside `CLAUDE.md`, keep the full block in `AGENTS.md` and reduce `CLAUDE.md` to the **secondary one-line pointer** ([README](README.md#multi-platform-repos)) so the two don't drift.
