package main

func (s *Server) SetupRoutes() {
	r := s.Root

	r.HandleFunc("/intern/user/lookup", s.lookupUser).Methods("GET")
	r.HandleFunc("/intern/repos/access", s.repoAccess).Methods("POST")

	r.HandleFunc("/intern/hooks/pre-receive", s.hookPreReceive).Methods("POST")
	r.HandleFunc("/intern/hooks/update", s.hookUpdate).Methods("POST")
	r.HandleFunc("/intern/hooks/post-receive", s.hookPostReceive).Methods("POST")

	r.HandleFunc("/repos/public", s.listPublicRepos).Methods("GET")
	r.HandleFunc("/repos/shared", s.listSharedRepos).Methods("GET")

	r.HandleFunc("/users/{user}/repos", s.createRepo).Methods("POST")
	r.HandleFunc("/users/{user}/repos", s.listRepos).Methods("GET")

	r.HandleFunc("/users/{user}/repos/{repo}/visibility", s.getRepoVisibility).Methods("GET")
	r.HandleFunc("/users/{user}/repos/{repo}/visibility", s.setRepoVisibility).Methods("PUT")

	r.HandleFunc("/users/{user}/repos/{repo}/branches/{branch}", s.getBranch).Methods("GET")
	r.HandleFunc("/users/{user}/repos/{repo}/objects/{object}", s.getObject).Methods("GET")
	r.HandleFunc("/users/{user}/repos/{repo}/browse/{branch}/{path:.*}", s.browseRepo).Methods("GET")
}
