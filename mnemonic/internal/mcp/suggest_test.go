package mcp

import "testing"

func TestSuggestTool(t *testing.T) {
	registered := []string{"code_search", "code_grep", "code_read", "code_status", "code_index"}

	cases := []struct {
		query string
		want  string
	}{
		{"code_scan", "code_read"},   // closest: dist 3 (code_grep/code_search/code_status all dist 4)
		{"code_seach", "code_search"}, // one-char transposition
		{"mem_savev", "mem_save"},     // extra char
		{"zzz", ""},                   // nothing close
	}

	for _, tc := range cases {
		// Add the query's siblings to make the test realistic
		all := append([]string{}, registered...)
		all = append(all, "mem_save")
		got := suggestTool(tc.query, all)
		if got != tc.want {
			t.Errorf("suggestTool(%q) = %q, want %q", tc.query, got, tc.want)
		}
	}
}

func TestSuggestTool_NoMatch(t *testing.T) {
	// A name that is not close to anything returns empty.
	if got := suggestTool("totally_different_name", []string{"code_search", "mem_save"}); got != "" {
		t.Errorf("expected empty suggestion, got %q", got)
	}
}

func TestExtractToolName(t *testing.T) {
	cases := []struct {
		msg  string
		want string
	}{
		{"tool 'code_scan' not found: tool not found", "code_scan"},
		{"tool 'code_scan' not found", "code_scan"},
		{"no quotes here", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := extractToolName(tc.msg); got != tc.want {
			t.Errorf("extractToolName(%q) = %q, want %q", tc.msg, got, tc.want)
		}
	}
}

func TestToolSuggestion(t *testing.T) {
	// code_scan → code_read (dist 3, within maxEditDistance)
	registered := []string{"code_search", "code_grep", "code_read", "mem_save"}
	if got := toolSuggestion("code_scan", registered); got != "; did you mean code_read?" {
		t.Errorf("toolSuggestion(code_scan) = %q, want %q", got, "; did you mean code_read?")
	}
	// No close match → empty
	if got := toolSuggestion("zzz", registered); got != "" {
		t.Errorf("expected no suggestion for zzz, got %q", got)
	}
}
