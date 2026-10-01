---
name: akca
description: Runs AKCA, an evidence-oriented DAST scanner that crawls an app (HTTP + headless browser), then adaptively tests the discovered surface and verifies each signal with baselines/controls before reporting. Use when you need a thorough scan of a running web app or API with replayable proof per finding (HTTP evidence, payloads, confidence), authenticated or client-rendered apps, or when you want fewer false positives than a broad template fuzzer.
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: akha-security/akca
---

# AKCA

**Announce at start:** "I'm using the skillgrid:akca skill to scan this app and verify the findings."

AKCA is an evidence-first DAST scanner. Unlike a broad fuzzer that hurls every payload at every endpoint, it first **fingerprint** the stack and WAF, **discover** the real attack surface (routes, JS-loaded endpoints, hidden parameters, API ops, access-controlled paths), then **select** tests by context and **verify** each signal with baselines, negative controls, state/identity checks, or OAST callbacks before it is promoted to a finding. Every result carries replayable proof: the HTTP transaction, the payload, a confidence level, and a proof-policy status. A verified finding is closer to a confirmed vuln than a raw matcher hit — but still read the evidence before you report it.

## When to Use

- You want a deep scan of a **running** web app or API with replayable evidence, not just a template match.
- The app is **client-rendered** (SPA, JS-loaded routes) or sits behind **authentication** — AKCA keeps a persistent browser session and honors cookies/headers.
- You have an API definition (OpenAPI/Swagger, GraphQL, Postman, HAR) to import as scope.
- You care about **fewer false positives** and want proof (baseline/control comparison) for each result.

**When NOT to use:**
- Static review of code/config/dependencies — use `skillgrid:owasp-security` or a SAST/SCA tool.
- You need fast broad CVE/exposure coverage over many hosts — use `skillgrid:nuclei`.
- You want a quick, lightweight single-target fuzzer with no browser and fast reports — use `skillgrid:wapiti`.

## Quick Start

```bash
akca --version                                   # Go 1.25+; prebuilt binaries are fine too
akca -u https://example.com                      # default profile is `full`
akca -u https://example.com -f html -o report.html
akca -u https://example.com -f json -o findings.json   # machine-readable for triage
```

Browser-backed checks (client-rendered routes, DOM XSS, JS analysis) need **Chrome, Chromium, or Edge** installed. Confirm the binary and, on a fresh install, that a browser is present.

## Scan Workflow

```
1. Confirm binary + browser         akca --version; headless Chrome available for full/client-rendered
2. Scope the target                 authorized URL; API spec as scope if you have one
3. Choose profile + budget          -m (modules), --request-budget / --time-budget
4. Add auth if the app is gated      -c "cookie=..." or -H "Authorization: Bearer ..."
5. Run and export                   -f json (or html/sarif) -o out
6. Triage with evidence             read proof policy + confidence; replay to confirm
7. Report                             severity-first, with the stored proof
```

### Choosing modules (`-m`)

Comma-separate to run a focused subset; default is `full`.

| Module | Covers |
|--------|--------|
| `full` | all active + passive modules (default) |
| `sql` | SQL and NoSQL injection |
| `xss` | reflected, stored, DOM, blind XSS + client-side checks |
| `rce` | command injection, SSTI, deserialization |
| `api` | API exposure, BOLA/IDOR, BFLA, mass assignment, token checks |
| `graphql` | GraphQL schema and operation checks |
| `ssrf` | SSRF, XXE, out-of-band checks |
| `auth` | authentication, authorization, CSRF, cookie/header checks |
| `passive` | metadata, TLS, security headers, secrets, components |
| `fuzz` | paths, exposed artifacts, traversal |

Prefer the narrowest module set that matches the change (e.g. `-m api` after an API change) — it's faster and more attributable than `full`.

### Budget and pacing

Full scans are coverage-driven, not speed-driven — they take longer on purpose (browser session, replay verification, OAST windows). Bound them so a big app doesn't run unbounded, and so a partial run is reported as **incomplete coverage** rather than a false clean:

```bash
akca -u https://example.com --request-budget 5000 --time-budget 30m
akca -u https://example.com --requests-per-target 200    # budget derived from discovered URL/methods
akca -u https://example.com --rate-limit 5 --concurrency 4   # pace a production target
```

`--request-budget` caps total requests (discovery + retries + redirects); `--requests-per-target` derives the module budget from the discovered surface. A positive `--request-budget` wins. Crawl defaults cap at 1,500 requests / 1,000 pages.

### Authenticated scans

```bash
akca -u https://app.example.com -c "session=YOUR_COOKIE"
akca -u https://app.example.com -H "Authorization: Bearer YOUR_TOKEN"
```

Some authorization checks (true BOLA ownership proofs) need **multiple identities** beyond one session. If the report marks an auth check as needing extra identities, that's a coverage note, not a miss.

### API scope

```bash
akca -u https://api.example.com --api-spec ./openapi.yaml -m api
```

Discovery accepts OpenAPI/Swagger, RAML, Postman, HAR, GraphQL, WSDL, protobuf, and AsyncAPI (incl. supported ZIP bundles). Coverage depends on the imported protocol/operation.

## Triage with Evidence

AKCA already filters hard, so a finding is usually stronger than a nuclei template hit — but read the proof before reporting:

1. **Proof policy / verification status.** Did it pass the baseline/control comparison, or is it a candidate that couldn't be replayed (missing browser, OAST window, or extra identity)? Unverified items are leads.
2. **Confidence level.** Use it to set Confirmed / Likely / Needs verification.
3. **Reachability.** Confirm the endpoint is reachable by the attacker (auth state, network boundary) the way `skillgrid:owasp-security` requires.
4. **Replay to confirm.** `akca replay --finding 42` re-sends the stored request — use it when you want to personally confirm a high-severity item before reporting.

Reports carry CWE and OWASP mappings per finding; cite them in the finding block.

## Key Flags

| Flag | Purpose |
|------|---------|
| `-u`, `--url` | Target URL (required) |
| `-m`, `--modules` | Comma-separated scan profile (`full` default) |
| `-f`, `--format` | `html`, `json`, `markdown`, `csv`, `sarif` |
| `-o`, `--output` | Report file path |
| `-c`, `--cookie` | Session cookie for authenticated scanning |
| `-H`, `--header` | Extra request header (e.g. `Authorization: Bearer …`) |
| `-p`, `--proxy` | Proxy, e.g. `http://127.0.0.1:8080` |
| `--api-spec` | API definition file to import as scope |
| `--request-budget` | Cap total requests (incl. discovery/retries/redirects) |
| `--requests-per-target` | Derive module budget from discovered URL/method combos |
| `--crawler-budget` | Limit discovery requests (default 1500) |
| `--time-budget` | Limit scan duration, e.g. `30m` |
| `--rate-limit` | Requests/sec (default conservative) |
| `--concurrency` | Concurrent workers |
| `--include-linked-api-subdomains` | Include linked subdomains under the same root |
| `replay --finding N` | Replay a stored finding's request |
| `-h` / `--help` | Concise / full option reference |

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "`full` is the right default for every scan" | It's the slowest and noisiest. Pick the modules matching the change; add `full` only when coverage is the explicit goal. |
| "AKCA verified it, so it's definitely a bug" | Verification is strong but not infallible. Read the proof policy and confidence; replay the high-severity ones before reporting. |
| "No browser installed, so just skip browser checks" | You'll miss client-rendered routes and DOM XSS. Install headless Chrome for `full` and client-rendered apps, or note the gap as incomplete coverage. |
| "Unauthenticated scan is fine" | Auth-gated apps hide most of the interesting surface. Pass `-c`/`-H`; some authz checks also need multiple identities. |
| "A partial scan that finished is a clean result" | Budget/time limits report **incomplete coverage**, not a pass. Never present a truncated run as a clean bill. |

## Red Flags

- Reporting a finding with a `candidate`/unverified proof policy as Confirmed without reading the evidence.
- A `full` scan with no `--request-budget`/`--time-budget` against a large or production app.
- Scanning an auth-gated app with no `-c`/`-H` and presenting the limited coverage as complete.
- No headless browser present while claiming DOM/SPA coverage.
- Treating a budget-limited/partial run as a pass instead of incomplete coverage.
- A finding reported without its CWE/OWASP mapping or the stored HTTP evidence.

## Verification

- [ ] `akca --version` succeeded (Go 1.25+), and a headless browser is present when browser/client-rendered checks are needed.
- [ ] The target is an authorized URL; an API spec was imported as scope when available.
- [ ] The module set (`-m`) matches the change/coverage goal, and budgets/pacing were set for the target.
- [ ] Output was exported (`-f json` for triage) and each reported finding was read against its proof policy and confidence.
- [ ] High-severity findings were confirmed by evidence (stored HTTP transaction, payload) and optionally `akca replay`.
- [ ] Any partial/budget-limited run is reported as incomplete coverage, not a clean result.

## References

- [akha-security/akca](https://github.com/akha-security/akca) — engine source, full option reference (`akca --help`), and changelog.
- `FEATURES.md` / `engine/docs/ARCHITECTURE.md` and `FALSE_POSITIVE_AUDIT.md` in the repo — full capability map and verification limitations.
