package git

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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

//AnnexKey represents an annex key. Key, Backend and Keyname
// fields are guaranteed to be there. Presence of other fields
// depends on the used backend.
type AnnexKey struct {
	Key      string
	Backend  string
	Bytesize int64
	Keyname  string
	MTime    *time.Time

	hash [16]byte
}

//HashDirLower is the new key hash format. It uses two directories,
// each dir name consisting of three lowercase letters.
// (c.f. http://git-annex.branchable.com/internals/hashing/)
func (key *AnnexKey) HashDirLower() string {
	hs := hex.EncodeToString(key.hash[:])
	return string(hs[:3]) + string(os.PathSeparator) +
		string(hs[3:6]) + string(os.PathSeparator)
}

const (
	annexKeyChars = "0123456789zqjxkmvwgpfZQJXKMVWGPF"
)

//HashDirMixed is the old key hash format. It uses two directories,
// each dir name consisting of two letters.
// (c.f. http://git-annex.branchable.com/internals/hashing/)
func (key *AnnexKey) HashDirMixed() string {
	var l [4]string
	w := binary.LittleEndian.Uint32(key.hash[:4])
	for i := uint(0); i < 4; i++ {
		l[i] = string(annexKeyChars[int(w>>(6*i)&0x1F)])
	}
	return l[1] + l[0] + string(os.PathSeparator) +
		l[3] + l[2] + string(os.PathSeparator)
}

//AnnexExamineKey parses the key and extracts all available information
//from the key string. See AnnexKey for more details.
//(cf. http://git-annex.branchable.com/internals/key_format/)
func AnnexExamineKey(keystr string) (*AnnexKey, error) {

	key := AnnexKey{Key: keystr, hash: md5.Sum([]byte(keystr))}

	front, name := split2(keystr, "--")
	key.Keyname = name

	parts := strings.Split(front, "-")

	if len(parts) < 1 {
		// key error
		return nil, fmt.Errorf("git: bad annex key (need backend--name)")
	}

	key.Backend = parts[0]
	for i := 1; i < len(parts); i++ {
		part := parts[i]

		if len(part) < 1 {
			continue
		}

		k, v := part[0], part[1:]

		switch k {
		case 's':
			i, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				continue
			}
			key.Bytesize = i
		case 'm':
			i, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				continue
			}
			t := time.Unix(i, 0)
			key.MTime = &t
		}
	}

	return &key, nil
}

//IsAnnexFile returns true if the file at path is
//managed by git annex, false otherwise. Does not check
//if the file is actually present
func IsAnnexFile(path string) bool {
	return strings.HasPrefix(path, ".git/annex")
}

type AnnexStat struct {
	Name string
	Size int64
	Have bool
}

func (repo *Repository) Astat(target string) (*AnnexStat, error) {

	sbuf := AnnexStat{Name: filepath.Base(target)}
	ki, err := AnnexExamineKey(sbuf.Name)

	if err != nil {
		return nil, err
	}

	// we are in a bare repository, therefore we use hasdirlower
	p := filepath.Join(repo.Path, "annex", "objects", ki.HashDirLower(), ki.Key, ki.Key)
	fi, err := os.Stat(p)

	if err != nil {
		sbuf.Have = true
		sbuf.Size = fi.Size()
	} else if os.IsNotExist(err) {
		sbuf.Have = false
	} else if err != nil {
		return nil, err
	}

	return &sbuf, nil
}
