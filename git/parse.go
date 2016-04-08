package git

import (
	"bufio"
	"compress/zlib"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

func OpenObject(path string) (Object, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	//TODO: we "leak" fd here, because we
	// will pass r into the ParseXXX functions
	// and clients will close the zlib reader but
	// fd will not be closed until garbage collected
	r, err := zlib.NewReader(fd)
	if err != nil {
		return nil, fmt.Errorf("git: could not create zlib reader: %v", err)
	}

	// general object format is
	// [type][space][length {ASCII}][\0]

	line, err := readUntilNul(r)
	if err != nil {
		return nil, err
	}

	tstr, lstr := split2(line, " ")
	size, err := strconv.ParseInt(lstr, 10, 64)

	if err != nil {
		return nil, fmt.Errorf("git: object parse error: %v", err)
	}

	switch tstr {
	case "tree":
		return ParseTree(r, size)

	case "commit":
		obj, err := ParseCommit(r, size)
		r.Close()
		fd.Close() // hmm ignoring the errors here ..
		return obj, err

	case "blob":
		return ParseBlob(r, size)

	case "tag":
		return ParseTag(r, size)

	}

	r.Close()
	fd.Close()

	return nil, fmt.Errorf("git: unsupported object")
}

func ParseCommit(r io.Reader, size int64) (Object, error) {
	c := &Commit{gitObject: gitObject{ObjCommit, size}}

	lr := &io.LimitedReader{R: r, N: size}
	br := bufio.NewReader(lr)

	var err error
	for {
		var l string
		l, err = br.ReadString('\n')
		head, tail := split2(l, " ")

		switch head {
		case "tree":
			c.Tree, err = ParseSHA1(tail)
		case "parent":
			c.Parent, err = ParseSHA1(tail)
		case "author":
			c.Author = strings.Trim(tail, "\n")
		case "committer":
			c.Committer = strings.Trim(tail, "\n")
		}

		if err != nil || head == "\n" {
			break
		}
	}

	if err != nil && err != io.EOF {
		return nil, err
	}

	data, err := ioutil.ReadAll(br)

	if err != nil {
		return nil, err
	}

	c.Message = string(data)
	return c, nil
}

func ParseTree(rc io.ReadCloser, size int64) (Object, error) {
	tree := Tree{gitObject{ObjCommit, size}, nil, nil, rc}
	return &tree, nil
}

func ParseTreeEntry(r io.Reader) (*TreeEntry, error) {
	//format is: [mode{ASCII, octal}][space][name][\0][SHA1]
	entry := &TreeEntry{}

	l, err := readUntilNul(r) // read until \0

	if err != nil {
		return nil, err
	}

	mstr, name := split2(l, " ")
	mode, err := strconv.ParseUint(mstr, 8, 32)
	if err != nil {
		return nil, err
	}

	//TODO: this is not correct because
	// we need to shift the "st_mode" file
	// info bits by 16
	entry.Mode = os.FileMode(mode)

	if entry.Mode == 040000 {
		entry.Type = ObjTree
	} else {
		entry.Type = ObjBlob
	}

	entry.Name = name

	n, err := r.Read(entry.ID[:])

	if err != nil && err != io.EOF {
		return nil, err
	} else if err == io.EOF && n != 20 {
		return nil, fmt.Errorf("git: unexpected EOF")
	}

	return entry, nil
}

func ParseBlob(r io.Reader, size int64) (Object, error) {
	blob := &Blob{gitObject{ObjBlob, size}, r}
	return blob, nil
}

func ParseTag(r io.Reader, size int64) (Object, error) {
	c := &Tag{gitObject: gitObject{ObjCommit, size}}

	lr := &io.LimitedReader{R: r, N: size}
	br := bufio.NewReader(lr)

	var err error
	for {
		var l string
		l, err = br.ReadString('\n')
		head, tail := split2(l, " ")

		switch head {
		case "object":
			c.Object, err = ParseSHA1(tail)
		case "type":
			c.ObjType, err = ParseObjectType(tail)
		case "tagger":
			c.Tagger = strings.Trim(tail, "\n")
		}

		if err != nil || head == "\n" {
			break
		}
	}

	if err != nil && err != io.EOF {
		return nil, err
	}

	data, err := ioutil.ReadAll(br)

	if err != nil {
		return nil, err
	}

	c.Message = string(data)
	return c, nil
}
