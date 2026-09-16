package main

import (
	"reflect"
	"testing"
)

func TestReorderArgs(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"empty", nil, nil},
		{"no flags", []string{"install"}, []string{"install"}},
		{"flags only", []string{"--yes", "--verbose"}, []string{"--yes", "--verbose"}},
		{"flags before sub", []string{"--skip-clone", "install"}, []string{"--skip-clone", "install"}},
		{"reported: flags after sub", []string{"install", "--skip-clone"}, []string{"--skip-clone", "install"}},
		{"mixed", []string{"install", "--yes", "--verbose"},
			[]string{"--yes", "--verbose", "install"}},
		{"string value stays attached", []string{"install", "--sync-repo", "/tmp/x"},
			[]string{"--sync-repo", "/tmp/x", "install"}},
		{"inline = value", []string{"install", "--agents=opencode"},
			[]string{"--agents=opencode", "install"}},
		{"short flags", []string{"install", "-s", "-y", "-n"},
			[]string{"-s", "-y", "-n", "install"}},
		{"double dash separator", []string{"install", "--", "weird"},
			[]string{"install", "--", "weird"}},
		{"positional after sep", []string{"install", "--", "-not-a-flag"},
			[]string{"install", "--", "-not-a-flag"}},
	}
	for _, tc := range cases {
		got := reorderArgs(tc.in)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}
