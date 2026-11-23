# Scout Contract — Zero-Inference Code Reading

The single source for how a read-only scout (fresh-context subagent or inline
equivalent) reads source code and reports back. The persona brief lives at
`.agents/agents/scout.md`. Dispatching skills (blueprint research,
codebase-details REVIEW, repo-mapping scouts, execute fan-out scoping) name
this contract and the persona; they do not restate it.

## Citation rule (zero inference)

- Every reported symbol, flag, route, table, or invariant MUST cite the exact
  `[file:symbol_or_flag]` location (e.g. `string_utils.py:sanitize`,
  `pkg/BUILD:java_library`).
- **Never infer from directory or file names.** A directory named
  `refund_service/` proves nothing about a `RefundService` class or an
  `issue_refund` RPC. The citation is the claim; the name is not.
- **Absence is data.** If no explicit invariant or side effect exists in the
  code read, report `None` — not a plausible-sounding one. An invariant a
  scout cannot point at does not exist.
- Before attributing an RPC service, schema, or message type to a local file,
  check build-manifest deps and `import` / `#include` statements. A type
  declared in another package is not this package's interface.

## Chunking cap

Bound each scout reading to **~1,500 lines of source** per scout call:
list top-level definitions (class/func/def/route/flag lines) with a search
tool, cut files into chunks at those boundaries (never mid-body; small files
share a chunk), dispatch one scout per chunk in parallel (at most 5 at a
time, queue the rest), and merge the JSON. Give each chunk the file's imports
and the top-level definition list so visibility can be judged.

## Read order

1. Contracts, schemas, build manifests (`.h/.hpp/.proto/.graphql/.thrift/
   .prisma/.sql` DDL, interface classes, `BUILD`, `Cargo.toml`,
   `package.json`): extract exact DDL table names and exported build targets.
2. Module entry points (`main.*`, `index.*`, `lib.*`, `mod.*`,
   `__init__.py`), then production source alphabetically.
3. Templates and configs: up to 200 lines each (800 in config-only
   directories); exclude `.css/.scss`.
4. Test files (`*_test.*`, `*Test.*`, `test_*.*`, `*.spec.*`): ignore
   entirely — including in hybrid directories.

## Return format

Scouts return JSON, not prose, so the parent can merge mechanically:

```json
{
  "interface": ["[file:symbol]", "..."],
  "invariants": ["[file:symbol_or_flag]", "None"],
  "side_effects": ["[file:symbol_or_flag]", "None"]
}
```

`None` is the literal string in the array, per the absence rule.

## Link hygiene

A scout report (and any doc generated from it) may link to a child contract
or file **only if it exists on disk**. Never emit a speculative link to a
file that does not exist; state it unlinked. A broken link in a generated
contract is a scout failure, not a formatting issue.

## Verification commands

When a scout reports a test/verification command, target syntax is per
build system. Bazel: `//pkg/...` or `//pkg:all` — **never** `//pkg:...` or
`//pkg...`. (Other systems: nearest-manifest command — `cargo test -p`,
`go test ./pkg/...`, `pytest pkg/`, `dotnet test path/to/*.csproj`.)
