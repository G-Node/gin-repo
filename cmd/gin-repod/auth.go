package main

import (
	"net/http"

	"github.com/G-Node/gin-repo/auth"
	"github.com/G-Node/gin-repo/store"
)

func (s *Server) checkAccess(w http.ResponseWriter, r *http.Request, rid store.RepoId, want store.AccessLevel) (*store.User, bool) {

	user, err := s.users.UserForRequest(r)

	if err != nil && err != auth.ErrNoAuth {
		// This means something went wrong with the token decoding
		// TODO: what about but expired tokens? http.StatusUnauthorized?
		http.Error(w, "Authorization error!", http.StatusForbidden)
		s.log(DEBUG, "Auth error: %v", err)
		return nil, false
	}

	if want == store.NoAccess {
		return user, true
	}

	uid := ""
	if user != nil {
		uid = user.Uid
	}

	have, err := s.repos.GetAccessLevel(rid, uid)

	if err != nil {
		// This should really not happen, since GetAccessLevel is pretty robust
		http.Error(w, "Internal server error :(", http.StatusInternalServerError)
		return user, false
	}

	s.log(DEBUG, "U: %s w: %v; h: %v -> %v", uid, want, have, want < have)
	if want > have {

		if have < store.PullAccess {
			//TODO: all 404 messages should be the same, otherwise you can infer information
			// from them
			http.Error(w, "Nothing here. Move along.", http.StatusNotFound)
		} else {
			http.Error(w, "No access", http.StatusForbidden)
		}
		return user, false
	}

	return user, true
}
