package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, protocolRelPath)); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root containing " + protocolRelPath)
		}
		dir = parent
	}
}

func TestProtocolMarkdownFromRepo(t *testing.T) {
	md := ProtocolMarkdownFromRepo(testRepoRoot(t))
	if md == "" {
		t.Fatal("ProtocolMarkdownFromRepo returned empty string")
	}
	for _, needle := range []string{
		"mem_save",
		"mem_search",
		"mem_context",
		"mem_session_summary",
		"code_search",
		"code_status",
		"code_read",
		"web_cache_lookup",
		"web_cache_save",
		"Memory Taxonomy",
		"Tool Routing",
	} {
		if !strings.Contains(md, needle) {
			t.Errorf("protocol missing %q", needle)
		}
	}
}
