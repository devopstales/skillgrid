package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"skillgrid-cli/internal/mnemonic/codeindex"
	"skillgrid-cli/internal/mnemonic/config"
	"skillgrid-cli/internal/mnemonic/project"
	"skillgrid-cli/internal/mnemonic/store"
)

func runIndex(args []string) error {
	fs := flag.NewFlagSet("index", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	status := fs.Bool("status", false, "print index stats")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	root := cwd
	if fs.NArg() > 0 {
		root = fs.Arg(0)
	} else if gitRoot, err := gitRoot(cwd); err == nil && gitRoot != "" {
		root = gitRoot
	}

	projectID, err := project.Resolve(cwd)
	if err != nil {
		return err
	}

	dataDir, err := mnemonicDataDir()
	if err != nil {
		return err
	}

	st, err := store.Open(dataDir, projectID)
	if err != nil {
		return err
	}
	defer st.Close()

	if *status {
		s, err := codeindex.GetStatus(st)
		if err != nil {
			return err
		}
		fmt.Printf("files: %d\nchunks: %d\nlast_indexed: %s\n", s.FileCount, s.ChunkCount, s.LastIndexed)
		return nil
	}

	cfg := config.Load(root)
	idxCfg := codeindex.Config{
		Include:      cfg.Include,
		Exclude:      cfg.Exclude,
		ChunkLines:   cfg.ChunkLines,
		ChunkOverlap: cfg.ChunkOverlap,
	}

	idx := codeindex.New(st)
	stats, err := idx.Run(context.Background(), root, idxCfg)
	if err != nil {
		return err
	}

	fmt.Printf("indexed: %d skipped: %d deleted: %d chunks: %d\n",
		stats.FilesIndexed, stats.FilesSkipped, stats.FilesDeleted, stats.ChunksAdded)
	return nil
}

func gitRoot(cwd string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func mnemonicDataDir() (string, error) {
	if v := os.Getenv("SKILLGRID_MNEMONIC_DATA_DIR"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".skillgrid", "mnemonic"), nil
}
