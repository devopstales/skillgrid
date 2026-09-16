# PRD scope is the whole Hub Product, not the binary alone

---
status: "accepted"
supersedes: none
date: 2026-09-16
---

## Context and Problem Statement

The repo has drifted: the README describes `skillgrid` as "an installer that
sets up AI-agent tooling on a machine," but the binary now ships 20+ subcommands
across install, sync, setup, doctor, serve/mcp, index/search/code-intel,
mem/session/trail/eval. A PRD scoped to "the `skillgrid-cli/` directory" or "the
installer" would under-describe the product the code actually is. The PRD's
scope boundary had to be decided before writing, because it changes the entire
document.

## Considered Options

- **The whole Hub Product** (CLI binary + Mnemonic Engine + Distribution Surface + Hub Content) — one product document
- **The `skillgrid` binary as one product** — everything the binary does, but framed as the binary
- **Only the installer** (install/sync/setup), with Mnemonic getting its own PRD

## Decision Outcome

Chosen option: "The whole Hub Product," because the PRD is a product
description, not a module description: the user installs one thing (`skillgrid`)
and gets the engine + surface + content together. The Hub Content (skills/hooks)
is repo content shipped *by* the installer, not a runtime component, so it is
covered in the Installer section, not as a peer component. Scoping to "the
binary" or "the installer" both slice the product along an artificial seam.

### Consequences

- Good, because the PRD matches what a user actually receives: one binary that
  installs, then runs a local engine exposed over three transports.
- Good, because the component map (Installer / Mnemonic Engine / Distribution
  Surface / Hub Content) is a clean 4-way decomposition that future changes can
  reference.
- Bad, because the PRD is larger than a binary-only doc and must stay honest
  about current state (stub UI, 501s, LLM stub) — a "Known gaps" section is
  mandatory, not optional.
