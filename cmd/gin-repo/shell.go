package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/G-Node/gin-repo/client"
)

func execGitCommand(program string, path string) int {
	cmd := exec.Command(program, path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()

	status := 0
	if err != nil {
		ee := err.(*exec.ExitError)
		status = ee.Sys().(syscall.WaitStatus).ExitStatus()
	}

	return status
}

func gitUploadPack(arg string, uid string) int {

	client := client.NewClient("http://localhost:8888")
	path, _, err := client.RepoAccess(arg, uid)

	if err != nil {
		fmt.Fprintf(os.Stderr, "[E] repo access error: %v\n", err)
		return -10
	}

	return execGitCommand("git-upload-pack", path)
}

func gitUploadArchive(arg string, uid string) int {

	client := client.NewClient("http://localhost:8888")
	path, _, err := client.RepoAccess(arg, uid)

	if err != nil {
		fmt.Fprintf(os.Stderr, "[E] repo access error: %v\n", err)
		return -10
	}

	return execGitCommand("git-upload-archive", path)
}

func gitReceivePack(arg string, uid string) int {

	client := client.NewClient("http://localhost:8888")
	path, push, err := client.RepoAccess(arg, uid)

	if err != nil {
		fmt.Fprintf(os.Stderr, "[E] repo access error: %v\n", err)
		return -10
	} else if !push {
		fmt.Fprintf(os.Stderr, "[E] repository is read only!\n")
		return -11
	}

	return execGitCommand("git-receive-pack", path)
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
		res = gitUploadPack(strings.Join(argv[1:], " "), uid)

	case "git-upload-archive":
		res = gitUploadArchive(strings.Join(argv[1:], " "), uid)

	case "git-receive-pack":
		res = gitReceivePack(strings.Join(argv[1:], " "), uid)

	default:
		fmt.Fprintf(os.Stderr, "[E] unhandled command: %s\n", cmd)
		res = 23
	}

	os.Exit(res)

}
