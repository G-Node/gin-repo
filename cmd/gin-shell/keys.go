package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/G-Node/gin-repo/client"
)

func cmdKeysSSHd(fingerprint string) {
	client := client.NewClient("http://localhost:8888")

	if token, err := makeServiceToken(); err == nil {
		client.AuthToken = token
	}

	user, err := client.LookupUserByFingerprint(fingerprint)

	if err != nil {
		fmt.Fprintf(os.Stderr, "No key found: %v", err)
		os.Exit(-10)
	}

	for _, key := range user.Keys {
		path, err := exec.LookPath("gin-shell")

		if err != nil {
			path = "/gin-shell"
		}

		out := bytes.NewBufferString("no-port-forwarding,no-X11-forwarding,no-agent-forwarding,no-pty,")
		out.WriteString("command=\"")
		out.WriteString(path)
		out.WriteString(" shell ")
		out.WriteString(user.Uid)
		out.WriteString("\" ")
		out.Write(key.Keydata)

		os.Stdout.Write(out.Bytes())
	}
}

func cmdKeys(args map[string]interface{}) {

	if val, ok := args["sshd"]; ok && val.(bool) {
		fingerprint := args["<fingerprint>"].(string)
		cmdKeysSSHd(fingerprint)
	} else {
		os.Exit(-11)
	}

}
