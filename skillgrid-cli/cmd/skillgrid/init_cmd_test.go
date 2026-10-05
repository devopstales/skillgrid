package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// TestMain points blockMDPath at the repo's real block.md for the whole test
// binary. init's tests run against a temp project dir that has no .agents tree,
// so without this the resolver would fall back to the installed ~/.skillgrid
// mirror, which can be stale. The test binary runs from skillgrid-cli/cmd/
// skillgrid, so the repo root is ../../../.
func TestMain(m *testing.M) {
	prevBlock := blockMDPath
	prevPreamble := agentsPreamblePath
	blockMDPath = func(dir string) string {
		return "../../../.agents/skills/_shared/agent-config/block.md"
	}
	agentsPreamblePath = func(dir string) string {
		return "../../../.agents/skills/lifecycle/onboarding/templates/agents-preamble.md"
	}
	code := m.Run()
	blockMDPath = prevBlock
	agentsPreamblePath = prevPreamble
	os.Exit(code)
}

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
	if !strings.Contains(s, "<!-- skillgrid-preamble:start -->") || !strings.Contains(s, "## Commands") {
		t.Fatalf("missing preamble: %s", s)
	}
	if strings.Count(s, "<!-- skillgrid:start -->") != 1 || strings.Count(s, "<!-- skillgrid:end -->") != 1 {
		t.Fatalf("sentinel count: %s", s)
	}
}

// TestInitKeepsOnboardingRenderedSentinel: the onboarding skill renders the
// canonical ## Skillgrid block into AGENTS.md from block.md before step 9 runs
// `skillgrid init`. init must not clobber that block with its own (older,
// config-blind) sentinelTemplate — only the preamble and the index are init's job.
func TestInitKeepsOnboardingRenderedSentinel(t *testing.T) {
	dir := t.TempDir()
	onboarded := "Project: **MyRealName**.\nOnboarding custom row: keep this."
	agents := "<!-- skillgrid:start -->\n## Skillgrid\n\n" + onboarded + "\n<!-- skillgrid:end -->\n"
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(agents), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := projectInit(context.Background(), service.New(t.TempDir()), dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(res.BootFile)
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	if !strings.Contains(s, onboarded) {
		t.Fatalf("onboarding-rendered block was clobbered by init:\n%s", s)
	}
	if strings.Count(s, "<!-- skillgrid:start -->") != 1 || strings.Count(s, "<!-- skillgrid:end -->") != 1 {
		t.Fatalf("sentinel count wrong:\n%s", s)
	}
}

// TestInitCreatePathUsesConfigProject: on greenfield (no prior AGENTS.md) init
// renders the sentinel from scratch; {project} must come from the config's
// project: field, not the directory name.
func TestInitCreatePathUsesConfigProject(t *testing.T) {
	dir := t.TempDir()
	cfg := "project: ConfiguredName\nmnemonic:\n  enabled: true\n"
	if err := os.MkdirAll(filepath.Join(dir, ".skillgrid"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := projectInit(context.Background(), service.New(t.TempDir()), dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(res.BootFile)
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	if !strings.Contains(s, "Project: **ConfiguredName**.") {
		t.Fatalf("create path did not use config project:\n%s", s)
	}
	if strings.Contains(s, "Project: **"+filepath.Base(dir)+"**.") {
		t.Fatalf("create path fell back to dir name:\n%s", s)
	}
}

// TestInitCreatePathHonorsMnemonicDisabled: when mnemonic is disabled in the
// config, the rendered memory line must say inactive — init must not hardcode
// the enabled line.
func TestInitCreatePathHonorsMnemonicDisabled(t *testing.T) {
	dir := t.TempDir()
	cfg := "project: ConfiguredName\nmnemonic:\n  enabled: false\n"
	if err := os.MkdirAll(filepath.Join(dir, ".skillgrid"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := projectInit(context.Background(), service.New(t.TempDir()), dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(res.BootFile)
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	const enabled = "Persistent memory is active (Mnemonic)."
	if strings.Contains(s, enabled) {
		t.Fatalf("disabled mnemonic rendered the enabled line:\n%s", s)
	}
	const inactive = "Mnemonic not detected — persistent memory is inactive."
	if !strings.Contains(s, inactive) {
		t.Fatalf("disabled mnemonic missing the inactive line:\n%s", s)
	}
}

// TestInitSentinelTemplateComesFromBlockMD proves the block's structure is read
// from block.md (the single source of truth), not from a template baked into the
// CLI. The canonical block.md's Artifacts table has 9 rows; the old CLI template
// had 4. If init rendered from its own embedded copy, this would see 4 rows.
func TestInitSentinelTemplateComesFromBlockMD(t *testing.T) {
	// Point the resolver at the repo's real block.md (the test binary runs from
	// skillgrid-cli/cmd/skillgrid, so the repo root is ../../../). This proves
	// init renders from block.md, not a baked-in CLI template. The current
	// block.md Artifacts table has 4 rows; an old/different CLI template would
	// give a different count (and the rows would not match block.md's text).
	prev := blockMDPath
	t.Cleanup(func() { blockMDPath = prev })
	blockMDPath = func(dir string) string {
		return "../../../.agents/skills/_shared/agent-config/block.md"
	}
	tmpl, _, err := loadSentinelTemplate(t.TempDir())
	if err != nil {
		t.Fatalf("loadSentinelTemplate: %v", err)
	}
	rows := countArtifactsRows(tmpl)
	if rows != 4 {
		t.Fatalf("template Artifacts table has %d rows, want 4 (block.md is the source of truth)\n%s", rows, tmpl)
	}
	// The rendered rows must carry block.md's exact row text, proving the
	// template came from block.md (the old CLI template's rows differ).
	const wantRow = "| Project knowledge (PRD, architecture, terms, ADRs, constraints, research) | `.skillgrid/artifacts/` |"
	if !strings.Contains(tmpl, wantRow) {
		t.Fatalf("template missing block.md's Artifacts row:\n%s", tmpl)
	}
}

// countArtifactsRows counts the data rows of the Artifacts table inside a
// rendered template: lines within the "### Artifacts" section that start with
// "|" and hold a backticked path cell. The header and |---| separator are
// excluded. Scoped to the section so block.md's own "Placeholders" tracker
// table (not part of the template) cannot inflate the count.
func countArtifactsRows(tmpl string) int {
	n := 0
	inSection := false
	for _, line := range strings.Split(tmpl, "\n") {
		t := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(t, "### Artifacts"):
			inSection = true
		case inSection && strings.HasPrefix(t, "##") && t != "### Artifacts":
			return n
		case inSection && strings.HasPrefix(t, "|"):
			if strings.Count(t, "`") >= 2 && !strings.Contains(t, "----") && !strings.Contains(t, "| Artifact |") {
				n++
			}
		}
	}
	return n
}

// TestInitPreambleIsMinimalCommandsFirst asserts the preamble follows the
// AGENTS.md best-practice research: a short project statement + exact commands,
// not the old 6-section generic boilerplate. The generic sections
// (Engineering Standards / Dependency Policies / Security & Escalation) raised
// inference cost without changing agent behavior, so init no longer writes them.
func TestInitPreambleIsMinimalCommandsFirst(t *testing.T) {
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
	// Commands-first: the skeleton leads with the Commands section.
	if !strings.Contains(s, "## Commands") || !strings.Contains(s, "- Test: <detect>") {
		t.Fatalf("preamble missing commands-first skeleton:\n%s", s)
	}
	// No literal-dot project name (the missing-config fallback).
	if strings.Contains(s, "Project: **.**") || strings.Contains(s, "# Project\n..") {
		t.Fatalf("preamble rendered a literal dot project name:\n%s", s)
	}
	// Generic boilerplate dropped.
	for _, gone := range []string{"# Engineering Standards", "# Dependency Policies", "# Security & Escalation Boundaries", "# Definition of Done", "# Architecture Constraints", "# Environment & Tooling"} {
		if strings.Contains(s, gone) {
			t.Fatalf("preamble still writes generic section %q:\n%s", gone, s)
		}
	}
	// The whole file should stay lean (< 150 lines per the research).
	if lines := len(strings.Split(s, "\n")); lines > 150 {
		t.Fatalf("boot file is %d lines, want < 150:\n%s", lines, s)
	}
}

func TestInitForceRewritesPreamble(t *testing.T) {
	dir := t.TempDir()
	res, err := projectInit(context.Background(), service.New(t.TempDir()), dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(res.BootFile)
	custom := strings.Replace(string(body), "# Project", "# Project\nUSER KEEP", 1)
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

// TestInitIndexDoesNotPrintUnresolvedMembers: init's result block is the
// operator status. An "unresolved member calls" line on stderr reads as a
// failure even when Errors is empty.
func TestInitIndexDoesNotPrintUnresolvedMembers(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	data := t.TempDir()
	svc := service.New(data)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\nfunc F() {}\n"), 0o644); err != nil {
		os.Stderr = old
		t.Fatal(err)
	}
	_, initErr := projectInit(context.Background(), svc, dir, false, nil)
	w.Close()
	os.Stderr = old
	if initErr != nil {
		t.Fatal(initErr)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "unresolved") {
		t.Fatalf("stderr looks like a failure: %q", buf.String())
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

func TestDetectProjectLanguagesGo(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x"), 0o644)
	langs := detectProjectLanguages(dir)
	found := map[string]bool{}
	for _, l := range langs {
		found[l] = true
	}
	if !found["go"] {
		t.Fatalf("langs = %v, want go", langs)
	}
	if !found["markdown"] {
		t.Fatalf("langs = %v, want markdown", langs)
	}
}

func TestDetectProjectLanguagesPython(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]"), 0o644)
	langs := detectProjectLanguages(dir)
	found := map[string]bool{}
	for _, l := range langs {
		found[l] = true
	}
	if !found["python"] {
		t.Fatalf("langs = %v, want python", langs)
	}
}

func TestDetectProjectLanguagesPolyglot(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x"), 0o644)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0o644)
	langs := detectProjectLanguages(dir)
	found := map[string]bool{}
	for _, l := range langs {
		found[l] = true
	}
	for _, want := range []string{"go", "typescript", "javascript"} {
		if !found[want] {
			t.Fatalf("langs = %v, want %s", langs, want)
		}
	}
}

func TestDetectProjectLanguagesEmpty(t *testing.T) {
	dir := t.TempDir()
	langs := detectProjectLanguages(dir)
	// Should still have markdown
	if len(langs) == 0 {
		t.Fatal("expected at least markdown")
	}
}

func TestWriteIndexingConfigGeneratesFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x"), 0o644)
	path := writeIndexingConfig(dir)
	if path == "" {
		t.Fatal("expected a written path")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	if !strings.Contains(s, "include:") {
		t.Fatalf("missing include: %s", s)
	}
	if !strings.Contains(s, "**/*.go") {
		t.Fatalf("missing go glob: %s", s)
	}
	if !strings.Contains(s, "**/*.md") {
		t.Fatalf("missing md glob: %s", s)
	}
	if !strings.Contains(s, "exclude:") {
		t.Fatalf("missing exclude: %s", s)
	}
}

func TestWriteIndexingConfigPython(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]"), 0o644)
	path := writeIndexingConfig(dir)
	if path == "" {
		t.Fatal("expected a written path")
	}
	body, _ := os.ReadFile(path)
	s := string(body)
	if !strings.Contains(s, "**/*.py") {
		t.Fatalf("missing python glob: %s", s)
	}
}

func TestWriteIndexingConfigSkipsExisting(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x"), 0o644)
	// Pre-create the config
	cfgDir := filepath.Join(dir, ".skillgrid", "config.d")
	os.MkdirAll(cfgDir, 0o755)
	os.WriteFile(filepath.Join(cfgDir, "indexing.yaml"), []byte("mnemonic:\n  include:\n    - \"**/*.custom\"\n"), 0o644)
	path := writeIndexingConfig(dir)
	if path != "" {
		t.Fatalf("expected skip, got %s", path)
	}
	// File should be unchanged
	body, _ := os.ReadFile(filepath.Join(cfgDir, "indexing.yaml"))
	if !strings.Contains(string(body), "**/*.custom") {
		t.Fatal("existing config was clobbered")
	}
}

func TestWriteIndexingConfigPolyglot(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x"), 0o644)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0o644)
	path := writeIndexingConfig(dir)
	if path == "" {
		t.Fatal("expected a written path")
	}
	body, _ := os.ReadFile(path)
	s := string(body)
	// package.json → typescript (.ts) + tsx (.tsx) + javascript (.js)
	for _, glob := range []string{"**/*.go", "**/*.ts", "**/*.tsx", "**/*.js", "**/*.md"} {
		if !strings.Contains(s, glob) {
			t.Fatalf("missing %s: %s", glob, s)
		}
	}
}

func TestInitGeneratesIndexingConfig(t *testing.T) {
	data := t.TempDir()
	svc := service.New(data)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x"), 0o644)
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\nfunc F() {}\n"), 0o644)
	res, err := projectInit(context.Background(), svc, dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.IndexingConfig == "" {
		t.Fatal("expected IndexingConfig to be set")
	}
	if _, err := os.Stat(res.IndexingConfig); err != nil {
		t.Fatal(err)
	}
}

func TestInitIndexingConfigNotClobbered(t *testing.T) {
	data := t.TempDir()
	svc := service.New(data)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x"), 0o644)
	// First init generates the config
	res1, err := projectInit(context.Background(), svc, dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res1.IndexingConfig == "" {
		t.Fatal("first init should generate config")
	}
	// Modify it
	body, _ := os.ReadFile(res1.IndexingConfig)
	modified := strings.Replace(string(body), "**/*.go", "**/*.custom", 1)
	os.WriteFile(res1.IndexingConfig, []byte(modified), 0o644)
	// Second init should not clobber
	res2, err := projectInit(context.Background(), svc, dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res2.IndexingConfig != "" {
		t.Fatalf("second init should skip, got %s", res2.IndexingConfig)
	}
	body2, _ := os.ReadFile(res1.IndexingConfig)
	if !strings.Contains(string(body2), "**/*.custom") {
		t.Fatal("config was clobbered on second init")
	}
}
