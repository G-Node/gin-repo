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

func gitUploadPack(arg string, uid string) {

	client := client.NewClient("http://localhost:8888")
	path, err := client.RepoAccess(arg, uid, "push")

	if err != nil {
		fmt.Fprintf(os.Stderr, "[E] repo access error: %v", err)
		os.Exit(-10)
	}

	cmd := exec.Command("git-upload-pack", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()

	if err != nil {
		ee := err.(*exec.ExitError)
		os.Exit(ee.Sys().(syscall.WaitStatus).ExitStatus())
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

	//case "git-receive-pack":
	default:
		fmt.Fprintf(os.Stderr, "[E] unhandled command: %s", gitcmd)
		os.Exit(23)
	}
}
