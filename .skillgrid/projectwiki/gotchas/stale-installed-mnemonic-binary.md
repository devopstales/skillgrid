---
type: Gotcha
title: "Stale installed mnemonic binary masks engine fixes"
status: stable
sources:
  - "code:mnemonic/cmd/mnemonic/main.go"
---

# Stale installed `mnemonic` binary masks engine fixes

The binary on `PATH` (`~/.local/bin/mnemonic`, v0.2.0) lags the repo build. Its
`projectwiki status` omits `by_type`/`orphans`/`stale`/`pending`, and `lint`
prints `clean` even when the vault has 233 findings. Any verification that relies
on the installed binary's output is unreliable until a fresh build is installed.

## Workaround

Build and run the repo binary directly:

```bash
go build -o /tmp/mnemonic-dev ./mnemonic/cmd/mnemonic
/tmp/mnemonic-dev projectwiki status --json
```

## Why it bites

A feature or fix lands in `mnemonic/`, tests pass against the fresh build, but the
operator's installed binary keeps serving the old behavior. The wiki gate's
`status --json` fields then silently disappear, making drift invisible.

## Related

- [[architecture/01-module-map]] — where the engine binary lives
- [[decisions/04-adr-0013-repo-source-of-truth]] — repo wins over installed state
- [[gotchas/projectwiki-generate-has-no-synthesizer]] — another mask that hides drift
- [[concepts/fail-open-floors]] — why degraded runs never crash
