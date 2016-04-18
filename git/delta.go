package git

import (
	"fmt"
	"io"
	"os"
)

type Delta struct {
	gitObject

	BaseRef SHA1
	BaseOff int64

	Offset int64
}

func (pf *PackFile) parseDelta(obj gitObject) (*Delta, error) {
	delta := Delta{gitObject: obj}

	var err error
	if obj.otype == OBjRefDelta {
		_, err = pf.Read(delta.BaseRef[:])
		//TODO: check n?

		if err != nil {
			return nil, err
		}

	} else {
		delta.BaseOff, err = readVarint(pf)
		if err != nil {
			return nil, err
		}
	}

	delta.Offset, err = pf.Seek(0, os.SEEK_CUR)
	if err != nil {
		return nil, err
	}

	//hmm not sure about this here
	delta.wrapSourceWithDeflate()
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

type DeltaDecoder interface {
	Setup() bool
	NextOp() bool
	Op() DeltaOp
	Err() error
	SourceSize() int64
	TargetSize() int64
}

type deltaDecoder struct {
	r io.Reader

	op  DeltaOp
	err error

	sizeSource int64
	sizeTarget int64
}

func NewDeltaDecoder(delta *Delta) DeltaDecoder {
	return &deltaDecoder{r: delta.source}
}

func (d *deltaDecoder) Setup() bool {

	d.sizeSource, d.err = readDeltaSize(d.r)
	if d.err != nil {
		return false
	}

	d.sizeTarget, d.err = readDeltaSize(d.r)
	if d.err != nil {
		return false
	}

	return true
}

func (d *deltaDecoder) Op() DeltaOp {
	return d.op
}

func (d *deltaDecoder) Err() error {
	return d.err
}

func (d *deltaDecoder) SourceSize() int64 {
	return d.sizeSource
}

func (d *deltaDecoder) TargetSize() int64 {
	return d.sizeTarget
}

func (d *deltaDecoder) NextOp() (ok bool) {
	var b [1]byte
	_, err := d.r.Read(b[:])

	if err != nil {
		return
	}

	if b[0]&0x80 != 0 {
		d.op.Op = DeltaOpCopy
		op := b[0]
		d.op.Offset, d.err = decodeInt(d.r, op, 4)
		if err != nil {
			return
		}

		d.op.Size, err = decodeInt(d.r, op>>4, 3)
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
