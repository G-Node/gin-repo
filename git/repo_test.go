package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepoInit(t *testing.T) {

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

	if err != nil {
		t.Fatalf("Expected to run git rev-parse, but failed with: %v", err)
	}

	if !strings.HasPrefix(string(body), "true") {
		t.Fatalf("Creating a proper git repository failed!")
	}

	err = exec.Command("rm", "-rf", repo.Path).Run()

	if err != nil {
		t.Log("[W] Could not remove test git dir: %q", repo.Path)
	}
}
