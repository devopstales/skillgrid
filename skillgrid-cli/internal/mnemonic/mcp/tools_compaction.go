package mcp

import (
	"context"
	"strings"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/facts"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/skills"
)

func registerCompactionTools(s *server.MCPServer) {
	s.AddTool(mnemonicCommitTool(), handleMnemonicCommit)
}

func mnemonicCommitTool() mcplib.Tool {
	return mcplib.NewTool("mnemonic_commit",
		mcplib.WithDescription("Explicitly commit lessons into long-term memory (L2 durable; L0/L1 async). Does not run on session end. Also extracts each lesson bullet as a Fact Memory row and auto-registers a skill when a lesson states a reusable pattern ([language] prefix)."),
		mcplib.WithString("title", mcplib.Description("Memory title")),
		mcplib.WithString("lessons_learned", mcplib.Description("Lesson body used as L2 when content is empty")),
		mcplib.WithString("content", mcplib.Description("Optional full L2 markdown")),
		mcplib.WithString("source_link", mcplib.Description("Optional source link")),
		mcplib.WithString("task_id", mcplib.Description("Optional team task id (future 001 hook)")),
		mcplib.WithString("project", mcplib.Description("Project id (defaults to CWD resolve)")),
		mcplib.WithString("session_id", mcplib.Description("Session id to attribute the extracted facts and skill to (required for fact extraction; commit still succeeds without it)")),
	)
}

func handleMnemonicCommit(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	projectID, err := projectIDFor(svc, req.GetString("project", ""))
	if err != nil {
		return toolError(err)
	}
	in := service.MnemonicCommitInput{
		Title:          req.GetString("title", ""),
		LessonsLearned: req.GetString("lessons_learned", ""),
		Content:        req.GetString("content", ""),
		SourceLink:     req.GetString("source_link", ""),
		TaskID:         req.GetString("task_id", ""),
	}
	// 003 compaction behavior: L2 + long_term_memories, unchanged.
	out, err := svc.MnemonicCommit(ctx, projectID, in, nil)
	if err != nil {
		return toolError(err)
	}

	// TICKET-05: extract facts from the committed body and optionally
	// auto-register a skill when a reusable pattern is detected. Extraction
	// failures never fail the commit — they surface as warnings.
	sessionID := strings.TrimSpace(req.GetString("session_id", ""))
	body := strings.TrimSpace(in.Content)
	if body == "" {
		body = strings.TrimSpace(in.LessonsLearned)
	}
	res := map[string]any{
		"memory_id": out.MemoryID,
		"paths": map[string]string{
			"full_path": out.FullPath,
		},
		"title": out.Title,
	}
	var warnings []string
	if sessionID == "" {
		res["facts_extracted"] = 0
		res["warnings"] = append(warnings, "session_id missing: fact extraction and auto-skill skipped")
		return JSONResult(res)
	}

	h, cleanup, err := svc.Open(projectID)
	if err != nil {
		warnings = append(warnings, "fact extraction skipped: "+err.Error())
		res["facts_extracted"] = 0
		res["warnings"] = warnings
		return JSONResult(res)
	}
	defer cleanup()

	bullets := lessonBullets(body)
	factStore := facts.New(h.Store().DB, projectID)
	extracted := 0
	for _, b := range bullets {
		if _, err := factStore.Add(ctx, sessionID, b); err != nil {
			warnings = append(warnings, "fact extract failed: "+err.Error())
			continue
		}
		extracted++
	}
	res["facts_extracted"] = extracted

	// Auto-skill: only a lesson that states a reusable pattern — a
	// [language] tag plus an imperative pattern statement — qualifies.
	if lang, pattern, ok := autoSkillFrom(bullets); ok {
		if _, err := skills.New(h.Store().DB, h.Root(), projectID).Write(ctx, skillNameFromPattern(pattern), lang, pattern, "", true); err != nil {
			warnings = append(warnings, "auto-skill skipped: "+err.Error())
		} else {
			res["skill_name"] = skillNameFromPattern(pattern)
		}
	} else {
		warnings = append(warnings, "auto-skill skipped: no reusable pattern detected (use a [language] bullet like '[go] write table-driven tests with t.Run subtests')")
	}
	if len(warnings) > 0 {
		res["warnings"] = warnings
	}
	return JSONResult(res)
}

// lessonBullets returns the lesson items of a committed body: markdown list
// items ("- " / "* " / "1. ") and, when the body has no list items, each
// non-empty line. Whitespace is trimmed; empty items are dropped.
func lessonBullets(body string) []string {
	var out []string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			line = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "-"), "*"))
		} else if len(line) >= 3 && line[1] == '.' && line[0] >= '0' && line[0] <= '9' {
			line = strings.TrimSpace(line[2:])
		}
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

// autoSkillFrom detects a reusable-pattern lesson: a bullet carrying a
// [language] tag whose remainder states an imperative pattern. Only the
// first qualifying bullet wins. ok=false means no auto-skill (warn+continue).
func autoSkillFrom(bullets []string) (language, pattern string, ok bool) {
	langs := map[string]bool{
		"go": true, "typescript": true, "javascript": true, "python": true,
		"rust": true, "java": true, "ruby": true, "csharp": true, "c": true,
		"cpp": true, "swift": true, "kotlin": true, "php": true, "elixir": true,
	}
	for _, b := range bullets {
		if !strings.HasPrefix(b, "[") {
			continue
		}
		rest, pattern, found := strings.Cut(strings.TrimPrefix(b, "["), "]")
		if !found {
			continue
		}
		lang := strings.ToLower(strings.TrimSpace(rest))
		if !langs[lang] {
			continue
		}
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		return lang, pattern, true
	}
	return "", "", false
}

// skillNameFromPattern derives a stable skill name from the pattern text:
// lowercased alphanumerics collapsed to hyphens, capped at 48 chars.
func skillNameFromPattern(pattern string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(pattern) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		case !prevDash && b.Len() > 0:
			b.WriteByte('-')
			prevDash = true
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		return "reusable-pattern"
	}
	if len(s) > 48 {
		s = strings.Trim(s[:48], "-")
	}
	return s
}
