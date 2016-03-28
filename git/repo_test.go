package git

import (
	"os"
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

	os.Setenv("GIT_DIR", path)
	body, err := exec.Command("git", "rev-parse", "--is-bare-repository").Output()
	os.Unsetenv("GIT_DIR")

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

	ok = IsBareRepository("/NONEXISTATNREPOHOPEFULLY")
	if ok {
		t.Fatalf("Bare repo check succeed for %q, but shouldn't!")
	}

	// Remove test directory again
	err = exec.Command("rm", "-rf", repo.Path).Run()

	if err != nil {
		t.Log("[W] Could not remove test git dir: %q", repo.Path)
	}
}
