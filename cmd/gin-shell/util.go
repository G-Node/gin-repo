package main

import (
	"bytes"
	"unicode"
)

func splitarg(args string) []string {
	var out []string
	q := rune(0)

	buf := bytes.NewBufferString("")
	for _, c := range args {
		switch {
		case q != rune(0):
			if c == q {
				q = rune(0)
			} else {
				buf.WriteRune(c)
			}
		case unicode.In(c, unicode.Quotation_Mark):
			q = c
		case unicode.IsSpace(c):
			if buf.Len() > 0 {
				out = append(out, buf.String())
				buf.Reset()
			}
		default:
			buf.WriteRune(c)
		}
	}

	if buf.Len() > 0 {
		out = append(out, buf.String())
	}

	return out
}

func head(args []string) string {
	if len(args) < 1 {
		return ""
	}
	return args[0]
}
