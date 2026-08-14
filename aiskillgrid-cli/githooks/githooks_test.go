package githooks

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupPack(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "packs", "git-hooks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	hook := `#!/bin/sh
# aiskillgrid-managed: commit-msg
MSG_FILE="$1"
# skillgrid:managed:start
sed -E -e '/^Co-authored-by: Cursor /d' -e '/^Co-authored-by:.*cursoragent@/d' "$MSG_FILE" >"$MSG_FILE.tmp" && mv "$MSG_FILE.tmp" "$MSG_FILE"
# skillgrid:managed:end
PREV="$(dirname "$0")/commit-msg.aiskillgrid-prev"
if [ -x "$PREV" ]; then
	exec "$PREV" "$@"
fi
exit 0
`
	if err := os.WriteFile(filepath.Join(dir, "commit-msg"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestInstallCommitMsgHookSkipsNonGit(t *testing.T) {
	pack := setupPack(t)
	dir := t.TempDir()
	written, err := InstallCommitMsgHook(dir, pack)
	if err != nil {
		t.Fatal(err)
	}
	if written != "" {
		t.Fatalf("expected empty written, got %q", written)
	}
}

func TestInstallCommitMsgHookWritesAndChains(t *testing.T) {
	pack := setupPack(t)
	project := t.TempDir()
	gitDir := filepath.Join(project, ".git", "hooks")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	prevBody := "#!/bin/sh\necho prev-ran >>\"$1\"\nexit 0\n"
	if err := os.WriteFile(filepath.Join(gitDir, "commit-msg"), []byte(prevBody), 0o755); err != nil {
		t.Fatal(err)
	}

	written, err := InstallCommitMsgHook(project, pack)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(project, ".git", "hooks", "commit-msg")
	if written != want {
		t.Fatalf("written=%q want %q", written, want)
	}
	data, err := os.ReadFile(written)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "aiskillgrid-managed: commit-msg") {
		t.Fatal("missing managed marker")
	}
	prevPath := want + ".aiskillgrid-prev"
	if _, err := os.Stat(prevPath); err != nil {
		t.Fatal(err)
	}

	// Reinstall must not clobber the preserved prev hook.
	if err := os.WriteFile(prevPath, []byte("#!/bin/sh\nexit 0\nKEEP\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallCommitMsgHook(project, pack); err != nil {
		t.Fatal(err)
	}
	prevData, _ := os.ReadFile(prevPath)
	if !strings.Contains(string(prevData), "KEEP") {
		t.Fatal("reinstall overwrote aiskillgrid-prev")
	}
}

func TestInstalledHookStripsCursorCoauthor(t *testing.T) {
	pack := setupPack(t)
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".git", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	written, err := InstallCommitMsgHook(project, pack)
	if err != nil || written == "" {
		t.Fatalf("install: written=%q err=%v", written, err)
	}

	msg := filepath.Join(project, "COMMIT_EDITMSG")
	body := "feat: something\n\nCo-authored-by: Cursor <cursoragent@cursor.com>\n"
	if err := os.WriteFile(msg, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", written, msg)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("hook failed: %v %s", err, out)
	}
	got, _ := os.ReadFile(msg)
	if strings.Contains(string(got), "Co-authored-by: Cursor") {
		t.Fatalf("co-author not stripped: %q", got)
	}
	if !strings.Contains(string(got), "feat: something") {
		t.Fatalf("subject lost: %q", got)
	}
}
