package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/docopt/docopt-go"

	"github.com/gorilla/mux"
)

func main() {
	usage := `gin repo daemon.

Usage:
  gin-repod
  gin-repod -h | --help
  gin-repod --version

Options:
  -h --help     Show this screen.
  `

	args, _ := docopt.Parse(usage, nil, true, "gin repod 0.1a", false)
	fmt.Println(args)

	r := mux.NewRouter()
	r.HandleFunc("/intern/user/lookup", lookupUser).Methods("GET")
	r.HandleFunc("/intern/repos/access", repoAccess).Methods("POST")

	r.HandleFunc("/users/{user}/repos", createRepo).Methods("POST")

	http.Handle("/", r)

	hostport := ":8888"
	log.Printf("Listening on " + hostport)
	log.Fatal(http.ListenAndServe(hostport, nil))
}
