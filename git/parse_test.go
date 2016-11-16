package git

import (
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
	"time"
)

var sigtests = []struct {
	in  string
	out *Signature
}{
	{"A U Thor <author@example.com> 1462210432 +0200", &Signature{Name: "A U Thor", Email: "author@example.com", Date: time.Unix(1462210432, 0), Offset: time.FixedZone("+0200", 7200)}},
	{"A U Thor <author@example.com 1462210432 +0200", nil},
	{"A U Thor author@example.com> 1462210432 +0200", nil},
	{"A U Thor <author@example.com> 1462210432 0200", nil},
	{"A U Thor <author@example.com> 1462210432", nil},
	{"A U Thor <author@example.com>", nil},
}

// String to create tags. they are without the
// [type][size][\0]
// part, such that they can be used by parseTag
var (
	fakeSignedTagTxt = `object 920514e51fdb27a8fcedb036391570788f5a6234
type commit
tag testTAg
tagger cgars <christian@stuebeweg50.de> 1479131540 +0100

uper beshcreibung
-----BEGIN PGP SIGNATURE-----
Version: GnuPG v1

iQIcBAABAgAGBQJYKcGeAAoJEHQliMt6qIEXo4IP/06SkTokaOEeBEjxMOvXQ5oZ
KOrljZ94TetPL5yreF98nCpdcaING4sKhnepOLUa4cZIvbexlIZmq5d8KXWcLF7l
uL/7aE1gViTZFRM/CYz2JhN8oQNaCSnBE30pnUfThkeDVr1ZPZAvV9QqEpvC2qgM
NdsQ+n7ymrf6ssUtwO4Ss/r49pM98dTwhCz4dTevb1G08y3JnkoKTSn98x8ow6XG
grpgCg62BKHeut0lRiIpkQ1zgJEbA8tpZ5Gozfzq1eYKjzeG+WkqA1+IHRE8jAqB
ZT2CqJAmagIyrxi4c2VgnawyRbXW59yPz7EBB5objLboJ/8xCaLQZIZ89xWFQIS+
dLqW4d3wuiwMbwrAgB7f/vMf+BZexHdoDu31s3+7PXSpKeojN95gHa+XvU8ZzJYd
nxY+6jHwhyWm6/Emi4tJI5CFk6vcIyQChe4K+yp21nVonqvHz7rAsbOWY0tkwspr
OYY+b2a5eyx83fA/CtGp3GcJ9UrxFfgDA6Ivu1qlHlZIQRXGzzG+Wvo8Th4M8aeJ
bJE4+eo3igMjzWCvY0in8z0Wv3iiWIN/oqo+coekTnwo1oeQC1gw13x+KqBgAHec
msV6PPWFlxJbL+QEbGIrHoKW7jSXaVzbDBEZIpYZ2WCP+0NXJPADG0kw3swU08dk
tW32G+ZLbLoRy6aRneO7
=BajW
-----END PGP SIGNATURE-----
`
	FakeUnsignedTagTxt = `object 920514e51fdb27a8fcedb036391570788f5a6234
type commit
tag tag3
tagger cgars <christian@stuebeweg50.de> 1479220880 +0100

some annotation for tag3
`
	tobj, _       = ParseSHA1("920514e51fdb27a8fcedb036391570788f5a6234")
	FakeSignedTag = Tag{
		gitObject: gitObject{
			otype:  ObjTag,
			size:   967,
			source: nil,
		},
		Object:  tobj,
		ObjType: ObjCommit,
		Tag:     "testTAg",
		Tagger: Signature{
			Name:   "cgars",
			Email:  "christian@stuebeweg50.de",
			Date:   time.Unix(1479131540, 0),
			Offset: time.FixedZone("+0100", 120),
		},
		Message: "super beschreibung\n",
		GPGSig: `-----BEGIN PGP SIGNATURE-----
Version: GnuPG v1

iQIcBAABAgAGBQJYKcGeAAoJEHQliMt6qIEXo4IP/06SkTokaOEeBEjxMOvXQ5oZ
KOrljZ94TetPL5yreF98nCpdcaING4sKhnepOLUa4cZIvbexlIZmq5d8KXWcLF7l
uL/7aE1gViTZFRM/CYz2JhN8oQNaCSnBE30pnUfThkeDVr1ZPZAvV9QqEpvC2qgM
NdsQ+n7ymrf6ssUtwO4Ss/r49pM98dTwhCz4dTevb1G08y3JnkoKTSn98x8ow6XG
grpgCg62BKHeut0lRiIpkQ1zgJEbA8tpZ5Gozfzq1eYKjzeG+WkqA1+IHRE8jAqB
ZT2CqJAmagIyrxi4c2VgnawyRbXW59yPz7EBB5objLboJ/8xCaLQZIZ89xWFQIS+
dLqW4d3wuiwMbwrAgB7f/vMf+BZexHdoDu31s3+7PXSpKeojN95gHa+XvU8ZzJYd
nxY+6jHwhyWm6/Emi4tJI5CFk6vcIyQChe4K+yp21nVonqvHz7rAsbOWY0tkwspr
OYY+b2a5eyx83fA/CtGp3GcJ9UrxFfgDA6Ivu1qlHlZIQRXGzzG+Wvo8Th4M8aeJ
bJE4+eo3igMjzWCvY0in8z0Wv3iiWIN/oqo+coekTnwo1oeQC1gw13x+KqBgAHec
msV6PPWFlxJbL+QEbGIrHoKW7jSXaVzbDBEZIpYZ2WCP+0NXJPADG0kw3swU08dk
tW32G+ZLbLoRy6aRneO7
=BajW
-----END PGP SIGNATURE-----`,
	}
	FakeUnsignedTag = Tag{
		gitObject: gitObject{
			otype:  ObjTag,
			size:   967,
			source: nil,
		},
		Object:  tobj,
		ObjType: ObjCommit,
		Tag:     "tag3",
		Tagger: Signature{
			Name:   "cgars",
			Email:  "christian@stuebeweg50.de",
			Date:   time.Unix(1479131540, 0),
			Offset: time.FixedZone("+0100", 120),
		},
		Message: "some annotation for tag3\n",
	}
)

func TestParseSignature(t *testing.T) {
	for _, tt := range sigtests {
		out, err := parseSignature(tt.in)

		switch {
		case tt.out == nil && err == nil:
			t.Errorf("parseSignature(%q) => success, want an error", tt.in)
		case tt.out != nil && err != nil:
			t.Errorf("parseSignature(%q) => error: %v, wanted success", tt.in, err)

		//Not an error condition
		case tt.out == nil && err != nil:
			t.Logf("parseSignature(%q) => error [OK!]", tt.in)
		//must be: out != nil, err === nil
		case !reflect.DeepEqual(out, *tt.out):
			t.Errorf("parseSignature(%q) => %v, want %q", tt.in, out, tt.out)
			//correct parsing and objects match
		default:
			t.Logf("parseSignature(%q) => %q [OK!]", tt.in, tt.out)
		}

	}
}

func TestParseTag(t *testing.T) {
	t.Log("Test Parse Signed Tag")
	tagReader := strings.NewReader(fakeSignedTagTxt)
	tagRC := ioutil.NopCloser(tagReader)
	fakeGitObject := gitObject{ObjTag, tagReader.Size(), tagRC}
	tag, err := parseTag(fakeGitObject)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if tag.Message != FakeSignedTag.Message &&
		tag.GPGSig != FakeSignedTag.GPGSig &&
		tag.Tag != FakeSignedTag.Tag &&
		tag.Tagger != FakeSignedTag.Tagger &&
		tag.Object != FakeSignedTag.Object &&
		tag.ObjType != FakeSignedTag.ObjType {
		t.Logf("[E] tag returned:\n%s\n", tag)
		t.Logf("[E] tag expected:\n%s\n", FakeSignedTag)
		t.Fail()
	}
	t.Log("Parse Signed Tag [OK!]")

	t.Log("Test Parse Unsigned Tag")
	tagReader = strings.NewReader(FakeUnsignedTagTxt)
	tagRC = ioutil.NopCloser(tagReader)
	fakeGitObject = gitObject{ObjTag, tagReader.Size(), tagRC}
	tag, err = parseTag(fakeGitObject)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if tag.Message != FakeUnsignedTag.Message &&
		tag.GPGSig != FakeUnsignedTag.GPGSig &&
		tag.Tag != FakeUnsignedTag.Tag &&
		tag.Tagger != FakeUnsignedTag.Tagger &&
		tag.Object != FakeUnsignedTag.Object &&
		tag.ObjType != FakeUnsignedTag.ObjType {
		t.Logf("[E] tag returned:\n%s\n", tag)
		t.Logf("[E] tag expected:\n%s\n", FakeSignedTag)
		t.Fail()
	}
	t.Log("Parse Unsigned Tag [OK!]")

}
