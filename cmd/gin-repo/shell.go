package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func translatePath(vpath string, uid string) string {
	dir := os.Getenv("GIN_REPO_DIR")

	if dir == "" {
		dir = "."
	}

	if strings.HasPrefix(vpath, "'") && strings.HasSuffix(vpath, "'") {
		vpath = vpath[1 : len(vpath)-1]
	}

	path := filepath.Join(dir, uid, vpath)

	if !strings.HasSuffix(path, ".git") {
		path += ".git"
	}

	fmt.Fprintf(os.Stderr, "[D] tp: %s@%s -> %s\n", uid, vpath, path)

	return path
}

func gitUploadPack(arg string, uid string) {

	path := translatePath(arg, uid)

	cmd := exec.Command("git-upload-pack", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	if err != nil {
		ee := err.(*exec.ExitError)
		fmt.Fprintf(os.Stderr, "[E] %v", err)
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
	var gitcmd, gitarg string

	if ok := splitarg(os.Getenv("SSH_ORIGINAL_COMMAND"), &gitcmd, &gitarg); !ok {
		log.Fatal("[E] :( (wrong ssh orignal command)")
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
		log.Fatalf("[E] unhandled command: %s", gitcmd)
	}
}
