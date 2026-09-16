package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

var errTrailNotFound = errors.New("trail not found")

type trailRow struct {
	ID          int64    `json:"id"`
	Query       string   `json:"query"`
	Directories []string `json:"directories"`
	Files       []string `json:"files"`
	ResultPath  string   `json:"result_path,omitempty"`
	Corpus      string   `json:"corpus,omitempty"`
	CreatedAt   string   `json:"created_at"`
}

// runTrail handles `skillgrid trail recent|show`.
func runTrail(version string, args []string) {
	_ = version
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: skillgrid trail recent|show <id> --project ID [--dir DATA_DIR] [--limit N]")
		os.Exit(2)
	}
	cmd := args[0]
	rest := reorderTrailArgs(args[1:])

	fs := flag.NewFlagSet("trail", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		dir     string
		project string
		limit   int
	)
	fs.StringVar(&dir, "dir", envOr("SKILLGRID_MNEMONIC_DATA_DIR", ""), "mnemonic data directory")
	fs.StringVar(&project, "project", "", "project id (required)")
	fs.IntVar(&limit, "limit", 20, "max rows for recent")
	if err := fs.Parse(rest); err != nil {
		os.Exit(2)
	}
	if project == "" {
		fmt.Fprintln(os.Stderr, "error: --project is required")
		os.Exit(2)
	}
	dataDir := dir
	if dataDir == "" {
		d, err := service.DefaultDataDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		dataDir = d
	}
	st, err := store.Open(dataDir, project)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer st.Close()

	switch cmd {
	case "recent":
		rows, err := listRecentTrails(st.DB, project, limit)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(rows)
	case "show":
		pos := fs.Args()
		if len(pos) < 1 {
			fmt.Fprintln(os.Stderr, "error: trail show requires an id")
			os.Exit(2)
		}
		id, err := strconv.ParseInt(pos[0], 10, 64)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: invalid trail id")
			os.Exit(2)
		}
		row, err := showTrail(st.DB, project, id)
		if err != nil {
			if errors.Is(err, errTrailNotFound) {
				fmt.Fprintln(os.Stderr, "not-found")
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(row)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown trail command %q\n", cmd)
		os.Exit(2)
	}
}

func listRecentTrails(db *sql.DB, project string, limit int) ([]trailRow, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := db.Query(`
		SELECT id, query, directories_json, files_json, COALESCE(result_path,''), COALESCE(corpus,''), created_at
		FROM retrieval_trails WHERE project = ?
		ORDER BY created_at DESC, id DESC LIMIT ?`, project, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []trailRow{}
	for rows.Next() {
		r, err := scanTrailRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func showTrail(db *sql.DB, project string, id int64) (trailRow, error) {
	row := db.QueryRow(`
		SELECT id, query, directories_json, files_json, COALESCE(result_path,''), COALESCE(corpus,''), created_at
		FROM retrieval_trails WHERE project = ? AND id = ?`, project, id)
	r, err := scanTrailRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return trailRow{}, errTrailNotFound
		}
		return trailRow{}, err
	}
	return r, nil
}

type trailScanner interface {
	Scan(dest ...any) error
}

func scanTrailRow(s trailScanner) (trailRow, error) {
	var r trailRow
	var dirsJSON, filesJSON string
	if err := s.Scan(&r.ID, &r.Query, &dirsJSON, &filesJSON, &r.ResultPath, &r.Corpus, &r.CreatedAt); err != nil {
		return trailRow{}, err
	}
	_ = json.Unmarshal([]byte(dirsJSON), &r.Directories)
	_ = json.Unmarshal([]byte(filesJSON), &r.Files)
	if r.Directories == nil {
		r.Directories = []string{}
	}
	if r.Files == nil {
		r.Files = []string{}
	}
	return r, nil
}

// reorderTrailArgs moves flags before positionals so `show 1 --project x` works.
func reorderTrailArgs(args []string) []string {
	var flags, pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-" || strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			if !strings.Contains(a, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		pos = append(pos, a)
	}
	return append(flags, pos...)
}
