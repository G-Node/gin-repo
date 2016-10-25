package main

import (
	"testing"

	"github.com/G-Node/gin-repo/store"
)

const repoUser = "alice"
const repoName = "exrepo"
const defaultRepoHead = "master"

func TestRepoToWire(t *testing.T) {

	id := store.RepoId{repoUser, repoName}

	repo, err := server.repos.OpenGitRepo(id)
	if err != nil {
		t.Fatalf("Error fetching repository %v: %v\n", id, err)
	}

	wired, err := server.repoToWire(id, repo)
	if err != nil {
		t.Fatalf("Error wiring repository settings: %v\n", err)
	}

	if wired.Name != repoName {
		t.Errorf("Expected repository name to be %q but was %q\n", repoName, wired.Name)
	}
	if wired.Owner != repoUser {
		t.Errorf("Expected repository owner to be %q but was %q\n", repoUser, wired.Owner)
	}
	if wired.Description == "" {
		t.Error("Expected repository description but got empty field")
	}
	if wired.Public {
		t.Errorf("Expected repository public setting to be false but was: %t\n", wired.Public)
	}
	if wired.Head != defaultRepoHead {
		t.Errorf("Expected repository head to be %q but got %q\n", defaultRepoHead, wired.Head)
	}
}

func Test_varsToRepoID(t *testing.T) {
	const user = "userName"
	const repo = "repoName"

	// test missing arguments
	vars := make(map[string]string)
	_, err := server.varsToRepoID(vars)
	if err == nil {
		t.Fatal("Expected error on missing arguments.")
	}
	// test missing user
	vars["repo"] = repo
	_, err = server.varsToRepoID(vars)
	if err == nil {
		t.Fatal("Expected error on missing user.")
	}

	// test missing repository
	delete(vars, "repo")
	vars["user"] = user
	_, err = server.varsToRepoID(vars)
	if err == nil {
		t.Fatal("Expected error on missing repository.")
	}

	// test existing user and repository
	vars["repo"] = repo
	id, err := server.varsToRepoID(vars)
	if err != nil {
		t.Fatalf("Error on exising user and repository: %v\n", err)
	}
	if id.Owner != user {
		t.Fatalf("Expected user %q, but got %q\n", user, id.Owner)
	}
	if id.Name != repo {
		t.Fatalf("Expected repository %q but got %q\n", repo, id.Name)
	}
}
