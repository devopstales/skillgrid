package eval

import (
	"regexp"
	"strings"
)

// QuerySet is one leak-free, git-history-derived evaluation query: the natural
// commit subject (the human's words, not a leak) paired with the files the
// commit changed (the ground truth the retriever should surface).
type QuerySet struct {
	Subject       string
	ExpectedFiles []string
}

// CORPUS_EXCLUDES are path substrings for the benchmark scaffolding itself.
// Files under any of these are dropped from the graded corpus (01.18): the eval
// harness is a measuring instrument, not a retrieval signal.
var CORPUS_EXCLUDES = []string{
	"/eval/",
	"internal/mnemonic/eval",
	".skillgrid/sdd",
}

var (
	reMerge   = regexp.MustCompile(`(?i)^\s*merge\b`)
	// reRevert is anchored to the start: only subjects that BEGIN with a revert
	// are noise. A position-free `revert\b` over-drops subjects like
	// "fix: avoid reverting state" (which merely mention reverting).
	reRevert = regexp.MustCompile(`(?i)^\s*(this\s+)?revert(s|ed)?\b`)
	reRelease = regexp.MustCompile(`(?i)^\s*(release|v?\d+\.\d+(?:\.\d+)?)\b`)
	reBump    = regexp.MustCompile(`(?i)\b(bump|bumped|update.*\b(?:version|go\.mod|go\.sum|deps?)\b|tidy\b)\b`)
	reFormat  = regexp.MustCompile(`(?i)\b(gofmt|format(ting)?|whitespace|reformat)\b`)
	reChangelogFile = regexp.MustCompile(`(?i)(^|/)changelog\.(md|rst|txt|log)$`)
	reBenchFile  = regexp.MustCompile(`(?i)(^|/)\w*bench\w*\.go$`)
)

// ExcludedFromCorpus reports whether path is benchmark scaffolding that must be
// dropped from the graded corpus (01.18).
func ExcludedFromCorpus(path string) bool {
	p := strings.ReplaceAll(path, "\\", "/")
	for _, ex := range CORPUS_EXCLUDES {
		if strings.Contains(p, ex) {
			return true
		}
	}
	return false
}

// isNoiseSubject reports whether a commit subject is merge/revert/release/
// bump/formatting/changelog-like noise (01.17).
func isNoiseSubject(subject string) bool {
	s := strings.TrimSpace(subject)
	if reMerge.MatchString(s) {
		return true
	}
	if reRevert.MatchString(s) {
		return true
	}
	if reRelease.MatchString(s) {
		return true
	}
	if reBump.MatchString(s) {
		return true
	}
	if reFormat.MatchString(s) {
		return true
	}
	return false
}

// isChangelogLike reports whether a commit is a changelog-style change: the
// subject names a changelog, or every changed file is a changelog file.
func isChangelogLike(subject string, files []string) bool {
	if reChangelogFile.MatchString(strings.TrimSpace(subject)) {
		return true
	}
	n := 0
	for _, f := range files {
		if reChangelogFile.MatchString(f) {
			n++
		}
	}
	return len(files) > 0 && n == len(files)
}

// isBenchmarkTouching reports whether a commit only touches benchmark files
// (the bench scaffolding, not real source).
func isBenchmarkTouching(files []string) bool {
	if len(files) == 0 {
		return false
	}
	n := 0
	for _, f := range files {
		if reBenchFile.MatchString(f) {
			n++
		}
	}
	return n == len(files)
}

// BuildQuerySet derives one leak-free query from a git commit: its subject +
// the changed files. It returns nil (no query) when the commit is noise — a
// merge, revert, release, version-bump, formatting-only change, a changelog-like
// commit, or a benchmark-only change (01.17) — or when every changed file is
// benchmark scaffolding (01.18).
func BuildQuerySet(subject string, changedFiles []string) (*QuerySet, error) {
	if isNoiseSubject(subject) {
		return nil, nil
	}
	var real []string
	for _, f := range changedFiles {
		if ExcludedFromCorpus(f) {
			continue
		}
		real = append(real, f)
	}
	if len(real) == 0 {
		return nil, nil
	}
	if isChangelogLike(subject, real) {
		return nil, nil
	}
	if isBenchmarkTouching(real) {
		return nil, nil
	}
	return &QuerySet{Subject: strings.TrimSpace(subject), ExpectedFiles: real}, nil
}

// DeriveQuerySet builds the full leak-free query set from a slice of commits,
// in commit order (deterministic).
func DeriveQuerySet(commits []Commit) []*QuerySet {
	var out []*QuerySet
	for _, c := range commits {
		q, err := BuildQuerySet(c.Subject, c.Files)
		if err != nil || q == nil {
			continue
		}
		out = append(out, q)
	}
	return out
}

// Commit is one git-history row fed to the query generator.
type Commit struct {
	Hash    string
	Subject string
	Files   []string
}
