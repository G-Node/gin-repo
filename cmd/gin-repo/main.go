package main

import (
	"github.com/docopt/docopt-go"
)

func main() {
	usage := `gin repo tool.

Usage:
  gin-repo keys list [--fingerprint=<fingerprint>]
  gin-repo keys sshd <fingerprint>
  
  gin-repo shell <uid>

  gin-repo -h | --help
  gin-repo --version

Options:
  -h --help     Show this screen.
  --version     Show version.
`
	args, _ := docopt.Parse(usage, nil, true, "gin repo 0.1a", false)

	if val, ok := args["keys"]; ok && val.(bool) {
		cmdKeys(args)
	} else if val, ok := args["shell"]; ok && val.(bool) {
		cmdShell(args)
	}

}
