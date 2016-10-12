package git

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

//HasAnnex returns true if the repository has git-annex initialized.
func (repo *Repository) HasAnnex() bool {
	d := filepath.Join(repo.Path, "annex")
	s, err := os.Stat(d)
	return err == nil && s.IsDir()
}

//InitAnnex initializes git-annex support for a repository.
func (repo *Repository) InitAnnex() error {
	cmd := exec.Command("git", fmt.Sprintf("--git-dir=%s", repo.Path), "annex", "init", "gin")
	body, err := cmd.Output()

	if err != nil {
		return fmt.Errorf("git: init annex failed: %q", string(body))
	}

	return nil
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
