package git

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepoBasic(t *testing.T) {

	const path = "test.git"
	repo, err := InitBareRepository(path)

	if err != nil {
		t.Fatalf("Creating repo failed with err: %v", err)
	}

	if !strings.HasSuffix(repo.Path, path) {
		t.Fatalf("Expected to see path suffix, got: %q", repo.Path)
	}

	if !filepath.IsAbs(repo.Path) {
		t.Fatalf("No absolute path in repo.Path, but: %q", repo.Path)
	}

	gd := "--git-dir=" + repo.Path
	body, err := exec.Command("git", gd, "rev-parse", "--is-bare-repository").Output()

	if err != nil {
		t.Fatalf("Expected to run git rev-parse, but failed with: %v", err)
	}

	if !strings.HasPrefix(string(body), "true") {
		t.Fatalf("Creating a proper git repository failed!")
	}

	ok := IsBareRepository(repo.Path)

	if !ok {
		t.Fatalf("Bare repo check that shouldn't fail, did fail! [%q]", repo.Path)
	}

	desc := "cooles repo!"
	err = repo.WriteDescription(desc)
	if err != nil {
		t.Fatalf("Could not write repo description: %v", err)
	}

	if ddisk := repo.ReadDescription(); ddisk != desc {
		t.Fatalf("Reop descriptions not as expected: got %q, wanted %q", desc, ddisk)
	}

	ok = IsBareRepository("/NONEXISTATNREPOHOPEFULLY")
	if ok {
		t.Fatalf("Bare repo check succeed for '/NONEXISTATNREPOHOPEFULLY', but shouldn't!")
	}

	rf, err := OpenRepository("/NONEXISTATNREPOHOPEFULLY")
	if err == nil || rf != nil {
		t.Fatalf("Could open Repository that shouldn't exists!")
	}

	r2, err := OpenRepository(repo.Path)
	if err != nil {
		t.Fatalf("Could not open existing bare repository %v", err)
	} else if r2 == nil {
		t.Fatalf("OpenRepository returned nil")
	}

	ok = r2.HasAnnex()
	if ok {
		t.Fatalf("Fresh bare repo and HasAnnex == true, should be false!")
	}

	// Remove test directory again
	err = exec.Command("rm", "-rf", repo.Path).Run()

	if err != nil {
		t.Logf("[W] Could not remove test git dir: %q", repo.Path)
	}
}

func TestRepoAnnexBasic(t *testing.T) {

	_, err := exec.LookPath("git-annex")
	if err != nil {
		t.Skip("[W] Could not find git-annex binary. Skipping test")
	}

	const path = "test.git"
	repo, err := InitBareRepository(path)
	if err != nil {
		t.Fatalf("Creating repo failed with err: %v", err)
	}

	err = repo.InitAnnex()
	if err != nil {
		t.Fatalf("Failed to initialize git annex support err: %v", err)
	}

	ok := repo.HasAnnex()
	if !ok {
		t.Fatalf("HasAnnex failed after repo.InitAnnex() succeded!")
	}

	// Remove test directory again
	err = exec.Command("rm", "-rf", repo.Path).Run()

	if err != nil {
		t.Logf("[W] Could not remove test git dir: %q", repo.Path)
	}
}

var ofptests = []struct {
	root  string
	path  string
	otype ObjectType
	err   bool
}{
	// 9c3409e9225137bcccf070e1bb583b808da37003 is a commit object
	{"9c3409e9225137bcccf070e1bb583b808da37003", "/git/repo_test.go", ObjBlob, false}, // this is us!
	{"9c3409e9225137bcccf070e1bb583b808da37003", "/git", ObjTree, false},              // our parent dir
	{"9c3409e9225137bcccf070e1bb583b808da37003", "/cmd/gin-shell/main.go", ObjBlob, false},
	{"9c3409e9225137bcccf070e1bb583b808da37003", "NONEXISTANT/PATH/FOOBAR", ObjTree, true},
	// 5ed35b298dad11daaaf5a497b5682ee53af41a9b is a tree object
	{"5ed35b298dad11daaaf5a497b5682ee53af41a9b", "/git/repo_test.go", ObjBlob, false}, // this is us, again
	{"5ed35b298dad11daaaf5a497b5682ee53af41a9b", "NONEXISTANT/PATH/FOOBAR", ObjTree, true},
}

func TestObjectForPath(t *testing.T) {
	repo, err := DiscoverRepository()

	if err != nil {
		t.Skip("[W] Not in git directory. Skipping test")
	}

	for _, tt := range ofptests {
		oid, _ := ParseSHA1(tt.root)
		root, err := repo.OpenObject(oid)

		if err != nil {
			t.Fatalf("ObjectForPath(%q, ...): could not object root obj: %v", tt.root, err)
		}

		obj, err := repo.ObjectForPath(root, tt.path)

		if tt.err && err == nil {
			t.Fatalf("ObjectForPath(%.7q, %q) => no error but expected one", tt.root, tt.path)
		} else if !tt.err && err != nil {
			t.Fatalf("ObjectForPath(%.7q, %q) => error: %v, wanted %s obj", tt.root, tt.path, err, tt.otype)
		} else if !tt.err && obj.Type() != tt.otype {
			t.Fatalf("Expected %s object, got %s", tt.otype, obj.Type())
		}

		//looking good

		if err == nil {
			t.Logf("ObjectForPath(%.7q, %q) => %s [OK!]", tt.root, tt.path, obj.Type())
		} else {
			t.Logf("ObjectForPath(%.7q, %q) => expected error(%q) [OK!]", tt.root, tt.path, err)
		}
	}

}
