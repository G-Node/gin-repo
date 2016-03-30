package main

import "unicode"

func splitarg(args string) []string {
	var out []string
	q := rune(0)
	s := 0
	for i, c := range args {
		switch {
		case q != rune(0):
			if c == q {
				q = rune(0)
			}
		case unicode.In(c, unicode.Quotation_Mark):
			q = c
		case unicode.IsSpace(c):
			if sub := args[s:i]; sub != "" {
				out = append(out, args[s:i])
			}
			s = i + 1
		}
	}

	if sub := args[s:]; sub != "" {
		out = append(out, args[s:])
	}

	return out
}

func head(args []string) string {
	if len(args) < 1 {
		return ""
	}
	return args[0]
}
