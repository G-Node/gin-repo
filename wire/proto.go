package wire

type RepoAccessQuery struct {
	User   string
	Path   string
	Method string
}

type CreateRepo struct {
	Name        string
	Description string
}
