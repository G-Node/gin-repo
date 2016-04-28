package git

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

type Delta struct {
	gitObject

	BaseRef    SHA1
	BaseOff    int64
	SizeSource int64
	SizeTarget int64

	pf  *PackFile
	op  DeltaOp
	err error
}

func parseDelta(obj gitObject) (*Delta, error) {
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

	delta.pf = delta.source.(*packReader).fd

	err = delta.wrapSourceWithDeflate()
	if err != nil {
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

type deltaChain struct {
	baseObj gitObject
	baseOff int64

	links []Delta
}

func (c *deltaChain) Len() int {
	return len(c.links)
}

type objectSource interface {
	openRawObject(id SHA1) (gitObject, error)
}

func buildDeltaChain(d *Delta, s objectSource) (*deltaChain, error) {
	var chain deltaChain
	var err error

	for err == nil {

		chain.links = append(chain.links, *d)

		var obj gitObject
		if d.otype == OBjRefDelta {
			obj, err = s.openRawObject(d.BaseRef)
		} else {
			obj, err = d.pf.readRawObject(d.BaseOff)
		}

		if err != nil {
			break
		}

		if IsStandardObject(obj.otype) {
			chain.baseObj = obj
			chain.baseOff = d.BaseOff
			break
		} else if !IsDeltaObject(obj.otype) {
			err = fmt.Errorf("git: unexpected object type in delta chain")
			break
		}

		d, err = parseDelta(obj)
	}

	if err != nil {
		//cleanup
		return nil, err
	}

	return &chain, nil
}

func (c *deltaChain) resolve() (Object, error) {

	ibuf := bytes.NewBuffer(make([]byte, 0, c.baseObj.Size()))
	n, err := io.Copy(ibuf, c.baseObj.source)
	if err != nil {
		return nil, err
	}

	if n != c.baseObj.Size() {
		return nil, io.ErrUnexpectedEOF
	}

	obuf := bytes.NewBuffer(make([]byte, 0, c.baseObj.Size()))

	for i := len(c.links); i > 0; i-- {
		lk := c.links[i-1]

		if lk.SizeTarget > int64(^uint(0)>>1) {
			return nil, fmt.Errorf("git: target to large for delta unpatching")
		}

		obuf.Grow(int(lk.SizeTarget))
		obuf.Truncate(0)

		err = lk.Patch(bytes.NewReader(ibuf.Bytes()), obuf)

		if err != nil {
			return nil, err
		}

		obuf, ibuf = ibuf, obuf
	}

	//ibuf is holding the data

	obj := gitObject{c.baseObj.otype, int64(ibuf.Len()), ioutil.NopCloser(ibuf)}
	return parseObject(obj)
}
