package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/G-Node/gin-repo/client"
	"github.com/G-Node/gin-repo/ssh"
)

func cmdKeysSSHd(client *client.Client, key ssh.Key) int {

	fingerprint, err := key.Fingerprint()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not create fingerprint: %v\n", err)
		return 1
	}

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
		out.WriteString("-S ")
		out.WriteString(client.Address)
		out.WriteString(" ")
		out.WriteString(user.Uid)
		out.WriteString("\" ")
		out.Write(key.MarshalAuthorizedKey())
		os.Stdout.Write(out.Bytes())
	}

	return 0
}
