package extract

import "testing"

func TestDeriveTypeInfoGo(t *testing.T) {
	cases := []struct {
		sig, name, wantRet, wantParams string
	}{
		{"func (s *Store) Query(ctx context.Context, q string) (*Row, error)", "Query", "*Row", "context.Context, string"},
		{"func Do(x int, y float64) (int, error)", "Do", "int", "int, float64"},
		{"func Greet() string", "Greet", "string", ""},
		{"func (s *S) Run()", "Run", "", ""},
		{"func Multi() (int, string, error)", "Multi", "int", ""},
	}
	for _, c := range cases {
		ti := deriveTypeInfo("go", c.name, c.sig)
		if ti.ReturnType != c.wantRet {
			t.Errorf("%s ret: got %q want %q", c.name, ti.ReturnType, c.wantRet)
		}
		if ti.ParamTypes != c.wantParams {
			t.Errorf("%s params: got %q want %q", c.name, ti.ParamTypes, c.wantParams)
		}
	}
}

func TestDeriveTypeInfoTS(t *testing.T) {
	ti := deriveTypeInfo("typescript", "load", "load(path: string): Promise<Doc>")
	if ti.ReturnType != "Promise<Doc>" {
		t.Errorf("ret: got %q want Promise<Doc>", ti.ReturnType)
	}
	if ti.ParamTypes != "string" {
		t.Errorf("params: got %q want string", ti.ParamTypes)
	}
}

func TestDeriveTypeInfoNonFunction(t *testing.T) {
	// A non-anchorable signature yields no type (never "unknown-but-present").
	if ti := deriveTypeInfo("go", "Foo", "const Foo = 1"); ti.ReturnType != "" || ti.ParamTypes != "" {
		t.Errorf("const: got %+v want empty", ti)
	}
}

func TestVisibilityGo(t *testing.T) {
	if v := visibilityOf("Query", "go"); v != "export" {
		t.Errorf("Query: got %q", v)
	}
	if v := visibilityOf("query", "go"); v != "private" {
		t.Errorf("query: got %q", v)
	}
	if v := visibilityOf("x", "typescript"); v != "unexported" {
		t.Errorf("ts x: got %q", v)
	}
	if v := visibilityOf("", "go"); v != "" {
		t.Errorf("empty: got %q", v)
	}
}

func TestIsExternalMember(t *testing.T) {
	if !isExternalMember("javascript", "p", "catch") {
		t.Error("js catch should be external")
	}
	if !isExternalMember("go", "rows", "Scan") {
		t.Error("go lowercase-receiver member should be external")
	}
	if isExternalMember("go", "MyStruct", "Method") {
		t.Error("go uppercase receiver should be internal (resolvable)")
	}
}
