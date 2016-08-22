package main

import (
	"fmt"
	"os"
)

func (s *Server) handleCommands(args map[string]interface{}) {
	res := 0
	hadCommand := false
	if args["make-token"].(bool) {
		hadCommand = true
		user := args["<user>"].(string)
		str, err := s.users.TokenForUser(user)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not create user token: %v", err)
			res = -12
		}

		fmt.Fprintf(os.Stderr, "Token for %q\n", user)
		fmt.Fprintf(os.Stdout, "%s\n", str)
	}

	if hadCommand {
		os.Exit(res)
	}
}
