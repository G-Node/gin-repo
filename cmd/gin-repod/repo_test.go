package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/G-Node/gin-repo/git"
	"github.com/G-Node/gin-repo/store"
	"github.com/G-Node/gin-repo/wire"
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
	type collaborator struct {
		User        string            `json:"User"`
		AccessLevel store.AccessLevel `json:"AccessLevel"`
	}

	const method = "GET"
	const urlTemplate = "/users/%s/repos/%s/collaborators"

	const invalidUser = "iDoNotExist"
	const invalidRepo = "iDoNotExist"
	const validUser = "alice"
	const validRepoEmpty = "auth"
	const validRepoCollaborator = "openfmri"
	const validRepoCollaboratorUser = "bob"
	const validRepoCollaboratorLevel = "is-admin"

	// test request fail for invalid user.
	url := fmt.Sprintf(urlTemplate, invalidUser, invalidRepo)
	_, err := RunRequest(method, url, nil, nil, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for valid user, invalid repository.
	url = fmt.Sprintf(urlTemplate, invalidUser, invalidRepo)
	_, err = RunRequest(method, url, nil, nil, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test existing user, existing repository, repository w/o collaborators.
	var repoCollaborators []collaborator

	url = fmt.Sprintf(urlTemplate, validUser, validRepoEmpty)
	resp, err := RunRequest(method, url, nil, nil, http.StatusOK)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	err = json.Unmarshal(resp.Body.Bytes(), &repoCollaborators)
	if err != nil {
		t.Fatal(err)
	}
	if len(repoCollaborators) != 0 {
		t.Errorf("Expected empty list but got: %v\n", repoCollaborators)
	}

	// test existing user, existing repository, repository with collaborators.
	url = fmt.Sprintf(urlTemplate, validUser, validRepoCollaborator)
	resp, err = RunRequest(method, url, nil, nil, http.StatusOK)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	err = json.Unmarshal(resp.Body.Bytes(), &repoCollaborators)
	if err != nil {
		t.Fatal(err)
	}
	if len(repoCollaborators) != 1 {
		t.Errorf("Expected one collaborator but got: %v\n", repoCollaborators)
	}
	if repoCollaborators[0].User != validRepoCollaboratorUser {
		t.Errorf("Expected user %q but got %q\n", validRepoCollaboratorUser, repoCollaborators[0].User)
	}
	if repoCollaborators[0].AccessLevel.String() != validRepoCollaboratorLevel {
		t.Errorf("Expected access level %q but got %q\n",
			validRepoCollaboratorLevel, repoCollaborators[0].AccessLevel.String())
	}

	// TODO add tests for private repository and tests for non owner
}

func Test_putRepoCollaborator(t *testing.T) {
	const method = "PUT"
	const urlTemplate = "/users/%s/repos/%s/collaborators/%s"

	const invalidUser = "iDoNotExist"
	const invalidRepo = "iDoNotExist"
	const validUser = "alice"

	const validRepoCollaborator = "openfmri"
	const validRepoEmpty = "auth"

	const invalidPutUser = "iDoNotExist"
	const validPutUserAdditional = "gicmo"
	const validPutUser = "bob"
	const defaultAccess = "can-pull"
	const putAccess = "can-push"

	headerMap := make(map[string]string)

	token, err := server.users.TokenForUser(validUser)
	if err != nil {
		t.Fatalf("Could not make token for %q: %v, %v", validUser, token, err)
	}

	// test request fail for missing authorization header.
	url := fmt.Sprintf(urlTemplate, invalidUser, invalidRepo, invalidPutUser)
	_, err = RunRequest(method, url, nil, nil, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for missing bearer token prefix in authorization header.
	headerMap["Authorization"] = ""
	url = fmt.Sprintf(urlTemplate, invalidUser, invalidRepo, invalidPutUser)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for missing bearer token.
	headerMap["Authorization"] = "Bearer "
	url = fmt.Sprintf(urlTemplate, invalidUser, invalidRepo, validPutUser)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusForbidden)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for non existing user, with authorization.
	headerMap["Authorization"] = "Bearer " + token
	url = fmt.Sprintf(urlTemplate, invalidUser, invalidRepo, validPutUser)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for existing user, non existing repository, with authorization.
	headerMap["Authorization"] = "Bearer " + token
	url = fmt.Sprintf(urlTemplate, validUser, invalidRepo, validPutUser)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// TODO rest request fail for non admin-access user

	// TODO test request fail for existing user, existing repository, invalid collaborator.

	// test request fail for existing user, existing repository, own user as collaborator.
	headerMap["Authorization"] = "Bearer " + token
	url = fmt.Sprintf(urlTemplate, validUser, validRepoCollaborator, validUser)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusBadRequest)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for existing user, existing repository, new collaborator, missing body.
	headerMap["Authorization"] = "Bearer " + token
	headerMap["Content-Type"] = "application/json"
	url = fmt.Sprintf(urlTemplate, validUser, validRepoCollaborator, validPutUserAdditional)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusBadRequest)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for existing user, existing repository, new collaborator, missing body content.
	headerMap["Authorization"] = "Bearer " + token
	headerMap["Content-Type"] = "application/json"
	url = fmt.Sprintf(urlTemplate, validUser, validRepoCollaborator, validPutUserAdditional)
	_, err = RunRequest(method, url, strings.NewReader(""), headerMap, http.StatusBadRequest)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for existing user, existing repository, new collaborator, invalid body: permission value.
	headerMap["Authorization"] = "Bearer " + token
	headerMap["Content-Type"] = "application/json"
	body := fmt.Sprint(`{"permission": "invalid value"}`)
	url = fmt.Sprintf(urlTemplate, validUser, validRepoCollaborator, validPutUserAdditional)
	_, err = RunRequest(method, url, strings.NewReader(body), headerMap, http.StatusBadRequest)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test valid request for existing user, existing repository, adding existing collaborator to non empty shared folder.
	repoId := store.RepoId{Owner: validUser, Name: validRepoCollaborator}
	// TODO maybe there is a better way to ensure a clean test environment for other tests
	defer os.Remove(filepath.Join(server.repos.IdToPath(repoId), "gin", "sharing", validPutUserAdditional))

	headerMap["Authorization"] = "Bearer " + token
	headerMap["Content-Type"] = "application/json"
	url = fmt.Sprintf(urlTemplate, validUser, validRepoCollaborator, validPutUserAdditional)
	body = fmt.Sprintf(`{"permission": %q}`, defaultAccess)
	_, err = RunRequest(method, url, strings.NewReader(body), headerMap, http.StatusOK)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	level, err := server.repos.GetAccessLevel(repoId, validPutUserAdditional)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	if level.String() != defaultAccess {
		t.Fatalf("Expected user to be present with access level %q but got %q.\n", defaultAccess, level.String())
	}

	// test valid request for existing user, existing repository, adding existing collaborator to empty shared folder.
	repoId = store.RepoId{Owner: validUser, Name: validRepoEmpty}
	// TODO maybe there is a better way to ensure a clean test environment for other tests
	defer os.Remove(filepath.Join(server.repos.IdToPath(repoId), "gin", "sharing", validPutUser))

	headerMap["Authorization"] = "Bearer " + token
	headerMap["Content-Type"] = "application/json"
	url = fmt.Sprintf(urlTemplate, validUser, validRepoEmpty, validPutUser)
	body = fmt.Sprintf(`{"permission": %q}`, defaultAccess)
	_, err = RunRequest(method, url, strings.NewReader(body), headerMap, http.StatusOK)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	level, err = server.repos.GetAccessLevel(repoId, validPutUser)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	if level.String() != defaultAccess {
		t.Fatalf("Expected user to be present with access level %q but got %q.\n", defaultAccess, level.String())
	}

	// test valid request for existing user, existing repository, update existing collaborator access level.
	repoId = store.RepoId{Owner: validUser, Name: validRepoEmpty}

	headerMap["Authorization"] = "Bearer " + token
	headerMap["Content-Type"] = "application/json"
	url = fmt.Sprintf(urlTemplate, validUser, validRepoEmpty, validPutUser)
	body = fmt.Sprintf(`{"permission": %q}`, putAccess)
	_, err = RunRequest(method, url, strings.NewReader(body), headerMap, http.StatusOK)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	level, err = server.repos.GetAccessLevel(repoId, validPutUser)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	if level.String() != putAccess {
		t.Fatalf("Expected user to be present with access level %q but got %q.\n", putAccess, level.String())
	}
}

func Test_deleteRepoCollaborator(t *testing.T) {
	const method = "DELETE"
	const urlTemplate = "/users/%s/repos/%s/collaborators/%s"

	const invalidUser = "iDoNotExist"
	const invalidRepo = "iDoNotExist"
	const validUser = "alice"

	const validRepoCollaborator = "openfmri"
	const validRepoEmpty = "auth"

	const invalidDeleteUser = "iDoNotExist"
	const validDeleteUser = "bob"

	headerMap := make(map[string]string)

	token, err := server.users.TokenForUser(validUser)
	if err != nil {
		t.Fatalf("Could not make token for %q: %v, %v", validUser, token, err)
	}

	// test request fail for missing authorization header.
	url := fmt.Sprintf(urlTemplate, invalidUser, invalidRepo, invalidDeleteUser)
	_, err = RunRequest(method, url, nil, nil, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for missing bearer token header in authorization header.
	headerMap["Authorization"] = ""
	url = fmt.Sprintf(urlTemplate, invalidUser, invalidRepo, invalidDeleteUser)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for missing bearer token in authorization header.
	headerMap["Authorization"] = "Bearer "
	url = fmt.Sprintf(urlTemplate, invalidUser, invalidRepo, invalidDeleteUser)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusForbidden)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for non existing user, with authorization.
	headerMap["Authorization"] = "Bearer " + token
	url = fmt.Sprintf(urlTemplate, invalidUser, invalidRepo, invalidDeleteUser)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for existing user, non existing repository, with authorization.
	headerMap["Authorization"] = "Bearer " + token
	url = fmt.Sprintf(urlTemplate, validUser, invalidRepo, invalidDeleteUser)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for existing user, existing repository, with authorization, empty sharing folder.
	headerMap["Authorization"] = "Bearer " + token
	url = fmt.Sprintf(urlTemplate, validUser, validRepoEmpty, invalidDeleteUser)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusConflict)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for existing user, existing repository, with authorization, invalid delete username.
	headerMap["Authorization"] = "Bearer " + token
	url = fmt.Sprintf(urlTemplate, validUser, validRepoCollaborator, invalidDeleteUser)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusConflict)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test valid request for existing user, existing repository, with authorization, valid delete username.
	repoId := store.RepoId{Owner: validUser, Name: validRepoCollaborator}
	oldShared, err := server.repos.ListSharedAccess(repoId)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	headerMap["Authorization"] = "Bearer " + token
	url = fmt.Sprintf(urlTemplate, validUser, validRepoCollaborator, validDeleteUser)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusOK)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	// TODO maybe there is a better way to ensure a clean test environment for other tests
	defer func() {
		filePath := filepath.Join(server.repos.IdToPath(repoId), "gin", "sharing", validDeleteUser)
		err := ioutil.WriteFile(filePath, []byte("is-admin"), 0664)
		if err != nil {
			fmt.Printf("Test cleanup error for repo %q : %v\n", filePath, err)
		}
	}()

	newShared, err := server.repos.ListSharedAccess(repoId)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	if len(newShared) != len(oldShared)-1 {
		t.Fatalf("Expected %d collaborators but got %d, list of collaborators: %v\n",
			len(newShared), len(oldShared), newShared)
	}
	_, exists := newShared[validDeleteUser]
	if exists {
		t.Fatalf("Collaborator %q was not deleted from shared folder of repo %q\n",
			validDeleteUser, validRepoCollaborator)
	}

	// TODO test fail due to not repo owner and insufficient repo access level

	// TODO test correct delete when not repo owner but with admin-rights
}

func Test_repoDescription(t *testing.T) {
	const method = "GET"
	const urlTemplate = "/users/%s/repos/%s"

	const invalidUser = "iDoNotExist"
	const invalidRepo = "iDoNotExist"
	const validUser = "alice"
	const validRepo = "auth"
	const validRepoShared = "openfmri"

	const validUserPrivate = "bob"
	const validRepoPrivate = "auth"

	// test request fail for invalid user.
	url := fmt.Sprintf(urlTemplate, invalidUser, invalidRepo)
	_, err := RunRequest(method, url, nil, nil, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for valid user, invalid repository.
	url = fmt.Sprintf(urlTemplate, invalidUser, invalidRepo)
	_, err = RunRequest(method, url, nil, nil, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test valid user, valid repository
	url = fmt.Sprintf(urlTemplate, validUser, validRepo)
	resp, err := RunRequest(method, url, nil, nil, http.StatusOK)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	result := wire.Repo{}
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	if result.Owner != validUser {
		t.Fatalf("Expected owner %q but got %q\n", validUser, result.Owner)
	}
	if result.Name != validRepo {
		t.Fatalf("Expected repository %q but got %q\n", validRepo, result.Name)
	}
	if result.Description == "" {
		t.Fatal("Received empty repository description")
	}
	vis, err := server.repos.GetRepoVisibility(store.RepoId{Owner: validUser, Name: validRepo})
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	if result.Public != vis {
		t.Fatalf("Expected visibility %t but got %t\n", vis, result.Public)
	}
	// Test non shared repository
	if result.Shared {
		t.Fatalf("Expected shared %t but got %t\n", false, result.Shared)
	}

	// test valid user, valid shared repository
	url = fmt.Sprintf(urlTemplate, validUser, validRepoShared)
	resp, err = RunRequest(method, url, nil, nil, http.StatusOK)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	result = wire.Repo{}
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	if !result.Shared {
		t.Fatalf("Expected shared %t but got %t\n", true, result.Shared)
	}

	// test valid user, valid private repository
	headerMap := make(map[string]string)
	token, err := server.users.TokenForUser(validUserPrivate)
	if err != nil {
		t.Fatalf("Could not make token for %q: %v, %v", validUserPrivate, token, err)
	}
	headerMap["Authorization"] = "Bearer " + token

	url = fmt.Sprintf(urlTemplate, validUserPrivate, validRepoPrivate)
	resp, err = RunRequest(method, url, nil, headerMap, http.StatusOK)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	result = wire.Repo{}
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	if result.Public {
		t.Fatalf("Expected public %t but got %t\n", false, result.Public)
	}
}

func Test_listRepoCommits(t *testing.T) {
	const method = "GET"
	const urlTemplate = "/users/%s/repos/%s/commits/%s"

	const invalidBranch = "iDoNotExist"
	const validBranch = "master"
	const validUser = "bob"
	const validRepo = "repod"

	headerMap := make(map[string]string)
	token, err := server.users.TokenForUser(validUser)
	if err != nil {
		t.Fatalf("Could not make token for %q: %v, %v", validUser, token, err)
	}
	headerMap["Authorization"] = "Bearer " + token

	// test request fail for insufficient access.
	url := fmt.Sprintf(urlTemplate, validUser, validRepo, validBranch)
	_, err = RunRequest(method, url, nil, nil, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test request fail for valid user, valid repository, invalid branch.
	url = fmt.Sprintf(urlTemplate, validUser, validRepo, invalidBranch)
	_, err = RunRequest(method, url, nil, headerMap, http.StatusNotFound)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test valid user, valid repository, valid branch
	url = fmt.Sprintf(urlTemplate, validUser, validRepo, validBranch)
	resp, err := RunRequest(method, url, nil, headerMap, http.StatusOK)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	// test content of resulting list of commits
	result := []wire.CommitSummary{}
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	if len(result) == 0 {
		t.Fatal("Expected a list of commits, but got none")
	}
}
