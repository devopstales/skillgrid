---
name: mnemonic-second-brain
description: "Skill trigger + capture contract for mnemonic second-brain: natural-language save cues that auto-infer type/topic_key via mem_save.infer."
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: skillgrid-v2:mnemonic
---
# Mnemonic Second Brain

Operates the mnemonic skill as a second brain: natural-language capture cues that auto-infer memory type and topic key via `mem_save.infer`, then persist via `mem_save`.

## Trigger Phrases

The following natural-language phrases trigger a `mem_save` call with `infer=true`, causing the tool to deterministically fill `type` and `topic_key` via the existing intent heuristic:

- `remember this`
- `don't forget that`
- `we decided to…`
- `note for next time`
- `important:`

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

## Related

- `.agents/skills/knowledge/mnemonic/SKILL.md` — the base mnemonic skill
- `.skillgrid/artifacts/05-locked-constraints.md` — second-brain capability layer ADR
- `.skillgrid/artifacts/09-claude-os-deep-dive.md` — verified steal list S1–S7