package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"syscall"

	"github.com/G-Node/gin-repo/auth"
	"github.com/G-Node/gin-repo/client"
	"github.com/G-Node/gin-repo/git"
)

func execGitCommand(name string, args ...string) int {
	// fmt.Fprintf(os.Stderr, "[D] ! %s, %#v", name, args)
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()

	if err == nil {
		return 0
	}

	if t, ok := err.(*exec.ExitError); ok {
		return t.Sys().(syscall.WaitStatus).ExitStatus()
	}

	return -1
}

func gitCommand(client *client.Client, args []string, push bool, uid string) int {

	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "ERROR: wrong arguments to %q", args[0])
		return -2
	}

	path, pok, err := client.RepoAccess(args[1], uid)

	if err != nil {
		fmt.Fprintf(os.Stderr, "[E] repo access error: %v\n", err)
		return -10
	} else if push && !pok {
		fmt.Fprintf(os.Stderr, "[E] repository is read only!\n")
		return -11
	}

	return execGitCommand(args[0], path)
}

func gitAnnex(client *client.Client, args []string, uid string) int {

	if len(args) < 3 {
		fmt.Fprintf(os.Stderr, "ERROR: wrong arguments to %q", args[0])
		return -2
	}

	path, pok, err := client.RepoAccess(args[2], uid)

	if err != nil {
		fmt.Fprintf(os.Stderr, "[E] repo access error: %v\n", err)
		return -10
	}

	_, err = git.OpenRepository(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Could not open repository.")
		return -15
	}

	args[2] = path

	// "If set, disallows running git-shell to handle unknown commands."
	os.Setenv("GIT_ANNEX_SHELL_LIMITED", "True")

	// "If set, git-annex-shell will refuse to run commands
	//  that do not operate on the specified directory."
	os.Setenv("GIT_ANNEX_SHELL_DIRECTORY", path)

	if !pok {
		os.Setenv("GIT_ANNEX_SHELL_READONLY", "True")
	}

	return execGitCommand(args[0], args[1:]...)
}

func readSecret() ([]byte, error) {
	home := ""

	u, err := user.Current()
	if err == nil {
		home = u.HomeDir
	}

	if home == "" {
		home = os.Getenv("HOME")
	}

	path := filepath.Join(home, "gin.secret")
	secret, err := ioutil.ReadFile(path)

	return secret, err
}

func makeServiceToken() (string, error) {

	secret, err := auth.ReadSharedSecret()

	if err != nil {
		return "", fmt.Errorf("could not load secret: %v", err)
	}

	return auth.MakeServiceToken(secret)
}
