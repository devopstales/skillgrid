package memory

import "testing"

func TestShouldPromote(t *testing.T) {
	if !shouldPromote(true, "command_exec", 1, "", "go test") {
		t.Fatal("errors promote")
	}
	if shouldPromote(false, "file_read", 1, "ok", "") {
		t.Fatal("a successful read does not promote")
	}
	if !shouldPromote(false, "file_write", 2, "", "") {
		t.Fatal("a second write promotes")
	}
	if shouldPromote(false, "file_write", 1, "", "") {
		t.Fatal("a first write does not promote")
	}
	if !shouldPromote(false, "tool_use", 0, "we decided to skip the daemon", "") {
		t.Fatal("an explicit decision promotes")
	}
}
