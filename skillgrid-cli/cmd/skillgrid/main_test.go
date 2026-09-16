package main

import (
	"reflect"
	"testing"
)

func TestParseAgents(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", []string{}},
		{"opencode", []string{"opencode"}},
		{"opencode,kilo", []string{"opencode", "kilo"}},
		{" kilo , OPENCODE ", []string{"kilo", "opencode"}},
	}
	for _, tc := range cases {
		got := parseAgents(tc.in)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("parseAgents(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
