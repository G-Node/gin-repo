package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/dgrijalva/jwt-go"
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

func readSecret() ([]byte, error) {

	path := "."
	_, err := os.Stat("gin.secret")
	if err != nil {
		path = ""
	}

	if path == "" {
		u, err := user.Current()
		if err == nil {
			path = u.HomeDir
		}
	}

	if path == "" {
		path = os.Getenv("HOME")
	}

	filename := filepath.Join(path, "gin.secret")
	secret, err := ioutil.ReadFile(filename)

	return secret, err
}

func makeToken(user string) int {
	token := jwt.New(jwt.SigningMethodHS256)

	host, err := os.Hostname()
	if err != nil {
		host = "localhost"
	}

	token.Claims["iss"] = "gin-repo@" + host
	token.Claims["iat"] = time.Now().Unix()
	token.Claims["exp"] = time.Now().Add(time.Minute * 5).Unix()
	token.Claims["role"] = user

	secret, err := readSecret()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not load secret: %v", err)
		return -1
	}

	str, err := token.SignedString(secret)
	fmt.Fprintf(os.Stdout, "Token for %q\n%s\n", user, str)
	return 0
}
