# Onboarding: write and verify

Step 3 and Step 4 of `references/pipeline/onboarding.md`. Detect and confirm happen there first; this file only writes what was confirmed.

## Step 3: Write

**1. Write `.skillgrid/config.yaml`:**
- Copy `references/patterns/config.yaml` from this skill's directory
- Fill in all detected values, including the `agents:` block (directories, security_boundaries, dependency_policies, architecture_constraints, definition_of_done, engineering_standards) from the preamble facts detected in Step 1 — render these into the AGENTS.md preamble (see `references/patterns/agent-config.md`). 
- If the file already exists, merge: keep existing values the programmer confirmed, update changed ones
- Do NOT overwrite `rules:` sections the programmer has customized

**2. Write `.skillgrid/state.yaml` (create if absent; merge if present):**
- Schema per `skillgrid:_shared/rules/sdd-structure.md` (§ `state.yaml` and the knowledge bundle).
- Initialize: `schema: skillgrid/state/v2`, `changes: {}` (empty map — filled as changes start), `progress.completed_changes: <count of `.skillgrid/change-archive/*/` dirs>`, `progress.blocked_changes: 0`, `constraints_ref: .skillgrid/projectwiki/ASSUMPTIONS.md`. No freeform `notes:` / journal keys in `state.yaml` — session prose goes to `.skillgrid/projectwiki/log.md` (newest-first dated entries).
- If the file already exists (re-onboarding), keep the existing `changes` + `progress` values; only fill missing keys. Never reset `completed_changes`. If the file is still v1 (has `pipeline:` instead of `changes:`), migrate: move `pipeline.current_change` + `pipeline.current_phase` into a single `changes.<slug>` entry if `current_change` is non-empty; remove the `pipeline:` key.

**3. Create the durable-knowledge zone (create if absent; merge if present):**
- Create the wiki directories if absent: `raw/`, `raw/assets/`, `sources/`, `entities/`, `concepts/`, `syntheses/`. Add a `.gitkeep` to each empty dir so git tracks them.
- If `.skillgrid/projectwiki/index.md` is absent, create it: a catalog grouped by directory (ASSUMPTIONS, ARCHITECTURE, glossary, invariants, decisions, sources, entities, concepts, syntheses), with a one-line summary per page. Seed it with the detected facts (stack, entry points, key directories) from Step 1.
- If `.skillgrid/projectwiki/log.md` is absent, create it with the first entry: `## [YYYY-MM-DD] ingest | projectwiki seeded by onboarding`.
- If `.skillgrid/projectwiki/ASSUMPTIONS.md` is absent, create it from the shape in `skillgrid:_shared/rules/sdd-structure.md` — the four tiers `## VERIFIED`, `## INFERRED (HYPOTHESIS)`, `## LOCKED` (the ADR index: one pointer row per in-force decision, path to `decisions/NNNN-slug.md`; `status`/`supersedes`/`date` live in the file's frontmatter, not the row), and `## Open Questions`. Operational rules are NOT a tier here — they live in AGENTS.md's `### Rules` section. On a fresh project the tiers start empty.
- If `.skillgrid/projectwiki/ARCHITECTURE.md` is absent, `skillgrid init` scaffolds it from `references/patterns/architecture.md` (the deterministic structure: header + 17-section skeleton with `<detect>` markers). Onboarding then **fills** it — see *Step 1.7* above. Do not hand-write the 17-section skeleton; let init scaffold it and replace the `<detect>`/`<detect: …>` markers with detected facts. If `skillgrid` is not on PATH, fall back to copying `references/patterns/architecture.md` yourself (filling `{project}`/`{version}`/`{last_updated}` by hand) so the zone is complete.
- If `.skillgrid/projectwiki/glossary/01-business-terms.md` is absent, create it from the **People** and **Work** tables in `references/patterns/glossary.md` (not the skill-internal Genesis table): copy those tables and add `type: Reference` + `title: "Glossary — Business"` frontmatter. This fixes the project's people and workflow vocabulary — most importantly that **programmer** builds and **user** runs the finished program. If the file exists, merge: keep the programmer's rows and add only missing terms.
- Research findings and programmer preferences land in the vault's type folders (`sources/` `syntheses/` `concepts/` `preferences/` — 013), scaffolded by the engine's `projectwiki bootstrap` (001 §8). No standalone findings file: the pre-013 `reference/06-research-findings.md` stub is gone.
- Brownfield migration of the old paths: if a legacy `00-architecture.md` exists and `ARCHITECTURE.md` does not, fold the architecture into `ARCHITECTURE.md` and `git mv` the old file to `.skillgrid/change-archive/`. `reference/00-prd.md` stays where it is (it is the requirements reference); if `ASSUMPTIONS.md` is missing, create it from `references/patterns/assumptions.md` with the product statement and a link to `00-prd.md`. If ADR bodies are inlined in `ASSUMPTIONS.md` and the matching `decisions/NNNN-slug.md` is missing, write each body out to that file and leave only the path in the `## LOCKED` ADR index. Do not fold ADR files or the PRD back into `ASSUMPTIONS.md`. If a legacy `### Locked constraints` section (or `invariants/05-locked-constraints.md`) still exists, move those bullets into AGENTS.md's `### Rules` section (the operational rules now live there), then archive/delete the old section and `git mv` `05-locked-constraints.md` to `invariants/` (or `.skillgrid/change-archive/` if unlocking). A legacy `.skillgrid/projectwiki/glossary/` → `glossary/01/02-*-terms.md`.
- The ADR files, `reference/00-prd.md`, the wiki pages (`entities/`, `concepts/`, `sources/`, `syntheses/`), and the ARCHITECTURE record are created lazily by their owning skills — onboarding only guarantees the zone + wiki directories + `index.md` + `log.md` + the `ASSUMPTIONS.md` skeleton + the glossary exist.

**4. Write AGENTS.md:**
- Render from `references/patterns/agent-config.md` (fill `{project}`, `{tracker_line}`, `{memory_line}`, `{rules_block}`)
- `{tracker_line}` is copied exactly from the `{tracker_line}` table in that file for `ticketing.type`. It points at that tracker's convention file. Do not paraphrase storage paths, ID rules, or CLI flags into the block.
- `{rules_block}` is owned by AGENTS.md's own `### Rules` section (one `-` bullet per rule) — it is the real source of the operational rules, so edit it there. If AGENTS.md has no `### Rules` section yet, render `No operational rules yet — add them to AGENTS.md's `### Rules` section.` (decisions are ADRs in `ASSUMPTIONS.md` § LOCKED, not rules).
- Target decision: if `AGENTS.md` exists → update it. Else if `CLAUDE.md` exists → update it. Else → create `AGENTS.md`.
- **Idempotent upsert** (per `references/patterns/agent-config.md`):
  1. Search the target file for `<!-- skillgrid:start -->`.
  2. **Found** → replace everything from that marker to `<!-- skillgrid:end -->` (inclusive) with the freshly rendered block. Never append a second copy.
  3. **Not found** → append the block at the end of the file.
- **Multi-platform rule:** if both `AGENTS.md` and `CLAUDE.md` exist, write the full block to `AGENTS.md` (source of truth) and put only a one-line pointer in `CLAUDE.md`. Two full blocks = two sources that drift.
  - **Pointer, not symlink.** Skillgrid's choice is a one-line pointer (`See AGENTS.md — the Skillgrid block there is the source of truth.`), not a symlink (`ln -s AGENTS.md CLAUDE.md`), because symlinks can break under git/CI and some editors. A symlink is a fine programmer-side alternative if they insist — but onboarding writes a pointer by default and never creates a second full block.
- The block is idempotent — running onboarding twice never duplicates it.
- **Nested AGENTS.md:** for monorepos or distinct areas (a `payments` module, `apps/api` vs `apps/web`), create a nested `<area>/AGENTS.md` for that area's specific rules — the agent loads root + nested for the folder it's in. The root block stays global and lean; do not push area-specific conventions up into it.

**5. Update `.gitignore`:**
- Ensure `.skillgrid/work/` is gitignored (scratch, ephemeral)
- Ensure `.skillgrid/brainstorm/` is gitignored (visual companion state)
- Do NOT gitignore `.skillgrid/config.yaml`, `.skillgrid/state.yaml`, `.skillgrid/change-specs/`, or the `.skillgrid/` knowledge bundle (committed artifacts)

**6. Tracker bootstrap (if `ticketing.enabled: true`):**
- If `ticketing.type: backlogmd`: run `backlog init "<project>" --integration-mode cli --backlog-dir .skillgrid/projectwiki/backlog --config-location root --zero-padded-ids 3`, verify with `backlog overview`. (The backlog directory lives under the project wiki vault per the skillgrid convention; override with `backlog config set backlog_directory <path>` if needed.) Copy each existing `.skillgrid/projectwiki/decisions/NNNN-slug.md` into `<backlog_directory>/decisions/` (shape in `_shared/adr-records/README.md` → Backlog.md copy).
- If `ticketing.type: gh`: verify `gh` CLI is available (`gh --version`)
- If `ticketing.type: glab`: verify `glab` CLI is available (`glab --version`)
- If `ticketing.type: jira`: verify `jira` CLI is available (`jira config`)
- If `ticketing.type: hermes-kanban`: verify `hermes` CLI is available (`hermes --version`); the `ticketing.kanban.*` block (board slug + assignee) is written by `skillgrid install --agents hermes` — if the block is absent from `config.yaml`, run that install or record the board/assignee from the onboarding override question. Verify the board: `hermes kanban boards list` — board missing → `hermes kanban boards create <slug>` (non-fatal; a missing board must not block onboarding, see `slicing/references/ticketing/hermes-kanban/hermes-kanban.md` § Discovery)
- If `ticketing.enabled: false`: skip all tracker bootstrap

**7. Mnemonic saves (if `mnemonic.enabled: true`):** follow the save shape + session protocol in `skillgrid:_shared/rules/mnemonic-memory.md`. For onboarding specifically:
- `mem_session_start(title: "skillgrid-init/{project}")`
- `mem_save` topic_key: `skillgrid-init/{project}`, type: `architecture` — detected project context (name, stack, repo, deploy)
- `mem_save` topic_key: `skillgrid/{project}/issue_tracker`, type: `config` — tracker id + CLI + storage path
- `mem_save` topic_key: `skillgrid/{project}/testing-capabilities`, type: `config` — testing capabilities table
- `mem_session_summary` + `mem_session_end`
- If the session fails (MCP not wired yet), note it and continue — the saves will be made on first `skillgrid:mnemonic` invocation

**8. Write `.skillgrid/config.yaml` (embedder config, only when Mnemonic is enabled and the programmer chose a non-default provider):**
- Write the `embedder:` block into `config.yaml`. If the file already exists, merge — keep any existing keys (include, exclude, chunk_lines, web_cache, etc.) and add/update only the `embedder:` section.
- The YAML schema (under the `mnemonic:` key, matching `config.EmbedderConfig`):

```yaml
mnemonic:
  embedder:
    provider: ollama            # ollama | onnx | local | external | off
    base_url: http://localhost:11434  # ollama/external only
    model: nomic-embed-code     # model name
    dimension: 768              # output vector dimension
    api_key: ""                 # external only
    # indexing_params / query_params: asymmetric input types (optional)
    # indexing_params:
    #   input_type: passage
    # query_params:
    #   input_type: query
```

- **Ollama** example:
```yaml
mnemonic:
  embedder:
    provider: ollama
    base_url: http://localhost:11434
    model: nomic-embed-code
    dimension: 768
```
- **External API** example:
```yaml
mnemonic:
  embedder:
    provider: external
    base_url: https://api.openai.com/v1
    model: text-embedding-3-small
    dimension: 1536
    api_key: sk-...
```
- **ONNX** (default) — no block needed; the built-in default applies. Write the block only if the programmer explicitly chose ONNX and wants it visible:
```yaml
mnemonic:
  embedder:
    provider: onnx
    model: nomic-embed-code
    dimension: 768
```
- **Off** — write the block to disable vector search explicitly:
```yaml
mnemonic:
  embedder:
    provider: off
```
- **Precedence note:** this repo-local file takes precedence over `~/.skillgrid/config.d/indexing.yaml` (home-local). The home-local file is a per-key fallback — it supplies any key the repo-local file leaves unset. Machine-specific endpoints (e.g. a non-default Ollama host) can live in the home-local file to avoid committing them. Mention this to the programmer if they express concern about committing a base_url.
- If the programmer chose the default (ONNX) and no `indexing.yaml` exists in the repo, skip the write — the built-in default applies without a file.

**9. Commit:**
- `git add .skillgrid/config.yaml` + `.skillgrid/state.yaml` + the `.skillgrid/` knowledge bundle files + the AGENTS.md file
- If `indexing.yaml` was written: `git add .skillgrid/config.yaml`
- `git commit -m "chore: add Skillgrid config + state + artifacts"`

**10. Project init (deterministic finish):**
- Run `skillgrid init` from the project root (add `--force` only if the programmer asked to rebuild the preamble). `skillgrid init` renders the rich AGENTS.md preamble (`references/patterns/agent-config.md`) from the `agents:` block you just filled **and scaffolds `.skillgrid/projectwiki/ARCHITECTURE.md`** from `references/patterns/architecture.md` — the deterministic finish that produces the `# Standards` region and the 17-section architecture skeleton.
- `skillgrid init` only *scaffolds* ARCHITECTURE.md (it writes the skeleton once and keeps an existing file). The **filling** of its `<detect>` markers is onboarding's job (Step 1.7) — do it before you consider onboarding complete, or flag the remaining `<detect>` markers to the programmer.
- If the programmer named extra documentation paths, pass each as `--docs <path>`.
- Do not walk files or call `mem_save` for docs here — the CLI owns ingest, the preamble render, the architecture scaffold, and the code index.
- If `skillgrid` is not on PATH, tell the programmer to install the CLI and re-run `skillgrid init`; fall back to copying `references/patterns/architecture.md` by hand so the zone is complete.

## Step 4: Verify

Show the programmer:
- **Recall evidence (re-runs only)** and **Step 1.5b bootstrap outcome (brownfield only)** — per the SKILL.md verification items (what was recalled; each draft's anchor, confirm/drop outcome, no unconfirmed draft survives)
- The written config (summarized, not full file)
- The `state.yaml` + knowledge bundle zone created
- The generated `.skillgrid/projectwiki/glossary/01-business-terms.md` — confirm it came from `references/patterns/glossary.md` and keeps **programmer** (builds) distinct from **user** (runs the program)
- The AGENTS.md block (with the `### Rules` section as the real source of the operational rules; decisions live in `ASSUMPTIONS.md` § LOCKED)
- The rendered AGENTS.md preamble (`# Standards` region) — confirm it contains the rich sections with no leftover `<detect>` placeholder when the `agents:` config was filled
- The filled `.skillgrid/projectwiki/ARCHITECTURE.md` — confirm the 17 sections are present and the `<detect>` markers are replaced with verified facts for everything that exists (greenfield: only the sections with real content are filled, the rest are honestly marked)
- "Skills now read from `.skillgrid/config.yaml` + `.skillgrid/state.yaml` + `.skillgrid/projectwiki/ASSUMPTIONS.md` + `.skillgrid/projectwiki/ARCHITECTURE.md` + `.skillgrid/projectwiki/index.md` + the `.skillgrid/` knowledge bundle. Run `skillgrid:brainstorm` to start your first project."
