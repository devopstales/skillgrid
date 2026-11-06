### Task 1: OpenAI-compatible Completer

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/llm/client.go`
- Test: `skillgrid-cli/internal/mnemonic/llm/client_test.go`

**Interfaces:**
- Consumes: none
- Produces:
  ```go
  type Completer interface {
      Complete(ctx context.Context, system, user string) (string, error)
  }
  type Config struct {
      BaseURL string
      Model   string
      APIKey  string
      Timeout time.Duration // 0 → 3s
  }
  func New(cfg Config) *Client // implements Completer
  ```
- Seam: Completer (HTTP Client + test stub)
- Adapters: ≥2
- Deletion test: without this package, every seam invents its own HTTP client

**SATISFIES:** `happy path openai-compatible complete succeeds`

- [ ] **Step 1: Write the failing test**

```go
package llm_test

func TestCompleteSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" && r.URL.Path != "/chat/completions" {
			t.Errorf("path=%s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": "hello"}}},
		})
	}))
	defer srv.Close()
	c := llm.New(llm.Config{BaseURL: srv.URL + "/v1", Model: "m", Timeout: time.Second})
	got, err := c.Complete(context.Background(), "sys", "user")
	if err != nil || got != "hello" {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestCompleteHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()
	c := llm.New(llm.Config{BaseURL: srv.URL, Model: "m", Timeout: time.Second})
	if _, err := c.Complete(context.Background(), "s", "u"); err == nil {
		t.Fatal("want error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./skillgrid-cli/internal/mnemonic/llm -count=1 -run 'TestComplete'
```

Expected: FAIL (package / symbol missing)

- [ ] **Step 3: Write minimal implementation**

```go
package llm

// Client POSTs OpenAI chat completions to {BaseURL}/chat/completions.
type Client struct { /* http.Client, baseURL, model, apiKey */ }

func New(cfg Config) *Client { /* trim trailing /, timeout default 3s */ }

func (c *Client) Complete(ctx context.Context, system, user string) (string, error) {
	// POST JSON: model, messages[{role:system},{role:user}]
	// Authorization: Bearer <apiKey> if non-empty
	// return choices[0].message.content or error on non-2xx / empty
}
```

Also add `TestCompleteTimeout` (handler sleeps past Timeout).

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./skillgrid-cli/internal/mnemonic/llm -count=1 -run 'TestComplete'
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/llm/
git commit -m "$(cat <<'EOF'
feat(mnemonic): openai-compatible chat Completer

EOF
)"
```

---

