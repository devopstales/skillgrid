# Hosts

How to run the phases and the isolated critics inside the agent the programmer already uses, and what to do when a host cannot isolate.

When to load: Phase 0 (choose the independence setup), any phase that spawns a critic. Role: orchestrator.

## The contract every host must meet

1. **Read and write** the programmer's `.skillgrid/` folder and working tree.
2. **Spawn** a subagent or a separate process with a filtered packet.
3. **Wait** for its result and save it verbatim.
4. **Run commands** in the project and capture their output.

A host that meets 1, 3 and 4 but not 2 can still run the pipeline. Its critics are grade C and every verdict says so.

## Hosts

| Host | Isolated critic | How | Grade |
|---|---|---|---|
| Claude Code | subagent tool, fresh context | `Task`/`Agent` with the packet in the prompt; no session history | B |
| Codex CLI | separate process, separate session | `codex exec` with the packet file, `--cd` the project | A when the writer was another family; B otherwise |
| Cursor / Windsurf | new chat, file-scoped context | a new composer with only the packet files attached | B |
| OpenCode | `subagent` with a fresh session | pass the packet in the prompt; the child starts with empty context | B |
| A plain chat | new conversation | paste the packet; nothing else | C |

Grade A needs a different **model family**, not just a different session. Two Claude sessions are grade B. Claude writing and Codex reading is grade A.

## Asking the programmer

Before using a second tool, ask in the intake check-in:

> "I found `<tool>` on this machine. It can read the change blind as a second opinion. It uses your `<tool>` quota and sends the packet to `<provider>`. Use it?"

Record the answer in `genesis.decisions`. Never use a paid second tool without it, and never send a packet containing secrets or the programmer's private paths.

## Packet delivery

- Write the packet to a file under `.skillgrid/work/<YYYY-MM-DD-topic>/packets/` and point the critic at it. A packet pasted into a prompt is easy to trim accidentally; a file is auditable.
- The packet contains only the allow-list in `references/roles/blind-reviewer.md` and `references/roles/auditor.md`.
- Ask the critic to list the files it received before it judges. That list is the isolation check.

## When no isolation is available

Say so once, plainly: "This host cannot spawn an isolated reviewer, so the audit is grade C — the same context that wrote the code is judging it." Then run the audit anyway, label every verdict, and let the programmer decide whether to run a second tool manually. A labelled weak audit beats no audit; an unlabelled one is a lie.
