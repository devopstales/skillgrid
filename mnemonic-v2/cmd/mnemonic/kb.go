// kb.go — the kb command group (the KB engine's CLI surface).
package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/project"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/kb"
)

func runKb(fl config.Flags, cwd string, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(outW(), "usage: mnemonic kb <subcommand> [flags]")
		return exitUsage
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "bootstrap":
		return runKbBootstrap(fl, cwd, rest)
	case "index":
		return runKbIndex(fl, cwd, rest)
	case "query":
		return runKbQuery(fl, cwd, rest)
	case "validate":
		return runKbValidate(fl, cwd, rest)
	case "lint":
		return runKbLint(fl, cwd, rest)
	case "status":
		return runKbStatus(fl, cwd, rest)
	case "cite-source":
		return runKbCiteSource(fl, cwd, rest)
	case "eval":
		return runKbEval(fl, cwd, rest)
	default:
		fmt.Fprintf(errW(), "error: unknown kb subcommand %q (see: mnemonic kb --help)\n", sub)
		return exitUsage
	}
}

func kbStore(cwd string, fl config.Flags, explicit string) (store.Store, string, error) {
	cfg, err := config.Load(cwd, fl)
	if err != nil {
		return nil, "", err
	}
	det := project.Detector{EnvVar: cfg.Project.EnvVar}
	p, err := det.Resolve(cwd, explicit, gitToplevel)
	if err != nil {
		return nil, "", err
	}
	cfg.Project.Name = p
	st, err := storeFor(cfg, p)
	if err != nil {
		return nil, "", err
	}
	return st, p, nil
}

// kbRoot returns the knowledge vault root for this project (spec
// 001-structure: the vault lives under a single sentinel dir,
// .skillgrid/knowledge/).
func kbRoot(cwd string) string {
	return filepath.Join(cwd, ".skillgrid", "knowledge")
}

func runKbBootstrap(fl config.Flags, cwd string, args []string) int {
	st, proj, err := kbStore(cwd, fl, "")
	if err != nil {
		fmt.Fprintf(errW(), "error: %v\n", err)
		return exitUsage
	}
	defer st.Close()
	res, err := kb.Bootstrap(context.Background(), st, kbRoot(cwd), kb.Options{Project: proj})
	if err != nil {
		fmt.Fprintf(errW(), "error: %v\n", err)
		return 1
	}
	fmt.Fprintln(outW(), "kb bootstrap:", res)
	return 0
}

func runKbIndex(fl config.Flags, cwd string, args []string) int {
	st, proj, err := kbStore(cwd, fl, "")
	if err != nil {
		fmt.Fprintf(errW(), "error: %v\n", err)
		return exitUsage
	}
	defer st.Close()
	res, err := kb.Reindex(context.Background(), st, kbRoot(cwd), proj)
	if err != nil {
		fmt.Fprintf(errW(), "error: %v\n", err)
		return 1
	}
	fmt.Fprintln(outW(), "kb index:", "scanned", res.Scanned, "changed", res.Changed)
	return 0
}

func runKbQuery(fl config.Flags, cwd string, args []string) int {
	st, proj, err := kbStore(cwd, fl, "")
	if err != nil {
		fmt.Fprintf(errW(), "error: %v\n", err)
		return exitUsage
	}
	defer st.Close()
	res, err := kb.Query(context.Background(), st, "auth", kb.QueryFilters{Project: proj, Limit: 10})
	if err != nil {
		fmt.Fprintf(errW(), "error: %v\n", err)
		return 1
	}
	fmt.Fprintln(outW(), "kb query:", len(res))
	return 0
}

func runKbValidate(fl config.Flags, cwd string, args []string) int {
	st, proj, err := kbStore(cwd, fl, "")
	if err != nil {
		fmt.Fprintf(errW(), "error: %v\n", err)
		return exitUsage
	}
	defer st.Close()
	res, err := kb.Validate(context.Background(), st, "decisions/auth-model.md", proj)
	if err != nil {
		fmt.Fprintf(errW(), "error: %v\n", err)
		return 1
	}
	fmt.Fprintln(outW(), "kb validate:", res.OK, res.Reason)
	return 0
}

func runKbLint(fl config.Flags, cwd string, args []string) int {
	st, proj, err := kbStore(cwd, fl, "")
	if err != nil {
		fmt.Fprintf(errW(), "error: %v\n", err)
		return exitUsage
	}
	defer st.Close()
	issues, err := kb.Lint(context.Background(), st, proj)
	if err != nil {
		fmt.Fprintf(errW(), "error: %v\n", err)
		return 1
	}
	fmt.Fprintln(outW(), "kb lint:", len(issues))
	return 0
}

func runKbStatus(fl config.Flags, cwd string, args []string) int {
	st, proj, err := kbStore(cwd, fl, "")
	if err != nil {
		fmt.Fprintf(errW(), "error: %v\n", err)
		return exitUsage
	}
	defer st.Close()
	res, err := kb.Status(context.Background(), st, proj)
	if err != nil {
		fmt.Fprintf(errW(), "error: %v\n", err)
		return 1
	}
	fmt.Fprintln(outW(), "kb status:", res.Indexed, res.Phase)
	return 0
}

func runKbCiteSource(fl config.Flags, cwd string, args []string) int {
	st, _, err := kbStore(cwd, fl, "")
	if err != nil {
		fmt.Fprintf(errW(), "error: %v\n", err)
		return exitUsage
	}
	defer st.Close()
	res, err := kb.CiteSource(context.Background(), st, "decisions/auth-model.md")
	if err != nil {
		fmt.Fprintf(errW(), "error: %v\n", err)
		return 1
	}
	fmt.Fprintln(outW(), "kb cite-source:", res.Source)
	return 0
}

func runKbEval(fl config.Flags, cwd string, args []string) int {
	st, _, err := kbStore(cwd, fl, "")
	if err != nil {
		fmt.Fprintf(errW(), "error: %v\n", err)
		return exitUsage
	}
	defer st.Close()
	rep, err := kb.Eval(context.Background(), st, nil, 10, 1.0)
	if err != nil {
		fmt.Fprintf(errW(), "error: %v\n", err)
		return 1
	}
	fmt.Fprintln(outW(), "kb eval:", rep.Cases, rep.RecallAtK, rep.Failed)
	return 0
}
