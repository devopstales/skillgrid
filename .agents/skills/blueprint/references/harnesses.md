# Harnesses (blueprint) — dispatch points for the shared Goldfish protocol

The R36 freshness STOP, the R37 four-part isolation contract, and the R38
harness table live in `_shared/elephant-goldfish/protocol.md` — the single
source. This file names only the **blueprint-specific dispatch points** where
the protocol's freshness check fires; it does not restate the contract.

## Where the freshness check applies (blueprint tracks)

Per `protocol.md` § Freshness rule (R36), the STOP check runs at each track's
first dispatch point:

- **plan review (Track B):** before Stage 5 is dispatched (see
  `goldfish-review.md` § Dispatch protocol)
- **drift:** before the first `codebase-details` reviewer dispatches

If the harness cannot spawn a fresh-context subagent, halt and name R36 — no
partial run, no same-context count.

Everything else — the verbatim STOP wording, the four isolation parts, the
per-harness mechanisms, and how to add a fifth harness — is in
`_shared/elephant-goldfish/protocol.md`.
