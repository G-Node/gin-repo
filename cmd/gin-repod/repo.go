package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/G-Node/gin-repo/git"
	"github.com/G-Node/gin-repo/store"
	"github.com/G-Node/gin-repo/wire"
	"github.com/gorilla/mux"
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

	rid := store.RepoId{Owner: owner, Name: creat.Name}

	_, ok := s.checkAccess(w, r, rid, store.OwnerAccess)
	if !ok {
		return
	}

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

func (s *Server) repoToWire(id store.RepoId, repo *git.Repository) (wire.Repo, error) {
	public, err := s.repos.GetRepoVisibility(id)
	if err != nil {
		s.log(WARN, "could not get repo visibility: %v", err)
		public = false
	}

	wr := wire.Repo{
		Name:        id.Name,
		Owner:       id.Owner,
		Description: repo.ReadDescription(),
		Head:        "master",
		Public:      public,
	}

	return wr, nil
}

func (s *Server) listRepos(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	owner := vars["user"]
	//TODO: sanitize ownername

	user, ok := s.checkAccess(w, r, store.RepoId{}, store.NoAccess)
	if !ok {
		return
	}

	ids, err := s.repos.ListReposForUser(owner)

	if os.IsExist(err) || len(ids) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	uid := ""
	if user != nil {
		uid = user.Uid
	}

	var repos []wire.Repo
	for _, p := range ids {

		level, err := s.repos.GetAccessLevel(p, uid)

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

		wr, err := s.repoToWire(p, repo)

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

func (s *Server) listSharedRepos(w http.ResponseWriter, r *http.Request) {
	user, ok := s.checkAccess(w, r, store.RepoId{}, store.NoAccess)
	if !ok {
		return
	}

	ids, err := s.repos.ListSharedRepos(user.Uid)

	//TODO: the following is duplicated code
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

		wr, err := s.repoToWire(p, repo)

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

		wr, err := s.repoToWire(p, repo)

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

// varsToRepoID checks if a map contains the entries "user" and "repo" and
// returns them as a store.RepoId or an error if they are missing.
func (s *Server) varsToRepoID(vars map[string]string) (store.RepoId, error) {
	iuser := vars["user"]
	irepo := vars["repo"]

	repoId := store.RepoId{iuser, irepo}
	if iuser == "" || irepo == "" {
		return repoId, errors.New("Missing arguments.")
	}

	return repoId, nil
}

func (s *Server) getRepoVisibility(w http.ResponseWriter, r *http.Request) {
	ivars := mux.Vars(r)
	rid, err := s.varsToRepoID(ivars)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Return StatusBadRequest if an error occurs or if the repository does not exist.
	// Returning StatusNotFound for non existing repositories could lead to inference
	// of private repositories later on.
	exists, err := s.repos.RepoExists(rid)
	if err != nil || !exists {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, ok := s.checkAccess(w, r, rid, store.PullAccess)
	if !ok {
		return
	}

	public, err := s.repos.GetRepoVisibility(rid)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("{%q: %t}", "Public", public)))
}

func (s *Server) setRepoVisibility(w http.ResponseWriter, r *http.Request) {
	ivars := mux.Vars(r)
	rid, err := s.varsToRepoID(ivars)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, ok := s.checkAccess(w, r, rid, store.AdminAccess)
	if !ok {
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

	_, ok := s.checkAccess(w, r, rid, store.PullAccess)
	if !ok {
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

	_, ok := s.checkAccess(w, r, rid, store.PullAccess)
	if !ok {
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

	s.objectToWire(w, repo, obj)
}

func (s *Server) objectToWire(w http.ResponseWriter, repo *git.Repository, obj git.Object) {
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
			out.WriteString(fmt.Sprintf("%q: %q,\n", "id", entry.ID))
			out.WriteString(fmt.Sprintf("%q: %q,\n", "name", entry.Name))
			out.WriteString(fmt.Sprintf("%q: \"%08o\"\n", "mode", entry.Mode))
			out.WriteString("}")
		}
		out.WriteString("]}")

	case *git.Blob:
		n := int64(512)
		if m := obj.Size() - 1; m < n {
			n = m
		}
		buf := make([]byte, n)
		_, err := obj.Read(buf[:])
		if err != nil {
			panic("IO error")
		}
		mtype := http.DetectContentType(buf)
		w.Header().Set("Content-Type", mtype)

		w.Write(buf)
		_, err = io.Copy(w, obj)
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

func (s *Server) browseRepo(w http.ResponseWriter, r *http.Request) {
	ivars := mux.Vars(r)
	rid, err := s.varsToRepoID(ivars)

	ibranch := ivars["branch"]
	ipath := ivars["path"]

	//for now we only support master :(
	if err != nil || ibranch != "master" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, ok := s.checkAccess(w, r, rid, store.PullAccess)
	if !ok {
		return
	}

	s.log(DEBUG, "branch: %q, ipath: %q", ibranch, ipath)

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
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	root, err := repo.OpenObject(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	obj, err := repo.ObjectForPath(root, ipath)
	if err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	s.objectToWire(w, repo, obj)
}
