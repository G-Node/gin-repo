package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

type Repository struct {
	Path string
}

func InitBareRepository(path string) (*Repository, error) {

	path, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("Could not determine absolute path: %v", err)
	}

	cmd := exec.Command("git", "init", "--bare", path)
	err = cmd.Run()

	if err != nil {
		return nil, err
	}

	return &Repository{Path: path}, nil
}
