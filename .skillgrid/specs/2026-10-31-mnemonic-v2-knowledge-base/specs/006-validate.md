# 006 — Validate (the pin-check, repocodex first-class)

**Status:** DRAFT · **Owner:** knowledge-base change
**Depends on:** `001-structure` (§5.2 `anchors:`, §5.3 `claims:`), `004-index`
(`kb_claims` — the read side), the `2026-10-31-mnemonic-v2-code-index` change (the code
graph for `symbol:` anchors), `008-transport` (the `--strict` / CI integration).
**Reference:** repocodex pinned-why + ripgrep anchors + pin-check that fails the turn.

---

## 1. Capability

The pin-check is what makes a grounded page **trustworthy**: it attests that the *why*
(the page) still matches the *what still holds* (the code). This is repocodex's core
idea, adopted as **first-class** (decision KB-2): every grounded page (decision /
invariant / workflow) carries `anchors:` + `claims:`; `kb_validate` re-reads the live
code and checks them, **failing the turn on divergence**.

A page that pins code is only as good as the code it pins. If the code moves (the
function is renamed, the literal changes, the line shifts), the page is **stale** — and
the agent must know *before* it relies on the page's "why."

## 2. What is checked

For each **grounded** page, `kb_validate` checks every anchor + claim:

| Check | Against | Result |
|---|---|---|
| `all_of: [token, …]` (token) | ripgrep over the anchor `path` (live working tree) | `ok` if every token present; `unresolved` if the file is missing |
| `line: N` + `version` (span) | the **relocating version token** (§2.1) vs the live file | `ok` (resolved, incl. relocated — line drift free) / `stale` (content changed at a unique site) / `unresolved` (ambiguous / no match) |
| `symbol: Name` (symbol) | the **code-index graph** (live), via `code_resolve` (`02-code-index/004` §3) | `ok` if `exists` (+ call-site holds for a call-anchor — `called_from` non-empty / the pinned file is in `called_from`); `stale` if it exists but the pinned call-site is gone; `unresolved` if `exists: false` or `ambiguous` |
| `claim: {subject, literal}` | the live code at the anchor site | `ok` if the literal holds; `stale` if it no longer holds (e.g. `enterprise_grace_attempts` is no longer `"3"`) |

**The anchor engine (KB-O1, default `auto`):**

- **token / span** → **ripgrep + the version-token resolver** (fast, no graph dependency,
  works on any text). The token/span path is the repocodex path *upgraded* with
  openwiki's relocating version (below) so line-number drift is free.
- **symbol** → the **code-index graph** (symbol-level, call-graph aware). This is the
  v2 addition — ripgrep can find a token but cannot tell you "is `capturePayment` still
  called from `enterprise.go`." The graph answers that.

Both engines are available; a page may mix token/span anchors (ripgrep) and symbol
anchors (graph). `config.json.anchor_engine` can force one engine (`auto` (default) |
`ripgrep` | `graph`). The **engine × anchor-kind matrix** (the complete behavior, so the
pre-code-index phase and forced-engine configs are unambiguous):

| | `all_of` (token) | `line`+`version` (span) | `symbol` |
|---|---|---|---|
| **`auto`** (default) | ripgrep | ripgrep + version-token resolver | graph; **if no graph** → ripgrep occurrence check, `degraded: true` (call-site not checked) |
| **forced `ripgrep`** | ripgrep | ripgrep + version-token resolver | ripgrep occurrence check, **`degraded: true`** (call-site not checked) |
| **forced `graph`** | graph symbol-presence (token = symbol lookup) | graph symbol-presence + line re-resolve, **`degraded: true`** (no token relocating without the file bytes) | graph |

The degradation is always **explicit** (`degraded: true`, INV-3) — never a silent
fallback. The common pre-code-index case (`auto` + no graph built yet) degrades only the
`symbol:` checks; token/span checks are unaffected (they never need the graph).

### 2.1 The relocating version token (openwiki's line-range model)

A `line: N` anchor alone is brittle — the pin-check fails the *instant* the line moves,
even if the construct is unchanged. openwiki's **relocating version token** fixes this.
A span anchor is pinned by a version token of the form:

```
kb-lines-v1:sha256:<contentHash>:<base64url(metadata)>
```

where `<contentHash>` is the SHA-256 of the selected line range's bytes, and
`metadata` carries the span length + **6 relocation anchors** (the count + 5 hashes).
The metadata is a fixed-order base64url-encoded JSON object:

| Field | Role in re-resolution |
|---|---|
| `selectedLineCount` | sizes the candidate windows (a *length*, not an anchor) |
| `firstSelectedLineHash` / `lastSelectedLineHash` | **prefilter only** — narrow candidate windows by first/last line |
| `precedingContextLineCount` / `precedingContextHash` | **decision anchor** — hash of up to 3 lines *before* the span |
| `followingContextLineCount` / `followingContextHash` | **decision anchor** — hash of up to 3 lines *after* the span |

**Re-resolution is an explicit decision procedure** (the pin-check's read side). The
first/last selected-line hashes are **prefilter-only** (they narrow candidates); only the
preceding/following **context hashes** decide. For the hinted file:

1. **Hit at the hinted lines.** If the bytes at the token's original location still
   hash to `<contentHash>` → resolved in place. Check result **`ok`** (page contributes
   nothing to `stale`). *(The code may sit at the same lines; the construct is unchanged.)*
2. **Relocate (unchanged content, moved).** Otherwise scan all `selectedLineCount`-line
   windows, **prefiltered** by first/last selected-line hashes. For each prefilter
   survivor, verify the **context hashes** (preceding + following) match *and* the
   window content hashes to `<contentHash>`:
   - **exactly one** such window → resolved at the new location, content unchanged.
     Check result **`ok`**. *(Line-number drift is free — this is the win.)*
   - **zero or two-or-more** → fall to step 3.
3. **Content changed or ambiguous.** If no unchanged-content window resolved, look for a
   window whose **context hashes** match (the exterior is intact) but whose content
   differs:
   - **exactly one** → the construct's *text* changed at a unique location. Check
     result **`stale`**; the result carries the new live value (so the agent fixes the
     *knowledge*). **The token is NOT re-written** — case 3 always reports `stale`;
     updating the pin is the knowledge-fix loop (`006` §6), not a side effect of
     validate.
   - **zero or two-or-more** → **`unresolved`** (fail closed; the pin is *never*
     silently re-pointed to a guess).

> **Prefilter-vs-context precedence (the "which is it?" resolved):** first/last selected-
> line hashes *never* decide. A single prefilter survivor whose context hashes do not
> match is **`unresolved`**, not `stale` — the context hashes are the only decision
> anchors. This is the fail-closed stance: when the exterior context doesn't confirm,
> the pin is unresolved, never guessed.

This is the crown-jewel upgrade over repocodex's bare ripgrep anchors: it makes
*line-number drift free* (step 2) while keeping *content drift* precise (step 3 unique)
and *ambiguity* explicit (step 3 non-unique → `unresolved`).

> **`line:` without a `version:` token** (a plain line anchor) degrades to the old
> repocodex behavior: the check is **`ok`** iff the line still contains every token of
> the page's co-located `all_of:` anchor (the "expected construct" is the token set from
> the sibling `all_of:` anchor on the same `path`; if there is none, the check is
> `unresolved` — a bare `line:` with no tokens to match against is under-specified). The
> version token is the *preferred* form for grounded pages; `kb_generate` emits it when
> it writes an anchor, and `kb_lint` flags a grounded page that pins a span by bare
> `line:` (no token) as a *weak pin* (warn).

## 3. The result (per page)

`kb_validate` returns, per grounded page:

```json
{
  "rel_path": "invariants/enterprise-grace",
  "status": "stale",                  // "fresh" | "stale" | "unresolved"
  "checks": [
    { "kind": "token",  "path": "internal/memory/service.go", "result": "ok" },
    { "kind": "span",   "line": 482, "result": "stale",
      "version": "kb-lines-v1:sha256:…:…",
      "expected": "enterprise_grace_attempts = 3",
      "actual": "   enterprise_grace_attempts := grace.Attempts",
      "note": "text changed; pin re-resolved to a unique new location" },
    { "kind": "claim",  "subject": "enterprise_grace_attempts", "literal": "3", "result": "stale" },
    { "kind": "symbol", "symbol": "capturePayment", "result": "ok" }
  ],
  "stale_checks": 2
}
```

Each check reports one of `ok` / `stale` / `unresolved`; the page's overall `status` is
the **worst** of its checks (`unresolved` > `stale` > `ok`). A page `status`:

- **`fresh`:** every check is `ok` (span tokens resolved in place or relocated — line
  drift free). The page's why still matches the code. A clean pin-check appends a
  `verified` event to the page's frontmatter (`001-structure` §5.1) — **but only in the
  explicit `kb_validate` call, never during a no-op proof** (which is read-only,
  `003-generate` §3 layer 2) and **routed through the KB write queue** (`008-transport`
  §4).
- **`stale`:** one or more checks are `stale` — the construct's *content* changed (a span
  token advanced, a claim's literal no longer holds). The page is outdated; the result
  names *which* checks are stale and the live value (so the agent fixes the *knowledge*,
  not just the code).
- **`unresolved`:** a check could not be resolved at all (the path does not exist, the
  span token is ambiguous, the symbol is not in the graph, ripgrep found nothing). This
  is stronger than stale — the pinned construct may have been deleted/renamed, or the
  relocation was ambiguous. **Fail closed: never a silent re-point.**

## 4. Fail the turn (KB-2, KB-O3)

The pin-check **fails the turn on divergence** (repocodex's "fails the turn"). The
mechanism and its strictness are controlled by `config.json.strict` + the `--strict`
flag (KB-O3, default):

- **Default (flag, non-strict):** `kb_validate` **reports** drift but does not fail the
  turn. The agent sees `status: stale` / `unresolved` and is expected (by the management
  skill, `009-skill`) to fix the knowledge. This is the interactive default — a stale
  pin is a *signal*, not a *blocker*, during exploration.
- **`--strict` / `config.strict: true` / CI:** `kb_validate` **fails** on any `stale`
  or `unresolved` grounded page. **Exit codes: `0`** if every grounded page is `fresh`;
  **`1`** if any page is `stale` *or* `unresolved` (one code for any divergence — the
  pre-commit hook and CI both key on "non-zero"); **`2`** on an operational error
  (vault missing, store unreachable). The CI / pre-commit hook blocks on non-zero. This
  is the "the why must match the what" gate.

> **The philosophy:** a grounded page is a *contract*. In strict mode (CI, a pre-commit
> hook, a `--strict` invocation) a broken contract **fails the build** — the agent
> cannot commit code that contradicts the documented why. In interactive mode, drift is
> a nudge. The default is **flag, not fail** (KB-O3) so that the pin-check is a useful
> *signal* in the flow, and a hard *gate* when the team opts in (CI / `--strict`).

## 5. The hook / CI integration

`kb_validate` is wired three ways:

1. **Interactive (agent tool / CLI):** `kb_validate [rel_path…]` — validates one or all
   grounded pages; returns the per-page result. The agent runs it **before editing
   grounded code** (session discipline, `009-skill`).
2. **`--strict` CLI:** `mnemonic knowledge validate --strict` — fails the process on
   drift. Usable in a pre-commit hook or a CI step.
3. **CI:** a CI job runs `kb_validate --strict` over all grounded pages. A stale /
   unresolved page fails the pipeline. This is the "the wiki is verified" gate — the single
   synthesized surface is *proven* to match the code, not just asserted to.

**Scope of the check:** by default `kb_validate` checks **all** grounded pages in the
project. `kb_validate <rel_path>` checks one. In a pre-commit hook, the check is
bounded to the pages whose anchors point at the changed files (the hook passes the
changed paths; `kb_validate` filters to the affected grounded pages) — so a commit that
touches `internal/memory/` only validates the pages pinning `internal/memory/`, keeping
the hook fast.

## 6. The "fix the knowledge, not just the code" loop

When `kb_validate` reports a stale / unresolved pin, the correct response is to **update the page** (the
knowledge), not just the code:

- The page's `anchors:` / `claims:` are stale → update them to the new code (the page
  still holds, the pins just moved).
- The page's *why* is stale → the code changed meaning; the page's body must be updated
  to reflect the new why (this is a real knowledge change, often a new decision).

The management skill (`009-skill`) encodes this: **before relying on a grounded page,
`kb_validate` it; if it is stale, fix the knowledge.** This closes the loop: the pin-check
is not just a gate, it is the *feedback* that keeps the brain honest.

## 7. Invariants

INV-1 (the pin-check result carries provenance: which checks, what was expected vs
actual), INV-3 (a `symbol:` check with no graph is `degraded` → falls back to ripgrep
token check, or `unresolved` if the symbol can't be found), KB-INV-2 (grounded pages are
grounded — a grounded page with no resolvable anchor is a finding).

## 8. Open items

KB-O1 (anchor engine default = `auto`: ripgrep for token/line, graph for symbol) and
KB-O3 (default = flag, not fail; fail in CI / `--strict`) are **resolved by default**
here. The one remaining tuning is the pre-commit hook's changed-file scoping (how to
map a changed path to the grounded pages that pin it) — defaulted to "pages whose any
anchor `path` is a prefix of a changed path," refined in the review.
