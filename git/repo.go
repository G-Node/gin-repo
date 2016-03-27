package git

import (
	"os/exec"
)

type Repository struct {
	Path string
}

func InitBareRepository(path string) (*Repository, error) {

	cmd := exec.Command("git", "init", "--bare", path)
	err := cmd.Run()

	if err != nil {
		return nil, err
	}

	return &Repository{Path: path}, nil
}
