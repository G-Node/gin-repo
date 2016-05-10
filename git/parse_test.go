package git

import (
	"reflect"
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
