# PRD framing is engine-first, not installer-first

---
status: "accepted"
supersedes: none
date: 2026-09-16
---

## Context and Problem Statement

The README's one-paragraph framing is installer-first: "installs the hub onto
a machine." The user guide and config context tell a richer story: a
local-first engine for agent memory + code intelligence, distributed as a
CLI/MCP/HTTP server, with pipeline discipline (skills/hooks) on top. A PRD that
leads with the installer undersells the product and mis-sequences the component
map (installer becomes the hero instead of the engine). The framing decision is
hard to reverse: it determines the product vision sentence, the persona
ordering, and which component the architecture leads with.

## Considered Options

- **"An installer that sets up AI-agent tooling on a machine"** — installer-first (the current README)
- **"A local-first engine for agent memory + code intelligence, distributed as a CLI/MCP/HTTP server"** — engine-first
- **"A hub for opinionated AI-assisted development: pipeline discipline + persistent memory + code orientation, all local"** — hub-first (the config context)

## Decision Outcome

Chosen option: "A local-first engine for agent memory + code intelligence,
distributed as a CLI/MCP/HTTP server," because the code has drifted
engine-first and the PRD should name the product for what it is, not for what
the README says. The pipeline discipline (skills/hooks) is a property of the
installed Hub Content, not the engine's selling point; leading with "opinionated
pipeline" (option 3) makes the PRD sound like a methodology rather than a
product. Engine-first (option 2) keeps the user's primary pain (agent context
doesn't survive sessions; "done" is a claim without evidence) as the through-line.

### Consequences

- Good, because the product vision, personas, and success metrics all derive
  from a single through-line (persistent memory + code intelligence), which
  makes the PRD internally consistent.
- Good, because the secondary persona (AI-tooling builders who embed the
  capability) falls out naturally from the MCP/HTTP surface — it is a
  consequence of engine-first, not a separate product.
- Bad, because the README is now stale by construction; aligning it is a
  follow-up ticket, not part of this PRD (ADR-0001 scope).
