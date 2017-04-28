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
	Public      bool
}

// Repo is used to export basic information about a repository.
// Public states whether a repository is publicly available.
// Shared states whether a repository is shared with a collaborator.
type Repo struct {
	Name        string
	Owner       string
	Description string
	Head        string
	Public      bool
	Shared      bool
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

// CommitListItem represents a subset of information from a git commit
type CommitListItem struct {
	Commit       string   `json:"commit"`
	Committer    string   `json:"committer"`
	Author       string   `json:"author"`
	DateIso      string   `json:"dateiso"`
	DateRelative string   `json:"daterel"`
	Subject      string   `json:"subject"`
	Changes      []string `json:"changes"`
}
