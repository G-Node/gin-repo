package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"path/filepath"
	"strings"

	"github.com/G-Node/gin-repo/store"
	"github.com/G-Node/gin-repo/wire"
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

func (s *Server) repoAccess(w http.ResponseWriter, r *http.Request) {
	log.Printf("repoAccess: %s @ %v", r.Method, r.URL.String())

	decoder := json.NewDecoder(r.Body)
	var query wire.RepoAccessQuery
	err := decoder.Decode(&query)

	if err != nil || query.Path == "" || query.User == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(os.Stderr, "Error precessing request: %v", err)
		return
	}

	rid, err := store.RepoIdParse(query.Path)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	repo, err := s.repos.OpenGitRepo(rid)

	if err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}

		return
	}

	access := wire.RepoAccessInfo{Path: repo.Path, Push: true}

	data, err := json.Marshal(access)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (s *Server) lookupUser(w http.ResponseWriter, r *http.Request) {
	log.Printf("lookupUser: %s @ %s", r.Method, r.URL.String())

	query := r.URL.Query()

	val, ok := query["key"]

	if !ok || len(val) != 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user, err := s.users.LookupUserBySSH(val[0])
	s.log(DEBUG, "lookupUser: fingerprint %q", val[0])

	if err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			s.log(PANIC, "lookupUser error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
		}

		return
	}

	data, err := json.Marshal(user)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = w.Write(data)
	if err != nil {
		s.log(PANIC, "lookupUser: IO error: %v", err)
	}
}
