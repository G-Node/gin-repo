package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"

	"regexp"

	"github.com/G-Node/gin-repo/auth"
	"github.com/G-Node/gin-repo/git"
	"github.com/G-Node/gin-repo/store"
	"github.com/G-Node/gin-repo/wire"
)

var nameChecker *regexp.Regexp

func init() {
	nameChecker = regexp.MustCompile("^[a-zA-Z0-9-_.]{3,}$")
}

func checkName(name string) bool {
	return nameChecker.MatchString(name)
}

func (s *Server) createRepo(w http.ResponseWriter, r *http.Request) {
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
	owner := vars["user"]

	user, err := s.users.UserForRequest(r)

	if err != nil && err == auth.ErrNoAuth {
		http.Error(w, "The owner must be logged in to create a repository", http.StatusUnauthorized)
		return
	} else if err != nil {
		http.Error(w, "Authorization error", http.StatusForbidden)
		s.log(DEBUG, "Auth error: %v", err)
		return
	} else if user.Uid != owner {
		http.Error(w, "Only the owner can create a repo", http.StatusForbidden)
		return
	}

	rid := store.RepoId{Owner: owner, Name: creat.Name}

	repo, err := s.repos.CreateRepo(rid)

	if err != nil {

		if os.IsExist(err) {
			w.WriteHeader(http.StatusConflict)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
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
		return wire.Repo{}, err
	}

	//HEAD must be a symbolic ref
	symhead := head.(*git.SymbolicRef)
	targetName := symhead.Fullname()

	//FIXME: git rev-parse based OpenRef doesn't work with
	// empty repos, workaround: return HEAD
	target, err := repo.OpenRef(symhead.Symbol)
	if err == nil {
		targetName = target.Fullname()
	}

	wr := wire.Repo{
		Name:        name,
		Description: repo.ReadDescription(),
		Head:        targetName}

	return wr, nil
}

func (s *Server) listRepos(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	owner := vars["user"]

	user, err := s.users.UserForRequest(r)
	if err != nil && err != auth.ErrNoAuth {
		http.Error(w, "Authorization error", http.StatusForbidden)
		return
	}
	//TODO: sanitize username

	ids, err := s.repos.ListReposForUser(owner)

	if os.IsExist(err) || len(ids) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var repos []wire.Repo
	for _, p := range ids {

		level, err := s.repos.GetAccessLevel(p, user.Uid)

		if err != nil {
			s.log(WARN, "Getting access level for %q failed: %v", p, err)
			continue
		}

		if level < store.PullAccess {
			continue
		}

		repo, err := s.repos.OpenGitRepo(p)

		if err != nil {
			s.log(WARN, "could not open repo @ %q", p)
			continue
		}

		wr, err := repoToWire(repo)

		if err != nil {
			s.log(WARN, "repo serialization error for %q [%v]", p, err)
			continue
		}

		repos = append(repos, wr)
	}

	if len(repos) == 0 {
		http.Error(w, "No repositories found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	js := json.NewEncoder(w)
	err = js.Encode(repos)

	if err != nil {
		s.log(WARN, "Error while encoding, status already sent. oh oh.")
	}
}

func (s *Server) listPublicRepos(w http.ResponseWriter, r *http.Request) {
	ids, err := s.repos.ListPublicRepos()

	if os.IsExist(err) || len(ids) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var repos []wire.Repo
	for _, p := range ids {
		repo, err := s.repos.OpenGitRepo(p)

		if err != nil {
			s.log(WARN, "could not open repo @ %q", p)
			continue
		}

		wr, err := repoToWire(repo)

		if err != nil {
			s.log(WARN, "repo serialization error for %q [%v]", p, err)
			continue
		}

		repos = append(repos, wr)
	}

	if len(repos) == 0 {
		http.Error(w, "No repositories found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	js := json.NewEncoder(w)
	err = js.Encode(repos)

	if err != nil {
		s.log(WARN, "Error while encoding, status already sent. oh oh.")
	}

}

func (s *Server) varsToRepoID(vars map[string]string) (store.RepoId, error) {
	iuser := vars["user"]
	irepo := vars["repo"]

	//TODO: check name and stuff

	return store.RepoId{iuser, irepo}, nil
}

func (s *Server) getRepoVisibility(w http.ResponseWriter, r *http.Request) {
	ivars := mux.Vars(r)
	rid, err := s.varsToRepoID(ivars)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	public, err := s.repos.GetRepoVisibility(rid)
	if err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
		return
	}

	var visibility string
	if public {
		visibility = "public"
	} else {
		visibility = "private"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("{%q: %q}", "visibility", visibility)))
}

func (s *Server) setRepoVisibility(w http.ResponseWriter, r *http.Request) {
	ivars := mux.Vars(r)
	rid, err := s.varsToRepoID(ivars)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	decoder := json.NewDecoder(r.Body)
	var req struct {
		Visibility string
	}

	err = decoder.Decode(&req)

	var public bool

	parser := func() bool {
		if req.Visibility == "public" {
			public = true
		} else if req.Visibility == "private" {
			public = false
		} else {
			return false
		}

		return true
	}

	if err != nil || req.Visibility == "" || !parser() {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = s.repos.SetRepoVisibility(rid, public)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	//TODO: return the visibility
}

func (s *Server) getBranch(w http.ResponseWriter, r *http.Request) {
	ivars := mux.Vars(r)
	ibranch := ivars["branch"]

	rid, err := s.varsToRepoID(ivars)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	repo, err := s.repos.OpenGitRepo(rid)

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

func (s *Server) getObject(w http.ResponseWriter, r *http.Request) {
	ivars := mux.Vars(r)
	isha1 := ivars["object"]

	rid, err := s.varsToRepoID(ivars)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	repo, err := s.repos.OpenGitRepo(rid)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	oid, err := git.ParseSHA1(isha1)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	obj, err := repo.OpenObject(oid)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	out := bufio.NewWriter(w)
	switch obj := obj.(type) {
	case *git.Commit:
		w.Header().Set("Content-Type", "application/json")
		out.WriteString("{")
		out.WriteString(fmt.Sprintf("%q: %q,\n", "type", "commit"))
		out.WriteString(fmt.Sprintf("%q: %q,\n", "tree", obj.Tree))
		for _, parent := range obj.Parent {
			out.WriteString(fmt.Sprintf("%q: %q,\n", "parent", parent))
		}
		out.WriteString(fmt.Sprintf("%q: %q,\n", "author", obj.Author))
		out.WriteString(fmt.Sprintf("%q: %q,\n", "commiter", obj.Committer))
		out.WriteString(fmt.Sprintf("%q: %q", "message", obj.Message))
		out.WriteString("}")

	case *git.Tree:
		w.Header().Set("Content-Type", "application/json")
		out.WriteString("{")
		out.WriteString(fmt.Sprintf("%q: %q,", "type", "tree"))
		out.WriteString(fmt.Sprintf("%q: [", "entries"))
		first := true // maybe change Tree.Next() sematics, this is ugly
		for obj.Next() {
			if first {
				first = false
			} else {
				out.WriteString(",\n")
			}
			entry := obj.Entry()
			out.WriteString("{")
			symlink := entry.Mode == 00120000
			if symlink {
				//refactor: too much nesting
				target, err := repo.Readlink(entry.ID)
				if err != nil {
					s.log(WARN, "could not resolve symlink for %s: %v", entry.ID, err)
					symlink = false
				} else if git.IsAnnexFile(target) {
					out.WriteString(fmt.Sprintf("%q: %q,\n", "type", "annex"))

					fi, err := repo.Astat(target)
					var state string
					if err != nil {
						s.log(WARN, "repo.Astat failed [%s]: %v", target, err)
						state = "error"
					} else if fi.Have {
						state = "have"
					} else {
						state = "missing"
					}

					out.WriteString(fmt.Sprintf("%q: %q,\n", "status", state))
				} else {
					out.WriteString(fmt.Sprintf("%q: %q,\n", "type", "symlink"))
				}
			}

			if !symlink {
				out.WriteString(fmt.Sprintf("%q: %q,\n", "type", entry.Type))
			}
			out.WriteString(fmt.Sprintf("%q: %q\n", "id", entry.ID))
			out.WriteString(fmt.Sprintf("%q: %q,\n", "name", entry.Name))
			out.WriteString(fmt.Sprintf("%q: \"%08o\",\n", "mode", entry.Mode))
			out.WriteString("}")
		}
		out.WriteString("]}")

	case *git.Blob:
		n := int64(512)
		if m := obj.Size() - 1; m < n {
			n = m
		}
		buf := make([]byte, n)
		_, err = obj.Read(buf[:])
		if err != nil {
			panic("IO error")
		}
		mtype := http.DetectContentType(buf)
		w.Header().Set("Content-Type", mtype)

		w.Write(buf)
		_, err := io.Copy(w, obj)
		if err != nil {
			s.log(WARN, "io error, but data already written")
		}
		obj.Close()

	default:
		w.Header().Set("Content-Type", "application/json")
		out.WriteString("{")
		out.WriteString(fmt.Sprintf("%q: %q", "type", obj.Type()))
		out.WriteString("}")
	}

	out.Flush() // ignoring error
	obj.Close() // ignoring error

}
