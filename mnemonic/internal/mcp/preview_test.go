package mcp

import "testing"

func TestPreviewUnfold(t *testing.T) {
	long := stringsRepeat("x", previewRunes+10)
	got := previewContent(long, "", 7)
	if !stringsHas(got, "mem_get_observation id=7") {
		t.Fatalf("preview missing retrieve id: %q", got)
	}
	full := previewContent(long, "7", 7)
	if full != long {
		t.Fatal("unfold did not return full text")
	}
	short := previewContent("hi", "", 7)
	if short != "hi" {
		t.Fatalf("short content changed: %q", short)
	}
}

func stringsRepeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}

func stringsHas(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
