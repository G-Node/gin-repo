package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
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

	_, ok := s.checkAccess(w, r, rid, store.NoAccess)
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

	// Make sure that the request body contains field "public"
	// with a proper boolean value.
	var setPublic bool
	var vis interface{}

	if r.Body == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(b, &vis)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	m := vis.(map[string]interface{})
	if val, ok := m["public"]; ok {
		switch val.(type) {
		case bool:
			setPublic = val.(bool)
		default:
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = s.repos.SetRepoVisibility(rid, setPublic)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("{%q: %t}", "Public", setPublic)))
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

// patchRepoSettings patches repository description and public status.
// The request header has to contain an authorization header with a valid token.
// The request body has to contain valid JSON containing "description" as key
// and a string as value, "public" as key and a boolean value as value or both.
// A request that does not contain any of these two keys will result in a BadRequest status.
func (s *Server) patchRepoSettings(w http.ResponseWriter, r *http.Request) {
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

	// Pointer fields are used to make sure, that boolean values are
	// not updated, when the corresponding field was not
	// included in the submitted JSON.
	var patch struct {
		Description *string `json:"description,omitempty"`
		Public      *bool   `json:"public,omitempty"`
	}

	if r.Body == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(b, &patch)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// TODO The following code currently does not deal with the problem
	// if an error occurs, after part of the patch has already
	// been applied.
	responseCode := http.StatusOK

	// The following code deviates from normal style of
	// return on error, since we always want to return
	// the latest state if the repository settings,
	// even if an error occurs.
	if patch.Description == nil && patch.Public == nil {
		responseCode = http.StatusBadRequest
	}
	repository, err := git.OpenRepository(s.repos.IdToPath(rid))
	if err != nil {
		responseCode = http.StatusInternalServerError
	}

	if patch.Description != nil {
		err = repository.WriteDescription(*patch.Description)
		if err != nil {
			responseCode = http.StatusInternalServerError
		}
	}

	if patch.Public != nil {
		err = s.repos.SetRepoVisibility(rid, *patch.Public)
		if err != nil {
			responseCode = http.StatusInternalServerError
		}
	}

	var resp struct {
		Public      bool
		Description string
	}

	resp.Description = repository.ReadDescription()
	resp.Public, err = s.repos.GetRepoVisibility(rid)
	if err != nil {
		responseCode = http.StatusInternalServerError
	}

	respBody, err := json.Marshal(resp)
	if err != nil {
		responseCode = http.StatusInternalServerError
	}

	w.WriteHeader(responseCode)
	w.Header().Set("Content-Type", "application/json")
	w.Write(respBody)
}

// listRepoCollaborators returns a JSON array containing all collaborators
// of the requested repository.
func (s *Server) listRepoCollaborators(w http.ResponseWriter, r *http.Request) {
	ivars := mux.Vars(r)
	rid, err := s.varsToRepoID(ivars)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, ok := s.checkAccess(w, r, rid, store.PullAccess)
	if !ok {
		return
	}

	repoAccess, err := s.repos.ListSharedAccess(rid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	respBody := []byte("[]")
	if len(repoAccess) > 0 {
		users := make([]string, len(repoAccess))
		i := 0
		for k := range repoAccess {
			users[i] = k
			i++
		}
		respBody, err = json.Marshal(users)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respBody)
}

// putRepoCollaborator adds a user with the submitted access level to the sharing folder
// of a repository. If the user already exists, an http.StatusConflict is returned.
func (s *Server) putRepoCollaborator(w http.ResponseWriter, r *http.Request) {
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

	username := ivars["username"]
	if username == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// TODO check if provided username actually exists in the user store (local and auth).

	if rid.Owner == username {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	repoAccess, err := s.repos.ListSharedAccess(rid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, exists := repoAccess[username]; exists {
		// TODO return error message along the lines "already exists".
		w.WriteHeader(http.StatusConflict)
		return
	}

	var accessLevel struct{ Permission string }

	if r.Body == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(b, &accessLevel)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	level, err := store.ParseAccessLevel(accessLevel.Permission)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = s.repos.SetAccessLevel(rid, username, level)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// deleteRepoCollaborator removes a user from the sharing folder of a repository.
// If the user is not found within the folder, an http.StatusConflict is returned.
func (s *Server) deleteRepoCollaborator(w http.ResponseWriter, r *http.Request) {
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

	username := ivars["username"]
	if username == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	repo, err := git.OpenRepository(s.repos.IdToPath(rid))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = repo.DeleteCollaborator(username)
	if err != nil && os.IsNotExist(err) {
		w.WriteHeader(http.StatusConflict)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// repoDescription returns the repositories description. If the client does not
// provide at least pull access an http.StatusNotFound is returned.
func (s *Server) repoDescription(w http.ResponseWriter, r *http.Request) {
	ivars := mux.Vars(r)
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

	desc, err := s.repoToWire(rid, repo)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	body, err := json.Marshal(desc)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(body)
}
