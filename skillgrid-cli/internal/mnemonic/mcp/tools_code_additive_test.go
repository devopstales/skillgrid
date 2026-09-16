package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// codeAdditiveFixture indexes a small go project containing a secret-like value
// in a source file, pins the project, and returns the root for handler calls.
func codeAdditiveFixture(t *testing.T) (dataDir, root string) {
	t.Helper()
	dataDir = t.TempDir()
	raw := t.TempDir()
	abs, err := filepath.Abs(raw)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	root = abs
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	// A source file with a secret-like AWS access key and a function to search.
	write("auth.go", "package auth\n\nconst awsKey = \"AKIAIOSFODNN7EXAMPLE\"\n\nfunc LoadConfig() string {\n\treturn awsKey\n}\n")
	// A second file to give search a hit target.
	write("client.go", "package http\n\nfunc NewClient() *Client {\n\treturn &Client{}\n}\n\nfunc (c *Client) Get(url string) error {\n\treturn nil\n}\n")
	t.Setenv("MNEMONIC_PROJECT", "codeadditive-probe")
	svc := service.New(dataDir)
	SetService(svc)
	t.Cleanup(func() { SetService(nil) })
	codeindex.ResetFileFirstSymbol()
	if _, err := svc.RunCodeIndex(context.Background(), root); err != nil {
		t.Fatalf("index: %v", err)
	}
	oldDir, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(oldDir) })
	return dataDir, root
}

// TestCodeSearchAdditiveGains covers @step-01 (Scenarios: Search response
// carries confidence action rerank reasons and fallbacks; Snippets are
// skeletonized and secrets are redacted in output): the code_search response
// GAINS confidence + action + rerank reasons + redaction state additively,
// while its 005 required `query` param and hit fields stay unchanged.
func TestCodeSearchAdditiveGains(t *testing.T) {
	codeAdditiveFixture(t)
	res, err := handleCodeSearch(context.Background(), newCallTool("code_search", map[string]any{"query": "NewClient"}))
	if err != nil {
		t.Fatalf("handleCodeSearch: %v", err)
	}
	text := callResultText(t, res)
	var out struct {
		Hits []struct {
			Path       string `json:"path"`
			StartLine  int    `json:"start_line"`
			EndLine    int    `json:"end_line"`
			Snippet    string `json:"snippet"`
			Score      float64 `json:"score"`
			Confidence string `json:"confidence"`
			Action     string `json:"action"`
			Reasons    string `json:"rerank_reasons"`
			Redacted   bool   `json:"redacted"`
		} `json:"hits"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal code_search: %v (text %s)", err, text)
	}
	if len(out.Hits) == 0 {
		t.Fatalf("expected code_search hits, got none: %s", text)
	}
	// 005 required fields must remain (additive, not a rewrite).
	for i, h := range out.Hits {
		if h.Path == "" {
			t.Errorf("hit %d lost its 005 'path' field", i)
		}
		// NEW additive fields must be present.
		if h.Confidence != "high" && h.Confidence != "medium" && h.Confidence != "low" {
			t.Errorf("hit %d confidence is not categorical high|medium|low, got %q", i, h.Confidence)
		}
		if h.Action == "" {
			t.Errorf("hit %d is missing the additive 'action' field", i)
		}
	}
	// A code_search over "NewClient" is an exact-symbol hit → high confidence.
	if out.Hits[0].Confidence != "high" {
		t.Errorf("exact-symbol search should be high confidence, got %q", out.Hits[0].Confidence)
	}
	if out.Hits[0].Reasons == "" {
		t.Errorf("exact-symbol search should carry rerank reasons, got empty")
	}
}

// TestCodeReadOutputRedaction covers @step-01 (Scenario: Snippets are
// skeletonized and secrets are redacted in output): the code_read response text
// for a file containing a secret-like value is redacted (the secret never
// appears raw) and carries an additive redaction-state field.
func TestCodeReadOutputRedaction(t *testing.T) {
	codeAdditiveFixture(t)
	res, err := handleCodeRead(context.Background(), newCallTool("code_read", map[string]any{"path": "auth.go"}))
	if err != nil {
		t.Fatalf("handleCodeRead: %v", err)
	}
	text := callResultText(t, res)
	// The secret must NOT appear raw in the response text.
	if strings.Contains(text, "AKIAIOSFODNN7EXAMPLE") {
		t.Errorf("code_read output leaked the raw secret: %s", text)
	}
	// An additive redaction-state field must be present.
	var out struct {
		Text     string `json:"text"`
		Redacted bool   `json:"redacted"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal code_read: %v (text %s)", err, text)
	}
	if !out.Redacted {
		t.Errorf("code_read output with a redacted secret must set redacted=true, got: %s", text)
	}
}

// TestCodeToolRequiredParamsUnchanged covers @step-01 (005 tool NAMES and
// REQUIRED params must NOT change — additive response gains only): the
// code_search/code_read tools keep their exact required-param contract even
// after the additive response fields are wired in.
func TestCodeToolRequiredParamsUnchanged(t *testing.T) {
	assertCodeToolStable(t, codeSearchTool(), "code_search", []string{"query"})
	assertCodeToolStable(t, codeReadTool(), "code_read", []string{"path"})
}
