package http

import (
	"context"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// gitMaxOutput bounds a single git command's combined output so a huge diff,
// blame, or log cannot hold multi-MB strings in memory per request (review A3).
const gitMaxOutput = 4 << 20 // 4 MiB

// gitTimeout bounds a single git subprocess (review A3).
const gitTimeout = 10 * time.Second

// shaRE validates a git revision: 7-40 hex chars. Rejects anything else (in
// particular a leading `-` or a `--flag=`-shaped value) so the value is never
// parsed as a git option (review A2).
var shaRE = regexp.MustCompile(`^[0-9a-fA-F]{7,40}$`)

// isSHA reports whether s is a plausible git revision (see shaRE).
func isSHA(s string) bool { return shaRE.MatchString(s) }

// ctxIsTimeout reports whether err is a git subprocess timeout (review A3).
func ctxIsTimeout(err error) bool { return err == context.DeadlineExceeded }

// isRelPath reports whether s is a safe relative repo path: non-empty, no
// leading `-` (so it is never parsed as a git option), no `..` segment, and no
// absolute path. Used for the `?path=` query arg (review A2).
func isRelPath(s string) bool {
	if s == "" || s[0] == '-' {
		return false
	}
	if strings.HasPrefix(s, "/") || len(s) >= 2 && s[1] == ':' {
		return false
	}
	for _, seg := range strings.Split(s, "/") {
		if seg == ".." {
			return false
		}
	}
	return true
}

// gitCmd runs a git command in root and returns stdout, or an error. root is
// the repo (sddRoot()). Used by all the read-only /git handlers. Output is
// capped at gitMaxOutput and the subprocess is bounded by gitTimeout.
func gitCmd(root string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return string(out), context.DeadlineExceeded
		}
		return string(out), err
	}
	if len(out) > gitMaxOutput {
		return string(out[:gitMaxOutput]) + "\n…[truncated]", nil
	}
	return string(out), nil
}

// isGitRepo reports whether root is inside a git worktree.
func isGitRepo(root string) bool {
	_, err := gitCmd(root, "rev-parse", "--is-inside-work-tree")
	return err == nil
}

// handleGitCommits serves GET /git/commits?limit=N — the commit log (newest
// first) with per-commit +/- stats and author. Not-a-repo → 503.
func (s *Server) handleGitCommits(w http.ResponseWriter, r *http.Request) {
	root := sddRoot()
	if !isGitRepo(root) {
		writeError(w, http.StatusServiceUnavailable, "not a git repository: "+root)
		return
	}
	limit := queryInt(r, "limit", 50)
	if limit < 1 {
		limit = 1
	}
	if limit > 500 {
		limit = 500
	}

	// %H sha | %an author | %ad date | %s subject | %b body — one record per
	// commit, \x1e (RS) separator between records, \x1f (US) between fields.
	// --numstat appends the per-commit +/- stats to the SAME single call, so
	// there is no per-commit `git show` fan-out (review A4: was N+1 spawns).
	const sep = "\x1e"
	const fsep = "\x1f"
	out, err := gitCmd(root, "log",
		"--max-count", strconv.Itoa(limit),
		"--numstat",
		"--pretty=format:"+sep+"%H"+fsep+"%an"+fsep+"%ad"+fsep+"%s"+fsep+"%b",
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "git log: "+err.Error())
		return
	}

	commits := []gitCommit{}
	for _, rec := range strings.Split(out, sep) {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		f := strings.SplitN(rec, fsep, 5)
		if len(f) < 4 {
			continue
		}
		c := gitCommit{SHA: f[0], Author: f[1], Date: f[2], Message: f[3]}
		if len(f) == 5 {
			c.Body = f[4]
		}
		// +/- stats from the numstat block appended after the body (same call).
		c.Additions, c.Deletions = parseNumstat(c.Body)
		commits = append(commits, c)
	}
	writeJSON(w, http.StatusOK, map[string]any{"commits": commits, "limit": limit})
}

// handleGitCommit serves GET /git/commits/{sha} — commit detail (metadata +
// changed files). Unknown sha → 404.
func (s *Server) handleGitCommit(w http.ResponseWriter, r *http.Request) {
	root := sddRoot()
	if !isGitRepo(root) {
		writeError(w, http.StatusServiceUnavailable, "not a git repository: "+root)
		return
	}
	sha := r.PathValue("sha")
	if !isSHA(sha) {
		writeError(w, http.StatusBadRequest, "invalid commit sha: "+sha)
		return
	}
	out, err := gitCmd(root, "show", "--numstat", "--format=%H%x1f%an%x1f%ad%x1f%s%x1f%b%x1e", sha)
	if err != nil {
		if ctxIsTimeout(err) {
			writeError(w, http.StatusGatewayTimeout, "git show timed out: "+sha)
			return
		}
		if strings.Contains(err.Error(), "unknown revision") || strings.Contains(out, "unknown revision") {
			writeError(w, http.StatusNotFound, "unknown commit: "+sha)
			return
		}
		writeError(w, http.StatusInternalServerError, "git show: "+err.Error())
		return
	}
	rec := strings.TrimSpace(out)
	if rec == "" {
		writeError(w, http.StatusNotFound, "unknown commit: "+sha)
		return
	}
	f := strings.SplitN(rec, "\x1f", 5)
	c := gitCommit{}
	if len(f) >= 4 {
		c.SHA, c.Author, c.Date, c.Message = f[0], f[1], f[2], f[3]
		if len(f) == 5 {
			c.Body = f[4]
		}
	}
	// The numstat block is everything after the message body; parse files.
	numstatPart := c.Body
	// Separate message body from numstat: numstat lines are "adds\tdels\tpath".
	var files []gitFileStat
	var msgLines []string
	for _, line := range strings.Split(numstatPart, "\n") {
		if isNumstatLine(line) {
			f := strings.Fields(line)
			if len(f) >= 3 {
				files = append(files, gitFileStat{Path: f[2], Additions: atoi(f[0]), Deletions: atoi(f[1])})
			}
		} else {
			msgLines = append(msgLines, line)
		}
	}
	c.Message = strings.TrimSpace(c.Message + "\n" + strings.Join(msgLines, "\n"))
	c.Files = files
	add, del := 0, 0
	for _, fs := range files {
		add += fs.Additions
		del += fs.Deletions
	}
	c.Additions, c.Deletions = add, del
	writeJSON(w, http.StatusOK, c)
}

// handleGitDiff serves GET /git/diff/{sha} — the unified diff of a commit.
// Unknown sha → 404.
func (s *Server) handleGitDiff(w http.ResponseWriter, r *http.Request) {
	root := sddRoot()
	if !isGitRepo(root) {
		writeError(w, http.StatusServiceUnavailable, "not a git repository: "+root)
		return
	}
	sha := r.PathValue("sha")
	if !isSHA(sha) {
		writeError(w, http.StatusBadRequest, "invalid commit sha: "+sha)
		return
	}
	diff, err := gitCmd(root, "show", "--format=", sha)
	if err != nil {
		if ctxIsTimeout(err) {
			writeError(w, http.StatusGatewayTimeout, "git show timed out: "+sha)
			return
		}
		if strings.Contains(err.Error(), "unknown revision") || strings.Contains(diff, "unknown revision") {
			writeError(w, http.StatusNotFound, "unknown commit: "+sha)
			return
		}
		writeError(w, http.StatusInternalServerError, "git show: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sha": sha, "diff": diff})
}

// handleGitFileHistory serves GET /git/file-history?path=... — the commit
// history for one file. Unknown path → 404.
func (s *Server) handleGitFileHistory(w http.ResponseWriter, r *http.Request) {
	root := sddRoot()
	if !isGitRepo(root) {
		writeError(w, http.StatusServiceUnavailable, "not a git repository: "+root)
		return
	}
	path := r.URL.Query().Get("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "path required")
		return
	}
	if !isRelPath(path) {
		writeError(w, http.StatusBadRequest, "invalid path: "+path)
		return
	}
	hlimit := queryInt(r, "limit", 50)
	if hlimit < 1 {
		hlimit = 1
	}
	if hlimit > 500 {
		hlimit = 500
	}
	out, err := gitCmd(root, "log", "--max-count", strconv.Itoa(hlimit), "--pretty=format:%H%x1f%an%x1f%ad%x1f%s", "--", path)
	if err != nil {
		if ctxIsTimeout(err) {
			writeError(w, http.StatusGatewayTimeout, "git log timed out: "+path)
			return
		}
		// A path with no history errors out.
		writeError(w, http.StatusNotFound, "no history for: "+path)
		return
	}
	hist := []gitCommit{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		f := strings.SplitN(line, "\x1f", 4)
		if len(f) < 4 {
			continue
		}
		hist = append(hist, gitCommit{SHA: f[0], Author: f[1], Date: f[2], Message: f[3]})
	}
	if len(hist) == 0 {
		writeError(w, http.StatusNotFound, "no history for: "+path)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"path": path, "history": hist})
}

// handleGitBlame serves GET /git/blame?path=... — per-line blame. Unknown
// path → 404.
func (s *Server) handleGitBlame(w http.ResponseWriter, r *http.Request) {
	root := sddRoot()
	if !isGitRepo(root) {
		writeError(w, http.StatusServiceUnavailable, "not a git repository: "+root)
		return
	}
	path := r.URL.Query().Get("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "path required")
		return
	}
	if !isRelPath(path) {
		writeError(w, http.StatusBadRequest, "invalid path: "+path)
		return
	}
	// %h sha | %l line# | %an author | %s summary | line content (the last
	// token after the 4th tab is the file line).
	out, err := gitCmd(root, "blame", "--porcelain", "--", path)
	if err != nil {
		if ctxIsTimeout(err) {
			writeError(w, http.StatusGatewayTimeout, "git blame timed out: "+path)
			return
		}
		writeError(w, http.StatusNotFound, "blame: "+path+" ("+err.Error()+")")
		return
	}
	lines := []gitBlameLine{}
	sha, author, summary := "", "", ""
	lineNo := 0
	for _, raw := range strings.Split(out, "\n") {
		// Metadata lines are tab-prefixed; the content line is \t<text> too, so
		// distinguish by shape. The hunk header has NO leading tab.
		if !strings.HasPrefix(raw, "\t") {
			trimmed := strings.TrimSpace(raw)
			if trimmed == "" {
				continue
			}
			if isBlameHeader(trimmed) {
				sha = strings.Fields(trimmed)[0]
				author, summary = "", ""
			}
			continue
		}
		body := strings.TrimPrefix(raw, "\t")
		switch {
		case strings.HasPrefix(body, "author "):
			author = strings.TrimPrefix(body, "author ")
		case strings.HasPrefix(body, "summary "):
			summary = strings.TrimPrefix(body, "summary ")
		case strings.HasPrefix(body, "prev ") ||
			strings.HasPrefix(body, "filename "):
			// ignore
		default:
			// Content line.
			lineNo++
			lines = append(lines, gitBlameLine{Line: lineNo, SHA: sha, Author: author, Summary: summary, Text: body})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"path": path, "lines": lines})
}

// isBlameHeader reports whether a porcelain-blame line is the per-hunk header
// (starts with a 40-hex sha followed by spaces).
var blameHeaderRE = regexp.MustCompile(`^[0-9a-f]{40} \d+ \d+`)

func isBlameHeader(s string) bool { return blameHeaderRE.MatchString(s) }

// gitCommit is one commit in the log / detail.
type gitCommit struct {
	SHA       string        `json:"sha"`
	Author    string        `json:"author"`
	Date      string        `json:"date"`
	Message   string        `json:"message"`
	Body      string        `json:"body,omitempty"`
	Additions int           `json:"additions"`
	Deletions int           `json:"deletions"`
	Files     []gitFileStat `json:"files,omitempty"`
}

type gitFileStat struct {
	Path      string `json:"path"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

type gitBlameLine struct {
	Line    int    `json:"line"`
	SHA     string `json:"sha"`
	Author  string `json:"author"`
	Summary string `json:"summary"`
	Text    string `json:"text"`
}

// parseNumstat sums additions/deletions from a `git --numstat` block.
func parseNumstat(s string) (add, del int) {
	for _, line := range strings.Split(s, "\n") {
		if isNumstatLine(line) {
			f := strings.Fields(line)
			if len(f) >= 2 {
				add += atoi(f[0])
				del += atoi(f[1])
			}
		}
	}
	return add, del
}

// isNumstatLine reports whether a line is a `git --numstat` record
// ("adds\tdels\tpath", where adds/dels are digits or "-" for binary).
func isNumstatLine(line string) bool {
	f := strings.Fields(line)
	if len(f) < 3 {
		return false
	}
	return isNum(f[0]) && isNum(f[1])
}

func isNum(s string) bool {
	if s == "-" {
		return true
	}
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}


