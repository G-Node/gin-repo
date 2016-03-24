package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/G-Node/gin-repo/ssh"
)

func cmdKeysList(args map[string]interface{}, keys map[string]ssh.Key) {
	if fingerprint := args["--fingerprint"]; fingerprint != nil {
		if key, ok := keys[fingerprint.(string)]; ok {

			uid := key.Comment

			fmt.Printf("%s: %s [%s]\n", uid, key.Keysize, key.Fingerprint)
			fmt.Printf("%s: [%s]\n", uid, string(key.Keydata))
		}
	} else {
		for _, key := range keys {
			uid := key.Comment

			fmt.Printf("%s: %s [%s]\n", uid, key.Keysize, key.Fingerprint)
			fmt.Printf("%s: [%s]\n", uid, string(key.Keydata))
		}
	}
}

func cmdKeysSSHd(fingerprint string, keys map[string]ssh.Key) {
	if key, ok := keys[fingerprint]; ok {
		path, err := exec.LookPath("gin-repo")

		if err != nil {
			path = "/gin-repo"
		}

		out := bytes.NewBufferString("no-port-forwarding,no-X11-forwarding,no-agent-forwarding,no-pty,")
		out.WriteString("command=\"")
		out.WriteString(path)
		out.WriteString(" shell ")
		out.WriteString(key.Comment)
		out.WriteString("\" ")
		out.Write(key.Keydata)

		os.Stdout.Write(out.Bytes())
	} else {
		fmt.Fprintf(os.Stderr, "No key for fingerprint")
		os.Exit(-10)
	}
}

func cmdKeys(args map[string]interface{}) {

	dir := os.Getenv("GIN_REPO_KEYDIR")

	if dir == "" {
		dir = "."
	}

	keys := ssh.ReadKeysInDir(dir)

	if val, ok := args["list"]; ok && val.(bool) {
		cmdKeysList(args, keys)
	} else if val, ok := args["sshd"]; ok && val.(bool) {
		fingerprint := args["<fingerprint>"].(string)
		cmdKeysSSHd(fingerprint, keys)
	} else {
		os.Exit(-11)
	}

}
