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
