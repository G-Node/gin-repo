package git

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//Repository represents an on disk git repository.
type Repository struct {
	Path string
}

//InitBareRepository creates a bare git repository at path.
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

//IsBareRepository checks if path is a bare git repository.
func IsBareRepository(path string) bool {

	cmd := exec.Command("git", fmt.Sprintf("--git-dir=%s", path), "rev-parse", "--is-bare-repository")
	body, err := cmd.Output()

	if err != nil {
		return false
	}

	status := strings.Trim(string(body), "\n ")
	return status == "true"
}

//OpenRepository opens the repository at path. Currently
//verifies that it is a (bare) repository and returns an
//error if the check fails.
func OpenRepository(path string) (*Repository, error) {

	path, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("git: could not determine absolute path")
	}

	if !IsBareRepository(path) {
		return nil, fmt.Errorf("git: not a bare repository")
	}

	return &Repository{Path: path}, nil
}

//DiscoverRepository returns the git repository that contains the
//current working directory, or and error if the current working
//dir does not lie inside one.
func DiscoverRepository() (*Repository, error) {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	data, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	path := strings.Trim(string(data), "\n ")
	return &Repository{Path: path}, nil
}

//ReadDescription returns the contents of the description file.
func (repo *Repository) ReadDescription() string {
	path := filepath.Join(repo.Path, "description")

	dat, err := ioutil.ReadFile(path)
	if err != nil {
		return ""
	}

	return string(dat)
}

//WriteDescription writes the contents of the description file.
func (repo *Repository) WriteDescription(description string) error {
	path := filepath.Join(repo.Path, "description")

	// not atomic, fine for now
	return ioutil.WriteFile(path, []byte(description), 0666)
}

//OpenObject returns the git object for a give id (SHA1).
func (repo *Repository) OpenObject(id SHA1) (Object, error) {
	obj, err := repo.openRawObject(id)

	if err != nil {
		return nil, err
	}

	if IsStandardObject(obj.otype) {
		return parseObject(obj)
	}

	//not a standard object, *must* be a delta object,
	// we know of no other types
	if !IsDeltaObject(obj.otype) {
		return nil, fmt.Errorf("git: unsupported object")
	}

	delta, err := parseDelta(obj)
	if err != nil {
		return nil, err
	}

	chain, err := buildDeltaChain(delta, repo)

	if err != nil {
		return nil, err
	}

	//TODO: check depth, and especially expected memory usage
	// beofre actually patching it

	return chain.resolve()
}

func (repo *Repository) openRawObject(id SHA1) (gitObject, error) {
	idstr := id.String()
	opath := filepath.Join(repo.Path, "objects", idstr[:2], idstr[2:])

	obj, err := openRawObject(opath)

	if err == nil {
		return obj, nil
	} else if err != nil && !os.IsNotExist(err) {
		return obj, err
	}

	indicies := repo.loadPackIndices()

	for _, f := range indicies {

		idx, err := PackIndexOpen(f)
		if err != nil {
			continue
		}

		//TODO: we should leave index files open,
		defer idx.Close()

		off, err := idx.FindOffset(id)

		if err != nil {
			continue
		}

		pf, err := idx.OpenPackFile()
		if err != nil {
			return gitObject{}, err
		}

		obj, err := pf.readRawObject(off)

		if err != nil {
			return gitObject{}, err
		}

		return obj, nil
	}

	// from inspecting the os.isNotExist source it
	// seems that if we have "not found" in the message
	// os.IsNotExist() report true, which is what we want
	return gitObject{}, fmt.Errorf("git: object not found")
}

func (repo *Repository) loadPackIndices() []string {
	target := filepath.Join(repo.Path, "objects", "pack", "*.idx")
	files, err := filepath.Glob(target)

	if err != nil {
		panic(err)
	}

	return files
}

//OpenRef returns the Ref with the given name or an error
//if either no maching could be found or in case the match
//was not unique.
func (repo *Repository) OpenRef(name string) (Ref, error) {

	if name == "HEAD" {
		return repo.parseRef("HEAD")
	}

	matches := repo.listRefWithName(name)

	//first search in local heads
	var locals []Ref
	for _, v := range matches {
		if IsBranchRef(v) {
			if name == v.Fullname() {
				return v, nil
			}
			locals = append(locals, v)
		}
	}

	// if we find a single local match
	// we return it directly
	if len(locals) == 1 {
		return locals[0], nil
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("git: ref matching %q not found", name)
	case 1:
		return matches[0], nil
	}
	return nil, fmt.Errorf("git: ambiguous ref name, multiple matches")
}

//Readlink returns the destination of a symbilc link blob object
func (repo *Repository) Readlink(id SHA1) (string, error) {

	b, err := repo.OpenObject(id)
	if err != nil {
		return "", err
	}

	if b.Type() != ObjBlob {
		return "", fmt.Errorf("id must point to a blob")
	}

	blob := b.(*Blob)

	//TODO: check size and don't read unreasonable large blobs
	data, err := ioutil.ReadAll(blob)

	if err != nil {
		return "", err
	}

	return string(data), nil
}
