package store

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

func Test_RepoExists(t *testing.T) {
	const invalidUser = "iDoNotExist"
	const invalidRepo = "iDoNotExist"
	const validUser = "alice"
	const validRepo = "auth"

	// Test empty RepoId
	id := RepoId{"", ""}
	exists, err := repos.RepoExists(id)
	if err != nil {
		t.Fatalf("Unexpected error on empty RepoId: %v\n", err)
	}
	if exists {
		t.Fatal("Did not expect true on empty RepoId.")
	}

	// Test invalid user
	id = RepoId{invalidUser, ""}
	exists, err = repos.RepoExists(id)
	if err != nil {
		t.Fatalf("Unexpected error on invalid user: %v\n", err)
	}
	if exists {
		t.Fatal("Did not expect true on invalid user.")
	}

	// Test missing repo name with existing user
	id = RepoId{validUser, ""}
	exists, err = repos.RepoExists(id)
	if err != nil {
		t.Fatalf("Unexpected error on missing repo: %v\n", err)
	}
	if exists {
		t.Fatal("Did not expect true on missing repo.")
	}

	// Test invalid repo name with existing user
	id = RepoId{validUser, invalidRepo}
	exists, err = repos.RepoExists(id)
	if err != nil {
		t.Fatalf("Unexpected error on invalid repo: %v\n", err)
	}
	if exists {
		t.Fatal("Did not expect true on invalid repo.")
	}

	// Test valid user with valid repository
	id = RepoId{validUser, validRepo}
	exists, err = repos.RepoExists(id)
	if err != nil {
		t.Fatalf("Unexpected error on valid RepoId: %v\n", err)
	}
	if !exists {
		t.Fatal("Did not expect false on valid RepoId.")
	}
}

func TestRepoStore_IdToPath(t *testing.T) {
	const repoOwner = "alice"
	const repoName = "auth"

	r := RepoId{Owner: repoOwner, Name: repoName}
	path := repos.IdToPath(r)

	if !strings.Contains(path, fmt.Sprintf(filepath.Join("repos", "git", repoOwner, repoName+".git"))) {
		t.Fatalf("Received unexpected repository path: %q\n", path)
	}
}

func TestRepoStore_RepoShared(t *testing.T) {
	const repoOwner = "alice"
	const repoInvalid = "iDoNotExist"
	const repoValid = "repod"
	const repoShared = "openfmri"

	// Test false on error when opening non existing repository
	rid := RepoId{Owner: repoOwner, Name: repoInvalid}
	ok := repos.RepoShared(rid)
	if ok {
		t.Fatalf("Expected fail when opening %v\n", rid)
	}

	// Test false on non shared repository
	rid.Name = repoValid
	ok = repos.RepoShared(rid)
	if ok {
		t.Fatalf("Expected fail when opening %v\n", rid)
	}

	// Test true on shared repository
	rid.Name = repoShared
	ok = repos.RepoShared(rid)
	if !ok {
		t.Fatalf("Expected success when opening %v\n", rid)
	}
}
