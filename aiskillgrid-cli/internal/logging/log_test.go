package logging

import (
	"os"
	"strings"
	"testing"
)

func TestInitCreatesLogFile(t *testing.T) {
	tmp := t.TempDir()
	err := Init(tmp)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	p := Path()
	if !strings.HasPrefix(p, tmp) {
		t.Fatalf("Path() = %q, expected prefix %q", p, tmp)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("log file not created: %v", err)
	}
}

func TestWritesAppend(t *testing.T) {
	tmp := t.TempDir()
	ResetForTest()
	Init(tmp)
	Warn("hello")
	Info("world")
	data, _ := os.ReadFile(Path())
	if !strings.Contains(string(data), "hello") {
		t.Fatalf("expected warn message in log, got: %s", string(data))
	}
	if !strings.Contains(string(data), "world") {
		t.Fatalf("expected info message in log, got: %s", string(data))
	}
}
