package affected

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// Area is one affected area in a --base result: a changed (or transitively
// touched) file, the collapsed hop detail that produced its affected tests,
// the affected test files, and the git-history owners ("who to tag") derived
// from `git blame` on the changed lines. Per-symbol detail is collapsed
// under the area — the area is the reporting unit for a ready PR comment.
type Area struct {
	Name      string   `json:"name"`
	Changed   bool     `json:"changed"`
	TestFiles []string `json:"test_files"`
	Hops      int      `json:"hops"`
	Owners    []string `json:"owners"`
}

// BaseResult extends Result with the --base grouping: affected areas and the
// resolved merge-base commit.
type BaseResult struct {
	Result
	BaseRef   string `json:"base_ref"`
	MergeBase string `json:"merge_base,omitempty"`
	Areas     []Area `json:"areas"`
}

// gitChangedSet returns the changed file set for base ref: the files touched
// between `git merge-base <base> HEAD` and HEAD (git diff --name-only),
// repo-relative. An unknown ref is a clear error (no invented changed set).
func gitChangedSet(ctx context.Context, repo, baseRef string) ([]string, string, error) {
	mergeBase, err := gitOutput(ctx, repo, "merge-base", baseRef, "HEAD")
	if err != nil {
		return nil, "", fmt.Errorf("merge-base %s: %v", baseRef, err)
	}
	files, err := gitOutput(ctx, repo, "diff", "--name-only", mergeBase, "HEAD")
	if err != nil {
		return nil, mergeBase, fmt.Errorf("diff %s: %v", mergeBase, err)
	}
	var out []string
	for _, line := range strings.Split(files, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return dedupeSorted(out), mergeBase, nil
}

// AffectedBase derives the changed set from `git merge-base <base> HEAD` +
// diff (no --stdin plumbing), runs the same Affected traversal, and groups
// the result into affected areas with git-history owners (git blame on the
// changed lines of each changed file).
func AffectedBase(ctx context.Context, db *sql.DB, repo, baseRef string, opts Options) (BaseResult, error) {
	changed, mergeBase, err := gitChangedSet(ctx, repo, baseRef)
	if err != nil {
		return BaseResult{}, err
	}
	opts.Changed = changed
	res, err := Affected(ctx, db, opts)
	if err != nil {
		return BaseResult{}, err
	}
	out := BaseResult{Result: res, BaseRef: baseRef, MergeBase: mergeBase, Areas: []Area{}}
	for _, p := range res.Changed {
		owners, oerr := gitBlameOwners(ctx, repo, p)
		if oerr != nil {
			owners = []string{}
		}
		out.Areas = append(out.Areas, Area{Name: p, Changed: true, TestFiles: []string{}, Owners: owners})
	}
	// Collapse hop detail + affected tests under each area. A hop lands in
	// the source area (when the source is a changed file) or the target's
	// area; affected test files attach to the areas whose hops reached them.
	areaByName := map[string]*Area{}
	for i := range out.Areas {
		areaByName[out.Areas[i].Name] = &out.Areas[i]
	}
	testSeen := map[string]map[string]bool{}
	for _, r := range res.Relationships {
		var target *Area
		if a, ok := areaByName[r.From]; ok {
			target = a
		} else if a, ok := areaByName[r.DepPath]; ok {
			target = a
		}
		if target == nil {
			continue
		}
		target.Hops++
		if isTestPath(r.DepPath) && matchFilter(r.DepPath, opts.Filter) {
			if testSeen[r.DepPath] == nil {
				testSeen[r.DepPath] = map[string]bool{}
			}
			testSeen[r.DepPath][target.Name] = true
			if !hasString(target.TestFiles, r.DepPath) {
				target.TestFiles = append(target.TestFiles, r.DepPath)
			}
		}
	}
	for i := range out.Areas {
		sort.Strings(out.Areas[i].TestFiles)
	}
	sort.SliceStable(out.Areas, func(i, j int) bool {
		return out.Areas[i].Name < out.Areas[j].Name
	})
	return out, nil
}

// gitBlameOwners returns the git-history owners ("who to tag") for the
// changed lines of path: `git blame HEAD -- <path>` last-author-wins over the
// whole file (the changed lines are a subset of it; when the file is new the
// blame covers the added lines). Deduped, stable order. A missing file (new
// on the base ref) yields the file's introducing author via
// `git log --diff-filter=A`.
func gitBlameOwners(ctx context.Context, repo, path string) ([]string, error) {
	out, err := gitOutput(ctx, repo, "blame", "HEAD", "--", path)
	if err != nil {
		return gitNewFileOwners(ctx, repo, path)
	}
	var owners []string
	seen := map[string]bool{}
	sc := bufio.NewScanner(strings.NewReader(out))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		author := parseBlameAuthor(sc.Text())
		if author == "" || seen[author] {
			continue
		}
		seen[author] = true
		owners = append(owners, author)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(owners) == 0 {
		return gitNewFileOwners(ctx, repo, path)
	}
	return owners, nil
}

// isDigits reports whether s is non-empty and all ASCII digits (the line
// number field of a `git blame` metadata group).
func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// isEpoch reports whether s is exactly 10 digits (a unix epoch timestamp).
func isEpoch(s string) bool {
	if len(s) != 10 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// isTz reports whether s is a git timestamp timezone offset: a 5-char signed
// offset like "+0200" or "-0700" (the " <tz>" field of regular `git blame`).
func isTz(s string) bool {
	if len(s) != 5 || (s[0] != '+' && s[0] != '-') {
		return false
	}
	for i := 1; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// isDateOrTime reports whether s is a "YYYY-MM-DD" date or an "H:MM[:SS]"
// time — the date/time fields of regular `git blame` metadata.
func isDateOrTime(s string) bool {
	if len(s) >= 5 && s[4] == '-' && s[0] >= '0' && s[0] <= '9' {
		return true // date-like (YYYY-MM-...)
	}
	dot := strings.IndexByte(s, ':')
	if dot > 0 && dot < len(s)-1 {
		for i := 0; i < len(s); i++ {
			if s[i] != ':' && (s[i] < '0' || s[i] > '9') {
				return false
			}
		}
		return true // time-like (H:MM or HH:MM:SS)
	}
	return false
}

func hasString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// gitNewFileOwners names the author that introduced a file (git log
// --diff-filter=A), used when blame has nothing to report (file absent on
// HEAD or blame formatting drift).
func gitNewFileOwners(ctx context.Context, repo, path string) ([]string, error) {
	out, err := gitOutput(ctx, repo, "log", "--diff-filter=A", "--format=%aN", "HEAD", "--", path)
	if err != nil {
		return []string{}, nil
	}
	var owners []string
	seen := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || seen[line] {
			continue
		}
		seen[line] = true
		owners = append(owners, line)
	}
	return owners, nil
}

// parseBlameAuthor extracts the author field from one `git blame` line. The
// line is "<hash> (<author> <timestamp> <linenum>) <content>" when the file
// is tracked, or "^<hash> (<author> <timestamp> <linenum>) <content>" for
// boundary commits (the leading ^ marks "not the result of a merge"; it is
// part of the hash, before the metadata parens). The first "(" opens the
// metadata group, the first ")" closes it; the author is between them (sans
// the timestamp/linenum).
func parseBlameAuthor(line string) string {
	open := strings.Index(line, "(")
	closeIdx := strings.Index(line, ")")
	if open < 0 || closeIdx < open {
		return ""
	}
	inner := line[open+1 : closeIdx]
	// The metadata group is "<author> <date> <tz> <linenum>" (regular `git
	// blame`) or "<author> <epoch> <linenum>" (epoch form). The line number
	// is the final group (always digits); the author is everything before
	// the date field — so author names containing digits ("Dev2") are
	// preserved.
	rest := inner
	if j := strings.LastIndex(rest, " "); j >= 0 && isDigits(rest[j+1:]) {
		rest = rest[:j] // drop the line number (always all-digits)
	}
	if i := strings.LastIndex(rest, " "); i >= 0 && isEpoch(rest[i+1:]) {
		rest = rest[:i] // drop the epoch date
		return strings.TrimSpace(rest)
	}
	// Regular form: "<author> <YYYY-MM-DD HH:MM:SS> <tz>". The tz is a
	// 5-char signed offset (e.g. +0200, -0700) — drop it.
	if i := strings.LastIndex(rest, " "); i >= 0 && isTz(rest[i+1:]) {
		rest = rest[:i]
	}
	// Drop the time (HH:MM:SS, 1:20) and date (YYYY-MM-DD) fields.
	for i := strings.LastIndex(rest, " "); i >= 0; i = strings.LastIndex(rest, " ") {
		field := rest[i+1:]
		if !isDateOrTime(field) {
			break
		}
		rest = rest[:i]
	}
	return strings.TrimSpace(rest)
}

// gitOutput runs a git command in repo and returns its trimmed stdout.
func gitOutput(ctx context.Context, repo string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repo
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%v: %s", strings.Join(args, " "), strings.TrimSpace(string(ee.Stderr)))
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
