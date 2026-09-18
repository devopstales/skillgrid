package memfs

import (
	"testing"
)

func TestResolveCodePath(t *testing.T) {
	cases := []struct {
		raw    string
		kind   string
		dir    string
		file   string
		symbol string
		want   error
	}{
		{raw: "", kind: "dir", dir: ""},
		{raw: ".", kind: "dir", dir: ""},
		{raw: "memfs://project/A/src/", kind: "dir", dir: "src"},
		{raw: "A/src/", kind: "dir", dir: "src"},
		{raw: "src/", kind: "dir", dir: "src"},
		{raw: "memfs://project/A/src/auth/login.go", kind: "file", dir: "src/auth", file: "login.go"},
		{raw: "A/src/auth/login.go", kind: "file", dir: "src/auth", file: "login.go"},
		{raw: "A/src/auth/login.go::Handler", kind: "symbol", dir: "src/auth", file: "login.go", symbol: "Handler"},
		{raw: "memfs://project/B/src/", want: errCodeScopeMismatch}, // id B != A
	}
	for _, c := range cases {
		got, err := ResolveCodePath("A", c.raw)
		if (err != nil) != (c.want != nil) {
			t.Fatalf("ResolveCodePath(%q) err=%v, want err=%v", c.raw, err, c.want)
		}
		if c.want != nil {
			if err != c.want {
				t.Fatalf("ResolveCodePath(%q) = %v, want %v", c.raw, err, c.want)
			}
			continue
		}
		if got.Kind != c.kind || got.Dir != c.dir || got.File != c.file || got.Symbol != c.symbol {
			t.Errorf("ResolveCodePath(%q) = %+v, want kind=%s dir=%q file=%q sym=%q",
				c.raw, got, c.kind, c.dir, c.file, c.symbol)
		}
	}
}
