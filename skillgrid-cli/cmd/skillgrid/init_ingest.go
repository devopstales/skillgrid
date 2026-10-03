package main

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// Default ingest roots (only when present). Directories are walked for regular
// files; missing roots are listed in Skipped (defaults) or Errors (extras).
var defaultIngestRoots = []string{
	"README.md",
	"docs",
	".skillgrid/ASSUMPTIONS.md",
	".skillgrid/ARCHITECTURE.md",
	".skillgrid/artifacts",
}

const maxIngestBytes = 512 * 1024

// ingestPaths upserts default project docs and jailed --docs paths into the
// project memory store via memory.Save. Topic keys are init/docs/<relpath>.
func ingestPaths(ctx context.Context, h *service.ProjectHandle, dir string, extra []string) (ingested int, skipped, errs []string) {
	if h == nil || h.Memory() == nil {
		return 0, nil, []string{"memory handle not available"}
	}
	mem := h.Memory()

	sid, err := mem.SessionStart(ctx, dir, "skillgrid-init")
	if err != nil {
		return 0, nil, []string{fmt.Sprintf("session start: %v", err)}
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return 0, nil, []string{fmt.Sprintf("resolve dir: %v", err)}
	}

	for _, root := range defaultIngestRoots {
		full := filepath.Join(absDir, filepath.FromSlash(root))
		info, statErr := os.Lstat(full)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				skipped = append(skipped, skipLabel(root))
			} else {
				errs = append(errs, fmt.Sprintf("%s: %v", root, statErr))
			}
			continue
		}
		if info.IsDir() {
			walkErr := filepath.WalkDir(full, func(path string, d fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					errs = append(errs, fmt.Sprintf("%s: %v", path, walkErr))
					return nil
				}
				if d.IsDir() {
					return nil
				}
				if !d.Type().IsRegular() {
					return nil
				}
				n, saveErr := saveIngestFile(ctx, mem, sid, absDir, path)
				if saveErr != nil {
					errs = append(errs, saveErr.Error())
					return nil
				}
				ingested += n
				return nil
			})
			if walkErr != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", root, walkErr))
			}
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}
		n, saveErr := saveIngestFile(ctx, mem, sid, absDir, full)
		if saveErr != nil {
			errs = append(errs, saveErr.Error())
			continue
		}
		ingested += n
	}

	for _, raw := range extra {
		n, errMsg := ingestExtra(ctx, mem, sid, absDir, raw)
		if errMsg != "" {
			errs = append(errs, errMsg)
			continue
		}
		ingested += n
	}

	return ingested, skipped, errs
}

func skipLabel(root string) string {
	if strings.HasSuffix(root, "/") {
		return root
	}
	// Prefer trailing slash for directory roots so tests accept "docs/" or "docs".
	if root == "docs" || root == ".skillgrid/artifacts" {
		return root + "/"
	}
	return root
}

func outsideJail(rel string) bool {
	relSlash := filepath.ToSlash(rel)
	return relSlash == "." || relSlash == ".." || strings.HasPrefix(relSlash, "../")
}

func ingestExtra(ctx context.Context, mem *memory.Service, sid, absDir, raw string) (ingested int, errMsg string) {
	candidate := raw
	if !filepath.IsAbs(raw) {
		candidate = filepath.Join(absDir, raw)
	}
	absPath, err := filepath.Abs(candidate)
	if err != nil {
		return 0, fmt.Sprintf("docs %s: %v", raw, err)
	}

	realDir, err := filepath.EvalSymlinks(absDir)
	if err != nil {
		return 0, fmt.Sprintf("docs %s: %v", raw, err)
	}
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Sprintf("docs missing: %s", raw)
		}
		return 0, fmt.Sprintf("docs %s: %v", raw, err)
	}

	rel, err := filepath.Rel(realDir, realPath)
	if err != nil || outsideJail(rel) {
		return 0, fmt.Sprintf("docs outside project: %s", raw)
	}

	info, err := os.Lstat(realPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Sprintf("docs missing: %s", raw)
		}
		return 0, fmt.Sprintf("docs %s: %v", raw, err)
	}
	if info.IsDir() {
		var n int
		walkErr := filepath.WalkDir(realPath, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if errMsg != "" {
					errMsg += "; "
				}
				errMsg += fmt.Sprintf("%s: %v", path, walkErr)
				return nil
			}
			if d.IsDir() || !d.Type().IsRegular() {
				return nil
			}
			// Topic Rel must use realDir: EvalSymlinks may rewrite /var → /private/var.
			saved, saveErr := saveIngestFile(ctx, mem, sid, realDir, path)
			if saveErr != nil {
				if errMsg != "" {
					errMsg += "; "
				}
				errMsg += saveErr.Error()
				return nil
			}
			n += saved
			return nil
		})
		if walkErr != nil {
			if errMsg != "" {
				errMsg += "; "
			}
			errMsg += fmt.Sprintf("%s: %v", raw, walkErr)
		}
		return n, errMsg
	}
	if !info.Mode().IsRegular() {
		return 0, fmt.Sprintf("docs not a regular file: %s", raw)
	}
	n, saveErr := saveIngestFile(ctx, mem, sid, realDir, realPath)
	if saveErr != nil {
		return 0, saveErr.Error()
	}
	return n, ""
}

func saveIngestFile(ctx context.Context, mem *memory.Service, sid, absDir, absPath string) (int, error) {
	rel, err := filepath.Rel(absDir, absPath)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", absPath, err)
	}
	relSlash := filepath.ToSlash(rel)

	info, err := os.Lstat(absPath)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", relSlash, err)
	}
	if !info.Mode().IsRegular() {
		return 0, fmt.Errorf("%s: not a regular file", relSlash)
	}
	if info.Size() > maxIngestBytes {
		return 0, fmt.Errorf("%s: exceeds 512KiB limit (%d bytes)", relSlash, info.Size())
	}

	body, err := os.ReadFile(absPath)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", relSlash, err)
	}
	probe := body
	if len(probe) > 512 {
		probe = probe[:512]
	}
	if bytes.IndexByte(probe, 0) >= 0 {
		return 0, fmt.Errorf("%s: binary content", relSlash)
	}
	content := string(body)
	if strings.TrimSpace(content) == "" {
		return 0, fmt.Errorf("%s: empty content", relSlash)
	}
	topicKey := "init/docs/" + relSlash
	_, err = mem.Save(ctx, memory.SaveInput{
		Title:         filepath.Base(absPath),
		Type:          ingestType(relSlash),
		Content:       content,
		Scope:         "project",
		TopicKey:      topicKey,
		SessionID:     sid,
		CapturePrompt: false,
		ToolName:      "skillgrid-init",
	})
	if err != nil {
		return 0, fmt.Errorf("%s: %w", topicKey, err)
	}
	return 1, nil
}

func ingestType(relSlash string) string {
	switch {
	case relSlash == ".skillgrid/ASSUMPTIONS.md",
		relSlash == ".skillgrid/ARCHITECTURE.md",
		strings.HasPrefix(relSlash, ".skillgrid/artifacts/"):
		return "architecture"
	default:
		return "discovery"
	}
}
