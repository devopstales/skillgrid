---
name: mnemonic-second-brain
description: "Use when the user says remember this, don't forget that, we decided to…, note for next time, important:, or so that next time…. Natural-language save that auto-infers type and topic_key via mem_save.infer. Also use after a bug fix, an architecture decision, or a non-obvious discovery."
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: skillgrid-v2:mnemonic
---
# Mnemonic Second Brain

**Announce at start:** "I'm using the skillgrid:mnemonic-second-brain skill for natural-language memory capture."

Operates the mnemonic skill as a second brain: natural-language capture cues that auto-infer memory type and topic key via `mem_save.infer`, then persist via `mem_save`.

## Trigger Phrases

The following natural-language phrases trigger a `mem_save` call with `infer=true`, causing the tool to deterministically fill `type` and `topic_key` via the existing intent heuristic:

- `remember this`
- `don't forget that`
- `we decided to…`
- `note for next time`
- `important:`
- `so that next time…`

## Capture Contract

When a trigger phrase is detected:

1. **Extract** the intent from the surrounding context
2. **Infer** `type` from the existing taxonomy (decision/bugfix/pattern/discovery/config/correction/learning/…) via `mem_save.infer`
3. **Infer** `topic_key` via `mem_suggest_topic_key`
4. **Assemble** `mem_save` content with What/Why/Where/Learned
5. **Call** `mem_save` with the stable `topic_key` so rephrasings upsert

**`mem_save.infer=true` flag (default false):** when set and `type`/`topic_key` are empty, fill `type` via the intent heuristic and `topic_key` via `mem_suggest_topic_key`. It only fills empty fields; agent-provided values always win.

The agent is the classifier; the flag makes that step testable without a live model.

## When to Use

- When you need to save a decision, bugfix, discovery, or convention in natural language
- When starting a new session and you need to recall prior context before re-deriving it
- When you want mnemonic to *feel* like a second brain — capture what happened, ask questions over the store with citations, and keep the store clean so it stays trustworthy

When NOT to use:

- Structured saves where you already know the exact `type`/`topic_key` — call `mem_save` directly via skillgrid:mnemonic
- When the mnemonic MCP server is unavailable — the capture contract has no file fallback
- For code orientation (symbols, callers, blast radius) — that is skillgrid:mnemonic's code index, not this skill

## Common Rationalizations

| Rationalization | Reality |
| --- | --- |
| "I'll save it later when I have a moment" | Rephrased-later saves get new hashes; the `topic_key` upsert only works when you save as you go |
| "The type is obvious, I'll skip infer" | The intent heuristic is deterministic and testable; a guessed type is a manual one that no test covers |
| "Two similar observations are fine, the store dedupes" | The store dedupes by hash within 24h only; `topic_key` upserts are the real merge path |

## Red Flags

- Saving a rephrased observation without reusing the original `topic_key`
- A save with a fabricated `type` outside the taxonomy
- Treating a failed `mem_save` as retry-safe without checking the store for the existing row

## Verification

After each capture: `mem_search` with a keyword from the saved content returns the new row (or the upserted one). `mem_get_observation` shows the full What/Why/Where/Learned body.

## Related

- skillgrid:mnemonic — the base mnemonic skill (`.agents/skills/knowledge/mnemonic/SKILL.md`)
- `.skillgrid/artifacts/05-locked-constraints.md` — second-brain capability layer ADR
- `.skillgrid/artifacts/09-claude-os-deep-dive.md` — verified steal list S1–S7