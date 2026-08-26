package main

import (
	"bytes"
	"flag"
	"os"
	"strings"
	"testing"
)

func TestUsagePrintsOnNoArgs(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	flag.CommandLine.SetOutput(w)
	os.Args = []string{"skillgrid-cli"}
	Run()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("expected usage output, got: %s", out)
	}
}

func TestMCPCommand(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	os.Args = []string{"skillgrid-cli", "mcp"}
	code := Run()

	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestIndexHelp(t *testing.T) {
	oldArgs := os.Args
	oldStdout := os.Stdout
	defer func() {
		os.Args = oldArgs
		os.Stdout = oldStdout
	}()

	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Args = []string{"skillgrid-cli", "index", "--help"}
	code := Run()
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("expected usage output, got: %s", out)
	}
	if !strings.Contains(out, "index") {
		t.Fatalf("expected index in help text, got: %s", out)
	}
}
