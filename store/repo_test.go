package store

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/G-Node/gin-repo/internal/testbed"
)

var users UserStore
var repos *RepoStore

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
	{"foo/bar.git", defaultRepo},
	{"/foo/bar.git", defaultRepo},
	{"/~/foo/bar.git", defaultRepo},
	{"/~/foo/bar.git/", defaultRepo},

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
	{"//foo//bar.", nil},
	{"foo//bar.gi", nil},
	{"foo//bar.git.", nil},

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

// TestMain sets up a temporary user store for store method tests.
// Currently the temporary files created by this function are not cleaned up.
func TestMain(m *testing.M) {
	repoDir, err := testbed.MkData("/tmp")
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not make test data: %v\n", err)
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get cwd: %v", err)
		os.Exit(1)
	}

	// Change to repoDir is required, since creating a local user store depends on
	// auth.ReadSharedSecret(), which reads a file "gin.secret" from the current directory
	// and fails otherwise.
	err = os.Chdir(repoDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not set cwd to repoDir: %v", err)
		os.Exit(1)
	}

	users, err = NewUserStore(repoDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not create local user store: %v\n", err)
		os.Exit(1)
	}

	repos, err = NewRepoStore(repoDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not create local repo store: %v\n", err)
		os.Exit(1)
	}

	res := m.Run()

	_ = os.Chdir(cwd)
	os.Exit(res)
}
