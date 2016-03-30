package main

import (
	"reflect"
	"testing"
)

var splitargtest = []struct {
	in  string
	out []string
}{
	{"", nil},
	{" ", nil},
	{"  ", nil},
	{"x", []string{"x"}},
	{"x x", []string{"x", "x"}},
	{"'x' 'x'", []string{"x", "x"}},
	{"'x'  'x'", []string{"x", "x"}},
	{"' x ' 'x'", []string{" x ", "x"}},
	{" ' x ' 'x'", []string{" x ", "x"}},
	{" ' x ' 'x' ", []string{" x ", "x"}},
	{" ' x ' 'x' x", []string{" x ", "x", "x"}},
	{" \" x \" 'x' x", []string{" x ", "x", "x"}},
	{" \" x \" 'x' x", []string{" x ", "x", "x"}},
	{" \" x \"x 'x'x xx", []string{" x x", "xx", "xx"}},
}

func TestSplitarg(t *testing.T) {
	for _, tt := range splitargtest {
		out := splitarg(tt.in)
		if !reflect.DeepEqual(out, tt.out) {
			t.Errorf("splitquoted(%q) => %q, want %q", tt.in, out, tt.out)
		}
	}
}
