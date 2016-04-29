package main

import (
	"github.com/docopt/docopt-go"
)

func main() {
	usage := `gin shell.

Usage:
  gin-shell keys list [--fingerprint=<fingerprint>]
  gin-shell keys sshd <fingerprint>
  
  gin-shell shell <uid>

  gin-shell -h | --help
  gin-shell --version

Options:
  -h --help     Show this screen.
  --version     Show version.
`
	args, _ := docopt.Parse(usage, nil, true, "gin shell 0.1a", false)

	if val, ok := args["keys"]; ok && val.(bool) {
		cmdKeys(args)
	} else if val, ok := args["shell"]; ok && val.(bool) {
		cmdShell(args)
	}

}
