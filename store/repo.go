package store

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/G-Node/gin-repo/git"
)

var idChecker *regexp.Regexp

func init() {
	idChecker = regexp.MustCompile("^(?:/~/|/)?([0-9a-zA-Z][0-9a-zA-Z._-]{2,})/([0-9a-zA-Z][0-9a-zA-Z._@+-]*?)(?:.git)?(?:/)?$")
}

type RepoId struct {
	Owner string
	Name  string
}

func (id RepoId) String() string {
	return path.Join(id.Owner, id.Name)
}

func RepoIdParse(str string) (RepoId, error) {

	res := idChecker.FindStringSubmatch(str)

	if res == nil || len(res) != 3 {
		return RepoId{}, fmt.Errorf("malformed repository id")
	}

	return RepoId{res[1], res[2]}, nil
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

type AccessLevel int

const (
	NoAccess    = 0
	PullAccess  = 1
	PushAccess  = 2
	AdminAccess = 3
	OwnerAccess = 4
)

func (level AccessLevel) String() string {
	switch level {

	case PullAccess:
		return "can-pull"
	case PushAccess:
		return "can-push"
	case AdminAccess:
		return "is-admin"
	case OwnerAccess:
		return "is-owner"
	}

	return "no-access"
}

func ParseAccessLevel(str string) (AccessLevel, error) {
	clean := strings.Trim(str, " \n")
	switch clean {
	case "no-access":
		return NoAccess, nil
	case "can-pull":
		return PullAccess, nil
	case "can-push":
		return PushAccess, nil
	case "is-admin":
		return AdminAccess, nil
	case "is-owner":
		return OwnerAccess, nil

	}

	return NoAccess, fmt.Errorf("unknown access level: %q", clean)
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

func (store *RepoStore) ListPublicRepos() ([]RepoId, error) {
	gitpath := store.gitPath()

	suffix := filepath.Join("gin", "public")
	pattern := filepath.Join(gitpath, "*", "*.git", suffix)
	names, err := filepath.Glob(pattern)

	if err != nil {
		panic("Bad glob pattern!")
	}

	repos := make([]RepoId, len(names))

	for i, name := range names {
		fmt.Fprintf(os.Stderr, "[D] public: %q\n", name)
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

func (store *RepoStore) SetAccessLevel(id RepoId, user string, level AccessLevel) error {
	if id.Owner == user {
		return fmt.Errorf("cannot set access level for owner")
	}

	//TODO: check user name
	path := filepath.Join(store.idToPath(id), "gin", "sharing", user)

	if level == NoAccess {
		err := os.Remove(path)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	err := ioutil.WriteFile(path, []byte(level.String()), 0664)
	return err
}

func (store *RepoStore) readAccessLevel(id RepoId, user string) (AccessLevel, error) {

	if user == "" {
		return NoAccess, nil
	}

	path := filepath.Join(store.idToPath(id), "gin", "sharing", user)

	data, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return NoAccess, nil
	} else if err != nil {
		return NoAccess, err
	}

	level, err := ParseAccessLevel(string(data))
	if err != nil {
		return NoAccess, err
	}

	return level, nil
}

func (store *RepoStore) GetAccessLevel(id RepoId, user string) (AccessLevel, error) {

	if id.Owner == user {
		return OwnerAccess, nil
	}

	level, err := store.readAccessLevel(id, user)
	if err != nil {
		//what now? besides logging it?
		fmt.Fprintf(os.Stderr, "error reading access level: %v", err)
	}

	// if we got any level other then NoAccess, which is the lowest,
	// then we are done. Otherwise, if the repo is public we could
	// still get PullAccess, the next higher one, so check for that.
	if level != NoAccess {
		return level, nil
	}

	public, err := store.GetRepoVisibility(id)
	if err != nil {
		return NoAccess, err
	} else if public {
		return PullAccess, nil
	}
	return NoAccess, nil
}

func (store *RepoStore) ListSharedAccess(id RepoId) (map[string]AccessLevel, error) {
	path := filepath.Join(store.idToPath(id), "gin", "sharing")

	dir, err := os.Open(path)

	if os.IsNotExist(err) {
		return make(map[string]AccessLevel), nil
	} else if err != nil {
		return nil, err
	}

	names, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, err
	}

	accessMap := make(map[string]AccessLevel)
	for _, name := range names {
		level, err := store.GetAccessLevel(id, name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[W] could not get level for %s\n", name)
			continue
		}

		accessMap[name] = level
	}

	return accessMap, nil
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
