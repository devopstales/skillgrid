# Skill Template (fill-in skeleton)

The skeleton every new `SKILL.md` starts from. Fill the braces, delete the
`←` comments before committing. This is a template, not a spec — the rules it
encodes live in `skill-creator` (§5c gates, §5d diagrams, §10 anatomy); cite
those, don't restate them.

```markdown
---
name: skill-name
description: What it does (third person). Use when {trigger}.
license: MIT
metadata:
  author: name
  version: "1.0"
  part-of: skillgrid
---

## Contents                    ← only if the body is > 100 lines
- §1 ...

## Workflow                    ← mermaid map of the steps + gates (§5d),
flowchart TD                   ←   a diamond per interactive gate
    A[Step 1] --> B{Gate?}
    B --|confirm| C[Step 2]
    B --|loop back| A

## Steps                       ← checklist with go-back lines
- [ ] Step 1: ...
- [ ] Step 2: ...
      If X fails: go back to Step 1
- [ ] Step 3: Verify (repeat until all pass)
- [ ] Step 4: Gate — chat summary first, then the single-select question
      (EGM format, §5c; payload in `references/interaction-gates.md`)

## References                  ← context pointers, not inlines
- For pricing details: `references/pricing.md`
- For edge cases: `references/edge-cases.md`

## Prerequisites               ← install line next to every script call
Requires: ...

## Model Notes                 ← only if model-specific behavior matters
- Haiku: ...
- Sonnet: ...
```

## Directory shape

```
skill-name/
├── SKILL.md              ← required: frontmatter + instructions (< 500 lines)
├── scripts/              ← optional: executable code (no token cost when loaded)
│   └── <name>.mjs
├── references/           ← optional: docs loaded on demand
│   └── <topic>.md
└── assets/               ← optional: files used in output (templates, icons)
```

## Writing rules (the ones that bite during the draft)

- Imperative form. Explain *why* in lieu of MUST/NEVER.
- One sentence per script call: "Run `node scripts/x.mjs`; exit 0 = pass,
  exit 1 = drift (report table)."
- No English if-statements. Multi-branch deterministic logic → a script.
- No "you are an elite X" preamble. No explaining what a domain concept is.
- Interactive steps use the EGM gate format: one question per gate, open-ended
  for exploration, single-select for decisions, chat summary before the gate.
- Multi-step workflows include a Mermaid diagram mapped to the steps.
- If a line doesn't change agent behavior, delete it. A line that names a
  command *and* explains what the command does in words is two lines in one —
  cut the explanation, keep the command.
