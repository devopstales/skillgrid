# Edge-case hunter — mnemonic-project-init

**Range:** `f739d371..17652346`  
**Lens:** writeBootFile, ingestPaths jail, symlink/--docs, empty files, chmod, second-init merges, index failure paths  
**Grade:** A (exhaustive path walk; only unhandled branches reported)

```json
[
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:113-154",
    "trigger_condition": "--docs path is in-tree symlink to outside file",
    "guard_snippet": "real, err := filepath.EvalSymlinks(absPath); jail Rel(absDir, real)",
    "potential_consequence": "Jail bypass; outside file bytes saved into project memory",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:118-122",
    "trigger_condition": "--docs path Rel is '.' (project root)",
    "guard_snippet": "if rel == \".\" || rel == \"\" { return jail-or-reject error }",
    "potential_consequence": "Walk ingests entire tree including secrets and .git",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:119",
    "trigger_condition": "In-project file/dir name starts with '..' e.g. ..foo",
    "guard_snippet": "outside := rel == \"..\" || strings.HasPrefix(rel, \"../\")",
    "potential_consequence": "Valid in-project --docs path falsely rejected as outside",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:131-145",
    "trigger_condition": "--docs targets symlink directory (Stat follows)",
    "guard_snippet": "info, err := os.Lstat(absPath); reject ModeSymlink or EvalSymlinks+re-jail",
    "potential_consequence": "WalkDir skips symlink root; silent zero ingest, no Errors",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:79-81",
    "trigger_condition": "Default root README.md is symlink or non-regular",
    "guard_snippet": "errs = append(errs, root+\": not a regular file\")",
    "potential_consequence": "Silent skip: neither Skipped nor Errors lists the path",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:63-65",
    "trigger_condition": "Symlink file under docs/ or artifacts walk",
    "guard_snippet": "if d.Type()&fs.ModeSymlink!=0 { errs=append(...); return nil }",
    "potential_consequence": "Symlinked docs silently omitted from ingest and report",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:133-141",
    "trigger_condition": "WalkDir walkErr or multiple saveErr under --docs dir",
    "guard_snippet": "if walkErr!=nil { errs=append(...)} ; collect all saveErrs",
    "potential_consequence": "Walk failures dropped; only last saveErr kept in errMsg",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:167-170",
    "trigger_condition": "File was ingested then becomes empty/whitespace on re-init",
    "guard_snippet": "on empty: mem delete-or-supersede topicKey, or Skipped+note",
    "potential_consequence": "Second-init leaves stale observation; Errors only, no upsert",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_boot.go:261-268",
    "trigger_condition": "AGENTS.md has skillgrid:start without matching :end",
    "guard_snippet": "if start>=0 && end<=start { return error mismatched sentinel markers }",
    "potential_consequence": "Every re-init appends another sentinel; duplicates accumulate",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_boot.go:239-258",
    "trigger_condition": "Preamble start marker present without valid end after it",
    "guard_snippet": "if start>=0 && end<=start { return error mismatched preamble markers }",
    "potential_consequence": "Second-init inserts another preamble region; markers multiply",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_boot.go:91-96",
    "trigger_condition": "AGENTS.md WriteFile OK then ensureClaudePointer WriteFile fails",
    "guard_snippet": "write CLAUDE pointer first, or rollback AGENTS on pointer error",
    "potential_consequence": "Fatal exit after AGENTS already mutated; partial boot state",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_boot.go:102-122",
    "trigger_condition": "AGENTS.md exists as symlink to path outside project",
    "guard_snippet": "if fi.Mode()&os.ModeSymlink!=0 { return err refuse symlink boot }",
    "potential_consequence": "WriteFile follows link and overwrites outside boot target",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_boot.go:157-161",
    "trigger_condition": "ASSUMPTIONS.md exists but chmod makes it unreadable",
    "guard_snippet": "if !os.IsNotExist(err) { return \"\", err } // surface EACCES",
    "potential_consequence": "Silent default Rules; locked constraints omitted from sentinel",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_cmd.go:120-125",
    "trigger_condition": "RunCodeIndex returns non-nil err with FilesIndexed > 0",
    "guard_snippet": "res.Indexed = indexed; if indexErr != nil { append Errors }",
    "potential_consequence": "Partial index progress reported as Indexed 0",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:33-36",
    "trigger_condition": "ingestPaths SessionStart succeeds; never SessionEnd",
    "guard_snippet": "defer mem.SessionEnd(ctx, sid, \"skillgrid-init\")",
    "potential_consequence": "Each init leaves an open dangling session row",
    "kind": "edge"
  }
]
```
