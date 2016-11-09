package main

import (
	"net/http"
	"testing"

	"encoding/json"
	"github.com/G-Node/gin-repo/wire"

	"bytes"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"os/exec"
	"reflect"
	"strings"
)

func makeCommand(env []string, args []string, out *bytes.Buffer,
	command string, stin io.Reader) *exec.Cmd {
	cmd := exec.Command(command)
	cmd.Env = env
	cmd.Stderr = out
	cmd.Stdout = out
	cmd.Args = args
	cmd.Stdin = stin
	return cmd
}

func makeFakeHandler(testHook wire.GitHook) func(http.ResponseWriter,
	*http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		hook := wire.GitHook{}
		decoder := json.NewDecoder(r.Body)
		decoder.Decode(&hook)
		if reflect.DeepEqual(hook, testHook) {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}
}

// Invoke gin-githooks inside a new process wrapped with the exec package
// Environment, stin and sterr are modified as needed.
// The trick is in the test env variable. The function executes main when test
// is set, which is (hopefully) only set when we are in a "sandboxed"
// environment. This is guarantied to on unixoid environments only.
func TestHooks(t *testing.T) {
	if os.Getenv("githooktest") == "1" {
		main()
		return
	}

	//Prepare  hooks to compare them later with the hooks as derived
	// in the webserver
	wd, err := os.Getwd()
	if err != nil {
		t.Fail()
	}
	updateHook := wire.GitHook{"udpate", []string{
		"502f31add03e446fcadb7041ae45c2f6fe5456d5",
		"225bb2c99652dc9d8bacd51110fa95896ea494c2",
		"1b73b2a32efb1efd394a1866fa6858f318fa8e22"}, wd, nil}
	preRecHook := wire.GitHook{"pre-receive", nil, wd,
		[]wire.RefLine{{
			"502f31add03e446fcadb7041ae45c2f6fe5456d5",
			"225bb2c99652dc9d8bacd51110fa95896ea494c2",
			"1b73b2a32efb1efd394a1866fa6858f318fa8e22"}}}
	postRecHook := wire.GitHook{"post-receive", nil, wd,
		[]wire.RefLine{{
			"502f31add03e446fcadb7041ae45c2f6fe5456d5",
			"225bb2c99652dc9d8bacd51110fa95896ea494c2",
			"1b73b2a32efb1efd394a1866fa6858f318fa8e22"}}}

	ts := httptest.NewServer(http.HandlerFunc(makeFakeHandler(updateHook)))
	defer ts.Close()
	FakeEnv := append(os.Environ(), "githooktest=1")
	buf := bytes.Buffer{}

	t.Log("Test update Hook")
	cmd := makeCommand(append(FakeEnv, fmt.Sprintf("GIN_REPO_URL=%s", ts.URL)),
		append([]string{updateHook.Name}, updateHook.HookArgs...),
		&buf, os.Args[0], nil)
	err = cmd.Run()
	ts.Close()
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	t.Log("Update Hook [OK!]")

	t.Log("Test pre-receive Hook")
	ts = httptest.NewServer(http.HandlerFunc(makeFakeHandler(preRecHook)))
	cmd = makeCommand(append(FakeEnv, fmt.Sprintf("GIN_REPO_URL=%s", ts.URL)),
		[]string{preRecHook.Name}, &buf, os.Args[0],
		strings.NewReader("502f31add03e446fcadb7041ae45c2f6fe5456d5 "+
			"225bb2c99652dc9d8bacd51110fa95896ea494c2 "+
			"1b73b2a32efb1efd394a1866fa6858f318fa8e22"))
	err = cmd.Run()
	ts.Close()
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	t.Log("Pre-receive Hook [OK!]")

	t.Log("Test post-receive Hook")
	ts = httptest.NewServer(http.HandlerFunc(makeFakeHandler(postRecHook)))
	cmd = makeCommand(append(FakeEnv, fmt.Sprintf("GIN_REPO_URL=%s", ts.URL)),
		[]string{postRecHook.Name}, &buf, os.Args[0],
		strings.NewReader("502f31add03e446fcadb7041ae45c2f6fe5456d5 "+
			"225bb2c99652dc9d8bacd51110fa95896ea494c2 "+
			"1b73b2a32efb1efd394a1866fa6858f318fa8e22"))
	err = cmd.Run()
	ts.Close()
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	t.Log("Post-receive Hook [OK!]")

	t.Log("Test server Denies")
	//Fake servers side "Bad request" with unmatched hooks
	ts = httptest.NewServer(http.HandlerFunc(makeFakeHandler(preRecHook)))
	cmd = makeCommand(append(FakeEnv, fmt.Sprintf("GIN_REPO_URL=%s", ts.URL)),
		[]string{postRecHook.Name}, &buf, os.Args[0], nil)
	err = cmd.Run()
	ts.Close()
	if err == nil {
		t.Fail()
	}
	t.Log("Server Denies [OK]")
}
