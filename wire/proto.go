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
	Description string
	Head        string
}

type Branch struct {
	Name   string
	Commit string
}
