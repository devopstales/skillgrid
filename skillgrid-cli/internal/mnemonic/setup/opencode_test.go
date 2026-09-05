package setup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tidwall/gjson"
)

func TestAppendJSONArrayUniqueWritesStringPaths(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tui.json")
	if err := os.WriteFile(path, []byte(`{"theme":"tokyonight"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	plugin := "/home/user/.config/opencode/tui-plugins/skillgrid-logo.tsx"
	if err := appendJSONArrayUnique(path, "plugin", plugin, false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	arr := gjson.GetBytes(data, "plugin")
	if !arr.IsArray() {
		t.Fatalf("plugin is not an array: %s", data)
	}
	if got := arr.Array(); len(got) != 1 || got[0].Type != gjson.String || got[0].Str != plugin {
		t.Fatalf("want string plugin path %q, got %s", plugin, data)
	}
	// Must not marshal gjson.Result struct fields.
	if gjson.GetBytes(data, "plugin.0.Type").Exists() || gjson.GetBytes(data, "plugin.0.Str").Exists() {
		t.Fatalf("plugin entry looks like marshaled gjson.Result: %s", data)
	}
}

func TestAppendJSONArrayUniqueHealsLegacyGjsonMarshal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tui.json")
	plugin := "/home/user/.config/opencode/tui-plugins/skillgrid-logo.tsx"
	legacy := `{"theme":"tokyonight","plugin":[{"Type":0,"Raw":"","Str":"` + plugin + `","Num":0,"Index":0,"Indexes":null}]}`
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := appendJSONArrayUnique(path, "plugin", plugin, false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := gjson.GetBytes(data, "plugin").Array()
	if len(got) != 1 || got[0].Type != gjson.String || got[0].Str != plugin {
		t.Fatalf("want healed string array, got %s", data)
	}
}

func TestAppendJSONArrayUniqueIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tui.json")
	plugin := "/tmp/logo.tsx"
	if err := os.WriteFile(path, []byte(`{"plugin":["`+plugin+`"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := appendJSONArrayUnique(path, "plugin", plugin, false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(gjson.GetBytes(data, "plugin").Array()); n != 1 {
		t.Fatalf("want 1 entry, got %d: %s", n, data)
	}
}
