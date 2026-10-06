package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/policy"
)

func TestPolicyInitValidateTest(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()

	path, err := policyInit(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := policyInit(dir, false); err == nil {
		t.Fatal("second init without --force must refuse")
	}

	var out bytes.Buffer
	if err := policyValidate(&out, dir); err != nil {
		t.Fatalf("starter must validate: %v", err)
	}
	if !strings.Contains(out.String(), "policy disabled") || !strings.Contains(out.String(), "no-secret-writes") {
		t.Errorf("validate output = %q", out.String())
	}

	d, err := policyTest(dir, policy.Input{Tool: "Write", Path: "secrets/db.env"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Effect != policy.Block || d.Rule != "no-secret-writes" || !strings.Contains(d.Message, "not enforced") {
		t.Errorf("disabled starter decision = %+v", d)
	}

	data, _ := os.ReadFile(path)
	if err := os.WriteFile(path, bytes.Replace(data, []byte("enabled: false"), []byte("enabled: true"), 1), 0o644); err != nil {
		t.Fatal(err)
	}
	d, _ = policyTest(dir, policy.Input{Action: "command_exec", Command: "rm -rf /tmp/x"})
	if d.Effect != policy.Warn || strings.Contains(d.Message, "not enforced") {
		t.Errorf("enabled warn decision = %+v", d)
	}
	d, _ = policyTest(dir, policy.Input{Tool: "Read", Path: filepath.Join(dir, "README.md")})
	if d.Effect != policy.Allow || d.Rule != "" {
		t.Errorf("unmatched decision = %+v", d)
	}

	if err := os.WriteFile(path, []byte("policy:\n  rules:\n    - effect: deny\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := policyValidate(&out, dir); err == nil || !strings.Contains(err.Error(), "unknown effect") {
		t.Errorf("broken file err = %v", err)
	}
}
