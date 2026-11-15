// cli_family_test.go — the family-first CLI cutover tests (TICKET-02,
// 2026-11-08-mnemonic-mcp-split R2/R4): the top-level tree is
// memory|code|projectwiki + the unchanged non-family commands, the old
// paths (mem/index/mcp/serve) refuse with a did-you-mean pointing at the
// replacement, and bare `code` requires a subcommand (like bare `memory`).
package main

import (
	"strings"
	"testing"
)

// TestFamilyDispatch proves each family verb reaches its runner: a
// real happy-path save under memory, and the family-specific unknown-
// subcommand error under code and projectwiki (the probe is
// fixture-independent: it asserts the dispatch reached the family's
// sub-dispatch, not the top-level "unknown command" fallthrough).
func TestFamilyDispatch(t *testing.T) {
	runCLI(t, "memory", "save", "--title", "T", "--type", "decision", "--content", "C",
		"--project", "p", "--session", "s")

	if err := runCLIError(t, "code", "nosuchsub"); err == nil ||
		!strings.Contains(err.Error(), "unknown code subcommand") {
		t.Fatalf("code nosuchsub: want the runIndex sub-dispatch error, got %v", err)
	}
	if err := runCLIError(t, "projectwiki", "nosuchsub"); err == nil ||
		!strings.Contains(err.Error(), "projectwiki: unknown subcommand") {
		t.Fatalf("projectwiki nosuchsub: want the runProjectWiki sub-dispatch error, got %v", err)
	}
}

// TestBareCodeRequiresSubcommand: `mnemonic code` with no subcommand is a
// usage error naming the subcommands (parity with bare `mnemonic memory`)
// — it no longer starts the index; `mnemonic code index` is the trigger.
func TestBareCodeRequiresSubcommand(t *testing.T) {
	if code := runCLIExit(t, "code"); code != exitUsage {
		t.Fatalf("bare `code`: want exit %d (subcommand required), got %d", exitUsage, code)
	}
	if err := runCLIError(t, "code"); err == nil ||
		!strings.Contains(err.Error(), "code: a subcommand is required") {
		t.Fatalf("bare `code`: want the subcommand-required usage error, got %v", err)
	}
}

// TestOldPathsSuggestTheirReplacement: the removed top-level paths (D3)
// refuse non-zero with a message naming the replacement — no alias, no
// silent success.
func TestOldPathsSuggestTheirReplacement(t *testing.T) {
	cases := []struct {
		cmd  string
		want string
	}{
		{"mem", "mnemonic memory"},
		{"index", "mnemonic code"},
		{"mcp", "mcp"}, // the replacement is `mnemonic <family> mcp`
		{"serve", "mcp"},
	}
	for _, tc := range cases {
		err := runCLIError(t, tc.cmd)
		if err == nil {
			t.Errorf("%q: want a non-zero refusal, got success", tc.cmd)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%q: want the replacement %q named, got %q", tc.cmd, tc.want, err.Error())
		}
	}
}

// TestUnknownProfileRefuses: a bad --profile on the family mcp verb is a
// usage error (exit 2), refused before the server starts (the test does
// not block on the stdio loop).
func TestUnknownProfileRefuses(t *testing.T) {
	if code := runCLIExit(t, "memory", "mcp", "--profile", "bogus"); code != exitUsage {
		t.Fatalf("memory mcp --profile bogus: want exit %d, got %d", exitUsage, code)
	}
}

// TestOldSubcommandPathsRefuse: the old group paths with a subcommand
// also refuse (the rename is at the top level; `mem save` is not an
// alias for `memory save`).
func TestOldSubcommandPathsRefuse(t *testing.T) {
	if err := runCLIError(t, "mem", "save", "no alias"); err == nil {
		t.Fatal("`mem save`: want a refusal, got success")
	}
	if err := runCLIError(t, "index", "status"); err == nil {
		t.Fatal("`index status`: want a refusal, got success")
	}
}
