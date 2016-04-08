package git

import (
	"bytes"
	"io"
	"strings"
)

func readUntilNul(r io.Reader) (string, error) {
	buf := bytes.NewBuffer(make([]byte, 0))
	for {
		var b [1]byte
		_, err := r.Read(b[:])
		if err != nil {
			return "", err
		} else if b[0] == 0 {
			break
		}
		buf.WriteByte(b[0])
	}

	return buf.String(), nil
}

func split2(s, sep string) (head, tail string) {
	comps := strings.SplitN(s, sep, 2)
	head = comps[0]
	if len(comps) > 1 {
		tail = comps[1]
	}
	return
}
