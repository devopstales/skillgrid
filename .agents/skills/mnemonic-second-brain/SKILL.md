---
name: mnemonic-second-brain
description: Capture what just happened into the second brain. Use when the user says "remember this", "don't forget that", "we decided to…", "note for next time", "important:", "so that next time…", or otherwise signals durable knowledge that must survive the session. Also use proactively after a bug fix, an architecture decision, or a non-obvious discovery.
---

# Second-Brain Capture

No questions. No ceremony. Just save it.

When you detect capture intent:

1. **Extract** the durable fact (what / why / where; add learned only if there's a gotcha).
2. **Infer** the type from the existing taxonomy (decision, bugfix, pattern, discovery, config, correction, learning, architecture, …). If the phrase is "we decided to…", it is a decision.
3. **Pick a stable topic_key** so rephrasings upsert, not duplicate (e.g. `architecture/store`, `bugfix/n-plus-one`). Reuse the same key when the same topic evolves.
4. **Call `mem_save`** with the structured content (What / Why / Where / Learned) and `topic_key`. Pass `infer=true` only when you are unsure of the type — let the tool fill empty metadata; never fight an explicit type you already know.

Wrap anything sensitive (tokens, PII, paths with secrets) in `<private>…</private>` — it is stripped before storage.

Do not narrate the save back to the user unless they ask.
