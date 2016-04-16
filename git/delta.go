package git

import (
	"fmt"
	"io"
)

type DeltaObject interface {
	NextOp() bool
	Op() DeltaOp
	Err() error
	SourceSize() int64
	TargetSize() int64
}

type deltaObject struct {
	gitObject

	op  DeltaOp
	err error

	sizeSource int64
	sizeTarget int64
}

type DeltaOfs struct {
	deltaObject

	Offset int64
}

type DeltaRef struct {
	deltaObject

	Base SHA1
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

func (o *deltaObject) Op() DeltaOp {
	return o.op
}

func (o *deltaObject) Err() error {
	return o.err
}

func (o *deltaObject) SourceSize() int64 {
	return o.sizeSource
}

func (o *deltaObject) TargetSize() int64 {
	return o.sizeTarget
}

func parseDelta(obj gitObject) (deltaObject, error) {
	delta := deltaObject{gitObject: obj}
	err := delta.wrapSourceWithDeflate()
	if err != nil {
		return delta, err
	}

	delta.sizeSource, err = readDeltaSize(delta.source)
	if err != nil {
		return delta, err
	}

	delta.sizeTarget, err = readDeltaSize(delta.source)
	if err != nil {
		return delta, err
	}

	return delta, nil
}

func parseDeltaOfs(obj gitObject) (Object, error) {
	offset, err := readVarint(obj.source)

	if err != nil {
		return nil, err
	}

	delta, err := parseDelta(obj)
	if err != nil {
		return nil, err
	}

	return &DeltaOfs{delta, offset}, nil
}

func parseDeltaRef(obj gitObject) (Object, error) {
	var ref SHA1
	_, err := obj.source.Read(ref[:])
	if err != nil {
		return nil, err
	}

	delta, err := parseDelta(obj)
	if err != nil {
		return nil, err
	}

	return &DeltaRef{delta, ref}, nil
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

func (o *deltaObject) NextOp() (ok bool) {
	var b [1]byte
	_, err := o.source.Read(b[:])

	if err != nil {
		return
	}

	if b[0]&0x80 != 0 {
		o.op.Op = DeltaOpInsert
		op := b[0]
		o.op.Offset, o.err = decodeInt(o.source, op, 4)
		if err != nil {
			return
		}

		o.op.Size, err = decodeInt(o.source, op>>4, 3)
		if err != nil {
			return
		}

		if o.op.Size == 0 {
			o.op.Size = 0x10000
		}
		ok = true
	} else if n := b[0]; n > 0 {
		o.op.Op = DeltaOpCopy
		o.op.Size = int64(n)
		ok = true
	} else {
		o.err = fmt.Errorf("git: unknown delta op code")
	}

	return
}
