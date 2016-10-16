package git

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestWriteCommit(t *testing.T) {

	tree, _ := ParseSHA1("55fc4f1f438ee7f1299afa564e124834f7f7641f")
	parent, _ := ParseSHA1("07f2bbad7e34a1efcde59ebe230b0942cf7957b6")
	author := Signature{
		Name:   "Christian Kellner",
		Email:  "christian@kellner.me",
		Date:   time.Unix(1476441561, 0),
		Offset: time.FixedZone("+0200", 120),
	}

	committer := Signature{
		Name:   "Christian Kellner",
		Email:  "christian@kellner.me",
		Date:   time.Unix(1476448221, 0),
		Offset: time.FixedZone("+0200", 120),
	}

	c := Commit{
		gitObject: gitObject{otype: ObjCommit, size: 273},
		Tree:      tree,
		Parent:    []SHA1{parent},
		Author:    author,
		Committer: committer,
		Message:   "[git] annex: Astat() fix non-error condition\n",
	}

	h := sha1.New()
	_, err := c.WriteTo(h)
	if err != nil {
		t.Fatalf("Commit.WriteTo() => %v ", err)
	}

	x := fmt.Sprintf("%x", h.Sum(nil))
	y := "af69da9aad950cec69faeca2543498075f8e3804"

	if x != y {
		f, err := os.OpenFile(x, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0666)
		if err == nil {
			n, err := c.WriteTo(f)
			if err != nil {
				t.Logf("could not dump commit to file: %v", err)
			}
			err = f.Close()
			if err != nil {
				t.Logf("could not close file: %v", err)
			}
			t.Logf("dumped commit to file: %q [%d]", x, n)
		} else {

		}
		t.Fatalf("sha1(commit) => %q expected %q", x, y)
	}
}

func TestWriteTree(t *testing.T) {
	// a tree object, trust me
	raw := []byte{0x31, 0x30, 0x30, 0x36, 0x34, 0x34, 0x20, 0x47,
		0x2d, 0x4e, 0x6f, 0x64, 0x65, 0x0, 0xe6, 0x9d, 0xe2, 0x9b, 0xb2,
		0xd1, 0xd6, 0x43, 0x4b, 0x8b, 0x29, 0xae, 0x77, 0x5a, 0xd8, 0xc2,
		0xe4, 0x8c, 0x53, 0x91, 0x31, 0x30, 0x30, 0x36, 0x34, 0x34, 0x20,
		0x61, 0x6b, 0x0, 0xe6, 0x9d, 0xe2, 0x9b, 0xb2, 0xd1, 0xd6, 0x43,
		0x4b, 0x8b, 0x29, 0xae, 0x77, 0x5a, 0xd8, 0xc2, 0xe4, 0x8c, 0x53,
		0x91, 0x31, 0x30, 0x30, 0x36, 0x34, 0x34, 0x20, 0x63, 0x67, 0x0,
		0xe6, 0x9d, 0xe2, 0x9b, 0xb2, 0xd1, 0xd6, 0x43, 0x4b, 0x8b, 0x29,
		0xae, 0x77, 0x5a, 0xd8, 0xc2, 0xe4, 0x8c, 0x53, 0x91, 0x31, 0x30,
		0x30, 0x36, 0x34, 0x34, 0x20, 0x63, 0x6b, 0x0, 0xe6, 0x9d, 0xe2,
		0x9b, 0xb2, 0xd1, 0xd6, 0x43, 0x4b, 0x8b, 0x29, 0xae, 0x77, 0x5a,
		0xd8, 0xc2, 0xe4, 0x8c, 0x53, 0x91, 0x31, 0x30, 0x30, 0x36, 0x34,
		0x34, 0x20, 0x6d, 0x73, 0x0, 0xe6, 0x9d, 0xe2, 0x9b, 0xb2, 0xd1,
		0xd6, 0x43, 0x4b, 0x8b, 0x29, 0xae, 0x77, 0x5a, 0xd8, 0xc2, 0xe4,
		0x8c, 0x53, 0x91, 0x31, 0x30, 0x30, 0x36, 0x34, 0x34, 0x20, 0x74,
		0x77, 0x0, 0xe6, 0x9d, 0xe2, 0x9b, 0xb2, 0xd1, 0xd6, 0x43, 0x4b,
		0x8b, 0x29, 0xae, 0x77, 0x5a, 0xd8, 0xc2, 0xe4, 0x8c, 0x53, 0x91}

	r := bytes.NewReader(raw)

	tree := Tree{
		gitObject: gitObject{otype: ObjTree,
			size:   int64(len(raw)),
			source: ioutil.NopCloser(r)},
	}

	h := sha1.New()

	n, err := tree.WriteTo(h)

	if err != nil {
		t.Fatalf("Tree.WriteTo() => %v ", err)
	} else if n < int64(len(raw)) {
		t.Fatalf("Tree.WriteTo() wrote %d bytes, expected at least %d", n, len(raw))
	}

	x := fmt.Sprintf("%x", h.Sum(nil))
	y := "2e2c935aff57a62eceb380f43ed39a09651711ff"

	if x != y {
		t.Fatalf("sha1(tree) => %q expected %q", x, y)
	}
}

func TestWriteBlob(t *testing.T) {
	var b bytes.Buffer
	blob := Blob{
		gitObject: gitObject{otype: ObjBlob,
			size:   0,
			source: ioutil.NopCloser(&b)},
	}

	h := sha1.New()

	_, err := blob.WriteTo(h)

	if err != nil {
		t.Fatalf("Blob.WriteTo() => %v ", err)
	}

	x := fmt.Sprintf("%x", h.Sum(nil))
	y := "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391"

	if x != y {
		t.Fatalf("sha1(blob) => %q expected %q", x, y)
	}
}

func TestWriteTag(t *testing.T) {
	tobj, _ := ParseSHA1("cd119b179d4be4629d8a2e605a8386a7b6fc2afa")
	tag := Tag{
		gitObject: gitObject{
			otype:  ObjTag,
			size:   151,
			source: nil,
		},
		Object:  tobj,
		ObjType: ObjCommit,
		Tag:     "paper/jossa",
		Tagger:  "gin repo <gin-repo@g-node.org> 1476609894 +0200",
		Message: "Tag as paper/jossa",
	}

	h := sha1.New()

	var b bytes.Buffer
	mw := io.MultiWriter(h, &b)

	_, err := tag.WriteTo(mw)
	if err != nil {
		t.Fatalf("Tag.WriteTo() => %v ", err)
	}

	x := fmt.Sprintf("%x", h.Sum(nil))
	y := "84c011b574ae42832efab99acb782f5d716f5097"

	if x != y {
		t.Logf("[E] tag object proof:\n%s\n", b.String())
		t.Fatalf("sha1(tag) => %q expected %q", x, y)
	}
}
