---
name: agent-reach
description: Read-only platform content with the agent-reach CLI: search, read posts, comments, subtitles, stock quotes, jobs. Use when the user pastes or names Xiaohongshu, Bilibili, Twitter/X, Reddit, YouTube, GitHub, V2EX, Xueqiu, LinkedIn, Boss Zhipin, Instagram, Facebook, Xiaoyuzhou content; or asks to install, update, add/remove, or configure Agent Reach. Never posts, likes, or logs in for the user.
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: Panniantong/Agent-Reach
---

# Agent Reach

**Announce at start:** "I'm using the skillgrid:agent-reach skill to read platform content."

## Overview

`agent-reach` is one read-only CLI for every platform: search, read posts, comments, subtitles, stock quotes, jobs. It prints one JSON object per call. Prefer your own search/read tools for public content; Reach fills the gaps — login-walled platforms, subtitle fetching, and sites your built-in tools cannot reach. `skillgrid install` installs the binary (uv); `agent-reach setup`/`update` manage it.

## When to Use

- The user pastes or names content from a supported platform (channel table below).
- "Search / look up / read this link" and your built-in web tools are unavailable, fail, or return a snippet or page shell without the body.
- The user wants to install, update, add/remove, log in/out, configure, or uninstall Agent Reach.
- First use on this machine: `agent-reach` is installed but not set up yet (no channels added) — run the setup below first.

**When NOT to use:**
- Posting, commenting, liking, messaging — Reach is read-only; use the platform's own tooling.
- General public web search — your own search/read tools first; Reach is the fallback.
- Content from a platform not in the channel table.

## Install and update

Run these only when `agent-reach` is missing, not set up, broken, or the user asks to update. `uv` is a prerequisite of `skillgrid install`; standalone: `curl -LsSf https://astral.sh/uv/install.sh | sh` (Windows: `powershell -ExecutionPolicy ByPass -c "irm https://astral.sh/uv/install.ps1 | iex"`).

```bash
agent-reach version    # JSON (1.6+) = current; plain "Agent Reach v1.5.x" = legacy; not found = fresh
```

- **Fresh install:** `uv tool install https://github.com/Panniantong/Agent-Reach/releases/latest/download/agent-reach.zip` then `agent-reach setup`. Show `result.installation_report.markdown` to the user as is — the program generates it; do not rewrite, reorder or add to it.
- **Update (1.6+):** `agent-reach update` — upgrades the program, every added channel's tools, and the skill. Tell the user each line of `result.user_actions`; show `result.installation_report.markdown` as is when present. On failure, `error.hint` holds the exact command to run.
- **Upgrade from 1.5:** tell the user "This is a big upgrade. Your keys and settings are kept; I'm moving Agent Reach to the new installer." Then `uv tool install --force <same URL>`, then `uv tool update-shell`, open a new shell, then `agent-reach setup`. Setup removes the old program (only Agent Reach itself, never other tools).
- **Never** install the PyPI package named `agent-reach` — it is a different project. Only the GitHub release URL above installs this tool.

## Core commands

```bash
agent-reach channels                     # actions and params per channel
agent-reach <channel> <action> [args]    # e.g. agent-reach bilibili search "rust async" --limit 5
agent-reach read <url>                   # picks the channel from the URL
```

Routing: public content → your own tools first (optionally `site:domain`); check body, comments and subtitles separately — when they are enough, answer without adding channels or logging in. Use Reach when your tools are unavailable, the result lacks what the user needs, or the user wants a site action a channel declares. Not-added channels first try anonymous public web access (results marked `backend: public-web`); `--backend <backend>` picks a site tool. A platform name or link only identifies the source — it never triggers install or login by itself.

## Results and errors

`ok: true` → content in `items` (each has `title / url / author / published_at / text`; upstream data in `raw`). `ok: false` → branch on `error.code`:

| error.code | What to do |
|---|---|
| `NEED_SETUP` | Missing dependency or key: `agent-reach add <channel>` as hinted; for a key, the user provides it, then `agent-reach configure <key>` |
| `NEED_LOGIN` | Keep existing results; `agent-reach login <channel>` opens the browser and the user logs in themselves. Never log in for them or ask for a password |
| `UPSTREAM_BROKEN` | Platform temporarily down; tell the user which, by name, and roughly why. Run `agent-reach update` once, after they agree, only if the message points to an outdated tool; never retry in a loop |
| `TIMEOUT` | Check `agent-reach doctor --text`; do not retry or reopen windows in a loop |
| `RATE_LIMITED` | Wait `retry_after_seconds`; do not switch backends |
| `PLATFORM_BLOCKED` | Verification (captcha/risk check); the user resolves it on the official page. Do not switch accounts or backends |
| `BUSY` | Browser full or a login window open; retry later or after the user says they logged in. Do not close the window or switch profiles |
| `NOT_FOUND` | Check the link or ID |
| `BAD_INPUT` | Wrong arguments; run `agent-reach channels` for usage |

On `PLATFORM_BLOCKED`, `RATE_LIMITED`, `NEED_LOGIN` or `BUSY`, stop this attempt — no retry with other accounts, backends or browsers. Snippets, page shells, comment counts and subtitle track lists are not body text; say which you have. If the user skips login, still deliver the public results already gathered.

## Login and the dedicated browser

All browser channels share one dedicated browser (headless by default, closes after 10 idle minutes). First use of a site copies that site's login once from the user's Chrome when the OpenCLI extension is connected; only when there is nothing to copy does the user run `agent-reach login <channel>` and log in themselves.

## Managing Agent Reach

| User says | You run |
|---|---|
| Add a platform | `agent-reach add <channel>` — deps install automatically; tell the user only what they must do (log in or provide a key) |
| Remove a platform | `agent-reach remove <channel>` |
| Log in / login expired | `agent-reach login <channel>` — the user logs in in the window that opens |
| Log out | `agent-reach logout <channel>` |
| Status / what works | `agent-reach doctor --text`; live test: `agent-reach check <channel> --text` |
| Update | `agent-reach update` |
| Uninstall | `agent-reach uninstall --dry-run` first, then `agent-reach uninstall` after the user confirms |
| Another AI should use it | `agent-reach skill link <that AI's skills folder>`; `agent-reach skill` shows where it is placed |
| Provide a key | `agent-reach configure <key>` — `groq-key` (Xiaoyuzhou transcription), `openai-key`, `github-token`, `proxy` |

## Channels

| channel | level | notes |
|---|---|---|
| `web` | default | web search, read pages |
| `github` | default | repos, README, issues/PRs, comments |
| `youtube` | default | search, info, subtitles |
| `bilibili` | default | search, info |
| `v2ex` | default | read topics, replies |
| `reddit` | optional | search, posts, comments |
| `xiaohongshu` | optional | text descriptions only — never claim to understand images or video |
| `linkedin` | optional | job search, public JD read |
| `boss` | optional | job search; `salary_encoded=true` → read the job link for the salary, never guess the digits |
| `xueqiu` | optional | stock quotes, name search |
| `xiaoyuzhou` | optional | podcast transcripts (needs `groq-key`) |
| `twitter` | experimental | search, read posts |
| `facebook` | experimental | search, profile |
| `instagram` | experimental | search users |

`default` channels are installed by `agent-reach setup` and can be removed; updates never add a removed one back. `agent-reach channels` is authoritative over this table.

## Update notices

Any result with `meta.update` means a newer version exists. Mention it once per conversation, in the user's language and plain words: the latest version plus 1–3 short points from `meta.update.notes`. Run `agent-reach update` only after the user agrees.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "I'll just web-scrape the platform myself" | Your tools fail on login-walled and JS-heavy platforms; Reach's site tools exist for that. Use it when your fetch misses the body. |
| "The snippet is enough" | A snippet, page shell or comment count is not the body. Deliver the body, or say which part you have. |
| "I'll log them in quickly for them" | Never log in for the user, never ask for or print credentials; `agent-reach login` opens a window they complete. |
| "Let me retry with another backend" | `PLATFORM_BLOCKED`/`RATE_LIMITED`/`NEED_LOGIN`/`BUSY` mean stop for this attempt; switching accounts or backends hides the platform's signal. |
| "I'll update it while I'm at it" | Ordinary queries never change dependencies or update automatically; update only on `meta.update` with the user's agreement, or when the user asks. |
| "pip install agent-reach is fine" | The PyPI package named `agent-reach` is a different project; only the GitHub release URL installs this tool. |
| "It's installed, so it must be set up" | `skillgrid install` only installs the binary; until `agent-reach setup` has run once, default channels are not added. Check with `agent-reach doctor --text` before assuming. |

## Red Flags

- Any write action attempted (post, comment, like, message) — Reach is read-only.
- A password, cookie or key asked of the user or printed in output.
- Retrying a blocked or rate-limited channel by switching accounts, backends or browsers.
- Claiming image/video understanding from a Xiaohongshu read (text only) or guessing Boss Zhipin salary digits.
- Rewriting or reordering `installation_report.markdown` before showing it.
- Installing from PyPI, or running `agent-reach update` without asking when `meta.update` is present.
- Using a channel right after a fresh `skillgrid install` without having run `agent-reach setup` on this machine.

## Verification

- [ ] `agent-reach version` prints JSON with `ok: true` (or the legacy line, in which case the upgrade path was taken).
- [ ] On a freshly installed machine, `agent-reach setup` has been run once and `result.installation_report.markdown` was shown as is.
- [ ] The called command returned `ok: true` with `items`, or a `error.code` was handled per the table.
- [ ] The user was told which delivered content is body vs snippet/page shell when the latter was used.
- [ ] Any login window was completed by the user, not the agent.
- [ ] `meta.update` was mentioned once and `agent-reach update` ran only after the user agreed.

## References

- [Panniantong/Agent-Reach](https://github.com/Panniantong/Agent-Reach) — source and releases.
- [Install guide](https://github.com/Panniantong/Agent-Reach/blob/main/docs/install.md) — the install/update/upgrade procedure this skill follows.
- [Troubleshooting](https://github.com/Panniantong/Agent-Reach/blob/main/docs/troubleshooting.md) — when an update fails or the dedicated browser will not start.
- `agent-reach channels` — live actions and params per channel (authoritative over the table above).
