package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"skillgrid-cli/internal/mnemonic/config"
	"skillgrid-cli/internal/mnemonic/project"
	"skillgrid-cli/internal/mnemonic/store"
	"skillgrid-cli/internal/mnemonic/webcache"
)

func runWeb(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: skillgrid web <search|status> [flags]")
	}

	switch args[0] {
	case "search":
		return runWebSearch(args[1:])
	case "status":
		return runWebStatus(args[1:])
	default:
		return fmt.Errorf("unknown web subcommand %q", args[0])
	}
}

func runWebSearch(args []string) error {
	fs := flag.NewFlagSet("web search", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	source := fs.String("source", "", "filter by source")
	freshOnly := fs.Bool("fresh-only", true, "only non-expired entries")
	limit := fs.Int("limit", 20, "maximum results")
	jsonOut := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: skillgrid web search <query> [-source=] [-fresh-only=true] [-limit=20]")
	}
	query := fs.Arg(0)

	svc, cleanup, err := openWebService()
	if err != nil {
		return err
	}
	defer cleanup()

	hits, err := svc.Search(context.Background(), query, *source, *freshOnly, *limit)
	if err != nil {
		return err
	}

	if *jsonOut {
		b, err := json.MarshalIndent(map[string]any{"entries": hits}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}

	if len(hits) == 0 {
		fmt.Println("no results")
		return nil
	}
	for _, hit := range hits {
		fmt.Printf("[%d] %s", hit.ID, hit.Source)
		if hit.Title != "" {
			fmt.Printf(" — %s", hit.Title)
		}
		if hit.Query != "" {
			fmt.Printf(" (%s)", hit.Query)
		}
		fmt.Println()
	}
	return nil
}

func runWebStatus(args []string) error {
	fs := flag.NewFlagSet("web status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	svc, cleanup, err := openWebService()
	if err != nil {
		return err
	}
	defer cleanup()

	st, err := svc.CacheStatus(context.Background())
	if err != nil {
		return err
	}

	if *jsonOut {
		b, err := json.MarshalIndent(st, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}

	fmt.Printf("total: %d\nexpired: %d\n", st.TotalEntries, st.ExpiredEntries)
	if st.OldestFetch != "" {
		fmt.Printf("oldest_fetch: %s\n", st.OldestFetch)
	}
	if st.NewestFetch != "" {
		fmt.Printf("newest_fetch: %s\n", st.NewestFetch)
	}
	for source, count := range st.BySource {
		fmt.Printf("%s: %d\n", source, count)
	}
	return nil
}

func openWebService() (*webcache.Service, func(), error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}

	projectID, err := project.Resolve(cwd)
	if err != nil {
		return nil, nil, err
	}

	dataDir, err := mnemonicDataDir()
	if err != nil {
		return nil, nil, err
	}

	st, err := store.Open(dataDir, projectID)
	if err != nil {
		return nil, nil, err
	}

	cfg := config.Load(cwd).WebCache
	return webcache.New(st, projectID, cfg), func() { st.Close() }, nil
}
