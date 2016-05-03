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
	"time"
)

func parseSignature(line string) (Signature, error) {
	//Format: "<name> <email> <unix timestamp> <time zone offset>"
	//i.e. "A U Thor <author@example.com> 1462210432 +0200"

	u := Signature{}

	//<name>
	start := strings.Index(line, " <")
	if start == -1 {
		return u, fmt.Errorf("invalid signature format")
	}
	u.Name = line[:start]

	//<email>
	end := strings.Index(line, "> ")
	if end == -1 {
		return u, fmt.Errorf("invalid signature format")
	}
	u.Email = line[start+2 : end]

	//<unix timestamp>
	tstr, off := split2(line[end+2:], " ")
	i, err := strconv.ParseInt(tstr, 10, 64)

	if err != nil || len(off) != 5 {
		return u, fmt.Errorf("invalid signature time format")
	}
	u.Date = time.Unix(i, 0)

	//<time zone offset>
	h, herr := strconv.Atoi(off[1:3])
	m, merr := strconv.Atoi(off[3:])

	if herr != nil || merr != nil {
		return u, fmt.Errorf("invalid signature offset format")
	}

	o := (h*60 + m) * 60

	if off[0] == '-' {
		o *= -1
	}

	u.Offset = time.FixedZone(off, o)

	return u, nil
}

func openRawObject(path string) (gitObject, error) {
	fd, err := os.Open(path)
	if err != nil {
		return gitObject{}, err
	}

	// we wrap the zlib reader below, so it will be
	// propery closed
	r, err := zlib.NewReader(fd)
	if err != nil {
		return gitObject{}, fmt.Errorf("git: could not create zlib reader: %v", err)
	}

	// general object format is
	// [type][space][length {ASCII}][\0]

	line, err := readUntilNul(r)
	if err != nil {
		return gitObject{}, err
	}

	tstr, lstr := split2(line, " ")
	size, err := strconv.ParseInt(lstr, 10, 64)

	if err != nil {
		return gitObject{}, fmt.Errorf("git: object parse error: %v", err)
	}

	otype, err := ParseObjectType(tstr)
	if err != nil {
		return gitObject{}, err
	}

	obj := gitObject{otype, size, r}
	obj.wrapSource(r)

	return obj, nil
}

func parseObject(obj gitObject) (Object, error) {
	switch obj.otype {
	case ObjCommit:
		return parseCommit(obj)

	case ObjTree:
		return parseTree(obj)

	case ObjBlob:
		return parseBlob(obj)

	case ObjTag:
		return parseTag(obj)
	}

	obj.Close()
	return nil, fmt.Errorf("git: unsupported object")
}

func parseCommit(obj gitObject) (*Commit, error) {
	c := &Commit{gitObject: obj}

	lr := &io.LimitedReader{R: obj.source, N: obj.size}
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
			parent, err := ParseSHA1(tail)
			if err == nil {
				c.Parent = append(c.Parent, parent)
			}
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

func parseTree(obj gitObject) (*Tree, error) {
	tree := Tree{obj, nil, nil}
	return &tree, nil
}

func parseTreeEntry(r io.Reader) (*TreeEntry, error) {
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

func parseBlob(obj gitObject) (*Blob, error) {
	blob := &Blob{obj}
	return blob, nil
}

func parseTag(obj gitObject) (*Tag, error) {
	c := &Tag{gitObject: obj}

	lr := &io.LimitedReader{R: c.source, N: c.size}
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
