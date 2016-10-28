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
	Public      bool
	Head        string
}

type Branch struct {
	Name   string
	Commit string
}

type GitHook struct {
	Name     string    `json:"name"`
	HookArgs []string  `json:"hookargs,omitempty"`
	RepoPath string    `json:"repopath"`
	RefLines []RefLine `json:"ref_lines,omitempty"`
}

type RefLine struct {
	OldRef  string `json:"oldref"`
	NewRef  string `json:"newref"`
	RefName string `json:"refname"`
}
