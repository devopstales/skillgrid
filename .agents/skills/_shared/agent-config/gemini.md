# GEMINI.md — agent config target

Google Gemini CLI's root instruction file. Primary only when `GEMINI.md` exists and neither `AGENTS.md` nor `CLAUDE.md` does ([decision matrix](README.md#decision-matrix--which-file-gets-the-full-block)).

- File name: `GEMINI.md` (repo root).
- Render: the full block from [block.md](block.md), verbatim with placeholders filled.
- Placement: existing sentinels → replace in place; none → append at end of file.
- Gemini CLI reads `GEMINI.md` per session — keep the block tight (already the case in [block.md](block.md)).

When an `AGENTS.md` (or `CLAUDE.md`) is the repo-wide standard, prefer keeping the full block there and putting only the **secondary one-line pointer** in `GEMINI.md` ([README](README.md#multi-platform-repos)).
