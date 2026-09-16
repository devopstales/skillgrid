# QA plan — 013-mnemonic-layered-memory-governance

Change-level verify verdict: **PASS** (34/34 scenarios COMPLIANT at runtime; 005/008/010/011 baselines intact; `-race` clean). Per-step verdicts: 01 PASS, 02 PASS, 03 PASS. No Ticket.

This is a Go library/CLI feature on the Mnemonic memory system: governance fields (owner/version/status/usage/visibility) + `mem_share`/`mem_governance`, session-close distillation (L0→L1/L2/L3) + `mem_layers`, and layered budgeted retrieval (`mem_search`/`mem_context`/`mem_timeline`). All three steps are opt-in/additive with the 005 `mem_*` tool surface stable (75→78). The automated suite proves every invariant at runtime — this plan is for a human to confirm the *experience* on a real Mnemonic store.

## What to exercise

### Happy path
1. `mem_save` a memory → it is `private` with an owner; the same owner's `mem_search` finds it; `mem_governance <id>` shows owner/versions/status/usage/visibility.
2. `mem_share <id> team` → a different owner/agent's `mem_search` now finds it; `mem_share <id> restricted` + an ACL grant → only that grant can read.
3. Close a session that has `## Key Learnings:` content (distillation enabled) → `mem_layers <session_id>` shows the L0→L1→L2→L3 chain; `mem_layers <topic_key>` by topic.
4. `mem_search` on a large store → results are budgeted snippets (truncated, "N chars omitted"), each with an id; `mem_get_observation <id>` returns the FULL content.

### Edge
5. A session with no new L1-able content closes as a no-op — `mem_layers` shows no new layers (nothing fabricated).
6. `mem_search` with a short `--timeout` on a slow store → returns a `truncated: true` partial with a reason, does not hang.
7. A `mem_update` on a distilled L1 atom → it survives (correctable, not deleted) and `mem_governance` shows the version history + L0 trace.

### Failure
8. `mem_share <id> galaxy` (unknown target) → rejected, visibility unchanged. Bad args on `mem_layers`/`mem_governance`/`mem_share` → clear error, no fabricated result.

## Environment / data
- A working Mnemonic store (`skillgrid mcp` / `skillgrid-cli`) with a small project and a few saved observations. For step 02, enable distillation (the opt-in) and a session with Key-Learnings content.

## Pass / fail criteria
- Pass: #1–#8 behave as described; non-shared private/restricted-no-grant memories are invisible to a 2nd owner; `mem_get_observation` is the only full-content path; budgeted reads truncate explicitly and never hang; 005 `mem_*` tool names/required params unchanged.
- Fail: a private memory leaks to a 2nd owner without a share; a finding/layer with an empty/fabricated provenance; a budgeted read hangs or silently crops; a `mem_*` tool name/required param changed.

## Waive
The automated suite (per-step Verdicts + change-level runtime proof, incl. the end-to-end owner-gated wiring tests and the enforced-timeout test) already proves every @p0/@p1 invariant at runtime, and all three steps are opt-in additive with a stable 005 surface + additive migrations. Human QA may be **waived** if the user accepts the residual (the exact "N chars omitted" count is not pinned in every path; mem_context/timeline item-cap uses their own `limit` param).
