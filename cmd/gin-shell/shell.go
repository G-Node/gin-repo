package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"syscall"
	"time"

	"github.com/G-Node/gin-repo/client"
	"github.com/G-Node/gin-repo/git"
	"github.com/dgrijalva/jwt-go"
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

	repo, err := git.OpenRepository(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Could not open repository.")
		return -15
	}

	if pok && !repo.HasAnnex() {
		err = repo.InitAnnex()
		if err != nil {
			//TODO: don't print the internal error, might give away the actual path
			fmt.Fprintf(os.Stderr, "ERROR: initialization of git-annex failed: %v", err)
			return -14
		}
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

	token := jwt.New(jwt.SigningMethodHS256)

	host, err := os.Hostname()
	if err != nil {
		host = "localhost"
	}

	token.Claims["iss"] = "gin-repo@" + host
	token.Claims["iat"] = time.Now().Unix()
	token.Claims["exp"] = time.Now().Add(time.Minute * 5).Unix()

	token.Claims["role"] = "service"

	secret, err := readSecret()

	if err != nil {
		return "", fmt.Errorf("Could not load secret: %v", err)
	}

	str, err := token.SignedString(secret)
	return str, err
}

func cmdShell(args map[string]interface{}) {
	log.SetOutput(os.Stderr)

	argv := splitarg(os.Getenv("SSH_ORIGINAL_COMMAND"))
	cmd := head(argv)

	if cmd == "" {
		fmt.Fprintf(os.Stderr, "ERROR: No shell access allowed.")
		return
	}

	if _, ok := args["<uid>"]; !ok {
		log.Fatal("[E] :( (no user)")
	}

	uid := args["<uid>"].(string)
	fmt.Fprintf(os.Stderr, "uid: %s\n", uid)
	fmt.Fprintf(os.Stderr, "cmd: %s %v\n", cmd, argv[1:])

	client := client.NewClient("http://localhost:8888")

	if token, err := makeServiceToken(); err == nil {
		client.AuthToken = token
	}

	res := 0
	switch cmd {
	case "git-upload-pack":
		fallthrough
	case "git-upload-archive":
		res = gitCommand(client, argv, false, uid)

	case "git-receive-pack":
		res = gitCommand(client, argv, true, uid)

	case "git-annex-shell":
		res = gitAnnex(client, argv, uid)

	default:
		fmt.Fprintf(os.Stderr, "[E] unhandled command: %s\n", cmd)
		res = 23
	}

	os.Exit(res)

}
