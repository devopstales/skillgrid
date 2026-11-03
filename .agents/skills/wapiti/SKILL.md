---
name: wapiti
description: Runs Wapiti, a black-box DAST fuzzer that crawls a deployed web app, extracts links and forms, and attacks scripts with payloads looking for errors and abnormal behavior. Use when you need a fast, lightweight, easy-to-run scan of a running web app for SQLi, XSS, file disclosure, command injection, XXE, CRLF, open redirects, and security headers; or when you want a quick report with no browser or template library required.
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: wapiti-scanner/wapiti
---

# Wapiti

**Announce at start:** "I'm using the skillgrid:wapiti skill to scan this app and triage the report."

Wapiti is a **black-box fuzzer**: it does not read source code. It crawls the deployed app like a user (following links, extracting and submitting forms), then **attacks** the scripts it finds — sending payloads and looking for error messages, special strings, or abnormal responses. It is the fastest path to a first security report on a running target: no browser required, no template library to download, a single command. It is less evidence-heavy than AKCA and less broad than Nuclei, so treat every hit as a fuzzer lead and confirm reachability before reporting.

## When to Use

- You want a **fast, lightweight** DAST report on a running web app with minimal setup.
- Quick pre-release or CI check for classic injection flaws (SQLi, XSS, command exec, file disclosure, XXE, CRLF) plus headers/cookies.
- You need a scan with **no browser and no template dependency** (Wapiti is pure Python; only optional `sslscan` for TLS).
- Enumerating CMS modules, WordPress plugins/themes, or known CVE fingerprints via Wappalyzer.

**When NOT to use:**
- Client-rendered SPAs or JS-loaded routes that a headless browser is needed to reach — use `skillgrid:akca`.
- Broad known-CVE / exposure / network coverage over many hosts — use `skillgrid:nuclei`.
- Deep evidence-backed verification (baseline/control replay, OAST) per finding — use `skillgrid:akca`.

## Quick Start

```bash
wapiti -u https://example.com                     # default attack set
wapiti -u https://example.com -o report.html      # report to a file (html/xml/json/txt/csv/md)
wapiti -u https://example.com -m sql,xss,exec     # only the modules you need
wapiti -u https://example.com --scope page        # scan one page, not the whole app
```

Wapiti needs **Python 3.12, 3.13, or 3.14** (`pip install wapiti3` or `uv tool install wapiti3`). Confirm the version before scanning. If `wapiti` is not in `$PATH`, check `~/.local/bin/wapiti` (uv tool install) or `~/.cargo/bin/wapiti` — add that directory to `$PATH` or use the full path.

```bash
wapiti --version
```

## Scan Workflow

```
1. Confirm binary                    wapiti --version  (or ~/.local/bin/wapiti; Python 3.12-3.14)
2. Scope the target                  live, authorized URL; pick scope width
3. Pick the attack set               -m modules, or default
4. Add auth if the app is gated       --basic-auth, -c cookie, or --form-script
5. Run + export                       -o report.json (or html/md/sarif-adjacent txt)
6. Triage every hit                   fuzzer lead → confirm reachability before reporting
7. Report                             severity-first, with the matched evidence
```

### Choosing scope (`--scope`)

Scope controls how far Wapiti crawls before attacking:

| Scope | Behavior |
|-------|----------|
| `url` (default) | The target and everything reachable from it |
| `page` | The single target page only |
| `folder` | The target directory |
| `domain` | The whole domain (broad, slower) |

Start at `page` or `folder` to keep a scan fast and attributable; widen to `domain` for full coverage. Restrict further with `--start-url`, `--exclude` (e.g. drop the logout URL), and `--exclude-param`.

### Choosing attack modules (`-m`)

Comma-separate a subset, or let Wapiti run its default set. Common modules:

| Module | Covers |
|--------|--------|
| `sql`, `timesql` | Error/boolean/time-based SQL injection |
| `xss`, `permanentxss` | Reflected + permanent (stored) XSS |
| `exec` | Code execution / command injection |
| `file` | Path traversal / file inclusion |
| `xxe` | XML external entity injection |
| `crlf` | CRLF injection |
| `ldap` | LDAP injection |
| `redirect` | Open redirects |
| `upload` | File upload vulnerabilities |
| `methods` | Uncommon HTTP methods (e.g. PUT) |
| `csrf` | Forms missing CSRF protection / weak anti-CSRF tokens |
| `http_headers` | Missing/weak HTTP security headers |
| `cookieflags` | Secure / HttpOnly cookie flags |
| `csp` | Missing or weak Content-Security-Policy |
| `ssl` | TLS configuration (requires `sslscan`) |
| `https_redirect` | HTTP→HTTPS redirects |
| `backup`, `nikto`, `buster` | Exposed copies, known vulns, directory/file enumeration |
| `htaccess` | Misconfigured `.htaccess` |
| `shellshock`, `log4shell`, `spring4shell` | Named CVE checks |
| `takeover` | Subdomain takeover |
| `cms`, `wapp`, `wp_enum` | Tech/CMS fingerprinting + related CVEs, WP plugin/theme enumeration |
| `ssrf` | Server-side request forgery |
| `brute_login_form` | Login brute force (dictionary) |
| `network_device`, `printer` | Network device / printer detection |

Run the narrowest set that matches the change (e.g. `-m sql,xss,exec` after a form/handler change); run the default set for a full first pass.

### Authenticated scans

```bash
wapiti -u https://app.example.com --basic-auth "user:pass"
wapiti -u https://app.example.com -c "session=YOUR_COOKIE"
wapiti -u https://app.example.com --form-script login.py   # custom Python for complex logins
```

For complex flows, import cookies from your browser or use the `wapiti-getcookie` tool.

### Limits and safety

```bash
wapiti -u https://example.com --max-time 30m      # cap total scan time
wapiti -u https://example.com --concurrency 10    # parallel HTTP requests
wapiti -u https://example.com --depth 3           # crawl depth
wapiti -u https://example.com --cookie -o report.json --info  # verbose report
```

### Report formats

`-o` writes the report as `html`, `xml`, `json`, `txt`, `csv`, or `md`. Export `json` or `xml` for machine triage; `html`/`md` for a human-readable handoff.

## Triage a Hit

Wapiti is a fuzzer — it fires payloads and matches on error strings and abnormal responses, so false positives are common. Apply the same rubric as `skillgrid:owasp-security` ("Before Reporting a Finding"):

1. **Is the parameter attacker-controlled and reachable?** A fuzzer match on an internal or unreachable value is not exploitable.
2. **Is it a real response or a coincidental error string?** Error-based SQLi and file-inclusion matches are the noisiest. Confirm the response actually reflects the injected payload.
3. **Blast radius.** Who can trigger it, what they get, which boundary it crosses.
4. **Can the attacker perform every step?** If a step needs a capability the attacker doesn't have, it's "Needs verification."

Report severity by exploitability, not by the module's nominal class. Mark Confirmed / Likely / Needs verification and say what you couldn't see.

## Store results in mnemonic

After a successful scan, persist the findings so they survive the session and become queryable memory. Use the mnemonic MCP tools (fail-open: if the store is unavailable, log and continue — a scan error never blocks the caller):

1. `scan_start` — open a scan row for `tool: "wapiti"`, the target URL, and the modules used. It returns the `scan_id` and the raw-artifact path.
2. Write the scan's `-o json` (or `-o xml`) output to the raw-artifact path.
3. `scan_store_findings` — pass the `scan_id` and `tool: "wapiti"`; the parser normalizes each report entry's type/info/url into a stable `dedup_hash` per finding.

```
wapiti --url <url> -o json > /tmp/wapiti-report.json
mnemonic: scan_start(tool="wapiti", target="<url>")        -> scan_id, raw_path
mnemonic: scan_store_findings(scan_id, tool="wapiti")      # parses raw, upserts findings
```

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "Wapiti hit it, so it's vulnerable" | It's a fuzzer matching on error strings. Error-based SQLi/file-inclusion hits are the most false-positive-prone; confirm the payload is actually reflected. |
| "No browser means it sees the whole app" | It only sees what a crawler can reach. SPAs and JS-loaded routes are invisible to Wapiti — use `skillgrid:akca` for those. |
| "`domain` scope is the safe default" | It's the broadest and slowest. Start at `page`/`folder`; widen deliberately. |
| "I'll just read the terminal" | Export `-o report.json` (or `xml`) and triage the records so no hit is missed and each is attributable. |
| "Default modules are enough for every change" | Narrow with `-m` to the modules matching the change; run the full default set only for a first broad pass. |

## Red Flags

- Reporting an error-based SQLi or file-inclusion hit as Confirmed without confirming the payload is reflected in the response.
- `--scope domain` on a large app with no `--max-time`/`--depth` cap.
- Scanning an SPA / client-rendered app and claiming full coverage with no browser.
- Scanning an auth-gated app with no `--basic-auth`/`-c`/`--form-script`.
- Triaging from scrolling terminal output instead of the exported `json`/`xml` report.
- A hit reported without reachability + severity + confidence.

## Verification

- [ ] `wapiti --version` succeeded (Python 3.12–3.14).
- [ ] The target is a reachable, authorized URL; scope width (`--scope`) and limits (`--max-time`, `--depth`) were chosen deliberately.
- [ ] The module set (`-m`) matches the change/coverage goal, not an unscoped default.
- [ ] Output was exported (`-o report.json` or `xml`) and read for triage.
- [ ] Every reported hit states severity, the concrete match, reachability, and confidence.
- [ ] No error-based SQLi/file-inclusion hit is reported as Confirmed without a confirmed reflected payload.

## References

- [wapiti-scanner/wapiti](https://github.com/wapiti-scanner/wapiti) — source, `wapiti -h` / manpage, and the module list.
- [Wapiti wiki](https://github.com/wapiti-scanner/wapiti/wiki) — exhaustive option reference and worked examples.
