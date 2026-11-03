package proxy

import "testing"

func TestParseModeValid(t *testing.T) {
	for _, s := range []string{"token", "cache", "passthrough", "TOKEN", " Cache "} {
		m, ok := parseMode(s)
		if !ok {
			t.Errorf("parseMode(%q): expected valid, got ok=false", s)
			continue
		}
		switch s {
		case "token", "TOKEN":
			if m != ModeToken {
				t.Errorf("parseMode(%q) = %q, want token", s, m)
			}
		case "cache", " Cache ":
			if m != ModeCache {
				t.Errorf("parseMode(%q) = %q, want cache", s, m)
			}
		case "passthrough":
			if m != ModePassthrough {
				t.Errorf("parseMode(%q) = %q, want passthrough", s, m)
			}
		}
	}
}

func TestParseModeInvalid(t *testing.T) {
	for _, s := range []string{"", "bogus", "fast", "token-cache"} {
		if _, ok := parseMode(s); ok {
			t.Errorf("parseMode(%q): expected invalid, got valid", s)
		}
	}
}

func TestResolveModePriority(t *testing.T) {
	// Override wins.
	if m := resolveMode("token", "cache", "passthrough"); m != ModeToken {
		t.Errorf("override should win: got %q", m)
	}
	// Env wins over config when override is empty/invalid.
	if m := resolveMode("", "cache", "token"); m != ModeToken {
		t.Errorf("env should win over config: got %q", m)
	}
	// Config wins when override and env are empty/invalid.
	if m := resolveMode("", "cache", ""); m != ModeCache {
		t.Errorf("config should win: got %q", m)
	}
	// Invalid env falls through to config.
	if m := resolveMode("", "passthrough", "bogus"); m != ModePassthrough {
		t.Errorf("invalid env should fall to config: got %q", m)
	}
	// All empty/invalid defaults to cache.
	if m := resolveMode("", "", ""); m != ModeCache {
		t.Errorf("all-empty should default to cache: got %q", m)
	}
}

func TestResolveModeFromEnv(t *testing.T) {
	envs := map[string]string{"CTX_PROXY_MODE": "token"}
	env := func(k string) string { return envs[k] }
	if m := ResolveMode("", "cache", env); m != ModeToken {
		t.Errorf("env override should win: got %q", m)
	}
	if m := ResolveMode("passthrough", "cache", env); m != ModePassthrough {
		t.Errorf("flag override should beat env: got %q", m)
	}
}

func TestHeadSplit(t *testing.T) {
	msgs := []message{
		{Role: "user", Content: "a"},
		{Role: "assistant", Content: "b"},
		{Role: "user", Content: "c"},
		{Role: "assistant", Content: "d"},
	}
	// Frozen head ends at the last user message (index 2) → head = 3.
	if got := headSplit(msgs); got != 3 {
		t.Errorf("headSplit = %d, want 3", got)
	}
	// No user message → head 0 (all mutable).
	if got := headSplit([]message{{Role: "assistant", Content: "x"}}); got != 0 {
		t.Errorf("headSplit(no user) = %d, want 0", got)
	}
	// Single user message → head 1.
	if got := headSplit([]message{{Role: "user", Content: "only"}}); got != 1 {
		t.Errorf("headSplit(single user) = %d, want 1", got)
	}
	// Case-insensitive role.
	if got := headSplit([]message{{Role: "User", Content: "u"}}); got != 1 {
		t.Errorf("headSplit(User) = %d, want 1", got)
	}
}
