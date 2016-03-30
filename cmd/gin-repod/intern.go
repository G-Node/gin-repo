package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	. "github.com/G-Node/gin-repo/common"
	"github.com/G-Node/gin-repo/git"
	"github.com/G-Node/gin-repo/ssh"
	"github.com/G-Node/gin-repo/wire"
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
	log.Printf("repoAccess: %s @ %v", r.Method, r.URL.String())

	decoder := json.NewDecoder(r.Body)
	var query wire.RepoAccessQuery
	err := decoder.Decode(&query)

	if err != nil || query.Path == "" || query.User == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(os.Stderr, "Error precessing request: %v", err)
		return
	}

	//TODO: check access here
	path := translatePath(query.Path, query.User)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// path does exists, but is it a bare repo?
	if !git.IsBareRepository(path) {
		// what is the right status here?
		//  for now we pretend the path doesnt exist
		w.WriteHeader(http.StatusNotFound)
		return
	}

	access := wire.RepoAccessInfo{Path: path, Push: true}

	data, err := json.Marshal(access)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func lookupUser(w http.ResponseWriter, r *http.Request) {
	log.Printf("lookupUser: %s @ %s", r.Method, r.URL.String())

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
