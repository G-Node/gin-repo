package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/docopt/docopt-go"

	"github.com/G-Node/gin-repo/ssh"
)

func getRepoDir() string {
	dir := os.Getenv("GIN_REPO_KEYDIR")

	if dir == "" {
		dir = "."
	}

	return dir
}

func repoInfo(w http.ResponseWriter, r *http.Request) {
	log.Printf("R: %s @ %v", r.Method, r.URL.String())

	w.WriteHeader(http.StatusNotImplemented)
}

func lookupUser(w http.ResponseWriter, r *http.Request) {
	log.Printf("R: %s @ %s", r.Method, r.URL.String())

	query := r.URL.Query()

	val, ok := query["key"]

	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	keys := ssh.ReadKeysInDir(getRepoDir())

	if key, ok := keys[val[0]]; ok {
		data, err := json.Marshal(key)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, err = w.Write(data)
		if err != nil {
			log.Panicf("[W] %v", err)
		}

	} else {
		w.WriteHeader(http.StatusNotFound)
	}

}

func main() {
	usage := `gin repo daemon.

Usage:
  gin-repo
  gin-repo -h | --help
  gin-repo --version

Options:
  -h --help     Show this screen.
  `

	args, _ := docopt.Parse(usage, nil, true, "gin repod 0.1a", false)
	fmt.Println(args)

	http.HandleFunc("/intern/user/lookup", lookupUser)
	http.HandleFunc("/intern/repo/", repoInfo)

	hostport := ":8888"
	log.Printf("Listening on " + hostport)
	log.Fatal(http.ListenAndServe(hostport, nil))
}
