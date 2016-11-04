package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/G-Node/gin-repo/git"
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

func Test_setRepoVisibility(t *testing.T) {
	const invalidUser = "iDoNotExist"
	const invalidRepo = "iDoNotExist"
	const validUser = "alice"
	const validPublicRepo = "auth"

	// Currently the test repositories are set up only once.
	// This means tests changing repository settings are not independent,
	// if repository settings are changed, they have to be reset after the test.
	// TODO organize the repo setup for tests so that they are reset for every test.
	defer server.repos.SetRepoVisibility(store.RepoId{Owner: validUser, Name: validPublicRepo}, true)

	token, err := server.users.TokenForUser(validUser)
	if err != nil {
		t.Fatalf("could not make token for %q: %v, %v", validUser, token, err)
	}

	// test request fail for non existing user
	url := fmt.Sprintf("/users/%s/repos/%s/visibility", invalidUser, invalidRepo)
	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = makeRequest(t, req, http.StatusBadRequest)
	if err != nil {
		t.Fatal(err)
	}

	// test request fail for existing user, non existing repository w/o authorization
	url = fmt.Sprintf("/users/%s/repos/%s/visibility", validUser, invalidRepo)
	req, err = http.NewRequest("PUT", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = makeRequest(t, req, http.StatusBadRequest)
	if err != nil {
		t.Fatal(err)
	}

	// test request fail for existing user, non existing repository with authorization
	url = fmt.Sprintf("/users/%s/repos/%s/visibility", validUser, invalidRepo)
	req, err = http.NewRequest("PUT", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer "+token)
	_, err = makeRequest(t, req, http.StatusBadRequest)
	if err != nil {
		t.Fatal(err)
	}

	// test request fail for existing user, existing repository w/o authorization
	url = fmt.Sprintf("/users/%s/repos/%s/visibility", validUser, validPublicRepo)
	req, err = http.NewRequest("PUT", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = makeRequest(t, req, http.StatusForbidden)
	if err != nil {
		t.Fatal(err)
	}

	// test request fail for existing user, existing repository with authorization and missing body
	url = fmt.Sprintf("/users/%s/repos/%s/visibility", validUser, validPublicRepo)
	req, err = http.NewRequest("PUT", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer "+token)
	_, err = makeRequest(t, req, http.StatusBadRequest)
	if err != nil {
		t.Fatal(err)
	}

	// test request fail for existing user, existing repository with authorization and empty body
	url = fmt.Sprintf("/users/%s/repos/%s/visibility", validUser, validPublicRepo)
	req, err = http.NewRequest("PUT", url, strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/json")
	_, err = makeRequest(t, req, http.StatusBadRequest)
	if err != nil {
		t.Fatal(err)
	}

	// test request fail for existing user, existing repository with authorization and invalid request body
	url = fmt.Sprintf("/users/%s/repos/%s/visibility", validUser, validPublicRepo)
	req, err = http.NewRequest("PUT", url, strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/json")
	_, err = makeRequest(t, req, http.StatusBadRequest)
	if err != nil {
		t.Fatal(err)
	}

	// test request fail for existing user, existing repository with authorization and invalid request field type
	url = fmt.Sprintf("/users/%s/repos/%s/visibility", validUser, validPublicRepo)
	req, err = http.NewRequest("PUT", url, strings.NewReader(`{"public":"True"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/json")
	_, err = makeRequest(t, req, http.StatusBadRequest)
	if err != nil {
		t.Fatal(err)
	}

	// test correct request, set public from true to false
	rid := store.RepoId{Owner: validUser, Name: validPublicRepo}
	err = server.repos.SetRepoVisibility(rid, true)
	if err != nil {
		t.Fatal("Unable to set repository visibility.")
	}
	check, err := server.repos.GetRepoVisibility(rid)
	if err != nil {
		t.Fatalf("Error fetching repository visibility: %v\n", err)
	}
	if !check {
		t.Fatalf("Invalid repository visibility: %t\n", check)
	}

	url = fmt.Sprintf("/users/%s/repos/%s/visibility", validUser, validPublicRepo)
	req, err = http.NewRequest("PUT", url, strings.NewReader(`{"public":false}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/json")
	resp, err := makeRequest(t, req, http.StatusOK)
	if err != nil {
		t.Fatal(err)
	}

	var visibility struct {
		Public bool
	}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&visibility)
	if err != nil {
		t.Fatalf("Error decoding response: %v\n", err)
	}
	if visibility.Public {
		t.Fatalf("Expected public false for repository %s\n", validPublicRepo)
	}

	check, err = server.repos.GetRepoVisibility(rid)
	if err != nil {
		t.Fatalf("Error fetching repository visibility: %v\n", err)
	}
	if check {
		t.Fatalf("Invalid repository visibility: %t\n", check)
	}
	// reset repository setting
	err = server.repos.SetRepoVisibility(rid, true)
	if err != nil {
		t.Fatal("Unable to set repository visibility.")
	}
}

func Test_patchRepoSettings(t *testing.T) {
	const settingsUrl = "/users/%s/repos/%s/settings"
	const invalidUser = "iDoNotExist"
	const invalidRepo = "iDoNotExist"
	const validUser = "alice"
	const validRepo = "auth"

	// Reset repository setting after the test
	defer server.repos.SetRepoVisibility(store.RepoId{Owner: validUser, Name: validRepo}, true)

	// TODO the following section is actually just a copy of the first part
	// TODO of the get/setRepoVisibility tests. Good enough for now, but unify at some point.

	token, err := server.users.TokenForUser(validUser)
	if err != nil {
		t.Fatalf("could not make token for %q: %v, %v", validUser, token, err)
	}

	// test request fail for missing authorization header
	url := fmt.Sprintf(settingsUrl, invalidUser, invalidRepo)
	req, err := http.NewRequest("PATCH", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = makeRequest(t, req, http.StatusForbidden)
	if err != nil {
		t.Fatal(err)
	}

	// test request fail for missing bearer token in authorization header
	url = fmt.Sprintf(settingsUrl, invalidUser, invalidRepo)
	req, err = http.NewRequest("PATCH", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "")
	_, err = makeRequest(t, req, http.StatusForbidden)
	if err != nil {
		t.Fatal(err)
	}

	// test request fail for non existing user.
	url = fmt.Sprintf(settingsUrl, invalidUser, invalidRepo)
	req, err = http.NewRequest("PATCH", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer ")
	_, err = makeRequest(t, req, http.StatusBadRequest)
	if err != nil {
		t.Fatal(err)
	}

	// test request fail for existing user, non existing repository w/o proper authorization.
	url = fmt.Sprintf(settingsUrl, validUser, invalidRepo)
	req, err = http.NewRequest("PATCH", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer ")
	_, err = makeRequest(t, req, http.StatusBadRequest)
	if err != nil {
		t.Fatal(err)
	}

	// test request fail for existing user, non existing repository with proper authorization.
	url = fmt.Sprintf(settingsUrl, validUser, invalidRepo)
	req, err = http.NewRequest("PATCH", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer "+token)
	_, err = makeRequest(t, req, http.StatusBadRequest)
	if err != nil {
		t.Fatal(err)
	}

	// test request fail for existing user, existing repository with authorization and missing body.
	url = fmt.Sprintf(settingsUrl, validUser, validRepo)
	req, err = http.NewRequest("PATCH", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer "+token)
	_, err = makeRequest(t, req, http.StatusBadRequest)
	if err != nil {
		t.Fatal(err)
	}

	// test request fail for existing user, existing repository with authorization and empty body.
	url = fmt.Sprintf(settingsUrl, validUser, validRepo)
	req, err = http.NewRequest("PATCH", url, strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/json")
	_, err = makeRequest(t, req, http.StatusBadRequest)
	if err != nil {
		t.Fatal(err)
	}

	// test request fail for existing user, existing repository with authorization and missing patchable field.
	url = fmt.Sprintf(settingsUrl, validUser, validRepo)
	req, err = http.NewRequest("PATCH", url, strings.NewReader(`{"otherfield":"has nowhere to patch"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/json")
	_, err = makeRequest(t, req, http.StatusBadRequest)
	if err != nil {
		t.Fatal(err)
	}

	// test request fail for existing user, existing repository with authorization and invalid patchable field value.
	url = fmt.Sprintf(settingsUrl, validUser, validRepo)
	req, err = http.NewRequest("PATCH", url, strings.NewReader(`{"public": "string"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/json")
	_, err = makeRequest(t, req, http.StatusBadRequest)
	if err != nil {
		t.Fatal(err)
	}

	// set up for valid PATCH tests
	const repoStartDesc = "description"
	const repoStartVisibility = false
	repoId := store.RepoId{Owner: validUser, Name: validRepo}
	repo, err := git.OpenRepository(server.repos.IdToPath(repoId))

	err = repo.WriteDescription(repoStartDesc)
	if err != nil {
		t.Fatalf("Error setting up repository description for test: %v\n", err)
	}
	err = server.repos.SetRepoVisibility(repoId, repoStartVisibility)
	if err != nil {
		t.Fatalf("Error setting up repository visibility for test: %v\n", err)
	}
	oldRepoDesc := repo.ReadDescription()

	// test request patch for existing user, existing repository with authorization and "public" only.
	const newPublic = true
	url = fmt.Sprintf(settingsUrl, validUser, validRepo)
	req, err = http.NewRequest("PATCH", url, strings.NewReader(fmt.Sprintf(`{"public": %t}`, newPublic)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/json")
	_, err = makeRequest(t, req, http.StatusOK)
	if err != nil {
		t.Fatal(err)
	}

	newRepoDesc := repo.ReadDescription()
	newRepoVis, _ := server.repos.GetRepoVisibility(repoId)
	if newRepoDesc != oldRepoDesc {
		t.Fatalf("Expected description to be unchanged %q, but was %q\n", oldRepoDesc, newRepoDesc)
	}
	if newRepoVis != newPublic {
		t.Fatal("Repository visibility was not updated.")
	}

	// test request patch for existing user, existing repository with authorization and "description" only.
	oldRepoVis, _ := server.repos.GetRepoVisibility(repoId)
	const newDesc = "a new description"

	url = fmt.Sprintf(settingsUrl, validUser, validRepo)
	req, err = http.NewRequest("PATCH", url, strings.NewReader(fmt.Sprintf(`{"description": %q}`, newDesc)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/json")
	_, err = makeRequest(t, req, http.StatusOK)
	if err != nil {
		t.Fatal(err)
	}

	newRepoDesc = repo.ReadDescription()
	newRepoVis, _ = server.repos.GetRepoVisibility(repoId)
	if newRepoDesc != newDesc {
		t.Fatalf("Expected description to be %q, but was %q\n", newDesc, newRepoDesc)
	}
	if newRepoVis != oldRepoVis {
		t.Fatalf("Expected visbility to be %t but was %t\n", oldRepoVis, newRepoVis)
	}

	// test request patch for existing user, existing repository with authorization and both "description" and "public".
	const anotherPublic = false
	const anotherDesc = "another new description"
	body := fmt.Sprintf(`{"description": %q, "public": %t}`, anotherDesc, anotherPublic)

	url = fmt.Sprintf(settingsUrl, validUser, validRepo)
	req, err = http.NewRequest("PATCH", url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/json")
	_, err = makeRequest(t, req, http.StatusOK)
	if err != nil {
		t.Fatal(err)
	}

	newRepoDesc = repo.ReadDescription()
	newRepoVis, _ = server.repos.GetRepoVisibility(repoId)
	if newRepoDesc != anotherDesc {
		t.Fatalf("Expected description to be %q, but was %q\n", anotherDesc, newRepoDesc)
	}
	if newRepoVis != anotherPublic {
		t.Fatalf("Expected visbility to be %t but was %t\n", anotherPublic, newRepoVis)
	}
}
