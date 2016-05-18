package git

import (
	"bytes"
	"testing"
)

var rdtest = []struct {
	in  []byte
	out int64
	err bool
}{
	{[]byte{0x0}, 0, false},
	{[]byte{0x7F}, 127, false},
	{[]byte{0x80, 0x0}, 128, false},
	{[]byte{0XFE, 0XFE, 0XFE, 0XFE, 0XFE, 0XFE, 0XFE, 0X7F}, 72057594037927935, false},
	{[]byte{0XFE, 0XFE, 0XFE, 0XFE, 0XFE, 0XFE, 0XFF, 0}, 72057594037927936, false},
	{[]byte{0XFE, 0XFE, 0XFE, 0XFE, 0XFE, 0XFE, 0XFE, 0XFE, 0X7F}, 9223372036854775807, false}, // last valid int64 ([0111 1111 1111 ...1111]
	{[]byte{0XFE, 0XFE, 0XFE, 0XFE, 0XFE, 0XFE, 0XFE, 0XFF, 0}, 0, true},                       // [1000 ...0000] (int64) would be -9223372036854775808
}

func TestReadVarint(t *testing.T) {
	for _, tt := range rdtest {
		r := bytes.NewReader(tt.in)
		out, err := readVarint(r)

		switch {
		case err == nil && tt.err:
			t.Errorf("readVarint(%#v) => %d, wanted overflow error", tt.in, out)

		case err != nil && !tt.err:
			t.Errorf("readVarint(%#v) => error: %v, wanted %d", tt.in, err, tt.out)

		//Not an error condition
		case err != nil && tt.err:
			t.Logf("readVarint(%#v) => error %v [OK!]", tt.in, err)

		case out != tt.out: // err == nil && !tt.err, i.e. results must match
			t.Errorf("readVarint(%#v) => %d, wanted %d", tt.in, out, tt.out)

		default:
			t.Logf("readVarint(%#v) => %d [OK!]", tt.in, out)
		}
	}
}
