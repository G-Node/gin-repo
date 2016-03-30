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
		t.Fatalf("Bare repo check succeed for %q, but shouldn't!")
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
		t.Log("[W] Could not remove test git dir: %q", repo.Path)
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
		t.Log("[W] Could not remove test git dir: %q", repo.Path)
	}
}
