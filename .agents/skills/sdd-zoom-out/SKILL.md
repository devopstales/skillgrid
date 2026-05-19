---
name: sdd-zoom-out
description: >
  Tell the agent to zoom out and give broader context or a higher-level perspective on an unfamiliar section of code.
  Trigger: User says "zoom out", "I don't understand this area", "what's the big picture here", or agent is unfamiliar with a code section.
license: Apache-2.0
metadata:
  author: devopstales
  version: "1.0"
triggers:
  - "zoom out"
  - "big picture"
  - "I don't understand this"
  - "what does this area do"
  - "how does this fit together"
  - "overview of this module"
tools:
  - file_system
  - execute_command
---

# Zoom Out

I don't know this area of code well. Go up a layer of abstraction. Give me a map of all the relevant modules and callers, using the project's domain glossary vocabulary.

## Process

1. **Identify the current scope** — what file, function, or module is the user looking at?
2. **Map the callers** — who calls this? What triggers this code path?
3. **Map the dependencies** — what does this code call into? What services, stores, or external systems does it reach for?
4. **Identify the module boundary** — where does this concept live in the architecture? What is its responsibility?
5. **Use domain language** — describe everything using terms from `.skillgrid/project/CONTEXT.md`. If a term isn't defined, note it as a candidate for the glossary.

## Output Format

```markdown
## Module Map: <concept name>

**Responsibility:** <one-sentence description>

### Callers (who triggers this)
- <caller 1> → <why it calls>
- <caller 2> → <why it calls>

### Dependencies (what this reaches for)
- <dependency 1> → <what it's used for>
- <dependency 2> → <what it's used for>

### Module boundary
- **Interface:** <what callers must know>
- **Hidden internals:** <what callers should NOT know>

### Related modules
- <related module 1> — <relationship>
- <related module 2> — <relationship>
```

## Return

- **Return:** the standard SDD envelope per `skills/_shared/sdd-return-envelope.md` (module map and friction notes in `detailed_report`).

## Rules

- Do NOT dive into implementation details — the point is the map, not the territory.
- If a module name doesn't match the domain glossary, translate it.
- If you find undocumented terminology, flag it for `CONTEXT.md` update.
- If the module has no clear responsibility or is a "god object," note that as architectural friction.
