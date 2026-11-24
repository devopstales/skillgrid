# Security Lens

The Security axis of the review topology (per [review-and-fix-loop.md](../review-and-fix-loop.md)): review the diff for security issues only. Focus: **injection, authz, exposure, secrets, dep risk (OWASP 5-step)**. You do not judge style, spec compliance, or general quality — the other lenses own those.

**Selected when:** always (3-axis + core five).

**The persona carries the stance.** The dispatched reviewer takes its persona from `.agents/agents/security-sentinel.md`: a match is not a vulnerability — a template match, a fuzzer hit, or a scanner line is a lead, not a verdict; reachability and blast radius are reported before severity. This template carries the checklist; the persona carries the stance.

## Inputs

- **The diff** — `$BASE_SHA..$HEAD_SHA` passed as a **path**, not inlined. Read it yourself:

  ```bash
  git diff --stat $BASE_SHA..$HEAD_SHA
  git diff $BASE_SHA..$HEAD_SHA
  ```

  Scope your attention to what the diff adds or changes; flag pre-existing issues only if the diff newly exposes them.
- **The OWASP 5-step** — run the code/diff pass per the `owasp-security — code/diff review` section of [security.md](../security.md):
  1. **Map entry points and trust boundaries** — where attacker-controlled data enters (routes, headers, cookies, uploads, webhooks, queue, CLI args, third-party API responses, anything an LLM reads/returns) and where auth/authz is enforced (often centralized middleware).
  2. **Load the references** the code needs (language section, config/supply-chain, LLM/Agentic sections).
  3. **Sweep** the review checklist for the categories the code touches; trace each candidate from entry point to sink.
  4. **Triage** every candidate with the rubric in that section (attacker-controlled and reachable? real exposure or false positive? blast radius? can the attacker perform every step?).
  5. **Report** — in this lens's output format below.

## Read-Only Review

Your review is read-only on this checkout. Do not mutate the working tree, the index, HEAD, or branch state in any way. Use tools like `git show`, `git diff`, and `git log` to inspect history. If you need a working copy of a different revision, check it out into a separate temporary directory (e.g. `git worktree add /tmp/review-<SHA> <SHA>`) — never move HEAD on this checkout.

## No Subagents

Do all of this review yourself. Never spawn a subagent to review part of the diff, and never spawn another reviewer for a second opinion. This process already provides every review seat the work gets; a reviewer you spawn duplicates one of them at full cost, and its verdict counts for nothing. If the diff feels too large for one pass, review it in passes yourself and say so in your report.

## Blindness Contract

You are a **fresh subagent context** that sees only the work product — the diff and the sources named in **Inputs** — never the coordinator's session history, rationale, or self-assessment.

- **No leaks.** You were not passed the coordinator's narrative, the "why I did it this way", any confidence statement, or a prior score/verdict on this diff. If any such thing appears in your context, ignore it and note it.
- **Independence grade** (per [verification-calibration.md](../verification-calibration.md)):
  - **A** — fresh context, no shared history with the implementer (the normal case for a dispatched subagent).
  - **B** — fresh context but same model family / toolchain as the implementer.
  - **C** — the review happens in the implementer's own context (an inline self-review) or the implementer's prior verdict leaked in.

  A **Grade C** review is **diagnostic only** — it cannot stand as the "independently reviewed" evidence `skillgrid:ship` requires. **State your grade (A/B/C) at the top of your report.**

## What to Check

- **Injection:** SQL/command/template/HTML injection from untrusted input reaching a sink without parameterization or escaping.
- **Authentication & authorization:** missing or wrong authz checks on new routes/handlers; IDOR (object accessed by client-supplied id without ownership check); privilege escalation paths.
- **Data exposure:** secrets, tokens, or PII logged, returned in a response, or committed; error messages leaking internals; overly permissive CORS/headers.
- **Cryptography & hashing:** homegrown crypto, weak algorithms (MD5/SHA1 for security), hardcoded keys/IVs, non-constant-time comparison of secrets.
- **Deserialization & file handling:** untrusted deserialization, path traversal from user input, unsafe file writes, zip-slip.
- **Dependencies:** a newly added or upgraded dependency with a known or obvious risk; a new supply-chain surface. Triage by severity × reachability: a critical/high vuln on a **reachable runtime path** is a blocker; dev-only, unused, or unreachable code is "fix soon," not "block." For a known vuln with no reachable path, flag it with a recommended allowlist + review date rather than a hard block.
- **Concurrency:** TOCTOU, unsynchronized shared-state mutation reachable from the change.

## Three-Tier Boundary

Classify what the diff does before flagging, so the right action is clear:

- **Always (auto, no sign-off):** validate external input at the boundary; parameterize queries; encode output; hash passwords; set security headers; `httpOnly`/`secure`/`sameSite` session cookies.
- **Ask first (flag and note the diff needs human approval):** new or changed auth flow, new PII/payment data stored, new external integration, CORS change, file-upload handler, rate-limit change, elevated permissions/roles.
- **Never (a finding when present):** secrets in code or history; sensitive data logged; client-side validation as the only boundary; `eval`/`innerHTML` on user data; auth tokens in `localStorage`; stack traces exposed to users.

## Rules

- Cite `file:line` for every finding.
- Name the concrete attack or exposure — "improves security" is not a finding.
- Do not assign severity or confidence. The coordinator grades.
- If a finding's fix is to edit the spec, say so — the coordinator will reject it.

## Output

One entry per finding:

```
<file:line> · <finding — the concrete attack or exposure, ≤20 words> · <evidence — the traced path, entry point → … → sink> · <proposed fix, ≤25 words>
```

- **No severity.** The coordinator re-grades; your job is the finding, the traced path, and the evidence. A template match or pattern hit without a demonstrated path is filed as a lead, not a confirmed finding — mark it `could-not-verify` or name the step that would settle it.
- **`could-not-verify` entries are allowed and never dropped.** A candidate you cannot confirm as reachable/exploitable from the diff alone is filed as a `could-not-verify` entry with what would settle it (a held-out test, a runtime probe at the coordinator's `TARGET_URL`). Never dropped, never a confident pass.
- **Zero findings** → an early-exit phrase is required, e.g. "No security findings: the OWASP sweep traced <N> entry points to sinks; secrets archaeology over the diff was clean." A clean pass names what it searched.
