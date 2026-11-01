package llm_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/intelligence/embedder"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/intelligence/llm"
)

func jsonString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func TestNewClientRejectsEmptyBaseURL(t *testing.T) {
	if _, err := llm.NewClient("", "m", "k", time.Second); err == nil {
		t.Fatal("expected error for empty base URL")
	}
}

func TestEmbedPostsToEmbeddingsEndpoint(t *testing.T) {
	var gotPath, gotModel, gotAPIKey, gotInput string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		gotPath = r.URL.Path
		gotModel = req.Model
		gotAPIKey = r.Header.Get("Authorization")
		if len(req.Input) > 0 {
			gotInput = req.Input[0]
		}
		// Two input texts → two embeddings.
		fmt.Fprint(w, `{"data":[{"embedding":[0.1,0.2,0.3]},{"embedding":[0.4,0.5,0.6]}]}`)
	}))
	defer srv.Close()
	c, err := llm.NewClient(srv.URL+"/v1", "emb-model", "key-123", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	vecs, err := c.Embed(context.Background(), []string{"alpha", "beta"})
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if gotPath != "/v1/embeddings" {
		t.Fatalf("path = %q, want /v1/embeddings", gotPath)
	}
	if gotModel != "emb-model" {
		t.Fatalf("model = %q, want emb-model", gotModel)
	}
	if gotAPIKey != "Bearer key-123" {
		t.Fatalf("auth = %q, want Bearer key-123", gotAPIKey)
	}
	if gotInput != "alpha" {
		t.Fatalf("input[0] = %q, want alpha", gotInput)
	}
	if len(vecs) != 2 || len(vecs[0]) != 3 {
		t.Fatalf("vecs = %v, want 2 vectors of dim 3", vecs)
	}
	if vecs[0][1] != 0.2 || vecs[1][2] != 0.6 {
		t.Fatalf("vector values wrong: %v", vecs)
	}
}

func TestEmbedNoAuthorizationWhenKeyEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization = %q, want empty", got)
		}
		fmt.Fprint(w, `{"data":[{"embedding":[1.0]}]}`)
	}))
	defer srv.Close()
	c, err := llm.NewClient(srv.URL, "m", "", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Embed(context.Background(), []string{"x"}); err != nil {
		t.Fatal(err)
	}
}

func TestClientImplementsBothSeams(t *testing.T) {
	var _ llm.LLM = (*llm.Client)(nil)
	var _ embedder.Embedder = (*llm.Client)(nil)
}

// [llm-seam-4-way-classify] TRIANGULATE — a different verdict ("noop", no
// candidate) and a second parse path (add with candidate null) must also map
// correctly. Different inputs from TestClassifyReturnsFourWayVerdict.
func TestClassifyNoopAndAddVerdicts(t *testing.T) {
	for _, tc := range []struct {
		name   string
		resp   string
		want   llm.DedupVerdict
		candID int64
	}{
		{"noop", `{"choices":[{"message":{"content":"{\"verdict\":\"noop\",\"candidate\":null}"}}]}`, llm.VerdictNoop, 0},
		{"add", `{"choices":[{"message":{"content":"{\"verdict\":\"add\",\"candidate\":null}"}}]}`, llm.VerdictAdd, 0},
		{"delete-first", `{"choices":[{"message":{"content":"{\"verdict\":\"delete\",\"candidate\":0}"}}]}`, llm.VerdictDelete, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, tc.resp)
			}))
			defer srv.Close()
			c, err := llm.NewClient(srv.URL, "m", "", time.Second)
			if err != nil {
				t.Fatal(err)
			}
			d, err := c.Classify(context.Background(), "new", []string{"one", "two"})
			if err != nil {
				t.Fatal(err)
			}
			if d.Verdict != tc.want || d.CandidateID != tc.candID {
				t.Fatalf("got %+v, want verdict=%s cand=%d", d, tc.want, tc.candID)
			}
		})
	}
}

// [degraded-bounded-retry] TRIANGULATE (HTTP level) — a failing seam (500 on
// every call) must produce exactly one retry: the client's Classify errors
// once per HTTP round-trip, and degrade.Run calls it twice (initial + one
// retry) before the floor. This is the end-to-end form of the bounded retry
// over the real HTTP provider, not just a stubbed seam.
func TestClassifyHTTP500IsASeamError(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"boom"}`)
	}))
	defer srv.Close()
	c, err := llm.NewClient(srv.URL, "m", "", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := c.Classify(context.Background(), "n", []string{"c"}); err == nil {
			t.Fatalf("call %d: want error from 500", i)
		}
	}
	if hits != 2 {
		t.Fatalf("hits = %d, want 2", hits)
	}
}

// Malformed JSON from the provider is a seam error (degrade floor applies),
// not a panic and not a silent zero-value.
func TestClassifyMalformedResponseIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"choices":[{"message":{"content":"not-json"}}]}`)
	}))
	defer srv.Close()
	c, err := llm.NewClient(srv.URL, "m", "", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Classify(context.Background(), "n", []string{"c"}); err == nil {
		t.Fatal("want error for malformed verdict JSON")
	}
}

// Judge/Compare: the strict-JSON judgment is parsed into the typed Judgment,
// and the client's model is recorded (INV-1 marked_by_model). Compare is the
// same call with no pending relation.
func TestJudgeAndCompareParseJudgment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"choices":[{"message":{"content":"{\"relation\":\"contradicts\",\"confidence\":0.8,\"evidence\":\"opposite claim\"}"}}]}`)
	}))
	defer srv.Close()
	c, err := llm.NewClient(srv.URL, "judge-model", "k", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	a := llm.Observation{ID: 1, Title: "A", Content: "x"}
	b := llm.Observation{ID: 2, Title: "B", Content: "y"}
	j, err := c.Judge(context.Background(), a, b, llm.Relation{Relation: "contradicts", Reason: "r"})
	if err != nil {
		t.Fatal(err)
	}
	if j.Relation != "contradicts" || j.Confidence != 0.8 || j.Evidence != "opposite claim" {
		t.Fatalf("bad judgment %+v", j)
	}
	if j.Model != "judge-model" {
		t.Fatalf("Model = %q, want judge-model (INV-1)", j.Model)
	}
	// Compare = Judge with no pending relation; same parse path.
	jc, err := c.Compare(context.Background(), a, b)
	if err != nil {
		t.Fatal(err)
	}
	if jc.Relation != "contradicts" || jc.Model != "judge-model" {
		t.Fatalf("bad compare judgment %+v", jc)
	}
}

// Extract: the strict-JSON learnings parse into CandidateObservations; blank
// items are dropped; an empty parse is an error (the regex floor decides).
func TestExtractParsesLearningsAndDropsBlank(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"choices":[{"message":{"content":"{\"learnings\":[{\"text\":\"  use WAL \",\"type\":\"pattern\"},{\"text\":\"   \",\"type\":\"\"}]}"}}]}`)
	}))
	defer srv.Close()
	c, err := llm.NewClient(srv.URL, "m", "", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.Extract(context.Background(), "some session text")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Content != "use WAL" || got[0].Type != "pattern" {
		t.Fatalf("bad extract %+v", got)
	}
}

func TestExtractEmptyLearningsIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"choices":[{"message":{"content":"{\"learnings\":[]}"}}]}`)
	}))
	defer srv.Close()
	c, err := llm.NewClient(srv.URL, "m", "", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Extract(context.Background(), "text"); err == nil {
		t.Fatal("want error for empty learnings")
	}
}

// The OpenAI-compatible /embeddings endpoint does not advertise its dimension,
// so the client infers it from the first round-trip and caches it; a vector
// whose length disagrees with a vector earlier in the same batch is an error
// (mismatch → error, never silent truncation).
func TestEmbedInconsistentVectorLengthIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// One batch, two vectors of different length: the client must error.
		fmt.Fprint(w, `{"data":[{"embedding":[0.1,0.2]},{"embedding":[0.3,0.4,0.5]}]}`)
	}))
	defer srv.Close()
	c, err := llm.NewClient(srv.URL, "m", "", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Embed(context.Background(), []string{"a", "b"}); err == nil {
		t.Fatal("want error for inconsistent vector lengths in one batch")
	}
	// A consistent batch pins the dimension.
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[{"embedding":[0.1,0.2,0.3]}]}`)
	}))
	defer srv2.Close()
	c2, _ := llm.NewClient(srv2.URL, "m", "", time.Second)
	if _, err := c2.Embed(context.Background(), []string{"a"}); err != nil {
		t.Fatal(err)
	}
	if c2.Dimensions() != 3 {
		t.Fatalf("Dimensions = %d, want 3", c2.Dimensions())
	}
}
