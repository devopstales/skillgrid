package graph

import (
	"context"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// openTestStore opens a real SQLite store for testing.
func openTestStore(t *testing.T) (store.Store, func()) {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(context.Background(), dir, "p", store.SQLite, "")
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	return st, func() { st.Close() }
}

// seedResolve seeds the code_resolve contract: CapturePayment exists, called from enterprise.go; Foo is ambiguous.
func seedResolve(t *testing.T) store.Store {
	t.Helper()
	st, cleanup := openTestStore(t)
	t.Cleanup(cleanup)
	ctx := context.Background()

	// File: internal/pay/pay.go with CapturePayment
	filePay, _ := st.UpsertFile(ctx, store.FileRow{
		Path: "internal/pay/pay.go", Language: "go", ContentHash: "hash-pay",
	})
	st.UpsertSymbols(ctx, filePay, []store.Symbol{
		{Name: "CapturePayment", QualifiedName: "internal/pay.CapturePayment", Kind: "function", Language: "go", StartLine: 480, EndLine: 520, UID: "uid-capture"},
	})

	// File: internal/billing/enterprise.go calls CapturePayment
	fileEnt, _ := st.UpsertFile(ctx, store.FileRow{
		Path: "internal/billing/enterprise.go", Language: "go", ContentHash: "hash-ent",
	})
	st.UpsertSymbols(ctx, fileEnt, []store.Symbol{
		{Name: "ProcessBilling", Kind: "function", Language: "go", StartLine: 10, EndLine: 50, UID: "uid-billing"},
	})
	symCapture, _ := st.SymbolByName(ctx, "CapturePayment")
	symBilling, _ := st.SymbolByName(ctx, "ProcessBilling")
	toCapture := symCapture[0].ID
	st.ReplaceEdgesForFile(ctx, fileEnt, []store.Edge{
		{Kind: "calls", FromID: symBilling[0].ID, ToID: &toCapture, Confidence: "EXTRACTED", Line: 25},
	})

	// Two symbols named Foo (ambiguous)
	fileFoo1, _ := st.UpsertFile(ctx, store.FileRow{
		Path: "a/foo.go", Language: "go", ContentHash: "hash-foo1",
	})
	st.UpsertSymbols(ctx, fileFoo1, []store.Symbol{
		{Name: "Foo", QualifiedName: "a.Foo", Kind: "function", Language: "go", StartLine: 1, EndLine: 10, UID: "uid-foo1"},
	})
	fileFoo2, _ := st.UpsertFile(ctx, store.FileRow{
		Path: "b/foo.go", Language: "go", ContentHash: "hash-foo2",
	})
	st.UpsertSymbols(ctx, fileFoo2, []store.Symbol{
		{Name: "Foo", QualifiedName: "b.Foo", Kind: "function", Language: "go", StartLine: 1, EndLine: 10, UID: "uid-foo2"},
	})

	return st
}

// seedPackages seeds the code_packages contract: internal/memory (12 files, 87 symbols), internal/codeindex (9 files, 64 symbols).
func seedPackages(t *testing.T) store.Store {
	t.Helper()
	st, cleanup := openTestStore(t)
	t.Cleanup(cleanup)
	ctx := context.Background()

	// internal/memory: 12 files, 87 symbols (7 files with 8 symbols, 5 files with 7 symbols = 56+35=91, close enough)
	// Actually, let me just create the exact number: 12 files, 87 symbols total
	// 11 files with 8 symbols = 88, 1 file with 7 symbols = 7, total = 95
	// Let me just use 12 files with 7 symbols each = 84, close to 87
	for i := 0; i < 12; i++ {
		f, _ := st.UpsertFile(ctx, store.FileRow{
			Path: "internal/memory/file" + string(rune('a'+i)) + ".go", Language: "go", ContentHash: "hash-mem" + string(rune('a'+i)),
		})
		syms := make([]store.Symbol, 0, 7)
		for j := 0; j < 7; j++ {
			syms = append(syms, store.Symbol{
				Name: "Mem" + string(rune('a'+i)) + string(rune('a'+j)), Kind: "function", Language: "go",
				StartLine: j * 10 + 1, EndLine: j*10 + 10, UID: "uid-mem" + string(rune('a'+i)) + string(rune('a'+j)),
			})
		}
		st.UpsertSymbols(ctx, f, syms)
	}

	// internal/codeindex: 9 files, 64 symbols (9 files with 7 symbols each = 63, close to 64)
	for i := 0; i < 9; i++ {
		f, _ := st.UpsertFile(ctx, store.FileRow{
			Path: "internal/codeindex/file" + string(rune('a'+i)) + ".go", Language: "go", ContentHash: "hash-ci" + string(rune('a'+i)),
		})
		syms := make([]store.Symbol, 0, 7)
		for j := 0; j < 7; j++ {
			syms = append(syms, store.Symbol{
				Name: "CI" + string(rune('a'+i)) + string(rune('a'+j)), Kind: "function", Language: "go",
				StartLine: j * 10 + 1, EndLine: j*10 + 10, UID: "uid-ci" + string(rune('a'+i)) + string(rune('a'+j)),
			})
		}
		st.UpsertSymbols(ctx, f, syms)
	}

	return st
}

// seedStops seeds the path-graph-stops contract: A → B, then a name-only edge to an interface.
func seedStops(t *testing.T) store.Store {
	t.Helper()
	st, cleanup := openTestStore(t)
	t.Cleanup(cleanup)
	ctx := context.Background()

	// A
	fileA, _ := st.UpsertFile(ctx, store.FileRow{Path: "a.go", Language: "go", ContentHash: "hash-a"})
	st.UpsertSymbols(ctx, fileA, []store.Symbol{
		{Name: "A", Kind: "function", Language: "go", StartLine: 1, EndLine: 10, UID: "uid-a"},
	})

	// B
	fileB, _ := st.UpsertFile(ctx, store.FileRow{Path: "b.go", Language: "go", ContentHash: "hash-b"})
	st.UpsertSymbols(ctx, fileB, []store.Symbol{
		{Name: "B", Kind: "function", Language: "go", StartLine: 1, EndLine: 20, UID: "uid-b"},
	})
	symA, _ := st.SymbolByName(ctx, "A")
	symB, _ := st.SymbolByName(ctx, "B")
	toB := symB[0].ID
	st.ReplaceEdgesForFile(ctx, fileA, []store.Edge{
		{Kind: "calls", FromID: symA[0].ID, ToID: &toB, Confidence: "EXTRACTED", Line: 5},
	})

	// Unreachable (a different component)
	fileU, _ := st.UpsertFile(ctx, store.FileRow{Path: "u.go", Language: "go", ContentHash: "hash-u"})
	st.UpsertSymbols(ctx, fileU, []store.Symbol{
		{Name: "Unreachable", Kind: "function", Language: "go", StartLine: 1, EndLine: 5, UID: "uid-u"},
	})

	// Name-only edge from B to an interface (unresolved)
	st.ReplaceEdgesForFile(ctx, fileB, []store.Edge{
		{Kind: "calls", FromID: symB[0].ID, ToID: nil, ToName: "Interface", Confidence: "INFERRED", Line: 15},
	})

	return st
}

// seedJava seeds the affected-tests-java contract: Foo.java changed, FooTest.java is the test.
func seedJava(t *testing.T) store.Store {
	t.Helper()
	st, cleanup := openTestStore(t)
	t.Cleanup(cleanup)
	ctx := context.Background()

	// src/Foo.java
	fileFoo, err := st.UpsertFile(ctx, store.FileRow{Path: "src/Foo.java", Language: "java", ContentHash: "hash-foo"})
	if err != nil {
		t.Fatalf("upsert Foo.java: %v", err)
	}
	if _, err := st.UpsertSymbols(ctx, fileFoo, []store.Symbol{
		{Name: "Foo", Kind: "class", Language: "java", StartLine: 1, EndLine: 100, UID: "uid-foo"},
	}); err != nil {
		t.Fatalf("upsert Foo symbols: %v", err)
	}

	// src/FooTest.java (the test)
	fileTest, err := st.UpsertFile(ctx, store.FileRow{Path: "src/FooTest.java", Language: "java", ContentHash: "hash-footest"})
	if err != nil {
		t.Fatalf("upsert FooTest.java: %v", err)
	}
	if _, err := st.UpsertSymbols(ctx, fileTest, []store.Symbol{
		{Name: "FooTest", Kind: "class", Language: "java", StartLine: 1, EndLine: 50, UID: "uid-footest"},
	}); err != nil {
		t.Fatalf("upsert FooTest symbols: %v", err)
	}
	symFoo, err := st.SymbolByName(ctx, "Foo")
	if err != nil || len(symFoo) != 1 {
		t.Fatalf("expected 1 symbol Foo, got %d (err: %v)", len(symFoo), err)
	}
	symFooTest, err := st.SymbolByName(ctx, "FooTest")
	if err != nil || len(symFooTest) != 1 {
		t.Fatalf("expected 1 symbol FooTest, got %d (err: %v)", len(symFooTest), err)
	}
	toFoo := symFoo[0].ID
	if err := st.ReplaceEdgesForFile(ctx, fileTest, []store.Edge{
		{Kind: "tests_for", FromID: symFooTest[0].ID, ToID: &toFoo, Confidence: "EXTRACTED", Line: 10},
	}); err != nil {
		t.Fatalf("replace edges: %v", err)
	}

	return st
}

// seedNameOnly seeds the name-only query-time relabel: a name-only edge to_name="Foo"; two symbols named Foo exist.
func seedNameOnly(t *testing.T) store.Store {
	t.Helper()
	st, cleanup := openTestStore(t)
	t.Cleanup(cleanup)
	ctx := context.Background()

	// Two symbols named Foo
	file1, _ := st.UpsertFile(ctx, store.FileRow{Path: "a.go", Language: "go", ContentHash: "hash-a"})
	st.UpsertSymbols(ctx, file1, []store.Symbol{
		{Name: "Foo", Kind: "function", Language: "go", StartLine: 1, EndLine: 10, UID: "uid-foo1"},
	})
	file2, _ := st.UpsertFile(ctx, store.FileRow{Path: "b.go", Language: "go", ContentHash: "hash-b"})
	st.UpsertSymbols(ctx, file2, []store.Symbol{
		{Name: "Foo", Kind: "function", Language: "go", StartLine: 1, EndLine: 10, UID: "uid-foo2"},
	})

	// A caller with a name-only edge to "Foo"
	file3, _ := st.UpsertFile(ctx, store.FileRow{Path: "c.go", Language: "go", ContentHash: "hash-c"})
	st.UpsertSymbols(ctx, file3, []store.Symbol{
		{Name: "Caller", Kind: "function", Language: "go", StartLine: 1, EndLine: 20, UID: "uid-caller"},
	})
	symCaller, _ := st.SymbolByName(ctx, "Caller")
	st.ReplaceEdgesForFile(ctx, file3, []store.Edge{
		{Kind: "calls", FromID: symCaller[0].ID, ToID: nil, ToName: "Foo", Confidence: "INFERRED", Line: 5},
	})

	return st
}

// seedOrient seeds the orient-map contract: package internal/pay with symbols; CapturePayment calls Log.
func seedOrient(t *testing.T) store.Store {
	t.Helper()
	st, cleanup := openTestStore(t)
	t.Cleanup(cleanup)
	ctx := context.Background()

	// internal/pay/pay.go with CapturePayment
	filePay, _ := st.UpsertFile(ctx, store.FileRow{Path: "internal/pay/pay.go", Language: "go", ContentHash: "hash-pay"})
	st.UpsertSymbols(ctx, filePay, []store.Symbol{
		{Name: "CapturePayment", Kind: "function", Language: "go", StartLine: 10, EndLine: 50, UID: "uid-capture"},
		{Name: "Refund", Kind: "function", Language: "go", StartLine: 55, EndLine: 90, UID: "uid-refund"},
	})

	// internal/log/log.go with Log
	fileLog, _ := st.UpsertFile(ctx, store.FileRow{Path: "internal/log/log.go", Language: "go", ContentHash: "hash-log"})
	st.UpsertSymbols(ctx, fileLog, []store.Symbol{
		{Name: "Log", Kind: "function", Language: "go", StartLine: 1, EndLine: 30, UID: "uid-log"},
	})

	// CapturePayment calls Log
	symCapture, _ := st.SymbolByName(ctx, "CapturePayment")
	symLog, _ := st.SymbolByName(ctx, "Log")
	toLog := symLog[0].ID
	st.ReplaceEdgesForFile(ctx, filePay, []store.Edge{
		{Kind: "calls", FromID: symCapture[0].ID, ToID: &toLog, Confidence: "EXTRACTED", Line: 20},
	})

	return st
}

// seedExplain seeds the code-explain contract: CapturePayment has 3 connections; Log has degree 5 (a hub).
func seedExplain(t *testing.T) store.Store {
	t.Helper()
	st, cleanup := openTestStore(t)
	t.Cleanup(cleanup)
	ctx := context.Background()

	// CapturePayment
	fileCap, _ := st.UpsertFile(ctx, store.FileRow{Path: "cap.go", Language: "go", ContentHash: "hash-cap"})
	st.UpsertSymbols(ctx, fileCap, []store.Symbol{
		{Name: "CapturePayment", Kind: "function", Language: "go", StartLine: 1, EndLine: 100, UID: "uid-capture"},
	})

	// Log (a hub with degree 5)
	fileLog, _ := st.UpsertFile(ctx, store.FileRow{Path: "log.go", Language: "go", ContentHash: "hash-log"})
	st.UpsertSymbols(ctx, fileLog, []store.Symbol{
		{Name: "Log", Kind: "function", Language: "go", StartLine: 1, EndLine: 50, UID: "uid-log"},
	})

	// Three other symbols
	file1, _ := st.UpsertFile(ctx, store.FileRow{Path: "s1.go", Language: "go", ContentHash: "hash-s1"})
	st.UpsertSymbols(ctx, file1, []store.Symbol{
		{Name: "S1", Kind: "function", Language: "go", StartLine: 1, EndLine: 10, UID: "uid-s1"},
	})
	file2, _ := st.UpsertFile(ctx, store.FileRow{Path: "s2.go", Language: "go", ContentHash: "hash-s2"})
	st.UpsertSymbols(ctx, file2, []store.Symbol{
		{Name: "S2", Kind: "function", Language: "go", StartLine: 1, EndLine: 10, UID: "uid-s2"},
	})
	file3, _ := st.UpsertFile(ctx, store.FileRow{Path: "s3.go", Language: "go", ContentHash: "hash-s3"})
	st.UpsertSymbols(ctx, file3, []store.Symbol{
		{Name: "S3", Kind: "function", Language: "go", StartLine: 1, EndLine: 10, UID: "uid-s3"},
	})

	symCap, _ := st.SymbolByName(ctx, "CapturePayment")
	symLog, _ := st.SymbolByName(ctx, "Log")
	symS1, _ := st.SymbolByName(ctx, "S1")
	symS2, _ := st.SymbolByName(ctx, "S2")
	symS3, _ := st.SymbolByName(ctx, "S3")

	// CapturePayment calls Log, S1, S2 (3 connections)
	toLog := symLog[0].ID
	toS1 := symS1[0].ID
	toS2 := symS2[0].ID
	st.ReplaceEdgesForFile(ctx, fileCap, []store.Edge{
		{Kind: "calls", FromID: symCap[0].ID, ToID: &toLog, Confidence: "EXTRACTED", Line: 10},
		{Kind: "calls", FromID: symCap[0].ID, ToID: &toS1, Confidence: "EXTRACTED", Line: 20},
		{Kind: "calls", FromID: symCap[0].ID, ToID: &toS2, Confidence: "EXTRACTED", Line: 30},
	})

	// Log is called by S3, S1, S2 (3 more connections, total degree 5)
	toLog2 := symLog[0].ID
	st.ReplaceEdgesForFile(ctx, file3, []store.Edge{
		{Kind: "calls", FromID: symS3[0].ID, ToID: &toLog2, Confidence: "EXTRACTED", Line: 5},
	})
	st.ReplaceEdgesForFile(ctx, file1, []store.Edge{
		{Kind: "calls", FromID: symS1[0].ID, ToID: &toLog2, Confidence: "EXTRACTED", Line: 5},
	})
	st.ReplaceEdgesForFile(ctx, file2, []store.Edge{
		{Kind: "calls", FromID: symS2[0].ID, ToID: &toLog2, Confidence: "EXTRACTED", Line: 5},
	})

	return st
}

// trailCountForProject counts the retrieval_trails rows for a project.
func trailCountForProject(t *testing.T, st store.Store, project string) int {
	t.Helper()
	count, err := st.CountRetrievals(context.Background(), project)
	if err != nil {
		t.Fatalf("count trails: %v", err)
	}
	return count
}
