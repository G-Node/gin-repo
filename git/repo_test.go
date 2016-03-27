package git

import (
	"os"
	"os/exec"
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

	os.Setenv("GIT_DIR", path)
	body, err := exec.Command("git", "rev-parse", "--is-bare-repository").Output()

	if err != nil {
		t.Fatalf("Expected to run git rev-parse, but failed with: %v", err)
	}

	if !strings.HasPrefix(string(body), "true") {
		t.Fatalf("Creating a proper git repository failed!")
	}

}
