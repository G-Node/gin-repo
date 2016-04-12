package git

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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

	cmd := exec.Command("git", fmt.Sprintf("--git-dir=%s", path), "rev-parse", "--is-bare-repository")
	body, err := cmd.Output()

	if err != nil {
		return false
	}

	status := strings.Trim(string(body), "\n ")
	return status == "true"
}

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

func (repo *Repository) ReadDescription() string {
	path := filepath.Join(repo.Path, "description")

	dat, err := ioutil.ReadFile(path)
	if err != nil {
		return ""
	}

	return string(dat)
}

func (repo *Repository) WriteDescription(description string) error {
	path := filepath.Join(repo.Path, "description")

	// not atomic, fine for now
	return ioutil.WriteFile(path, []byte(description), 0666)
}

func (repo *Repository) HasAnnex() bool {
	d := filepath.Join(repo.Path, "annex")
	s, err := os.Stat(d)
	return err == nil && s.IsDir()
}

func (repo *Repository) InitAnnex() error {
	cmd := exec.Command("git", fmt.Sprintf("--git-dir=%s", repo.Path), "annex", "init", "gin")
	body, err := cmd.Output()

	if err != nil {
		return fmt.Errorf("git: init annex failed: %q", string(body))
	}

	return nil
}

func (repo *Repository) OpenObject(id SHA1) (Object, error) {
	idstr := id.String()
	opath := filepath.Join(repo.Path, "objects", idstr[:2], idstr[2:])

	obj, err := OpenObject(opath)

	if err == nil {
		return obj, nil
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	indicies := repo.loadPackIndices()

	for _, f := range indicies {

		idx, err := PackIndexOpen(f)
		if err != nil {
			continue
		}

		obj, err := idx.OpenObject(id)
		if err == nil {
			return obj, nil
		}
	}

	// from inspecting the os.isNotExist source it
	// seems that if we have "not found" in the message
	// os.IsNotExist() report true, which is what we want
	return nil, fmt.Errorf("git: object not found")
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

//AnnexKeyInfo corresponds to the output for git annex examinekey
//see Repository.AnnexExamineKey
type AnnexKeyInfo struct {
	Key          string
	Backend      string
	Bytesize     string //should be int
	Humansize    string
	Keyname      string
	Hashdirlower string
	Hashdirmixed string
}

func (repo *Repository) AnnexExamineKey(name string) (AnnexKeyInfo, error) {
	gdir := fmt.Sprintf("--git-dir=%s", repo.Path)
	cmd := exec.Command("git", gdir, "annex", "examinekey", name, "--json")
	body, err := cmd.Output()

	var info AnnexKeyInfo
	if err != nil {
		return info, err
	}

	err = json.Unmarshal(body, &info)
	if err != nil {
		return info, err
	}

	return info, nil
}
