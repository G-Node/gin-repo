package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/G-Node/gin-repo/auth"
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

func makeToken(user string) int {
	token := jwt.New(jwt.SigningMethodHS256)

	host, err := os.Hostname()
	if err != nil {
		host = "localhost"
	}

	token.Claims = &auth.Claims{
		StandardClaims: &jwt.StandardClaims{
			Issuer:    "gin-repo@" + host,
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(time.Minute * 120).Unix(),
			Subject:   user,
		},
		TokenType: "user",
	}

	secret, err := auth.ReadSharedSecret()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not load secret: %v", err)
		return -1
	}

	str, err := token.SignedString(secret)
	fmt.Fprintf(os.Stdout, "Token for %q\n%s\n", user, str)
	return 0
}
