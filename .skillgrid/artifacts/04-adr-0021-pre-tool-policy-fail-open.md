# Pre-tool policy is opt-in, first-match field rules, and fails open

---
status: "accepted"
supersedes: none
date: 2026-10-02
---

## Context and Problem Statement

Sessions now record every tool call that Cursor, OpenCode, and Kilo make (Gryph-style observability). The next step is to act before a call runs: block a write to `secrets/`, warn on `rm -rf`, or nudge an agent toward a different tool. A pre-tool hook sits in the agent loop for every user of the hooks, so a bad rule, a slow server, or a crash changes what agents can do. Users who already have hooks installed would get enforcement they never asked for. The rule language and the failure mode are hard to change once policy files exist in repos.

## Considered Options

- Enforce by default with a CEL (or Rego) expression engine evaluated in the server
- Opt-in enforcement with first-match field matchers (action, path glob, command glob, tool, agent, project, counter thresholds) evaluated in the server; hooks fail open when the server is unreachable
- Opt-in enforcement evaluated inside each hook script (JavaScript) with no server round-trip

## Decision Outcome

Chosen option: "Opt-in enforcement with first-match field matchers evaluated in the server; hooks fail open", because the matchers cover the Gryph rule set without a new dependency (CEL would need its own ADR), one evaluator serves the CLI (`skillgrid policy test`), the HTTP API, and every harness, and failing open keeps an outage of `skillgrid serve` from stopping agents.

- Policy is disabled unless `policy.enabled: true` is set in `.skillgrid/policy.yaml` or `~/.skillgrid/policy.yaml`. The repo file's rules come first; the machine file's rules follow. With no file or `enabled: false`, every call is allowed and nothing is recorded as a decision.
- Rules evaluate in order and the first match wins. Effects are `block`, `warn`, `guide`, and `allow`. No match means allow.
- `POST /policy/evaluate` returns `{effect, message, rule}`. A `block`, `warn`, or `guide` decision is written to `session_events` with result `blocked`, `warned`, or `guided`; a block also increments `sessions.blocked_actions`.
- A hook that cannot reach the server, times out, or gets a malformed answer allows the call. Fail-closed is not offered by Skillgrid's hooks.

### Consequences

- Good, because nobody's agents change behavior on upgrade, a policy file is reviewable YAML, and `policy test` gives the same answer as the hooks.
- Good, because blocks and warnings show up in the same timeline, stats, and `skillgrid logs` as the calls themselves.
- Bad, because fail-open means a policy is advisory whenever `skillgrid serve` is down. It is a guard rail, not a security boundary.
- Bad, because field matchers cannot express compound logic (OR across fields, regex captures). A future need for that is a new ADR that supersedes this one.
- Bad, because every guarded call pays a localhost round-trip before it runs.
