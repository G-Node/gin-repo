package main

import (
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

	if val, ok := args["--keys"]; ok && val.(bool) {
		fingerprint := args["<fingerprint>"].(string)
		cmdKeysSSHd(fingerprint)
		return
	}

	cmdShell(args)
}
