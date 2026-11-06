# Review package: 76ec335..1086a2b

## Commits
1086a2b feat(http): prototype serve route (.skillgrid/prototype) + sandboxed iframe in decision card (TICKET-06)
4c884d2 test(http): RED acceptance tests for prototype serve + decision card iframe (TICKET-05)

## Files changed
 skillgrid-cli/internal/mnemonic/http/prototype.go  | 64 +++++++++++++++
 .../internal/mnemonic/http/prototype_test.go       | 92 ++++++++++++++++++++++
 skillgrid-cli/internal/mnemonic/http/server.go     |  6 ++
 .../src/features/decisions/DecisionCard.test.tsx   | 13 +++
 .../src/features/decisions/DecisionCard.tsx        | 14 +++-
 5 files changed, 185 insertions(+), 4 deletions(-)

## Diff
diff --git a/skillgrid-cli/internal/mnemonic/http/prototype.go b/skillgrid-cli/internal/mnemonic/http/prototype.go
new file mode 100644
index 0000000..1429705
--- /dev/null
+++ b/skillgrid-cli/internal/mnemonic/http/prototype.go
@@ -0,0 +1,64 @@
+package http
+
+import (
+	"net/http"
+	"os"
+	"path/filepath"
+	"strings"
+)
+
+// prototypeRoot returns the .skillgrid/prototype/ directory (under sddRoot,
+// the repo root). This is the on-disk store for the companion's throwaway
+// interactive prototypes (agent-authored HTML), separate from .stitch/.
+func prototypeRoot() string { return filepath.Join(sddRoot(), ".skillgrid", "prototype") }
+
+// prototypeFile resolves a prototype id to an absolute path sandboxed to
+// .skillgrid/prototype/ — the same path-traversal guard shape as stitchFile
+// (prototypes.go). ok=false for empty/absolute ids, '..' segments, or any
+// path that escapes the root after filepath.Clean.
+func prototypeFile(id string) (string, bool) {
+	id = strings.TrimSpace(id)
+	if id == "" {
+		return "", false
+	}
+	// Reject absolute ids (leading / or a Windows drive) up front.
+	if filepath.IsAbs(id) || id[0] == '/' || (len(id) >= 2 && id[1] == ':') {
+		return "", false
+	}
+	root := prototypeRoot()
+	full := filepath.Join(root, filepath.FromSlash(id))
+	// Clean both and ensure the resolved path stays under root.
+	cleanFull := filepath.Clean(full)
+	cleanRoot := filepath.Clean(root)
+	rel, err := filepath.Rel(cleanRoot, cleanFull)
+	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
+		return "", false
+	}
+	return cleanFull, true
+}
+
+// handlePrototypeDecision serves GET /prototype/{id...} — the HTML of one
+// companion prototype, sandboxed to .skillgrid/prototype/. Traversal/absolute
+// ids → 400; unknown → 404. Rendered client-side in a sandboxed iframe
+// (sandbox="allow-scripts", no allow-same-origin).
+func (s *Server) handlePrototypeDecision(w http.ResponseWriter, r *http.Request) {
+	id := r.PathValue("id")
+	full, ok := prototypeFile(id)
+	if !ok {
+		writeError(w, http.StatusBadRequest, "invalid prototype id: "+id)
+		return
+	}
+	data, err := os.ReadFile(full)
+	if err != nil {
+		if os.IsNotExist(err) {
+			writeError(w, http.StatusNotFound, "unknown prototype: "+id)
+			return
+		}
+		writeError(w, http.StatusInternalServerError, err.Error())
+		return
+	}
+	w.Header().Set("Content-Type", "text/html; charset=utf-8")
+	w.Header().Set("X-Content-Type-Options", "nosniff")
+	w.WriteHeader(http.StatusOK)
+	_, _ = w.Write(data)
+}
diff --git a/skillgrid-cli/internal/mnemonic/http/prototype_test.go b/skillgrid-cli/internal/mnemonic/http/prototype_test.go
new file mode 100644
index 0000000..63fa676
--- /dev/null
+++ b/skillgrid-cli/internal/mnemonic/http/prototype_test.go
@@ -0,0 +1,92 @@
+package http
+
+import (
+	"net/http"
+	"os"
+	"path/filepath"
+	"strings"
+	"testing"
+
+	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
+)
+
+// newProtoServer builds a server whose sddRoot points at a temp repo with a
+// .skillgrid/prototype/ tree (the companion's throwaway HTML prototypes,
+// distinct from the .stitch/ design prototypes) plus a sibling file outside
+// the root that must NOT be reachable (path-traversal threat).
+func newProtoServer(t *testing.T) http.Handler {
+	t.Helper()
+	repo := t.TempDir()
+	proto := filepath.Join(repo, ".skillgrid", "prototype", "demo")
+	if err := os.MkdirAll(proto, 0o755); err != nil {
+		t.Fatalf("mkdir prototype: %v", err)
+	}
+	if err := os.WriteFile(filepath.Join(proto, "a.html"),
+		[]byte("<html><body><h1>variant A</h1></body></html>"), 0o644); err != nil {
+		t.Fatalf("write a.html: %v", err)
+	}
+	// A secret file OUTSIDE .skillgrid/prototype/ that must not be servable.
+	if err := os.WriteFile(filepath.Join(repo, "secret.html"),
+		[]byte("<html><body>SECRET</body></html>"), 0o644); err != nil {
+		t.Fatalf("write secret: %v", err)
+	}
+
+	t.Setenv("SKILLGRID_DOCS_CWD", repo)
+	dataDir := t.TempDir()
+	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
+	svc := service.New(dataDir)
+	return NewServer(svc).Handler()
+}
+
+func TestPrototypeServeAndTraversal(t *testing.T) {
+	h := newProtoServer(t)
+
+	// --- happy path: serves the seeded file as HTML ---
+	rr := doGet(t, h, "/prototype/demo/a.html")
+	if rr.Code != http.StatusOK {
+		t.Fatalf("/prototype/demo/a.html: status = %d, want 200; body=%s", rr.Code, rr.Body.String())
+	}
+	if !strings.Contains(rr.Body.String(), "variant A") {
+		t.Errorf("content = %q, want 'variant A'", rr.Body.String())
+	}
+	ct := rr.Header().Get("Content-Type")
+	if !strings.Contains(ct, "text/html") {
+		t.Errorf("Content-Type = %q, want text/html", ct)
+	}
+	if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
+		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
+	}
+
+	// --- traversal / absolute ids → 400 ---
+	// Bare `..` ids are rewritten by the mux's clean-path redirect before
+	// reaching the handler (307 to a path outside the /prototype subtree);
+	// %-encoded forms reach the handler and must be 400. A successful
+	// response must never be the secret (no escape past the root either way).
+	wants := map[string]int{
+		"307": http.StatusTemporaryRedirect,
+		"400": http.StatusBadRequest,
+		"404": http.StatusNotFound,
+	}
+	for _, tc := range []struct{ id, want string }{
+		{"../a.html", "307"},
+		{"..%2F..%2Fa.html", "400"},
+		{"%2e%2e/secret.html", "400"},
+		{"%2Fetc%2Fpasswd", "400"},
+		{"demo/../../secret.html", "307"},
+		{"demo/../secret.html", "307"},
+	} {
+		rr = doGet(t, h, "/prototype/"+tc.id)
+		if rr.Code != wants[tc.want] {
+			t.Errorf("id %q: status = %d, want %s; body=%s", tc.id, rr.Code, tc.want, rr.Body.String())
+		}
+		if rr.Code == http.StatusOK && strings.Contains(rr.Body.String(), "SECRET") {
+			t.Errorf("id %q: served SECRET (escaped .skillgrid/prototype/)", tc.id)
+		}
+	}
+
+	// --- unknown file → 404 ---
+	rr = doGet(t, h, "/prototype/demo/nope.html")
+	if rr.Code != http.StatusNotFound {
+		t.Errorf("unknown: status = %d, want 404; body=%s", rr.Code, rr.Body.String())
+	}
+}
diff --git a/skillgrid-cli/internal/mnemonic/http/server.go b/skillgrid-cli/internal/mnemonic/http/server.go
index 99793a2..9e06c83 100644
--- a/skillgrid-cli/internal/mnemonic/http/server.go
+++ b/skillgrid-cli/internal/mnemonic/http/server.go
@@ -196,20 +196,26 @@ func (s *Server) registerGraphRoutes() {
 	s.mux.HandleFunc("GET /git/commits", s.handleGitCommits)
 	s.mux.HandleFunc("GET /git/commits/{sha}", s.handleGitCommit)
 	s.mux.HandleFunc("GET /git/diff/{sha}", s.handleGitDiff)
 	s.mux.HandleFunc("GET /git/file-history", s.handleGitFileHistory)
 	s.mux.HandleFunc("GET /git/blame", s.handleGitBlame)
 
 	// Phase 7 prototypes routes (sandboxed .stitch/ gallery readers).
 	s.mux.HandleFunc("GET /prototypes", s.shellOrJSON(s.handlePrototypes))
 	s.mux.HandleFunc("GET /prototypes/{id...}", s.handlePrototype)
 
+	// Visual companion prototype serve route (sandboxed .skillgrid/prototype/
+	// readers). A distinct subtree from the plural /prototypes above;
+	// registered here (before registerUIRoutes) so the SPA fallback can't
+	// swallow it.
+	s.mux.HandleFunc("GET /prototype/{id...}", s.handlePrototypeDecision)
+
 	// The SPA shell uses base './', so on a /mnemonic/... path its module
 	// script is requested at /mnemonic/assets/... — serve the embedded build
 	// assets there (the root /assets/ route covers the top-level roots).
 	s.mux.HandleFunc("GET /mnemonic/assets/{rest...}", s.handleUIAsset)
 	// The bare client paths stay SPA shells for the client router. The Phase 5
 	// API routes above own the exact /mnemonic/{memories,sessions,search}
 	// paths, so only the paths without an API handler keep an explicit shell.
 	for _, p := range []string{"/mnemonic/graph", "/mnemonic/files"} {
 		p := p
 		s.mux.HandleFunc("GET "+p, s.handleShellPage)
diff --git a/skillgrid-ui/src/features/decisions/DecisionCard.test.tsx b/skillgrid-ui/src/features/decisions/DecisionCard.test.tsx
index 3e63c8c..c38d0a6 100644
--- a/skillgrid-ui/src/features/decisions/DecisionCard.test.tsx
+++ b/skillgrid-ui/src/features/decisions/DecisionCard.test.tsx
@@ -39,20 +39,33 @@ describe('DecisionCard', () => {
     expect(screen.getByText('Two column')).toBeTruthy()
     const rec = screen.getByTestId('decision-option-a')
     expect(rec.className).toContain('recommended')
   })
 
   it('renders no iframe for a decision without visual', () => {
     const { container } = render(<DecisionCard decision={pending} onAnswer={vi.fn()} />)
     expect(container.querySelector('iframe')).toBeNull()
   })
 
+  it('renders the sandboxed iframe for a decision with visual', () => {
+    const withVisual: Decision = {
+      ...pending,
+      content: { ...pending.content, visual: 'demo/a.html' },
+    }
+    const { container } = render(<DecisionCard decision={withVisual} onAnswer={vi.fn()} />)
+    const iframe = container.querySelector('iframe')
+    expect(iframe).not.toBeNull()
+    expect(iframe?.getAttribute('src')).toBe('/prototype/demo/a.html')
+    expect(iframe?.getAttribute('sandbox')).toBe('allow-scripts')
+    expect(iframe?.getAttribute('sandbox')).not.toContain('allow-same-origin')
+  })
+
   it('posts the chosen option and note on submit', () => {
     const onAnswer = vi.fn()
     render(<DecisionCard decision={pending} onAnswer={onAnswer} />)
     fireEvent.change(screen.getByPlaceholderText(/optional note/i), {
       target: { value: 'prefer a' },
     })
     fireEvent.click(screen.getByTestId('decision-submit'))
     expect(onAnswer).toHaveBeenCalledWith(1, { optionId: 'a', note: 'prefer a' })
   })
 
diff --git a/skillgrid-ui/src/features/decisions/DecisionCard.tsx b/skillgrid-ui/src/features/decisions/DecisionCard.tsx
index 9e5dc22..5e33944 100644
--- a/skillgrid-ui/src/features/decisions/DecisionCard.tsx
+++ b/skillgrid-ui/src/features/decisions/DecisionCard.tsx
@@ -1,20 +1,21 @@
 import { useState } from 'react'
 
 // DecisionCard renders one decision from the inbox: the question, its
 // lettered options (the recommended one highlighted), an optional note, and a
 // Submit that hands {optionId, note} to onAnswer. On an answered decision it
 // shows the stored choice + note; the options stay re-answerable (the backend
 // appends a version) and re-selecting the picked option re-submits it.
 // A decision whose content carries a `visual` gets a sandboxed prototype
-// iframe in P3 — until then it renders a "visual available" placeholder,
-// never an iframe.
+// iframe (src=/prototype/{visual}, sandbox="allow-scripts", no
+// allow-same-origin) rendering the companion's throwaway HTML from
+// .skillgrid/prototype/ — the same sandbox contract as SandboxPreview.
 
 export interface DecisionOption {
   id: string
   label: string
   rationale?: string
 }
 
 export interface DecisionContent {
   question: string
   options: DecisionOption[]
@@ -130,22 +131,27 @@ export function DecisionCard({
       {answered && content.answerNote && (
         <div className="mt-2 text-xs text-zinc-500">
           Note: <span className="text-zinc-300">{content.answerNote}</span>
           {content.updatedBy && (
             <span className="ml-2 text-zinc-600">· {content.updatedBy}</span>
           )}
         </div>
       )}
 
       {hasVisual && (
-        <div className="mt-3 rounded-md border border-dashed border-edge px-3 py-2 text-xs text-zinc-500">
-          Visual prototype available (renders in P3).
+        <div className="mt-3 overflow-hidden rounded-md border border-edge bg-white">
+          <iframe
+            title={`Prototype: ${content.visual}`}
+            src={`/prototype/${content.visual}`}
+            sandbox="allow-scripts"
+            className="h-[420px] w-full border-0"
+          />
         </div>
       )}
 
       <div className="mt-3 flex items-center gap-2">
         <input
           value={note}
           onChange={(e) => setNote(e.target.value)}
           placeholder="Optional note for the agent…"
           aria-label="Optional note"
           className="min-w-0 flex-1 rounded-md border border-edge bg-bg px-2 py-1 text-sm text-zinc-200 placeholder:text-zinc-600 focus:border-accent/60 focus:outline-none"
