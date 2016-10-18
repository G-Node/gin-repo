package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/G-Node/gin-repo/internal/testbed"
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
			t.Fatalf("could not make token for alice: %v", token)
		}
	}

	if token != "" {
		req.Header.Add("Authorization", "Bearer "+token)
	}

	return req
}

func makeRequest(t *testing.T, req *http.Request, code int) (*httptest.ResponseRecorder, error) {
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if status := rr.Code; status != code {
		return rr, fmt.Errorf("wrong status code: got %v want %v", status, code)
	}

	return rr, nil
}

func TestBranchAccess(t *testing.T) {
	req := NewGet(t, "/users/alice/repos/exrepo/branches/master", "")
	_, err := makeRequest(t, req, http.StatusNotFound)
	if err != nil {
		t.Fatal(err)
	}

	req = NewGet(t, "/users/alice/repos/exrepo/branches/master", "alice")
	_, err = makeRequest(t, req, http.StatusOK)
	if err != nil {
		t.Fatal(err)
	}
}

func TestObjectAccess(t *testing.T) {
	// b1318dfe1d7926146f6d8ccf4b52bd7ab3b66431 is the first commit of exrepo
	url := "/users/alice/repos/exrepo/objects/b1318dfe1d7926146f6d8ccf4b52bd7ab3b66431"
	req := NewGet(t, url, "")
	_, err := makeRequest(t, req, http.StatusNotFound)
	if err != nil {
		t.Fatal(err)
	}

	req = NewGet(t, url, "alice")
	_, err = makeRequest(t, req, http.StatusOK)
	if err != nil {
		t.Fatal(err)
	}
}
