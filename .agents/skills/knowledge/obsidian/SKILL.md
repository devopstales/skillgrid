---
name: obsidian
description: "Read, search, create, and edit notes in an Obsidian vault: filesystem-first note work plus [[wikilinks]]. Use when the user points at a vault, or alongside wiki curation."
license: MIT
metadata:
  author: Teknium, Hermes Agent (ported; see ADR-0015)
  version: "1.0"
  part-of: skillgrid
---

# Obsidian Vault

**Announce at start:** "I'm using the skillgrid:obsidian skill to work in this vault."

## Overview

Filesystem-first Obsidian vault work: reading notes, listing notes, searching filenames and contents, creating notes, appending, targeted edits, and `[[wikilinks]]`. The vault is plain markdown on disk — no database, no plugin API needed. This skill pairs with `skillgrid:llm-wiki`: the wiki skill owns curation judgment, this skill owns vault mechanics (and any `~/wiki`-style vault doubles as an Obsidian vault out of the box).

Ported from the Hermes `obsidian` skill under MIT (see ADR-0015). Behavioral delta from upstream: Hermes tool names translated to skillgrid tools (table below); the `OBSIDIAN_VAULT_PATH` convention is kept.

## When to Use

- User points at an Obsidian vault (or `~/wiki`, or the project `.wiki/` as read context) and asks to read, search, create, or edit notes
- Curating wiki pages (`skillgrid:llm-wiki`) — vault mechanics live here
- Following or adding `[[wikilinks]]` between notes

**When NOT to use:** For project knowledge that belongs in `.skillgrid/` (decisions, ADRs, specs) — that is the repo source of truth, not the vault. For code search — use `skillgrid:mnemonic` code index. For writing the project `.wiki/` projection — only the ADR-0014 compiler writes there.

## Tool mapping (Hermes → skillgrid)

| Hermes (upstream wording) | skillgrid tool |
|---|---|
| `read_file` | Read (absolute path; paginated) |
| `search_files` (files) | Glob (`*.md` under the vault path) |
| `search_files` (content) | Grep (content regex, `*.md` glob) |
| `write_file` | Write (absolute path, full content) |
| `patch` (anchored edit) | Edit (exact-match replacement) |
| `terminal` (path resolution only) | Bash (resolving the vault path only) |

Prefer file tools over shell for everything except resolving the vault path itself. Vault paths may contain spaces — another reason to prefer file tools over shell quoting.

## The Process

### 1. Resolve the vault path

Convention: `OBSIDIAN_VAULT_PATH` environment variable. If unset, ask the user; fallback `~/Documents/Obsidian Vault`. Resolve to a concrete absolute path once, then pass absolute paths everywhere — never a `$VAR` string.

### 2. Read

Read with pagination. Before creating anything, search filenames and contents for the topic — vault work duplicates silently when you skip this.

### 3. Create

Write full markdown content. Link related notes with `[[Note Name]]` at creation time — an unlinked note is invisible in Graph View.

### 4. Append and edit

Prefer Edit with stable anchor context (a heading, a trailing block) over whole-file rewrites. For a context-free append, read-then-rewrite beats a fragile patch.

## Wikilinks

`[[Note Name]]` between notes; minimum 2 outbound links per new note (matches the `skillgrid:llm-wiki` page rule). Images/attachments go in the vault's attachment folder and are referenced `![[image.png]]`.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "I'll just cat the note" | Read gives line numbers + pagination; vault notes grow. Use the tool. |
| "grep -r is fine" | Grep tool with a `*.md` glob is the same search without shell-quoting risk on spaced paths. |
| "I'll link it later" | Later never comes — unlinkable notes rot. Link at creation. |
| "This belongs in the vault" (for ADRs/specs) | Project decisions live in `.skillgrid/` (repo source of truth). The vault is for notes and curation, not decisions. |

## Red Flags

- Passing `$OBSIDIAN_VAULT_PATH` unresolved into a file tool (resolve first)
- Shell heredocs/`echo` for note content (quoting bugs; use Write)
- Creating a note without searching for an existing one first
- Writing to the project `.wiki/` by hand (compiler-owned; ADR-0014)
- Attachments scattered outside the attachment folder

## Verification

- [ ] Vault path resolved to an absolute path before any note op
- [ ] Searched filenames + contents before creating (no duplicate)
- [ ] New note carries ≥ 2 `[[wikilinks]]` (or a recorded reason)
- [ ] Edits used anchored Edit; appends verified by re-read
- [ ] Nothing project-authoritative (ADRs, specs, constraints) landed in the vault instead of `.skillgrid/`
