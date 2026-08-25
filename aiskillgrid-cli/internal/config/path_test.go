package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWritePathInstructions(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir failed: %v", err)
	}
	baseDir := filepath.Join(home, ".aiskillgrid")
	var buf bytes.Buffer
	err = WritePathInstructions(baseDir, &buf)
	if err != nil {
		t.Fatalf("WritePathInstructions failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `export PATH="$HOME/.aiskillgrid/bin:$PATH"`) {
		t.Fatalf("missing bin path export:\n%s", out)
	}
	if !strings.Contains(out, `export PATH="$HOME/.aiskillgrid/npm/.bin:$PATH"`) {
		t.Fatalf("missing npm path export:\n%s", out)
	}
}
