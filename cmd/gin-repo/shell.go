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

func gitUploadPack(arg string, uid string) {

	client := client.NewClient("http://localhost:8888")
	path, err := client.RepoAccess(arg, uid, "pull")

	if err != nil {
		fmt.Fprintf(os.Stderr, "[E] repo access error: %v", err)
		os.Exit(-10)
	}

	res := execGitCommand("git-upload-pack", path)

	if res != 0 {
		os.Exit(res)
	}
}

func gitReceivePack(arg string, uid string) {

	client := client.NewClient("http://localhost:8888")
	path, err := client.RepoAccess(arg, uid, "push")

	if err != nil {
		fmt.Fprintf(os.Stderr, "[E] repo access error: %v", err)
		os.Exit(-10)
	}

	res := execGitCommand("git-receive-pack", path)

	if res != 0 {
		os.Exit(res)
	}
}

func splitarg(arg string, out ...*string) bool {
	comps := strings.Split(arg, " ")

	if len(comps) != len(out) {
		return false
	}

	for i, str := range comps {
		*out[i] = str
	}

	return true
}

func cmdShell(args map[string]interface{}) {
	log.SetOutput(os.Stderr)

	var gitcmd, gitarg string
	if ok := splitarg(os.Getenv("SSH_ORIGINAL_COMMAND"), &gitcmd, &gitarg); !ok {
		log.Fatal("[E] :( (no shell access allowed)")
	}

	if _, ok := args["<uid>"]; !ok {
		log.Fatal("[E] :( (no user)")
	}

	uid := args["<uid>"].(string)
	fmt.Fprintf(os.Stderr, "uid: %s\n", uid)
	fmt.Fprintf(os.Stderr, "git: %s [%s]\n", gitcmd, gitarg)

	switch gitcmd {
	case "git-upload-pack":
		gitUploadPack(gitarg, uid)

	case "git-receive-pack":
		gitReceivePack(gitarg, uid)

	default:
		fmt.Fprintf(os.Stderr, "[E] unhandled command: %s", gitcmd)
		os.Exit(23)
	}
}
