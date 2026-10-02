package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/loop"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/project"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

func runPrime(args []string) {
	dir, session := loopFlags("prime", args)
	text := primeText(dir, session)
	_ = os.MkdirAll(loop.PagesDir(dir), 0o755)
	_ = os.WriteFile(filepath.Join(loop.PagesDir(dir), "prime.txt"), []byte(text), 0o644)
	fmt.Fprint(os.Stdout, text)
}

func runCompact(args []string) {
	dir, session := loopFlags("compact", args)
	fmt.Fprint(os.Stdout, compactText(dir, session))
}

func runPages(args []string) {
	dir, _ := loopFlags("pages", args)
	h, cleanup, err := openLoop(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer cleanup()
	if err := loop.WritePages(context.Background(), h.Store().DB, dir, h.ProjectID()); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, loop.PagesDir(dir))
}

func runDiffImpact(args []string) {
	dir, _ := loopFlags("diff-impact", args)
	h, cleanup, err := openLoop(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer cleanup()
	files := loop.ChangedFiles(dir)
	lines, err := loop.DiffImpact(context.Background(), h.Store().DB, files)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	for _, line := range loop.FormatImpact(lines) {
		fmt.Fprintln(os.Stdout, line)
	}
}

func loopFlags(name string, args []string) (dir, session string) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&dir, "dir", ".", "repository directory")
	fs.StringVar(&session, "session", "", "session id")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: skillgrid %s [--dir path] [--session id]\n", name)
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if strings.TrimSpace(dir) == "" {
		dir = "."
	}
	return dir, strings.TrimSpace(session)
}

func primeText(dir, session string) string {
	h, cleanup, err := openLoop(dir)
	if err != nil {
		return loop.RenderPrime(loop.PrimeInput{})
	}
	defer cleanup()
	_ = loop.WriteActiveProject(h.ProjectID())
	step, _ := loop.LoadLastStep(context.Background(), h.Store().DB, h.ProjectID(), session)
	files := loop.ChangedFiles(dir)
	var impact []string
	if lines, ierr := loop.DiffImpact(context.Background(), h.Store().DB, files); ierr == nil {
		impact = loop.FormatImpact(lines)
	}
	return loop.RenderPrime(loop.PrimeInput{
		Project:      h.ProjectID(),
		Task:         step.Task,
		Branch:       loop.GitBranch(dir),
		StoppedAt:    step.StoppedAt,
		OneLiner:     step.OneLiner,
		ChangedFiles: files,
		ImpactLines:  impact,
	})
}

func compactText(dir, session string) string {
	branch := loop.GitBranch(dir)
	stat := loop.DiffStat(dir)
	task := ""
	h, cleanup, err := openLoop(dir)
	if err == nil {
		defer cleanup()
		_ = loop.WriteActiveProject(h.ProjectID())
		if step, lerr := loop.LoadLastStep(context.Background(), h.Store().DB, h.ProjectID(), session); lerr == nil {
			task = step.Task
		}
		text := renderCompact(task, branch, stat)
		if session != "" {
			_ = h.Memory().SessionSummary(context.Background(), session, text)
		}
		return text
	}
	return renderCompact(task, branch, stat)
}

func renderCompact(task, branch, stat string) string {
	return loop.RenderCompact(loop.CompactInput{
		Task:      task,
		Branch:    branch,
		StoppedAt: branch,
		OneLiner:  firstDiffLine(stat),
		DiffStat:  stat,
	})
}

func firstDiffLine(stat string) string {
	line := strings.TrimSpace(stat)
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	return strings.TrimSpace(line)
}

func openLoop(dir string) (*service.ProjectHandle, func(), error) {
	dataDir := envOr("SKILLGRID_MNEMONIC_DATA_DIR", "")
	svc, err := newMnemonicService(dataDir)
	if err != nil {
		return nil, nil, err
	}
	h, cleanup, err := svc.OpenForDirectory(dir)
	if err == nil {
		return h, cleanup, nil
	}
	if id := strings.TrimSpace(os.Getenv("MNEMONIC_PROJECT")); id != "" {
		return svc.OpenAt(project.NormalizeID(id), dir)
	}
	if id := loop.ReadActiveProject(); id != "" {
		return svc.OpenAt(id, dir)
	}
	return nil, nil, err
}
