# projectwiki CLAIMS

## Claim

- Claim = must-hold assertion: `subject` + `literal` + optional `path`. The literal must sit in the matched region of its owning anchor; an unpathed claim is checked against the file(s) of the page's anchors (006-validate).
- `claims[]` is REQUIRED on `InvariantContract` pages; optional elsewhere — an unanchored page is valid knowledge, it just doesn't arm the pin-check.
- **The subject is the claim's identity.** Revise keeps the subject; retract deletes it. The same assertion re-worded under a new subject is a new claim — the old attestation does not carry over.

## Anchor tiers — pin at the strongest that fits

| Tier | Frontmatter | Catches | Use for |
|---|---|---|---|
| Span token | `{ path, line, version }` | content change at the pinned span — **moved** (ok, line-drift free) vs **changed** (stale, live value reported) vs **ambiguous** (unresolved, fail closed) | behavioral claims: the logic must stay *this* logic |
| Symbol | `{ path, symbol, called_from? }` | API-surface move/rename via the code graph (check is `degraded: true` when the graph is absent) | function/type-level contracts |
| Token list | `{ path, all_of: [...] }` | literal disappearance | stable, low-churn anchors: config keys, table/column names, error strings, flag names |

An `all_of`-only page is a `weak_pin` (warn) — a refactor that keeps every literal passes silently. Upgrade the behavioral anchors to span or symbol; keep tokens for the stable tokens. A bare `line` with no `version` is also `weak_pin`.

## Span token — format and mint

Format: `projectwiki-lines-v1:sha256:<spanHash>:<metaB64>`

- `<spanHash>` — sha256 hex of the pinned lines joined with `\n`.
- `<metaB64>` — base64url **without padding** of a JSON object with exactly these keys, in this order: `selectedLineCount`, `firstSelectedLineHash`, `lastSelectedLineHash`, `precedingContextLineCount`, `precedingContextHash`, `followingContextLineCount`, `followingContextHash`.
- Each `*LineHash` — sha256 hex of the raw line (no trailing newline).
- `precedingContextHash` / `followingContextHash` — sha256 hex of up to **3** context lines before/after the span joined with `\n` (fewer at file edges; zero lines → sha256 of the empty string).
- `line` — 1-based start line of the span in the file at mint time (a hint, not a pointer — resolution relocates via the context hashes).

Mint (args: file, first line, last line inclusive):

```bash
python3 - "$FILE" "$FIRST" "$LAST" <<'EOF'
import hashlib, json, base64, sys
path, first, last = sys.argv[1], int(sys.argv[2]), int(sys.argv[3])
lines = open(path).read().split("\n")
sel = lines[first-1:last]
h = lambda s: hashlib.sha256(s.encode()).hexdigest()
p, f = lines[max(0, first-1-3):first-1], lines[last:min(len(lines), last+3)]
meta = {"selectedLineCount": len(sel), "firstSelectedLineHash": h(sel[0]),
        "lastSelectedLineHash": h(sel[-1]), "precedingContextLineCount": len(p),
        "precedingContextHash": h("\n".join(p)), "followingContextLineCount": len(f),
        "followingContextHash": h("\n".join(f))}
print(f"projectwiki-lines-v1:sha256:{h(chr(10).join(sel))}:"
      f"{base64.urlsafe_b64encode(json.dumps(meta, separators=(',', ':')).encode()).decode().rstrip('=')}")
EOF
```

The engine never rewrites the token (attestation, not a pointer). After minting, `projectwiki index` + `projectwiki validate <page>` must report the span `ok` before the pin counts.

## Re-attestation — sparse, per the report

`projectwiki validate` reports one `Check` per anchor/claim: `kind` (token|span|symbol|claim), `result` (ok|stale|unresolved), and `expected`/`actual` (the live value for a stale span). Work only what the report names:

- **stale span** — re-read the live code at the reported site. Behavior unchanged → re-pin (mint a token over the current lines). Behavior changed → update the why first, then re-pin to the new behavior, or retract.
- **unresolved** — the span is ambiguous or gone. Relocate it deliberately (new `line` + minted token) or drop the anchor; never guess.
- **stale `all_of` token / claim literal** — the literal is gone: confirm (the fact moved — re-anchor), revise-same-subject, or retract.
- **ok** — carry the claim forward untouched. Do not re-word, re-order, or "refresh" it; churn destroys the attestation trail.

Freshness is per run: a claim is fresh only when the current validate run says so — not because it was fresh last session.

# projectwiki MERMAID

- Structural diagrams only; Mermaid for flow/sequence/state/deps; SVG fallback for layout/mockups.
- Terminal renders ASCII; SVG uses `currentColor` + theme vars, no background.
