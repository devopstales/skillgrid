package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
	"github.com/devopstales/skillgrid/mnemonic/internal/policy"
)

// runPolicy handles `skillgrid policy init | validate | test` (ADR-0021).
func runPolicy(args []string) {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		printPolicyUsage(os.Stderr)
		if len(args) == 0 {
			os.Exit(2)
		}
		return
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	switch args[0] {
	case "init":
		fs := flag.NewFlagSet("policy init", flag.ContinueOnError)
		force := fs.Bool("force", false, "overwrite an existing .skillgrid/policy.yaml")
		dir := fs.String("dir", cwd, "repo directory")
		if err := fs.Parse(args[1:]); err != nil {
			os.Exit(2)
		}
		path, err := policyInit(*dir, *force)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s (enabled: false; set enabled: true to enforce)\n", path)
	case "validate":
		fs := flag.NewFlagSet("policy validate", flag.ContinueOnError)
		dir := fs.String("dir", cwd, "directory to resolve .skillgrid/policy.yaml from")
		if err := fs.Parse(args[1:]); err != nil {
			os.Exit(2)
		}
		if err := policyValidate(os.Stdout, *dir); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "test":
		fs := flag.NewFlagSet("policy test", flag.ContinueOnError)
		var in policy.Input
		var jsonOut bool
		dir := fs.String("dir", cwd, "directory to resolve .skillgrid/policy.yaml from")
		fs.StringVar(&in.Action, "action", "", "file_read, file_write, command_exec, tool_use (derived from --tool when empty)")
		fs.StringVar(&in.Path, "path", "", "file path")
		fs.StringVar(&in.Command, "command", "", "shell command")
		fs.StringVar(&in.Tool, "tool", "", "tool name")
		fs.StringVar(&in.Agent, "agent", "", "cursor, opencode, kilo")
		fs.StringVar(&in.Project, "project", "", "project id")
		fs.BoolVar(&jsonOut, "json", false, "emit JSON")
		if err := fs.Parse(args[1:]); err != nil {
			os.Exit(2)
		}
		d, err := policyTest(*dir, in)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		if jsonOut {
			printJSON(d)
			return
		}
		writeDecision(os.Stdout, d)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown policy command %q\n", args[0])
		printPolicyUsage(os.Stderr)
		os.Exit(2)
	}
}

func printPolicyUsage(w io.Writer) {
	fmt.Fprint(w, `usage: skillgrid policy <init|validate|test> [flags]

  init [--force]        write a disabled starter .skillgrid/policy.yaml
  validate              load the repo and ~/.skillgrid policy files and report errors
  test --action A --path P | --command C | --tool T [--agent A]
                        print the decision the hooks would get (same evaluator as
                        POST /policy/evaluate)
`)
}

func repoRootFor(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err == nil && strings.TrimSpace(string(out)) != "" {
		return strings.TrimSpace(string(out))
	}
	return dir
}

func policyInit(dir string, force bool) (string, error) {
	path := policy.RepoFile(repoRootFor(dir))
	if _, err := os.Stat(path); err == nil && !force {
		return "", fmt.Errorf("%s exists (use --force to overwrite)", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	return path, os.WriteFile(path, []byte(policy.Starter), 0o644)
}

func policyValidate(w io.Writer, dir string) error {
	p, err := policy.Load(dir)
	if err != nil {
		return err
	}
	if len(p.Files) == 0 {
		fmt.Fprintln(w, "no policy files (run `skillgrid policy init`); every call is allowed")
		return nil
	}
	state := "disabled"
	if p.Enabled {
		state = "enabled"
	}
	fmt.Fprintf(w, "policy %s: %d rules from %s\n", state, len(p.Rules), strings.Join(p.Files, ", "))
	for i, r := range p.Rules {
		fmt.Fprintf(w, "  %2d. %-6s %s\n", i+1, r.Effect, r.Name)
	}
	return nil
}

func policyTest(dir string, in policy.Input) (policy.Decision, error) {
	p, err := policy.Load(dir)
	if err != nil {
		return policy.Decision{}, err
	}
	in.Action = memory.ActionForTool(in.Tool, in.Action)
	if in.Directory == "" {
		in.Directory = repoRootFor(dir)
	}
	if in.Path != "" && !filepath.IsAbs(in.Path) {
		in.Path = filepath.ToSlash(filepath.Clean(in.Path))
	}
	if !p.Enabled && len(p.Files) > 0 {
		// Show what would happen once enabled, and say that it is not enforced.
		p.Enabled = true
		d := p.Evaluate(in)
		if d.Rule != "" {
			d.Message = strings.TrimSpace(d.Message + " (policy disabled: not enforced)")
		}
		return d, nil
	}
	return p.Evaluate(in), nil
}

func writeDecision(w io.Writer, d policy.Decision) {
	if d.Rule == "" {
		fmt.Fprintf(w, "%s (no rule matched)\n", d.Effect)
		return
	}
	fmt.Fprintf(w, "%s by rule %s", d.Effect, d.Rule)
	if d.Message != "" {
		fmt.Fprintf(w, ": %s", d.Message)
	}
	fmt.Fprintln(w)
}
