package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/G-Node/gin-repo/internal/testbed"
	"github.com/G-Node/gin-repo/store"
)

var server *Server

func TestMain(m *testing.M) {
	flag.Parse()

	repoDir, err := testbed.MkData("/tmp")

	if err != nil {
		fmt.Fprintf(os.Stderr, "could not make test data: %v\n", err)
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get cwd: %v", err)
		os.Exit(1)
	}

	err = os.Chdir(repoDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not set cwd to repoDir: %v", err)
		os.Exit(1)
	}

	server = NewServer("127.0.0.1:0")
	server.SetupServiceSecret()
	server.SetupRoutes()
	server.SetupStores()

	res := m.Run()

	_ = os.Chdir(cwd)
	os.Exit(res)
}

func NewGet(t *testing.T, url string, user string) *http.Request {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatal(err)
	}

	token := ""

	if user != "" {
		token, err = server.users.TokenForUser(user)
		if err != nil {
			t.Fatalf("could not make token for %q: %v, %v", user, token, err)
		}
	}

	if token != "" {
		req.Header.Add("Authorization", "Bearer "+token)
	}

	return req
}

func makeRequest(req *http.Request, code int) (*httptest.ResponseRecorder, error) {
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if status := rr.Code; status != code {
		return rr, fmt.Errorf("wrong status code: got %v want %v", status, code)
	}

	return rr, nil
}

func RunRequest(method string, url string, body io.Reader,
	header map[string]string, code int) (*httptest.ResponseRecorder, error) {

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if header != nil && len(header) > 0 {
		for k, v := range header {
			req.Header.Add(k, v)
		}
	}

	resp, err := makeRequest(req, code)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func TestBranchAccess(t *testing.T) {
	req := NewGet(t, "/users/alice/repos/exrepo/branches/master", "")
	_, err := makeRequest(req, http.StatusNotFound)
	if err != nil {
		t.Fatal(err)
	}

	req = NewGet(t, "/users/alice/repos/exrepo/branches/master", "alice")
	_, err = makeRequest(req, http.StatusOK)
	if err != nil {
		t.Fatal(err)
	}
}

func TestObjectAccess(t *testing.T) {
	//first find the commit id
	repo, err := server.repos.OpenGitRepo(store.RepoId{Name: "exrepo", Owner: "alice"})
	if err != nil {
		t.Fatal(err)
	}

	ref, err := repo.OpenRef("master")
	if err != nil {
		t.Fatal(err)
	}

	id, err := ref.Resolve()
	if err != nil {
		t.Fatal(err)
	}

	//now make some requests
	url := fmt.Sprintf("/users/alice/repos/exrepo/objects/%s", id)
	req := NewGet(t, url, "")
	_, err = makeRequest(req, http.StatusNotFound)
	if err != nil {
		t.Fatal(err)
	}

	req = NewGet(t, url, "alice")
	_, err = makeRequest(req, http.StatusOK)
	if err != nil {
		t.Fatal(err)
	}
}
