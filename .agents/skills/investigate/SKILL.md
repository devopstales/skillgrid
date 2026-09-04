---
name: investigate
description: "Investigate a question against high-trust primary sources and capture the findings as one cited Markdown file. Use when you need to research a topic, gather docs or API facts, compare library options, delegate the reading legwork to a background agent, or produce the upstream research for an SDD change."
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
---

# investigate

Investigate a question against **primary sources** and capture the findings as a single cited Markdown file.

## When to use

- The user asks you to **research** a topic, verify a claim, or gather docs / API facts.
- You need a decision input that only a primary source can give (a behavior, a default, a version constraint, a deprecation).
- An SDD change needs its `research.md` / upstream fact-gathering and the reading legwork should not block the main thread.

## How to run

Spin up a **background agent** to do the research, so you keep working while it reads (the platform primitive may be `task(...)` with a background flag, a sub-agent, or an async tool — pick whichever the runtime offers; never block the foreground turn on it).

The agent's job:

1. **Investigate against primary sources** — official docs, the source code, specs, first-party APIs, release notes. Not a secondary write-up of them. Follow every claim back to the source that owns it.
2. **Cite each claim.** Inline source (URL or file path) next to the claim. If you cannot find a primary source for a fact, say so and drop or downgrade it.
3. **Write it to one Markdown file** in the location below, matching the repo's existing notes convention. If none, put it somewhere sensible and say where.

## Where the file lives

Default: the existing research-notes location in the repo (a `notes/`, `docs/`, `adr/`, or similar directory the project already uses).

When invoked for an in-flight SDD change, the change owns the artifact location — write it here instead:

```
docs/skillgrid/changes/<NNN-slug>/research.md
```

and persist the same content to Mnemonic following the shared convention (see [mnemonic-memory.md](../_shared/conventions/mnemonic-memory.md)):

```
mem_save(
  title:      "sdd/<NNN-slug>/research",
  topic_key:  "sdd/<NNN-slug>/research",
  type:       "architecture",
  scope:      "project",
  session_id: {sid},
  content:    {full markdown content}
)
```

## What the file looks like

```markdown
# <one-line question>

Date: <ISO>
Question: <the exact question being answered, restated>

## Findings

- <claim 1> — <primary source URL / path>
- <claim 2> — <primary source URL / path>
- <claim 3 (no primary source found): stated as such>

## Unresolved

- <things we could not confirm; flag clearly>
```

Keep it terse. The file is a lookup, not a blog post.

## Rules

- **Primary sources only.** A secondary article or a Stack Overflow answer is a lead, not a citation.
- **One file.** If the question spans two areas, split into two research runs, not one file with two topics.
- **No code, no edits.** A research run may read anything but writes only the one artifact file (and the optional Mnemonic persistence — same file, not a second location).
- **If a primary source says one thing and a secondary source says another,** the primary source wins and the disagreement is recorded in Unresolved.
- **Background, not foreground.** The point of the skill is to keep the main thread unblocked. If your platform has no background primitive, do the research inline but tell the user it ran blocking.

## Gotchas

- `mem_search` returns 300-char previews. Never treat a preview as the full finding — always `mem_get_observation(id)` for the stored artifact.
- Mnemonic topic keys are namespaced per change: `sdd/<NNN-slug>/research`. Misspell the slug segment and `sdd-propose` searches into the void.
- A "research" file that includes the agent's opinion is not research. Opinions go to `change.md` (`sdd-propose`), not here.
- If the repo already keeps notes under a specific path, do not create a second one — match the existing convention exactly.
- At session end: `mem_session_summary` then `mem_session_end`.

## References

- [mnemonic-memory.md](../_shared/conventions/mnemonic-memory.md) — save shape, naming, session protocol.
- [sdd-structure.md](../_shared/conventions/sdd-structure.md) — where `research.md` sits in the change folder.
