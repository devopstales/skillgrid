# Parallel code review — mnemonic memory checkpoint

**Range:** `6a93d29a..8259669d` (+ fix `4a76da98`)  
**Date:** 2026-10-02  
**Specialists:** Standards, Spec, Edge, Verification-gap, Security, Accessibility, Performance, Red-team  
**Spec specialist:** retried on composer after model-limit failure on first dispatch

## Per specialist (pre-fix)

| Lens | Count | Worst |
|------|------:|-------|
| Standards | 9 | config.Load(".") vs project root |
| Spec | 11 | partial plugin registration / stream reconnect |
| Edge | 10 | lifecycle in EventsSinceWrite; path StripPrivate |
| Verification-gap | 8 | cursor-session-end.sh untested |
| Security | 4 | path StripPrivate; digest fence |
| Accessibility | 2 | aria-label / aria-hidden |
| Performance | 9 | content SELECT / indexes (deferred) |
| Red-team | 8 | **invalid mem_save types in prompt** (HIGH) |

## Deduped verdicts → routing

| Finding | Verdict | Route | Outcome |
|---------|---------|-------|---------|
| Prompt lists feature/refactor/change (invalid IsValidType) | high | fix now | fixed 4a76da98 |
| StripPrivate missing on Path (Go+hook) | medium MULTI | fix now | fixed |
| EventsSinceWrite includes lifecycle | medium | fix now | fixed |
| config.Load(".") for claim | medium MULTI | fix now | fixed |
| Digest/title fence injection | medium | fix now | fixed |
| ExistingTitles uncapped | medium | fix now | fixed (cap 20) |
| Plugin file not in config plugin[] | medium | fix now | fixed (upsertPluginKey) |
| A11y label + diamond | low | fix now | fixed |
| Plugin twins duplicate | low | noise | by design |
| Claim before client ack | low | defer | ADR design |
| Concurrent claim race | medium | defer | later |
| Performance indexes / content SELECT | low–medium | defer | later |
| stream-disconnected full reconnect | false/partial | noise | blueprint: no reconnect |
| SKILLGRID_HTTP_TOKEN on hooks | medium | defer | pre-existing transport |
| noop touchMemoryWrite | medium | defer | later |
| ## Memory vs Memory Index naming | low | defer | docs aligned |
| cursor-session-end.sh CI gap | medium | defer | script covered by tool-call-capture tests |

## Fix wave

SHA: `4a76da98` — gates re-verified (checkpoint/memory/http/setup filters + ToolTimeline vitest).

## Review floor

**MET** for merge of this epic after fix wave. Deferred items are non-blocking or out of scope.
