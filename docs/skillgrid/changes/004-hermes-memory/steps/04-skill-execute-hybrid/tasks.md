# Tasks: 004-hermes-memory — Step 04-skill-execute-hybrid

> Goal: Sandboxed `use_skill` + hybrid BM25+cosine (sqlite-vec).
> Depends on: 02-fact-tools, 03-skills-registry

> Change-level Review Workload Forecast: see `../01-facts-schema/tasks.md`
> Decision needed before apply: Yes · Chained PRs recommended: Yes · Chain strategy: pending · 400-line budget risk: High

## Execution

- [ ] 04.1 RED (Documentation-like paths / shell sandbox): `../evil.sh` or unknown language → error, no exec; timeout/cwd-jail/allowlist reject host-wide escape — expect fail until sandbox lands.
- [ ] 04.2 RED (Mnemonic tool surface): assert `use_skill` (+ hybrid search mode) registered **and** `mem_save` still works; soft-deleted facts stay absent from hybrid fact search.
- [ ] 04.3 Make 04.1 pass — create `skillgrid-cli/internal/mnemonic/skills/execute.go` — allowlisted runners, timeout, no network, cwd=skill dir; log `skill_usage`.
- [ ] 04.4 Make 04.2 pass — create `skillgrid-cli/internal/mnemonic/mcp/tools_skills_exec.go` with `use_skill` + hybrid BM25+sqlite-vec cosine for skills (and facts when mode=hybrid).
- [ ] 04.5 Modify `skillgrid-cli/internal/mnemonic/mcp/server.go` — wire `RegisterFactTools` + skill registry + exec registrars without dropping `mem_*`/`code_*`/`web_*`.
- [ ] 04.6 Cover WHAT: sandboxed `use_skill` returns captured IO and logs usage.
- [ ] 04.7 Cover WHAT: hybrid BM25 + sqlite-vec cosine for skills (and facts when mode=hybrid).
- [ ] 04.8 Cover WHAT edge: bad language / timeout / path escape → error, no host-wide exec.
