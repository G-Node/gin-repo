package store

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/G-Node/gin-repo/git"
)

type RepoId struct {
	Owner string
	Name  string
}

func (id RepoId) String() string {
	return path.Join(id.Owner, id.Name)
}

func RepoIdParse(str string) (RepoId, error) {
	if count := strings.Count(str, "/"); count != 1 {
		return RepoId{}, fmt.Errorf("malformed id: wrong number of components: %d", count)
	}

	comps := strings.Split(str, "/")

	if len(comps) != 2 {
		panic("Have not exactly two components!")
	}

	return RepoId{comps[0], comps[1]}, nil
}

func RepoIdFromPath(path string) (RepoId, error) {
	if !strings.HasSuffix(path, ".git") {
		return RepoId{}, fmt.Errorf("Not a valid git path: %q", path)
	}

	base := filepath.Base(path)
	name := base[:len(base)-4]

	dir := filepath.Dir(path)
	if len(dir) == 1 && (dir[0] == '.' || dir[0] == filepath.Separator) {
		return RepoId{}, fmt.Errorf("Malformed git path: %q", path)
	}

	uid := filepath.Base(dir)
	return RepoId{uid, name}, nil
}

type RepoStore struct {
	Path string
}

func (store *RepoStore) gitPath() string {
	return filepath.Join(store.Path, "git")
}

func (store *RepoStore) idToPath(id RepoId) string {
	return filepath.Join(store.gitPath(), id.Owner, id.Name+".git")
}

func (store *RepoStore) CreateRepo(id RepoId) (*git.Repository, error) {
	path := store.idToPath(id)

	_, err := os.Stat(path)

	if err == nil {
		return nil, os.ErrExist
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	repo, err := git.InitBareRepository(path)
	if err != nil {
		return nil, err
	}

	gin := filepath.Join(path, "gin")
	os.Mkdir(gin, 0775) //TODO: what to do about errors?

	return repo, nil
}

func (store *RepoStore) ListRepos() ([]RepoId, error) {
	gitpath := store.gitPath()
	rdir, err := os.Open(gitpath)

	if err != nil {
		return nil, err
	}

	defer rdir.Close()

	entries, err := rdir.Readdir(-1)

	if err != nil {
		return nil, err
	}

	var repos []RepoId

	for _, entry := range entries {
		owner := entry.Name()

		odir, err := os.Open(filepath.Join(gitpath, owner))
		if err != nil {
			fmt.Fprintf(os.Stderr, "[W] error opening %q\n", owner)
			continue
		}

		repoInfos, err := store.ListReposForUser(owner)

		if err != nil {
			fmt.Fprintf(os.Stderr, "[W] %v", err)
		} else {
			repos = append(repos, repoInfos...)
		}

		odir.Close()
	}

	return repos, nil
}

func (store *RepoStore) ListReposForUser(uid string) ([]RepoId, error) {
	userpath := filepath.Join(store.gitPath(), uid)

	info, err := os.Stat(userpath)

	if err != nil {
		return nil, err
	} else if !info.IsDir() {
		return nil, fmt.Errorf("%q is not a directory as expected", userpath)
	}

	names, err := filepath.Glob(filepath.Join(userpath, "*.git"))

	var repos []RepoId
	for _, path := range names {

		if !strings.HasSuffix(path, ".git") {
			continue
		}
		base := filepath.Base(path)
		name := base[:len(base)-4]
		repos = append(repos, RepoId{uid, name})
	}

	return repos, nil
}

func (store *RepoStore) ListSharedRepos(uid string) ([]RepoId, error) {
	gitpath := store.gitPath()

	suffix := filepath.Join("gin", "sharing", uid)
	pattern := filepath.Join(gitpath, "*", "*.git", suffix)
	names, err := filepath.Glob(pattern)

	if err != nil {
		panic("Bad glob pattern!")
	}

	repos := make([]RepoId, len(names))

	for i, name := range names {
		fmt.Fprintf(os.Stderr, "[D] shared: %q\n", name)
		rid, err := RepoIdFromPath(name[:len(name)-(len(suffix)+1)])
		if err != nil {
			fmt.Fprintf(os.Stderr, "[W] could not parse repo id: %v", err)
			continue
		}

		repos[i] = rid
	}

	return repos, nil
}

func (store *RepoStore) OpenGitRepo(id RepoId) (*git.Repository, error) {
	path := store.idToPath(id)
	return git.OpenRepository(path)
}

func (store *RepoStore) GetRepoVisibility(id RepoId) (bool, error) {
	base := store.idToPath(id)
	path := filepath.Join(base, "gin", "public")

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (store *RepoStore) SetRepoVisibility(id RepoId, public bool) error {
	cur, err := store.GetRepoVisibility(id)

	if err != nil {
		return err
	}

	if cur == public {
		return nil
	}

	path := filepath.Join(store.idToPath(id), "gin", "public")
	if public {
		_, err := os.Create(path)
		if err != nil {
			return err
		}
		return nil
	}

	return os.Remove(path)
}

func NewRepoStore(basePath string) (*RepoStore, error) {
	store := RepoStore{Path: filepath.Join(basePath, "repos")}

	gitpath := filepath.Join(store.Path, "git")

	info, err := os.Stat(gitpath)

	if err != nil {

		if os.IsNotExist(err) {
			err = os.MkdirAll(gitpath, 0777)

			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%q is not a directory as expected", gitpath)
	}

	return &store, nil
}
