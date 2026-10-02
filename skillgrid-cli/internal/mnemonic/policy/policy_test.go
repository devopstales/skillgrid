package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustPolicy(t *testing.T, yml string) *Policy {
	t.Helper()
	en, rules, err := Parse([]byte(yml), "test.yaml")
	if err != nil {
		t.Fatal(err)
	}
	p := &Policy{Rules: rules}
	if en != nil {
		p.Enabled = *en
	}
	return p
}

const sample = `
policy:
  enabled: true
  rules:
    - name: allow-fixtures
      match: { action: file_write, path: "secrets/fixtures/**" }
      effect: allow
    - name: no-secret-writes
      match: { action: file_write, path: ["secrets/**", "**/*.pem"] }
      effect: block
      message: blocked secrets
    - name: rm
      match: { action: command_exec, command: "*rm -rf *" }
      effect: warn
      message: careful
    - name: web
      match: { tool: [WebSearch], agent: cursor }
      effect: guide
      message: use mem_search
    - name: runaway
      match: { action: command_exec, counters: { commands_exec: "> 100" } }
      effect: block
      message: too many commands
`

func TestEvaluate_Effects(t *testing.T) {
	p := mustPolicy(t, sample)
	cases := []struct {
		name string
		in   Input
		want Decision
	}{
		{"block relative", Input{Action: "file_write", Path: "secrets/db.env"}, Decision{Block, "blocked secrets", "no-secret-writes"}},
		{"block absolute under dir", Input{Action: "file_write", Path: "/repo/secrets/db.env", Directory: "/repo"}, Decision{Block, "blocked secrets", "no-secret-writes"}},
		{"pem at root via **/", Input{Action: "file_write", Path: "key.pem"}, Decision{Block, "blocked secrets", "no-secret-writes"}},
		{"allow wins first", Input{Action: "file_write", Path: "secrets/fixtures/a.env"}, Decision{Allow, "", "allow-fixtures"}},
		{"read is not write", Input{Action: "file_read", Path: "secrets/db.env"}, Decision{Effect: Allow}},
		{"warn", Input{Action: "command_exec", Command: "sudo rm -rf /tmp/x"}, Decision{Warn, "careful", "rm"}},
		{"guide needs agent", Input{Tool: "websearch", Agent: "cursor"}, Decision{Guide, "use mem_search", "web"}},
		{"guide other agent", Input{Tool: "WebSearch", Agent: "opencode"}, Decision{Effect: Allow}},
		{"counter below", Input{Action: "command_exec", Command: "ls", Counters: map[string]int{"commands_exec": 100}}, Decision{Effect: Allow}},
		{"counter above", Input{Action: "command_exec", Command: "ls", Counters: map[string]int{"commands_exec": 101}}, Decision{Block, "too many commands", "runaway"}},
	}
	for _, c := range cases {
		if got := p.Evaluate(c.in); got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestEvaluate_DisabledAllowsEverything(t *testing.T) {
	p := mustPolicy(t, strings.Replace(sample, "enabled: true", "enabled: false", 1))
	if d := p.Evaluate(Input{Action: "file_write", Path: "secrets/x"}); d.Effect != Allow || d.Rule != "" {
		t.Errorf("disabled policy = %+v, want plain allow", d)
	}
	var nilPolicy *Policy
	if d := nilPolicy.Evaluate(Input{}); d.Effect != Allow {
		t.Errorf("nil policy = %+v", d)
	}
}

func TestParse_Validation(t *testing.T) {
	bad := map[string]string{
		"unknown effect":  "policy:\n  rules:\n    - match: {action: file_write}\n      effect: deny\n      message: x\n",
		"missing effect":  "policy:\n  rules:\n    - match: {action: file_write}\n",
		"unknown action":  "policy:\n  rules:\n    - match: {action: write}\n      effect: block\n      message: x\n",
		"unknown counter": "policy:\n  rules:\n    - match: {counters: {bananas: \"> 1\"}}\n      effect: warn\n      message: x\n",
		"bad threshold":   "policy:\n  rules:\n    - match: {counters: {errors: \"lots\"}}\n      effect: warn\n      message: x\n",
		"no message":      "policy:\n  rules:\n    - match: {action: file_write}\n      effect: block\n",
		"bad yaml":        "policy: [",
	}
	for name, y := range bad {
		if _, _, err := Parse([]byte(y), "p.yaml"); err == nil {
			t.Errorf("%s: want a validation error", name)
		}
	}
	if _, _, err := Parse([]byte(Starter), "starter"); err != nil {
		t.Errorf("starter must validate: %v", err)
	}
}

func TestLoad_RepoThenHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(home, ".skillgrid", "policy.yaml"), `
policy:
  enabled: true
  rules:
    - name: home-block-all-writes
      match: { action: file_write }
      effect: block
      message: home
`)
	p, err := Load(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !p.Enabled || p.Evaluate(Input{Action: "file_write", Path: "a.go"}).Rule != "home-block-all-writes" {
		t.Fatalf("home-only policy = %+v", p)
	}

	write(RepoFile(repo), `
policy:
  enabled: false
  rules:
    - name: repo-allow-go
      match: { action: file_write, path: "**/*.go" }
      effect: allow
`)
	sub := filepath.Join(repo, "pkg", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	p, err = Load(sub)
	if err != nil {
		t.Fatal(err)
	}
	if p.Enabled {
		t.Error("repo enabled:false must win over home enabled:true")
	}
	p.Enabled = true
	if d := p.Evaluate(Input{Action: "file_write", Path: "pkg/a.go"}); d.Rule != "repo-allow-go" {
		t.Errorf("repo rules come first: got %+v", d)
	}
	if d := p.Evaluate(Input{Action: "file_write", Path: "README.md"}); d.Rule != "home-block-all-writes" {
		t.Errorf("home rules follow: got %+v", d)
	}
	if len(p.Files) != 2 {
		t.Errorf("files = %v", p.Files)
	}

	write(RepoFile(repo), "policy:\n  rules:\n    - effect: nope\n")
	if _, err := Load(repo); err == nil {
		t.Error("a broken repo file must return an error")
	}
}
