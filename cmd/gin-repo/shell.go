package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/G-Node/gin-repo/client"
)

func execGitCommand(name string, args ...string) int {
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

	default:
		fmt.Fprintf(os.Stderr, "[E] unhandled command: %s\n", cmd)
		res = 23
	}

	os.Exit(res)

}
