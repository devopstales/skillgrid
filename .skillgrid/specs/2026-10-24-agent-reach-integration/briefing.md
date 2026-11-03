---
type: Spec
title: "2026-10-24-agent-reach-integration — briefing"
archive: "2026-10-24-agent-reach-integration"
status: draft

---

# Change: Agent-Reach First-Class Research Source Integration

> **STATUS:** `ready-for-tasks`
>
> **Classification:** trivial (fast-track eligible) — no behavior change to existing code paths, no new Go dependency, no trust boundary change, no migration. Adds a new optional research source alongside existing ones.

## Goal

Integrate [Agent-Reach](https://github.com/Panniantong/Agent-Reach) as a **first-class research source** in the skillgrid workflow. Agent-Reach is a Python CLI that provides read-only access to login-walled platforms, video transcripts, and structured data — surfaces that `webfetch` and `exa` cannot reach. It slots into the existing `research.sources` config, the `research` skill's sourcing rules, the `mnemonic` web cache, and the CLI install flow.

## Problem

The current research stack (`context7`, `deepwiki`, `exa`, `webfetch`) has a coverage gap:

- **Login-walled platforms** (Reddit, X/Twitter, XiaoHongShu, Boss, LinkedIn, Instagram, Facebook) return a login wall or JS-only page from `webfetch`.
- **Video/audio content** (YouTube, Bilibili, podcasts) has no transcript extraction in the current stack. The `youtube-transcript` skill exists but relies on manual `yt-dlp` invocation.
- **Structured data** (GitHub repo/issue metadata, stock quotes, V2EX threads) is poorly served by raw HTML scraping.

Agent-Reach fills all three gaps with a single CLI that wraps platform-specific tools with a preferred + fallback chain. It is MIT-licensed, read-only, zero API fees (Groq key optional for YouTube transcription), and manages its own lifecycle in `~/.agent-reach/`.

## Research Sources

| Source | URL | Key takeaway |
|---|---|---|
| Agent-Reach README | `github.com/Panniantong/Agent-Reach` | Python CLI, 14 channels (bilibili, v2ex, github, youtube, web, xiaohongshu, xueqiu, boss, xiaoyuzhou, linkedin, reddit, twitter, facebook, instagram). Default channels work without login. Optional channels need login (dedicated browser, reuses Chrome login). Experimental channels (twitter, facebook, instagram) may break. Install via `uv tool install agent-reach --from git+...`. PyPI `agent-reach` is a DIFFERENT project — must use GitHub URL. State in `~/.agent-reach/`. Read-only. |
| Agent-Reach install doc | `github.com/Panniantong/Agent-Reach/blob/main/docs/install.md` | Install is agent-driven: the agent reads install.md and executes the steps. Manages `uv` dependency, per-channel tools, login state. Upgrade/uninstall are also agent-driven one-liners. |
| skillgrid install flow | `skillgrid-cli/internal/install/install.go` | `installSecurityTools` pattern: `SecTool` struct with `Manager` (uv/go) + `InstallArgs` + `Bin` + `Hint`. Non-fatal: missing manager warns and skips. `uv` is already a hard prerequisite (line 216). |
| skillgrid research skill | `.agents/skills/knowledge/research/SKILL.md` | Sourcing rules name tooling inline. `web_cache_lookup`/`web_cache_save` bracket every remote call. `research.sources` config block does not yet exist. |
| skillgrid mnemonic skill | `.agents/skills/knowledge/mnemonic/SKILL.md` | Web cache source enum: `context7|exa|deepwiki|fetch|manual`. TTL 168h, cap 256 KB. |
| skillgrid config | `.skillgrid/config.yaml` | `research:` block has `default`, `output_dir`, `code_research_preset`. No `sources` list yet. |

## Design Decisions

1. **Agent-Reach is a first-class `research.sources` entry, not a lazy escalation.** The config list is the authoritative tooling budget. If it's in the list, the research skills may call it. If it's not, it's not available. This makes the source discoverable and auditable.

2. **No Go module dependency.** Agent-Reach is a Python CLI managed by `uv`, not a Go package. It sits alongside `trivy`, `wapiti3`, `akca`, `nuclei` in the install flow — shell-out, not `go.mod`.

3. **`SecTool` renamed to `ExternalTool`.** The struct is generic enough for both security scanners and research sources. The rename is mechanical (3 functions in `config.go` + references in `install.go` and tests). `SecurityTools()` and `ResearchTools()` both return `[]ExternalTool`.

4. **Install is non-fatal.** Same contract as `installSecurityTools`: skip if binary on PATH, warn if `uv` missing, respect `--dry-run`, capture output. Agent-Reach is a research source, not a build dependency — its absence degrades the research stack but does not break the install.

5. **PyPI name collision.** The install command must use `--from git+https://github.com/Panniantong/Agent-Reach.git`, not the PyPI package name. The `Hint` field documents this for manual install.

6. **Web cache source enum extended.** `agent-reach` is added to the `web_cache_lookup`/`web_cache_save` source list alongside `context7|exa|deepwiki|fetch|manual`. Same TTL (168h) and cap (256 KB).

7. **No new skill.** Agent-Reach is a transport, not a workflow. The existing `research`, `code-research`, and `deep-research` skills call it via shell when the surface demands it. The `youtube-transcript` skill delegates to `agent-reach youtube transcript` when available.

## Steps

| # | Step | Deliverable | Key files |
|---|---|---|---|
| 1 | Rename `SecTool` → `ExternalTool`, add `ResearchTools()` | Generic struct, two tool lists (security + research) | `skillgrid-cli/internal/install/config.go` |
| 2 | Add `installResearchTools()`, call in `Run` | Install step for research CLIs (non-fatal) | `skillgrid-cli/internal/install/install.go` |
| 3 | Add `TestResearchTools` | Unit test mirroring `TestSecurityTools` | `skillgrid-cli/internal/install/install_test.go` |
| 4 | Add `research.sources` to config | Authoritative tooling budget | `.skillgrid/config.yaml` |
| 5 | Update `research/SKILL.md` sourcing rules | Tooling line + routing note | `.agents/skills/knowledge/research/SKILL.md` |
| 6 | Update `mnemonic/SKILL.md` web cache source enum | Add `agent-reach` to source list | `.agents/skills/knowledge/mnemonic/SKILL.md` |
| 7 | Update `youtube-transcript/SKILL.md` | Delegate to `agent-reach` when available | `.agents/skills/craft/youtube-transcript/SKILL.md` |

**Zone order:** Steps 4–7 (spec zone) commit before steps 1–3 (code zone).

## Trust Boundaries

None. Agent-Reach is read-only, runs locally, and uses a dedicated browser profile. No new network trust boundary beyond what `webfetch`/`exa` already establish. No data leaves the machine except to the platform being read (same as `webfetch`).

## Verification Floor

L2 (build + unit test). The Go change is a struct rename + a new tool list + a new install function — all covered by existing test patterns. The spec-zone changes are documentation. No integration test needed (Agent-Reach is external, not in the Go test binary).

## Rollback

- **Code:** revert the 3 Go file changes. `ExternalTool` rename is atomic — one commit reverts it.
- **Config:** remove `agent-reach` from `research.sources` list. Research skills fall back to `webfetch`/`exa`.
- **Machine:** `uv tool uninstall agent-reach` removes the CLI. `~/.agent-reach/` can be deleted.

## Open Questions

None. All decisions resolved during planning.
