package git

import (
	"bufio"
	"fmt"
	"io"
)

func writeHeader(o Object, w *bufio.Writer) (n int64, err error) {

	x, err := w.WriteString(o.Type().String())
	n += int64(x)
	if err != nil {
		return n, err
	}

	x, err = w.WriteString(" ")
	n += int64(x)
	if err != nil {
		return n, err
	}

	x, err = w.WriteString(fmt.Sprintf("%d", o.Size()))
	n += int64(x)
	if err != nil {
		return n, err
	}

	err = w.WriteByte(0)
	if err != nil {
		return n, err
	}

	return n + 1, nil
}

//WriteTo writes the commit object to the writer in the on-disk format
//i.e. as it would be stored in the git objects dir (although uncompressed).
func (c *Commit) WriteTo(writer io.Writer) (int64, error) {
	w := bufio.NewWriter(writer)

	n, err := writeHeader(c, w)
	if err != nil {
		return n, err
	}

	x, err := w.WriteString(fmt.Sprintf("tree %s\n", c.Tree))
	n += int64(x)
	if err != nil {
		return n, err
	}

	for _, p := range c.Parent {
		x, err = w.WriteString(fmt.Sprintf("parent %s\n", p))
		n += int64(x)
		if err != nil {
			return n, err
		}
	}

	x, err = w.WriteString(fmt.Sprintf("author %s\n", c.Author))
	n += int64(x)
	if err != nil {
		return n, err
	}

	x, err = w.WriteString(fmt.Sprintf("committer %s\n\n", c.Committer))
	n += int64(x)
	if err != nil {
		return n, err
	}

	x, err = w.WriteString(fmt.Sprintf("%s", c.Message))
	n += int64(x)
	if err != nil {
		return n, err
	}

	err = w.Flush()
	return n, err
}
