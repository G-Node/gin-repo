package wire

type RepoAccessQuery struct {
	User string
	Path string
}

type RepoAccessInfo struct {
	Path string
	Push bool
}

type CreateRepo struct {
	Name        string
	Description string
}

type Repo struct {
	Name        string
	Owner       string
	Description string
	Visibility  bool
	Head        string
}

type Branch struct {
	Name   string
	Commit string
}

type GitHook struct {
	Name     string    `json:"name,omitempty"`
	HookArgs []string  `json:"hookargs,omitempty"`
	RepoPath string    `json:"repopath,omitempty"`
	RefLines []RefLine `json:"ref_lines,omitempty"`
}

type RefLine struct {
	OldRef  string `json:"oldref,omitempty"`
	NewRef  string `json:"newref,omitempty"`
	RefName string `json:"refname,omitempty"`
}
