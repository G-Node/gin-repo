package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/G-Node/gin-repo/store"
)

const repoUser = "alice"
const repoName = "exrepo"
const defaultRepoHead = "master"

func TestRepoToWire(t *testing.T) {

	id := store.RepoId{Owner: repoUser, Name: repoName}

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

func Test_getRepoVisibility(t *testing.T) {
	const invalidUser = "iDoNotExist"
	const invalidRepo = "iDoNotExist"
	const validUser = "alice"
	const validPublicRepo = "auth"
	const validPrivateRepo = "exrepo"

	// test request fail for non existing user
	url := fmt.Sprintf("/users/%s/repos/%s/visibility", invalidUser, invalidRepo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = makeRequest(t, req, http.StatusBadRequest)
	if err != nil {
		t.Fatal(err)
	}

	// test request fail for existing user, non existing repository w/o authorization
	url = fmt.Sprintf("/users/%s/repos/%s/visibility", validUser, invalidRepo)
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = makeRequest(t, req, http.StatusBadRequest)
	if err != nil {
		t.Fatal(err)
	}

	// test request fail for existing user, non existing repository with authorization
	token, err := server.users.TokenForUser(validUser)
	if err != nil {
		t.Fatalf("could not make token for %q: %v, %v", validUser, token, err)
	}
	req.Header.Add("Authorization", "Bearer "+token)
	_, err = makeRequest(t, req, http.StatusBadRequest)
	if err != nil {
		t.Fatal(err)
	}

	// test request for existing user, existing public repository with authorization
	url = fmt.Sprintf("/users/%s/repos/%s/visibility", validUser, validPublicRepo)
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	token, err = server.users.TokenForUser(validUser)
	if err != nil {
		t.Fatalf("could not make token for %q: %v, %v", validUser, token, err)
	}
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err := makeRequest(t, req, http.StatusOK)
	if err != nil {
		t.Fatalf("Unexpected error on public repository: %v\n", err)
	}

	var visibility struct {
		Public bool
	}

	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&visibility)
	if err != nil {
		t.Fatalf("Error decoding response: %v\n", err)
	}
	if !visibility.Public {
		t.Fatalf("Expected public true for repository %s\n", validPublicRepo)
	}

	// test request for existing user, existing private repository with authorization
	url = fmt.Sprintf("/users/%s/repos/%s/visibility", validUser, validPrivateRepo)
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	token, err = server.users.TokenForUser(validUser)
	if err != nil {
		t.Fatalf("could not make token for %q: %v, %v", validUser, token, err)
	}
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err = makeRequest(t, req, http.StatusOK)
	if err != nil {
		t.Fatalf("Unexpected error on public repository: %v\n", err)
	}

	dec = json.NewDecoder(resp.Body)
	err = dec.Decode(&visibility)
	if err != nil {
		t.Fatalf("Error decoding response: %v\n", err)
	}
	if visibility.Public {
		t.Fatalf("Expected public false for repository %s\n", validPrivateRepo)
	}
}
