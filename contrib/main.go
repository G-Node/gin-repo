package main

import (
	"fmt"
	"log"
	"os"

	"github.com/G-Node/gin-repo/store"
	"github.com/docopt/docopt-go"
)

func main() {
	usage := `contrib gin-repo tool

Usage:
  contrib token <user>

Options:
  -h --help     Show this screen.
  --version     Show version.
`

	args, _ := docopt.Parse(usage, nil, true, "contrib 0.1", false)
	log.SetOutput(os.Stderr)

	res := 0
	if args["token"].(bool) {
		user := args["<user>"]
		res = makeToken(user.(string))
	}

	os.Exit(res)
}

func makeToken(user string) int {

	var err error
	dir := os.Getenv("GIN_REPO_DIR")

	if dir == "" {
		dir = "."
	}

	store, err := store.NewUserStore(dir)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not setup user store: %v", err)
		return 11
	}

	str, err := store.TokenForUser(user)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not create user token: %v", err)
		return 12
	}

	fmt.Fprintf(os.Stderr, "Token for %q\n", user)
	fmt.Fprintf(os.Stdout, "%s\n", str)
	return 0
}
