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
	const method = "GET"
	const urlTemplate = "/users/%s/repos/%s/visibility"

	const invalidUser = "iDoNotExist"
	const invalidRepo = "iDoNotExist"
	const validUser = "alice"
	const validPublicRepo = "auth"
	const validPrivateRepo = "exrepo"

	headerMap := make(map[string]string)

	token, err := server.users.TokenForUser(validUser)
	if err != nil {
		t.Fatalf("Could not make token for %q: %v, %v", validUser, token, err)
	}

	// test request fail for missing authorization header
	url := fmt.Sprintf(urlTemplate, invalidUser, invalidRepo)
	_, err = RunRequest(method, url, nil, nil, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for invalid authorization header
	headerMap["Authorization"] = ""
	url = fmt.Sprintf(urlTemplate, invalidUser, invalidRepo)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for non existing user w/o authorization
	headerMap["Authorization"] = "Bearer "
	url = fmt.Sprintf(urlTemplate, invalidUser, invalidRepo)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusForbidden)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for non existing user, with authorization
	headerMap["Authorization"] = "Bearer " + token
	url = fmt.Sprintf(urlTemplate, validUser, invalidRepo)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for existing user, non existing repository with authorization
	headerMap["Authorization"] = "Bearer " + token
	url = fmt.Sprintf(urlTemplate, validUser, invalidRepo)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request for existing user, existing public repository with authorization
	headerMap["Authorization"] = "Bearer " + token
	url = fmt.Sprintf(urlTemplate, validUser, validPublicRepo)
	resp, err := RunRequest(method, url, nil, headerMap, http.StatusOK)
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
	headerMap["Authorization"] = "Bearer " + token
	url = fmt.Sprintf(urlTemplate, validUser, validPrivateRepo)
	resp, err = RunRequest(method, url, nil, headerMap, http.StatusOK)
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
	const method = "PUT"
	const urlTemplate = "/users/%s/repos/%s/visibility"

	const invalidUser = "iDoNotExist"
	const invalidRepo = "iDoNotExist"
	const validUser = "alice"
	const validPublicRepo = "auth"

	headerMap := make(map[string]string)

	token, err := server.users.TokenForUser(validUser)
	if err != nil {
		t.Fatalf("Could not make token for %q: %v, %v", validUser, token, err)
	}

	// Currently the test repositories are set up only once.
	// This means tests changing repository settings are not independent,
	// if repository settings are changed, they have to be reset after the test.
	// TODO organize the repo setup for tests so that they are reset for every test.
	defer server.repos.SetRepoVisibility(store.RepoId{Owner: validUser, Name: validPublicRepo}, true)

	// test request fail for missing authorization header
	url := fmt.Sprintf(urlTemplate, invalidUser, invalidRepo)
	_, err = RunRequest(method, url, nil, nil, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for invalid authorization header
	headerMap["Authorization"] = ""
	url = fmt.Sprintf(urlTemplate, invalidUser, invalidRepo)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for non existing user w/o authorization
	headerMap["Authorization"] = "Bearer "
	url = fmt.Sprintf(urlTemplate, invalidUser, invalidRepo)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusForbidden)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for non existing user, with authorization
	headerMap["Authorization"] = "Bearer " + token
	url = fmt.Sprintf(urlTemplate, invalidUser, invalidRepo)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for existing user, non existing repository, with authorization
	headerMap["Authorization"] = "Bearer " + token
	url = fmt.Sprintf(urlTemplate, validUser, invalidRepo)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for existing user, existing repository, with authorization, missing body
	headerMap["Authorization"] = "Bearer " + token
	url = fmt.Sprintf(urlTemplate, validUser, validPublicRepo)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusBadRequest)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for existing user, existing repository with authorization and empty body
	headerMap["Authorization"] = "Bearer " + token
	headerMap["Content-Type"] = "application/json"
	url = fmt.Sprintf(urlTemplate, validUser, validPublicRepo)
	_, err = RunRequest(method, url, strings.NewReader(""), headerMap, http.StatusBadRequest)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for existing user, existing repository with authorization and invalid request body
	headerMap["Authorization"] = "Bearer " + token
	headerMap["Content-Type"] = "application/json"
	url = fmt.Sprintf(urlTemplate, validUser, validPublicRepo)
	_, err = RunRequest(method, url, strings.NewReader("{}"), headerMap, http.StatusBadRequest)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for existing user, existing repository with authorization and invalid request field type
	headerMap["Authorization"] = "Bearer " + token
	headerMap["Content-Type"] = "application/json"
	url = fmt.Sprintf(urlTemplate, validUser, validPublicRepo)
	_, err = RunRequest(method, url, strings.NewReader(`{"public":"True"}`), headerMap, http.StatusBadRequest)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// set up test for valid put request
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

	// test correct request, set public from true to false
	headerMap["Authorization"] = "Bearer " + token
	headerMap["Content-Type"] = "application/json"
	url = fmt.Sprintf(urlTemplate, validUser, validPublicRepo)
	resp, err := RunRequest(method, url, strings.NewReader(`{"public":false}`), headerMap, http.StatusOK)
	if err != nil {
		t.Fatalf("%v\n", err)
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
}

func Test_patchRepoSettings(t *testing.T) {
	const method = "PATCH"
	const urlTemplate = "/users/%s/repos/%s/settings"

	const invalidUser = "iDoNotExist"
	const invalidRepo = "iDoNotExist"
	const validUser = "alice"
	const validRepo = "auth"

	headerMap := make(map[string]string)

	token, err := server.users.TokenForUser(validUser)
	if err != nil {
		t.Fatalf("Could not make token for %q: %v, %v", validUser, token, err)
	}

	// Reset repository setting after the test
	defer server.repos.SetRepoVisibility(store.RepoId{Owner: validUser, Name: validRepo}, true)

	// TODO the following section is actually just a copy of the first part
	// TODO of the get/setRepoVisibility tests. Good enough for now, but unify at some point.

	// test request fail for missing authorization header
	url := fmt.Sprintf(urlTemplate, invalidUser, invalidRepo)
	_, err = RunRequest(method, url, nil, nil, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for missing bearer token in authorization header
	headerMap["Authorization"] = ""
	url = fmt.Sprintf(urlTemplate, invalidUser, invalidRepo)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for non existing user with invalid authorization.
	headerMap["Authorization"] = "Bearer "
	url = fmt.Sprintf(urlTemplate, invalidUser, invalidRepo)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusForbidden)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for non existing user, with authorization.
	headerMap["Authorization"] = "Bearer " + token
	url = fmt.Sprintf(urlTemplate, invalidUser, invalidRepo)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for existing user, non existing repository, with authorization.
	headerMap["Authorization"] = "Bearer " + token
	url = fmt.Sprintf(urlTemplate, validUser, invalidRepo)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for existing user, existing repository, with authorization, missing body.
	headerMap["Authorization"] = "Bearer " + token
	url = fmt.Sprintf(urlTemplate, validUser, validRepo)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusBadRequest)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for existing user, existing repository with authorization and empty body.
	headerMap["Authorization"] = "Bearer " + token
	headerMap["Content-Type"] = "application/json"
	url = fmt.Sprintf(urlTemplate, validUser, validRepo)
	_, err = RunRequest(method, url, strings.NewReader(""), headerMap, http.StatusBadRequest)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for existing user, existing repository with authorization and missing patchable field.
	headerMap["Authorization"] = "Bearer " + token
	headerMap["Content-Type"] = "application/json"
	url = fmt.Sprintf(urlTemplate, validUser, validRepo)
	body := `{"otherfield":"has nowhere to patch"}`
	_, err = RunRequest(method, url, strings.NewReader(body), headerMap, http.StatusBadRequest)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for existing user, existing repository with authorization and invalid patchable field value.
	headerMap["Authorization"] = "Bearer " + token
	headerMap["Content-Type"] = "application/json"
	url = fmt.Sprintf(urlTemplate, validUser, validRepo)
	body = `{"public": "string"}`
	_, err = RunRequest(method, url, strings.NewReader(body), headerMap, http.StatusBadRequest)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// set up for valid PATCH tests
	const repoStartDesc = "description"
	const repoStartVisibility = false
	repoId := store.RepoId{Owner: validUser, Name: validRepo}
	repo, _ := git.OpenRepository(server.repos.IdToPath(repoId))
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

	headerMap["Authorization"] = "Bearer " + token
	headerMap["Content-Type"] = "application/json"
	url = fmt.Sprintf(urlTemplate, validUser, validRepo)
	body = fmt.Sprintf(`{"public": %t}`, newPublic)
	_, err = RunRequest(method, url, strings.NewReader(body), headerMap, http.StatusOK)
	if err != nil {
		t.Fatalf("%v\n", err)
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

	headerMap["Authorization"] = "Bearer " + token
	headerMap["Content-Type"] = "application/json"
	url = fmt.Sprintf(urlTemplate, validUser, validRepo)
	body = fmt.Sprintf(`{"description": %q}`, newDesc)
	_, err = RunRequest(method, url, strings.NewReader(body), headerMap, http.StatusOK)
	if err != nil {
		t.Fatalf("%v\n", err)
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

	headerMap["Authorization"] = "Bearer " + token
	headerMap["Content-Type"] = "application/json"
	url = fmt.Sprintf(urlTemplate, validUser, validRepo)
	body = fmt.Sprintf(`{"description": %q, "public": %t}`, anotherDesc, anotherPublic)
	_, err = RunRequest(method, url, strings.NewReader(body), headerMap, http.StatusOK)
	if err != nil {
		t.Fatalf("%v\n", err)
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

func Test_listRepoCollaborators(t *testing.T) {
	const method = "GET"
	const urlTemplate = "/users/%s/repos/%s/collaborators"

	const invalidUser = "iDoNotExist"
	const invalidRepo = "iDoNotExist"
	const validUser = "alice"
	const validRepoEmpty = "auth"
	const validRepoCollaborator = "openfmri"

	// test request fail for non existing user.
	url := fmt.Sprintf(urlTemplate, invalidUser, invalidRepo)
	_, err := RunRequest(method, url, nil, nil, http.StatusBadRequest)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for existing user, non existing repository.
	url = fmt.Sprintf(urlTemplate, validUser, invalidRepo)
	_, err = RunRequest(method, url, nil, nil, http.StatusBadRequest)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	var collaborators []string

	// test existing user, existing repository, repository w/o collaborators.
	url = fmt.Sprintf(urlTemplate, validUser, validRepoEmpty)
	resp, err := RunRequest(method, url, nil, nil, http.StatusOK)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	err = json.Unmarshal(resp.Body.Bytes(), &collaborators)
	if err != nil {
		t.Fatal(err)
	}
	if len(collaborators) != 0 {
		t.Errorf("Expected empty list but got: %v\n", collaborators)
	}

	// test existing user, existing repository, repository with collaborators.
	url = fmt.Sprintf(urlTemplate, validUser, validRepoCollaborator)
	resp, err = RunRequest(method, url, nil, nil, http.StatusOK)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	err = json.Unmarshal(resp.Body.Bytes(), &collaborators)
	if err != nil {
		t.Fatal(err)
	}
	if len(collaborators) != 1 {
		t.Errorf("Expected one collaborator but got: %v\n", collaborators)
	}
}
