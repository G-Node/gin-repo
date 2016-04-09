package git

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

//SHA1 is the object identifying checksum of
// the object data
type SHA1 [20]byte

func (oid SHA1) String() string {
	return hex.EncodeToString(oid[:])
}

func ParseSHA1(input string) (sha SHA1, err error) {
	data, err := hex.DecodeString(strings.Trim(input, " \n"))
	if err != nil {
		return
	} else if len(data) != 20 {
		err = fmt.Errorf("git: sha1 must be 20 bytes")
		return
	}

	copy(sha[:], data)
	return
}

type ObjectType byte

const (
	ObjCommit = 1
	ObjTree   = 2
	ObjBlob   = 3
	ObjTag    = 4

	ObjOFSDelta = 0x6
	OBjRefDelta = 0x7
)

func ParseObjectType(s string) (ObjectType, error) {
	s = strings.Trim(s, "\n ")
	switch s {
	case "commit":
		return ObjCommit, nil
	case "tree":
		return ObjTree, nil
	case "blob":
		return ObjBlob, nil
	case "tag":
		return ObjBlob, nil
	}

	return ObjectType(0), fmt.Errorf("git: unknown object: %q", s)
}

func (ot ObjectType) String() string {
	switch ot {
	case ObjCommit:
		return "commit"
	case ObjTree:
		return "tree"
	case ObjBlob:
		return "blob"
	case ObjTag:
		return "tag"
	case ObjOFSDelta:
		return "delta-ofs"
	case OBjRefDelta:
		return "delta-ref"
	}
	return "unknown"
}

type Object interface {
	Type() ObjectType
	Size() int64

	io.Closer
}

type gitObject struct {
	otype ObjectType
	size  int64

	source io.ReadCloser
}

func (o *gitObject) Type() ObjectType {
	return o.otype
}

func (o *gitObject) Size() int64 {
	return o.size
}

func (o *gitObject) Close() error {
	if o.source == nil {
		return nil
	}
	return o.source.Close()
}

type Commit struct {
	gitObject

	Tree      SHA1
	Parent    SHA1
	Author    string
	Committer string
	Message   string
}

type TreeEntry struct {
	Mode os.FileMode
	Type ObjectType
	ID   SHA1
	Name string
}

type Tree struct {
	gitObject

	entry *TreeEntry
	err   error
}

func (tree *Tree) Next() bool {
	tree.entry, tree.err = ParseTreeEntry(tree.source)
	return tree.err == nil
}

func (tree *Tree) Err() error {
	if err := tree.err; err != nil && err != io.EOF {
		return err
	}

	return nil
}

func (tree *Tree) Entry() *TreeEntry {
	return tree.entry
}

type Blob struct {
	gitObject
}

func (b *Blob) Read(data []byte) (n int, err error) {
	n, err = b.source.Read(data)
	return
}

type Tag struct {
	gitObject

	Object  SHA1
	ObjType ObjectType
	Tagger  string
	Message string
}

type DeltaOfs struct {
	gitObject

	Offset int64
}

type DeltaRef struct {
	gitObject

	Base SHA1
}
