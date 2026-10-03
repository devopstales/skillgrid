package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// countTopic counts live observations with the given topic_key.
// Uses Recent because observations_fts does not index topic_key, so Search("init")
// would miss init/docs/* rows whose title/content lack that token.
func countTopic(t *testing.T, mem *memory.Service, key string) int {
	t.Helper()
	hits, err := mem.Recent(context.Background(), 50)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, h := range hits {
		if h.TopicKey == key {
			n++
		}
	}
	return n
}

func TestInitIngestsDefaultPaths(t *testing.T) {
	data := t.TempDir()
	svc := service.New(data)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# App"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "guide.md"), []byte("g"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := projectInit(context.Background(), svc, dir, false, nil); err != nil {
		t.Fatal(err)
	}
	h, closeH, err := svc.OpenForDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	if countTopic(t, h.Memory(), "init/docs/README.md") != 1 {
		t.Fatal("README observation")
	}
	if countTopic(t, h.Memory(), "init/docs/docs/guide.md") != 1 {
		t.Fatal("docs observation")
	}
	closeH()
	// Second init upserts the same topic keys (no duplicates).
	if _, err := projectInit(context.Background(), svc, dir, false, nil); err != nil {
		t.Fatal(err)
	}
	h2, closeH2, err := svc.OpenForDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer closeH2()
	if countTopic(t, h2.Memory(), "init/docs/README.md") != 1 {
		t.Fatal("README duplicated on second init")
	}
	if countTopic(t, h2.Memory(), "init/docs/docs/guide.md") != 1 {
		t.Fatal("docs duplicated on second init")
	}
}

func TestInitSkipsMissingDocs(t *testing.T) {
	svc := service.New(t.TempDir())
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("r"), 0o644)
	res, err := projectInit(context.Background(), svc, dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range res.Skipped {
		if s == "docs/" || s == "docs" {
			found = true
		}
	}
	if !found {
		t.Fatalf("skipped = %v", res.Skipped)
	}
}

func TestInitIngestsExtraDocs(t *testing.T) {
	svc := service.New(t.TempDir())
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.sh"), []byte("echo hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := projectInit(context.Background(), svc, dir, false, []string{filepath.Join(dir, "README.sh")}); err != nil {
		t.Fatal(err)
	}
	h, closeH, err := svc.OpenForDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer closeH()
	if countTopic(t, h.Memory(), "init/docs/README.sh") != 1 {
		t.Fatal("extra docs")
	}
}

func TestInitIngestsRepeatedDocsFlags(t *testing.T) {
	svc := service.New(t.TempDir())
	dir := t.TempDir()
	a := filepath.Join(dir, "a.md")
	b := filepath.Join(dir, "b.md")
	if err := os.WriteFile(a, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := projectInit(context.Background(), svc, dir, false, []string{a, b}); err != nil {
		t.Fatal(err)
	}
	h, closeH, err := svc.OpenForDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer closeH()
	if countTopic(t, h.Memory(), "init/docs/a.md") != 1 {
		t.Fatal("expected init/docs/a.md")
	}
	if countTopic(t, h.Memory(), "init/docs/b.md") != 1 {
		t.Fatal("expected init/docs/b.md")
	}
}

func TestInitMissingExtraDocsIsNonFatal(t *testing.T) {
	svc := service.New(t.TempDir())
	dir := t.TempDir()
	res, err := projectInit(context.Background(), svc, dir, false, []string{filepath.Join(dir, "missing.md")})
	if err != nil {
		t.Fatal(err)
	}
	if res.BootFile == "" || len(res.Errors) == 0 {
		t.Fatalf("res = %+v", res)
	}
}

func TestInitRejectsDocsOutsideProject(t *testing.T) {
	svc := service.New(t.TempDir())
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "x.md")
	_ = os.WriteFile(outside, []byte("no"), 0o644)
	res, err := projectInit(context.Background(), svc, dir, false, []string{outside})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Errors) == 0 {
		t.Fatal("expected jail error")
	}
}

func TestInitRejectsDocsSymlinkEscape(t *testing.T) {
	svc := service.New(t.TempDir())
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.md")
	if err := os.WriteFile(outside, []byte("SECRET_HOST_CONTENT"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "leak.md")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	res, err := projectInit(context.Background(), svc, dir, false, []string{link})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Errors) == 0 {
		t.Fatal("expected symlink jail error")
	}
	h, closeH, err := svc.OpenForDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer closeH()
	if countTopic(t, h.Memory(), "init/docs/leak.md") != 0 {
		t.Fatal("symlink escape ingested host file")
	}
}

func TestInitRejectsSymlinkBootFile(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "host-agents.md")
	if err := os.WriteFile(outside, []byte("HOST"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	_, err := projectInit(context.Background(), service.New(t.TempDir()), dir, false, nil)
	if err == nil {
		t.Fatal("expected symlink boot error")
	}
	body, readErr := os.ReadFile(outside)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(body) != "HOST" {
		t.Fatalf("host file overwritten: %q", body)
	}
}

func TestInitRejectsDocsProjectRoot(t *testing.T) {
	svc := service.New(t.TempDir())
	dir := t.TempDir()
	res, err := projectInit(context.Background(), svc, dir, false, []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Errors) == 0 {
		t.Fatal("expected reject of project-root --docs")
	}
}

// D3: oversized / NUL-binary files are rejected (Errors, non-fatal) and not saved.
func TestInitRejectsOversizedAndBinaryDocs(t *testing.T) {
	svc := service.New(t.TempDir())
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	big := filepath.Join(dir, "docs", "big.md")
	if err := os.WriteFile(big, bytes.Repeat([]byte("a"), maxIngestBytes+1), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "docs", "blob.md")
	if err := os.WriteFile(bin, []byte("ok\x00bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	ok := filepath.Join(dir, "docs", "ok.md")
	if err := os.WriteFile(ok, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := projectInit(context.Background(), svc, dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.BootFile == "" {
		t.Fatal("boot file required; ingest rejects must be non-fatal")
	}
	joined := strings.Join(res.Errors, "; ")
	if !strings.Contains(joined, "exceeds 512KiB limit") {
		t.Fatalf("Errors missing size bound: %q", joined)
	}
	if !strings.Contains(joined, "binary content") {
		t.Fatalf("Errors missing binary reject: %q", joined)
	}

	h, closeH, err := svc.OpenForDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer closeH()
	if countTopic(t, h.Memory(), "init/docs/docs/big.md") != 0 {
		t.Fatal("oversized file was saved")
	}
	if countTopic(t, h.Memory(), "init/docs/docs/blob.md") != 0 {
		t.Fatal("NUL-binary file was saved")
	}
	if countTopic(t, h.Memory(), "init/docs/docs/ok.md") != 1 {
		t.Fatal("valid sibling should still ingest")
	}
}

// D6: relative --docs resolves under project absDir even when process cwd differs.
func TestInitRelativeDocsUsesProjectDirNotCwd(t *testing.T) {
	svc := service.New(t.TempDir())
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte("from project"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Decoy at cwd so a cwd-relative join would pick the wrong file (or fail).
	cwd := t.TempDir()
	if err := os.WriteFile(filepath.Join(cwd, "notes.md"), []byte("from cwd"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)

	res, err := projectInit(context.Background(), svc, dir, false, []string{"notes.md"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Errors) != 0 {
		t.Fatalf("unexpected Errors: %v", res.Errors)
	}
	if res.Ingested < 1 {
		t.Fatalf("Ingested = %d, want >= 1", res.Ingested)
	}

	h, closeH, err := svc.OpenForDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer closeH()
	if countTopic(t, h.Memory(), "init/docs/notes.md") != 1 {
		t.Fatal("relative --docs did not ingest under project")
	}
	hits, err := h.Memory().Recent(context.Background(), 50)
	if err != nil {
		t.Fatal(err)
	}
	for _, hit := range hits {
		if hit.TopicKey == "init/docs/notes.md" && strings.Contains(hit.Content, "from cwd") {
			t.Fatal("ingested cwd file instead of project file")
		}
	}
}

func TestInitHelpListsFlags(t *testing.T) {
	var buf bytes.Buffer
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	newInitFlagSet(fs)
	fs.SetOutput(&buf)
	fs.Usage()
	out := buf.String()
	if !strings.Contains(out, "--force") || !strings.Contains(out, "--docs") {
		t.Fatalf("usage = %q", out)
	}
}

func TestInitWritesBootFileAndReportsCounts(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := projectInit(context.Background(), service.New(t.TempDir()), dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.BootFile == "" {
		t.Fatal("expected boot file path")
	}
	if _, statErr := os.Stat(res.BootFile); statErr != nil {
		t.Fatal(statErr)
	}
}

func TestInitBootFileWriteFailureIsFatal(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "notdir")
	if err := os.WriteFile(dir, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := projectInit(context.Background(), service.New(t.TempDir()), dir, false, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInitUpsertsPreambleAndSentinel(t *testing.T) {
	dir := t.TempDir()
	res, err := projectInit(context.Background(), service.New(t.TempDir()), dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(res.BootFile)
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	if !strings.Contains(s, "<!-- skillgrid-preamble:start -->") || !strings.Contains(s, "# Definition of Done") {
		t.Fatalf("missing preamble: %s", s)
	}
	if strings.Count(s, "<!-- skillgrid:start -->") != 1 || strings.Count(s, "<!-- skillgrid:end -->") != 1 {
		t.Fatalf("sentinel count: %s", s)
	}
}

func TestInitForceRewritesPreamble(t *testing.T) {
	dir := t.TempDir()
	res, err := projectInit(context.Background(), service.New(t.TempDir()), dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(res.BootFile)
	custom := strings.Replace(string(body), "# Project Overview", "# Project Overview\nUSER KEEP", 1)
	if err := os.WriteFile(res.BootFile, []byte(custom+"\n\nUSER BELOW\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res2, err := projectInit(context.Background(), service.New(t.TempDir()), dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	keep, _ := os.ReadFile(res2.BootFile)
	if !strings.Contains(string(keep), "USER KEEP") || !strings.Contains(string(keep), "USER BELOW") {
		t.Fatalf("merge dropped user text: %s", keep)
	}
	if strings.Count(string(keep), "<!-- skillgrid:start -->") != 1 {
		t.Fatal("duplicated sentinel")
	}
	res3, err := projectInit(context.Background(), service.New(t.TempDir()), dir, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	forced, _ := os.ReadFile(res3.BootFile)
	if strings.Contains(string(forced), "USER KEEP") {
		t.Fatal("force left old preamble")
	}
	if !strings.Contains(string(forced), "USER BELOW") {
		t.Fatal("force wiped text outside regions")
	}
}

func TestInitIndexesTheProject(t *testing.T) {
	data := t.TempDir()
	svc := service.New(data)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\nfunc F() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := projectInit(context.Background(), svc, dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Indexed < 1 {
		t.Fatalf("Indexed = %d", res.Indexed)
	}
}

func TestInitIndexFailureIsNonFatal(t *testing.T) {
	prev := initRunIndex
	t.Cleanup(func() { initRunIndex = prev })
	initRunIndex = func(context.Context, *service.Service, string) (int, error) {
		return 0, errors.New("boom")
	}

	data := t.TempDir()
	svc := service.New(data)
	dir := t.TempDir()
	res, err := projectInit(context.Background(), svc, dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.BootFile == "" {
		t.Fatal("boot file required even when index fails")
	}
	if _, statErr := os.Stat(res.BootFile); statErr != nil {
		t.Fatal(statErr)
	}
	if len(res.Errors) < 1 {
		t.Fatal("expected index failure under Errors")
	}
	joined := strings.Join(res.Errors, "; ")
	if !strings.Contains(joined, "index failed") {
		t.Fatalf("Errors = %q, want index failed", joined)
	}
}

func TestOnboardingSkillCallsSkillgridInit(t *testing.T) {
	body, err := os.ReadFile(onboardingSkillPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(body, []byte("skillgrid init")) {
		t.Fatal("onboarding must name skillgrid init")
	}
}

func TestOnboardingSkillPassesDocsThrough(t *testing.T) {
	body, err := os.ReadFile(onboardingSkillPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(body, []byte("--docs")) {
		t.Fatal("onboarding must pass extra paths as --docs")
	}
}

func TestOnboardingSkillHasNoSecondIngest(t *testing.T) {
	body, err := os.ReadFile(onboardingSkillPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(body, []byte("walk docs/ and mem_save each file")) {
		t.Fatal("skill must not re-specify ingest")
	}
}

func onboardingSkillPath(t *testing.T) string {
	t.Helper()
	// repo root: this file is skillgrid-cli/cmd/skillgrid
	p := filepath.Join("..", "..", "..", ".agents", "skills", "lifecycle", "onboarding", "SKILL.md")
	if _, err := os.Stat(p); err != nil {
		t.Fatal(err)
	}
	return p
}
