package git

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func IsBareRepository(path string) bool {

	if os.Getenv("GIT_DIR") != "" {
		log.Printf("[W] $GIT_DIR defined! We will unset it now.")
		if err := os.Unsetenv("GIT_DIR"); err != nil {
			log.Printf("[W] Unsetting $GIT_DIR failed. :( ")
		}
	}

	env := os.Environ() // returns a copy, so safe to edit
	env = append(env, fmt.Sprintf("GIT_DIR=%s", path))
	cmd := exec.Command("git", "rev-parse", "--is-bare-repository")
	cmd.Env = env

	body, err := cmd.Output()

	if err != nil {
		return false
	}

	status := strings.Trim(string(body), "\n ")
	return status == "true"
}
