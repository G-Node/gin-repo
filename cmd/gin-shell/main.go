package main

import (
	"fmt"
	"log"
	"os"

	"github.com/G-Node/gin-repo/client"
	"github.com/docopt/docopt-go"
)

func main() {
	usage := `gin shell.

Usage:
  gin-shell --keys <fingerprint>
  
  gin-shell <uid>

  gin-shell -h | --help
  gin-shell --version

Options:
  -h --help     Show this screen.
  --version     Show version.
`
	args, _ := docopt.Parse(usage, nil, true, "gin shell 0.1a", false)

	log.SetOutput(os.Stderr)

	client := client.NewClient("http://localhost:8888")

	if token, err := makeServiceToken(); err == nil {
		client.AuthToken = token
	}

	if val, ok := args["--keys"]; ok && val.(bool) {
		fingerprint := args["<fingerprint>"].(string)
		ret := cmdKeysSSHd(client, fingerprint)
		os.Exit(ret)
	}

	argv := splitarg(os.Getenv("SSH_ORIGINAL_COMMAND"))
	cmd := head(argv)

	if cmd == "" {
		fmt.Fprintf(os.Stderr, "ERROR: No shell access allowed.")
		os.Exit(-9)
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
		res = gitCommand(client, argv, false, uid)

	case "git-receive-pack":
		res = gitCommand(client, argv, true, uid)

	case "git-annex-shell":
		res = gitAnnex(client, argv, uid)

	default:
		fmt.Fprintf(os.Stderr, "[E] unhandled command: %s\n", cmd)
		res = 23
	}

	os.Exit(res)
}
