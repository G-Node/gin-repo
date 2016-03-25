package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/docopt/docopt-go"

	. "github.com/G-Node/gin-repo/common"
	"github.com/G-Node/gin-repo/wire"
	"github.com/G-Node/gin-repo/ssh"
	"path/filepath"
	"strings"
)

func getRepoDir() string {
	dir := os.Getenv("GIN_REPO_KEYDIR")

	if dir == "" {
		dir = "."
	}

	return dir
}

func translatePath(vpath string, uid string) string {
	dir := os.Getenv("GIN_REPO_DIR")

	if dir == "" {
		dir = "."
	}

	if strings.HasPrefix(vpath, "'") && strings.HasSuffix(vpath, "'") {
		vpath = vpath[1 : len(vpath)-1]
	}

	path := filepath.Join(dir, uid, vpath)

	if !strings.HasSuffix(path, ".git") {
		path += ".git"
	}

	path, err := filepath.Abs(path)

	//TODO: propagate the error
	if err != nil {
		return path
	}

	fmt.Fprintf(os.Stderr, "[D] tp: %s@%s -> %s\n", uid, vpath, path)

	return path
}

func repoAccess(w http.ResponseWriter, r *http.Request) {
	log.Printf("R: %s @ %v", r.Method, r.URL.String())

	decoder := json.NewDecoder(r.Body)
	var query wire.RepoAccessQuery
	err := decoder.Decode(&query)

	if err != nil || query.Path == "" || query.User == "" || query.Method == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(os.Stderr, "Error precessing request: %v", err)
		return
	}

	//TODO: check access here

	path := translatePath(query.Path, query.User)
	w.Write([]byte(path))
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
		user := User{Uid: key.Comment, Keys: []ssh.Key{key}}

		data, err := json.Marshal(user)
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
	http.HandleFunc("/intern/repos/access", repoAccess)

	hostport := ":8888"
	log.Printf("Listening on " + hostport)
	log.Fatal(http.ListenAndServe(hostport, nil))
}
