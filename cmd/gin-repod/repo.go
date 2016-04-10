package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"

	"regexp"

	"github.com/G-Node/gin-repo/git"
	"github.com/G-Node/gin-repo/wire"
)

var nameChecker *regexp.Regexp

func init() {
	nameChecker = regexp.MustCompile("^[a-zA-Z0-9-_.]{3,}$")
}

func checkName(name string) bool {
	return nameChecker.MatchString(name)
}

func createRepo(w http.ResponseWriter, r *http.Request) {
	log.Printf("createRepo: %s @ %s", r.Method, r.URL.String())

	decoder := json.NewDecoder(r.Body)
	var creat wire.CreateRepo
	err := decoder.Decode(&creat)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(os.Stderr, "Error precessing request: %v", err)
		return
	} else if creat.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(os.Stderr, "Error precessing request: name missing")
		return
	} else if !checkName(creat.Name) {
		http.Error(w, "Invalid repository name", http.StatusBadRequest)
		fmt.Fprintf(os.Stderr, "Error precessing request: invalid name: %q", creat.Name)
		return
	}

	vars := mux.Vars(r)
	user := vars["user"]

	path := translatePath(creat.Name, user)

	repo, err := git.InitBareRepository(path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	// ignore error, because we created the repo
	//  which is more important
	repo.WriteDescription(creat.Description)

	wr := wire.Repo{Name: creat.Name, Description: repo.ReadDescription()}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	js := json.NewEncoder(w)
	err = js.Encode(wr)

	if err != nil {
		log.Printf("Error while encoding, status already sent. oh oh.")
	}
}

func repoToWire(repo *git.Repository) (wire.Repo, error) {
	basename := filepath.Base(repo.Path)
	name := basename[:len(basename)-len(filepath.Ext(basename))]

	head, err := repo.OpenRef("HEAD")
	if err != nil {
		return wire.Repo{}, nil
	}

	//HEAD must be a symbolic ref
	symhead := head.(*git.SymbolicRef)
	target, err := repo.OpenRef(symhead.Symbol)

	if err != nil {
		return wire.Repo{}, nil
	}

	wr := wire.Repo{
		Name:        name,
		Description: repo.ReadDescription(),
		Head:        target.Fullname()}

	return wr, nil
}

func (s *Server) listRepos(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := vars["user"]

	base := s.repoDir(user)
	names, err := filepath.Glob(filepath.Join(base, "*.git"))

	if err != nil {
		panic("Bad pattern for filepath.Glob(). Uh oh!")
	}

	if len(names) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var repos []wire.Repo
	for _, p := range names {
		repo, err := git.OpenRepository(p)

		if err != nil {
			s.log(WARN, "could not open repo @ %q", p)
			continue
		}

		wr, err := repoToWire(repo)

		if err != nil {
			s.log(WARN, "repo serialization error for %q", p)
			continue
		}

		repos = append(repos, wr)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	js := json.NewEncoder(w)
	err = js.Encode(repos)

	if err != nil {
		s.log(WARN, "Error while encoding, status already sent. oh oh.")
	}
}

func (s *Server) getBranch(w http.ResponseWriter, r *http.Request) {
	ivars := mux.Vars(r)
	iuser := ivars["user"]
	irepo := ivars["repo"]
	ibranch := ivars["branch"]

	base := filepath.Join(s.repoDir(iuser), irepo+".git")

	repo, err := git.OpenRepository(base)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	ref, err := repo.OpenRef(ibranch)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	id, err := ref.Resolve()
	if err != nil {
		panic("Could not resolve ref")
	}

	branch := wire.Branch{Name: ref.Name(), Commit: id.String()}
	js := json.NewEncoder(w)
	err = js.Encode(branch)

	if err != nil {
		s.log(WARN, "Error while encoding, status already sent. oh oh.")
	}
}

