package store

import (
	"reflect"
	"testing"
)

var defaultRepo = &RepoId{"foo", "bar"}

var repoids = []struct {
	in  string
	out *RepoId
}{
	//valid repos
	{"foo/bar", defaultRepo},
	{"/foo/bar", defaultRepo},
	{"foo/bar/", defaultRepo},
	{"/foo/bar/", defaultRepo},
	{"/~/foo/bar", defaultRepo},
	{"/~/foo/bar/", defaultRepo},

	//invalid paths
	{"foo", nil},
	{"~/foo/bar/", nil},
	{"//foo//", nil},
	{"/~/foo", nil},
	{"/~//foo/bar", nil},
	{"/foo/bar/se", nil},
	{"foo/bar/se/", nil},
	{"foo/bar/se/foo/bar/se", nil},
	{"//foo//bar//se", nil},

	//invalid names
	{"/~foo/bar/", nil},
	{"/foo~/bar/", nil},
	{"/a/b", nil},
}

func TestParseRepoId(t *testing.T) {

	for _, tt := range repoids {
		out, err := RepoIdParse(tt.in)
		if err != nil && tt.out != nil {
			t.Errorf("RepoIdParse(%q) => error, want: %v", tt.in, *tt.out)
		} else if err == nil && tt.out == nil {
			t.Errorf("RepoIdParse(%q) => %v, want error", tt.in, out)
		} else if err == nil && tt.out != nil && !reflect.DeepEqual(out, *(tt.out)) {
			t.Errorf("RepoIdParse(%q) => %v, want %v", tt.in, out, *tt.out)
		}
	}
}
