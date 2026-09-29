package wiki

import "testing"

func TestSlugify(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"normal words", "Hello World", "hello-world"},
		{"single word", "Hello", "hello"},
		{"already lowercase", "already-lower", "already-lower"},
		{"empty string", "", ""},
		{"only punctuation", "!!! ???", ""},
		{"leading and trailing punctuation", "  --Hello--  ", "hello"},
		{"mixed case", "HelloWorld", "helloworld"},
		{"spaces between", "a b c", "a-b-c"},
		{"underscore", "foo_bar", "foo-bar"},
		{"dotted", "foo.bar", "foo-bar"},
		{"slash", "foo/bar", "foo-bar"},
		{"multiple spaces", "a   b", "a-b"},
		{"consecutive punctuation", "a!!!b", "a-b"},
		{"reserved index", "Index", "page-index"},
		{"reserved log", "Log", "page-log"},
		{"reserved index lowercase", "index", "page-index"},
		{"reserved log lowercase", "log", "page-log"},
		{"reserved index with punctuation", "INDEX!", "page-index"},
		{"not reserved indexing", "Indexing", "indexing"},
		{"not reserved logger", "Logger", "logger"},
		{"page-index not double prefixed", "page-index", "page-index"},
		{"page-log not double prefixed", "page-log", "page-log"},
		{"unicode latin accented dropped", "Café au Lait", "caf-au-lait"},
		{"unicode umlaut dropped", "Über Cool", "ber-cool"},
		{"unicode cjk all dropped", "你好 世界", ""},
		{"unicode mixed latin cjk dropped", "Foo 你好 Bar", "foo-bar"},
		{"digits kept", "Go 1.25 Feature", "go-1-25-feature"},
		{"digit only", "12345", "12345"},
		{"tab and newline", "Foo\tBar\nBaz", "foo-bar-baz"},
		{"single alnum", "a", "a"},
		{"single non-alnum", "#", ""},
		{"all digits", "999", "999"},
		{"trailing boundary", "foo-", "foo"},
		{"leading boundary", "-foo", "foo"},
		{"boundary both ends", "--foo--", "foo"},
		{"uppercase all", "HELLO WORLD", "hello-world"},
		{"empty after trim", "   ", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Slugify(tc.input)
			if got != tc.want {
				t.Fatalf("Slugify(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestSlugifyIdempotentOnSlugs(t *testing.T) {
	// Slugging an already-slug should be stable.
	s := "hello-world-123"
	got := Slugify(s)
	if got != s {
		t.Fatalf("Slugify(%q) = %q, want stable %q", s, got, s)
	}
}

func TestTypeDir(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"ADR", "ADR", "adr"},
		{"Term", "Term", "concepts"},
		{"Constraint", "Constraint", "concepts"},
		{"Spec", "Spec", "entities"},
		{"Spike", "Spike", "entities"},
		{"State", "State", "entities"},
		{"Architecture", "Architecture", "entities"},
		{"Finding", "Finding", "entities"},
		{"unknown", "Bogus", "entities"},
		{"empty", "", "entities"},
		{"case sensitive adr lower", "adr", "entities"},
		{"case sensitive term lower", "term", "entities"},
		{"surrounding spaces", "  ADR  ", "adr"},
		{"tab around", "\tADR\t", "adr"},
		{"unknown long word", "NotARealType", "entities"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := TypeDir(tc.input)
			if got != tc.want {
				t.Fatalf("TypeDir(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
