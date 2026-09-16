package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/tiered"
)

// ErrMissingCommitSources is returned when mnemonic_commit lacks required content.
var ErrMissingCommitSources = errors.New("missing required commit sources")

// MnemonicCommitInput is the explicit compaction request.
type MnemonicCommitInput struct {
	Title          string
	LessonsLearned string
	Content        string // optional full L2 body; if empty, LessonsLearned is used
	SourceLink     string
	TaskID         string
}

// MnemonicCommitResult is returned after LTM + L2 are durable.
type MnemonicCommitResult struct {
	MemoryID int64  `json:"memory_id"`
	FullPath string `json:"full_path"`
	Title    string `json:"title,omitempty"`
}

// MnemonicCommit writes durable L2 + long_term_memories, then fires async tiering.
// It does not await L0/L1 generation.
func (s *Service) MnemonicCommit(ctx context.Context, projectID string, in MnemonicCommitInput, hookWG *sync.WaitGroup) (*MnemonicCommitResult, error) {
	body := strings.TrimSpace(in.Content)
	if body == "" {
		body = strings.TrimSpace(in.LessonsLearned)
	}
	title := strings.TrimSpace(in.Title)
	if body == "" && title == "" {
		return nil, fmt.Errorf("%w: need title or lessons_learned/content", ErrMissingCommitSources)
	}
	if body == "" {
		body = title
	}
	if title == "" {
		title = "committed memory"
	}

	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}

	root := filepath.Join(s.dataDir, projectID, "ltm")
	if err := os.MkdirAll(root, 0o755); err != nil {
		cleanup()
		return nil, fmt.Errorf("create ltm dir: %w", err)
	}
	slug := sanitizeFileSlug(title)
	fullPath := filepath.Join(root, fmt.Sprintf("%d-%s.md", time.Now().UnixNano(), slug))
	if err := os.WriteFile(fullPath, []byte(body), 0o644); err != nil {
		cleanup()
		return nil, fmt.Errorf("write L2: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	abs, over := tiered.SidecarPaths(fullPath)
	res, err := h.store.DB.Exec(`
		INSERT INTO long_term_memories (project, title, source_link, full_path, abstract_path, overview_path, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		projectID, title, nullEmpty(in.SourceLink), fullPath, abs, over, now, now)
	if err != nil {
		_ = os.Remove(fullPath)
		cleanup()
		return nil, fmt.Errorf("insert long_term_memories: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		cleanup()
		return nil, err
	}

	// Register L2 path immediately so load_full_details works before tiers finish.
	if err := (&tiered.Store{DB: h.store.DB}).Register(ctx, projectID, fullPath, "", "", title); err != nil {
		cleanup()
		return nil, err
	}

	// Release the store lock before background tiering reopens the DB.
	cleanup()

	dataDir := s.dataDir
	pid := projectID
	fp := fullPath
	ttl := title
	if hookWG != nil {
		hookWG.Add(1)
	}
	go func() {
		if hookWG != nil {
			defer hookWG.Done()
		}
		st2, err := store.Open(dataDir, pid)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: tiered hook open: %v\n", err)
			return
		}
		defer st2.Close()
		ts := &tiered.Store{
			DB:         st2.DB,
			Summarizer: tiered.HeuristicSummarizer{},
			Logf: func(format string, args ...any) {
				fmt.Fprintf(os.Stderr, "warn: "+format+"\n", args...)
			},
		}
		if err := ts.GenerateTiers(context.Background(), pid, fp, ttl); err != nil {
			fmt.Fprintf(os.Stderr, "warn: tiered hook: %v\n", err)
		}
	}()

	_ = in.TaskID // reserved for 001 wiring
	return &MnemonicCommitResult{MemoryID: id, FullPath: fullPath, Title: title}, nil
}

// CountLongTermMemories is a test/helper for auto-commit assertions.
func (s *Service) CountLongTermMemories(ctx context.Context, projectID string) (int, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return 0, err
	}
	defer cleanup()
	_ = ctx
	var n int
	if err := h.store.DB.QueryRow(`SELECT COUNT(*) FROM long_term_memories WHERE project = ?`, projectID).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func sanitizeFileSlug(title string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(title) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			b.WriteByte('-')
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		return "memory"
	}
	if len(s) > 48 {
		s = s[:48]
	}
	return s
}
