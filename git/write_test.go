package git

import (
	"crypto/sha1"
	"fmt"
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
