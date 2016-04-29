package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/G-Node/gin-repo/client"
)

func cmdKeysSSHd(client *client.Client, fingerprint string) int {

	user, err := client.LookupUserByFingerprint(fingerprint)

	if err != nil {
		fmt.Fprintf(os.Stderr, "No key found: %v", err)
		return -10
	}

	for _, key := range user.Keys {
		path, err := exec.LookPath("gin-shell")

		if err != nil {
			path = "/gin-shell"
		}

		out := bytes.NewBufferString("no-port-forwarding,no-X11-forwarding,no-agent-forwarding,no-pty,")
		out.WriteString("command=\"")
		out.WriteString(path)
		out.WriteString(" ")
		out.WriteString(user.Uid)
		out.WriteString("\" ")
		out.Write(key.Keydata)

		os.Stdout.Write(out.Bytes())
	}

	return 0
}
