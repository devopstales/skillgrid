# Review and verify artifacts

Paths produced by `/sdd-verify` and `/sdd-review` for gates and archive.

## Verify (Stage 1)

| Artifact | Path |
| --- | --- |
| Report | `openspec/changes/<change>/verify-report.md` |
| State | `.skillgrid/state/<change>/verification_status` → `passed` |
| Security tag | `.skillgrid/state/<change>/security_sensitive` (when classifier matches) |
| Checkpoint | `.skillgrid/tasks/checkpoints.log` — `verify-pass` |
| Persona | `.skillgrid/tasks/research/<change>/heimdall-security-review.md` (required when sensitive) |

Classifier: `.skillgrid/scripts/classify-security-sensitive.sh --change <change>`

## Review (Stage 2)

| Artifact | Path |
| --- | --- |
| Consolidated report | `openspec/changes/<change>/reviews/<timestamp>-review.md` |
| State | `.skillgrid/state/<change>/review_status` → `approved` \| `changes_requested` |
| Trivy JSON | `.agents/reviews/<change>/trivy-report.json` |
| Pattern scan JSON | `.agents/reviews/<change>/vulnerability-scan.json` |
| TrueCourse | `.agents/reviews/<change>/truecourse-analyze.txt`, `truecourse-violations.txt`, `truecourse-summary.json` (via `run-truecourse-review.sh`) |
| Checkpoint | `review-pass` in checkpoints.log |

## Gates

- `sdd-gate.sh review` — requires verify PASS first
- `sdd-gate.sh archive` — requires review APPROVED; warns if `security_sensitive` without scan artifacts

Config: [`.skillgrid/config.json`](04-config-reference.md) → `verify.security`, `review.security`, `review.architecture`
