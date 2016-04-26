package git

import (
	"fmt"
	"io"
	"os"
)

type Delta struct {
	gitObject

	BaseRef    SHA1
	BaseOff    int64
	SizeSource int64
	SizeTarget int64

	op  DeltaOp
	err error
}

func (pf *PackFile) parseDelta(obj gitObject) (*Delta, error) {
	delta := Delta{gitObject: obj}

	var err error
	if obj.otype == OBjRefDelta {
		_, err = delta.source.Read(delta.BaseRef[:])
		//TODO: check n?

		if err != nil {
			return nil, err
		}

	} else {
		off, err := readVarint(delta.source)
		if err != nil {
			return nil, err
		}

		r := delta.source.(*packReader)
		delta.BaseOff = r.start - off
	}

	err = delta.wrapSourceWithDeflate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "wrapSourceWithDeflate failed [%#v]\n", delta.gitObject.source)
		return nil, err
	}

	delta.SizeSource, err = readDeltaSize(delta.source)
	if err != nil {
		return nil, err
	}

	delta.SizeTarget, err = readDeltaSize(delta.source)
	if err != nil {
		return nil, err
	}

	return &delta, nil
}

func readDeltaSize(r io.Reader) (int64, error) {
	b := make([]byte, 1)
	size := int64(0)
	b[0] = 0x80

	for i := uint(0); b[0]&0x80 != 0; i += 7 {
		//TODO: overflow check
		_, err := r.Read(b)
		if err != nil {
			return 0, fmt.Errorf("git: io error: %v", err)
		}

		size |= int64(b[0]&0x7F) << i
	}

	return size, nil
}

func decodeInt(r io.Reader, b byte, l uint) (size int64, err error) {

	var d [1]byte
	for i := uint(0); i < l; i++ {
		if b&(1<<i) != 0 {
			_, err = r.Read(d[:])
			if err != nil {
				return
			}

			size |= int64(d[0]) << (i * 8)
		}
	}

	return
}

func readVarint(r io.Reader) (int64, error) {
	b := make([]byte, 1)

	_, err := r.Read(b)
	if err != nil {
		return 0, fmt.Errorf("git: io error: %v", err)
	}

	size := int64(b[0] & 0x7F)

	for b[0]&0x80 != 0 {
		//TODO: overflow check
		_, err := r.Read(b)
		if err != nil {
			return 0, fmt.Errorf("git: io error: %v", err)
		}

		size++
		size = (size << 7) + int64(b[0]&0x7F)
	}

	return size, nil
}

type DeltaOpCode byte

const (
	DeltaOpInsert = 1
	DeltaOpCopy   = 2
)

type DeltaOp struct {
	Op     DeltaOpCode
	Size   int64
	Offset int64
}

func (d *Delta) Op() DeltaOp {
	return d.op
}

func (d *Delta) Err() error {
	return d.err
}

func (d *Delta) NextOp() (ok bool) {
	var b [1]byte
	_, err := d.source.Read(b[:])

	if err != nil {
		return
	}

	if b[0]&0x80 != 0 {
		d.op.Op = DeltaOpCopy
		op := b[0]
		d.op.Offset, d.err = decodeInt(d.source, op, 4)
		if err != nil {
			return
		}

		d.op.Size, err = decodeInt(d.source, op>>4, 3)
		if err != nil {
			return
		}

		if d.op.Size == 0 {
			d.op.Size = 0x10000
		}
		ok = true
	} else if n := b[0]; n > 0 {
		d.op.Op = DeltaOpInsert
		d.op.Size = int64(n)
		ok = true
	} else {
		d.err = fmt.Errorf("git: unknown delta op code")
	}

	return
}

func (d *Delta) Patch(r io.ReadSeeker, w io.Writer) error {

	for d.NextOp() {
		op := d.Op()
		switch op.Op {
		case DeltaOpCopy:
			_, err := r.Seek(op.Offset, os.SEEK_SET)
			if err != nil {
				return err
			}

			_, err = io.CopyN(w, r, op.Size)
			if err != nil {
				return err
			}
		case DeltaOpInsert:
			_, err := io.CopyN(w, d.source, op.Size)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

type idResolver interface {
	FindOffset(SHA1) (int64, error)
}

type deltaChain struct {
	baseObj gitObject
	baseOff int64

	links []Delta
}

func (d *deltaChain) Len() int {
	return len(d.links)
}
