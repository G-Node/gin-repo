package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"

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

func gitCommand(args []string, push bool, uid string) int {

	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "ERROR: wrong arguments to %q", args[0])
		return -2
	}

	client := client.NewClient("http://localhost:8888")
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

func gitAnnex(args []string, uid string) int {

	if len(args) < 3 {
		fmt.Fprintf(os.Stderr, "ERROR: wrong arguments to %q", args[0])
		return -2
	}

	client := client.NewClient("http://localhost:8888")
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

	res := 0
	switch cmd {
	case "git-upload-pack":
		fallthrough
	case "git-upload-archive":
		res = gitCommand(argv, false, uid)

	case "git-receive-pack":
		res = gitCommand(argv, true, uid)

	case "git-annex-shell":
		res = gitAnnex(argv, uid)

	default:
		fmt.Fprintf(os.Stderr, "[E] unhandled command: %s\n", cmd)
		res = 23
	}

	os.Exit(res)

}
