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

func TestBranchAccess(t *testing.T) {
	req, err := http.NewRequest("GET", "/users/alice/repos/exrepo/branches/master", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Fatalf("wrong status code: got %v want %v", status, http.StatusNotFound)
	}

	token, err := server.users.TokenForUser("alice")
	if err != nil {
		t.Fatalf("could not make token for alice: %v", token)
	}

	req, err = http.NewRequest("GET", "/users/alice/repos/exrepo/branches/master", nil)
	req.Header.Add("Authorization", "Bearer "+token)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("wrong status code: got %v want %v", status, http.StatusOK)
	}
}
