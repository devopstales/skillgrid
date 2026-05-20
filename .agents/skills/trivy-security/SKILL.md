---
name: trivy-security
description: >
  Run Trivy filesystem scans via MCP for sdd-review Stage A. Maps CVEs, secrets, and
  misconfigurations to CRITICAL/IMPORTANT for merge gates.
license: Apache-2.0
metadata:
  author: aiskillgrid
  version: "1.0"
  triggers:
    - sdd-review Stage A
    - trivy_scan
---

# Trivy Security Scan

## When to use

- Invoked by `sdd-review` when `review.security.trivy_scan` is true or `--security` is set.
- Not a substitute for `vulnerability-scanner` — use that skill when Trivy MCP/binary is unavailable.

## Prerequisites

- MCP: `.configs/mcp/trivy.json` (`trivy mcp`)
- `trivy` binary in PATH
- Change ID `{change}` for artifact paths

## Execution

1. Create output dir: `mkdir -p .agents/reviews/{change}`

2. Call MCP tool **`trivy_scan_filesystem`** (read server tool schema before calling):

| Parameter | Typical value |
| --- | --- |
| `target` | Repository root (`.`) |
| `scanType` | `vuln`, `secret`, `config` (per project policy) |
| `severities` | From config `review.security.trivy_severity` — default CRITICAL, HIGH, MEDIUM |
| `outputFormat` | `json` |

3. Save raw JSON to `.agents/reviews/{change}/trivy-report.json`

4. Parse into markdown table for Stage D consolidation:

```markdown
## Security Scan (Trivy)

| Severity | Count | Notable |
|----------|-------|---------|
| CRITICAL | N | … |
| HIGH     | N | … |
```

## Verdict mapping

Read `.skillgrid/config.json`:

| Config | Behavior |
| --- | --- |
| `fail_on_cve: true` | Any CRITICAL/HIGH CVE → report as **CRITICAL** for review verdict |
| `fail_on_secret: true` | Any detected secret → **CRITICAL** |

Include package name, installed version, fixed version, CVE ID, and file:line for secrets.

## On failure

| Condition | Action |
| --- | --- |
| MCP unavailable | WARN; delegate to `vulnerability-scanner` (Stage A.5) |
| Binary missing | WARN; run Stage A.5 |
| Scan timeout | ERROR; retry once or fall back |

## Output

Return structured summary for parent `sdd-review`:

- `artifact_path`: `.agents/reviews/{change}/trivy-report.json`
- `critical_count`, `high_count`, `medium_count`
- `blocking`: boolean (from fail_on_* rules)

## See also

- `security-review` — manual OWASP pass on diff
- `vulnerability-scanner` — pattern/secret/deps fallback
- Docs: `docs/13-mcp-servers.md` (Trivy MCP)
