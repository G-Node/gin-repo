package git

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// Resources:
//  https://github.com/git/git/blob/master/Documentation/technical/pack-format.txt
//  http://schacon.github.io/gitbook/7_the_packfile.html

//PackHeader stores version and number of objects in the packfile
// all data is in network-byte order (big-endian)
type PackHeader struct {
	Sig     [4]byte
	Version uint32
	Objects uint32
}

//FanOut table where the "N-th entry of this table records the
// number of objects in the corresponding pack, the first
// byte of whose object name is less than or equal to N.
type FanOut [256]uint32

func (fo FanOut) Bounds(b byte) (s, e int) {
	e = int(fo[b])
	if b > 0 {
		s = int(fo[b-1])
	}
	return
}

type PackIndex struct {
	*os.File

	Version uint32
	FO      FanOut

	shaBase int64
}

type PackFile struct {
	*os.File

	Version  uint32
	ObjCount uint32
}

type Pack struct {
	Index *PackIndex
	Data  *PackFile
}

func OpenPack(path string) (*Pack, error) {
	pathpack := path + ".pack"
	pathidx := path + ".idx"

	idx, err := PackIndexOpen(pathidx)
	if err != nil {
		return nil, fmt.Errorf("git: error opening pack index: %v", err)
	}

	data, err := OpenPackFile(pathpack)
	if err != nil {
		idx.Close()
		return nil, fmt.Errorf("git: error opening pack data: %v", err)
	}

	return &Pack{idx, data}, nil
}

func (p *Pack) Close() error {

	err := p.Data.Close()
	if err != nil {
		p.Index.Close()
		return err
	}

	return p.Index.Close()
}

func (p *Pack) ObjectCount() uint32 {
	return p.Index.FO[255]
}

func (p *Pack) GetOID(pos int) (SHA1, error) {
	var sha SHA1
	err := p.Index.ReadSHA1(&sha, pos)
	return sha, err
}

func PackIndexOpen(path string) (*PackIndex, error) {
	fd, err := os.Open(path)

	if err != nil {
		return nil, fmt.Errorf("git: could not read pack index: %v", err)
	}

	idx := &PackIndex{File: fd, Version: 1}

	var peek [4]byte
	err = binary.Read(fd, binary.BigEndian, &peek)
	if err != nil {
		fd.Close()
		return nil, fmt.Errorf("git: could not read pack index: %v", err)
	}

	if bytes.Equal(peek[:], []byte("\377tOc")) {
		binary.Read(fd, binary.BigEndian, &idx.Version)
	}

	if idx.Version == 1 {
		_, err = idx.Seek(0, 0)
		if err != nil {
			fd.Close()
			return nil, fmt.Errorf("git: io error: %v", err)
		}
	} else if idx.Version > 2 {
		fd.Close()
		return nil, fmt.Errorf("git: unsupported pack index version: %d", idx.Version)
	}

	err = binary.Read(idx, binary.BigEndian, &idx.FO)
	if err != nil {
		idx.Close()
		return nil, fmt.Errorf("git: io error: %v", err)
	}

	idx.shaBase = int64((idx.Version-1)*8) + int64(binary.Size(idx.FO))

	return idx, nil
}

func (pi *PackIndex) ReadSHA1(chksum *SHA1, pos int) error {
	if version := pi.Version; version != 2 {
		return fmt.Errorf("git: v%d version incomplete", version)
	}

	start := pi.shaBase
	_, err := pi.ReadAt(chksum[0:20], start+int64(pos)*int64(20))
	if err != nil {
		return err
	}

	return nil
}

func (pi *PackIndex) ReadOffset(pos int) (int64, error) {
	if version := pi.Version; version != 2 {
		return -1, fmt.Errorf("git: v%d version incomplete", version)
	}

	//header[2*4] + FanOut[256*4] + n * (sha1[20]+crc[4])
	start := int64(2*4+256*4) + int64(pi.FO[255]*24) + int64(pos*4)

	var offset uint32

	_, err := pi.Seek(start, 0)
	if err != nil {
		return -1, fmt.Errorf("git: io error: %v", err)
	}

	err = binary.Read(pi, binary.BigEndian, &offset)
	if err != nil {
		return -1, err
	}

	//see if msb is set, if so this is an
	// offset into the 64b_offset table
	if val := uint32(1<<31) & offset; val != 0 {
		return -1, fmt.Errorf("git: > 31 bit offests not implemented. Meh")
	}

	return int64(offset), nil
}

func (pi *PackIndex) FindSHA1(target SHA1) (int, error) {

	s, e := pi.FO.Bounds(target[0])
	for s < e {
		midpoint := s + (e-s+1)/2

		var sha SHA1
		err := pi.ReadSHA1(&sha, midpoint-1)
		if err != nil {
			return 0, fmt.Errorf("git: io error: %v", err)
		}

		switch bytes.Compare(target[:], sha[:]) {
		case -1: // target < sha1
			e = midpoint
		case +1: //taget > sha1
			s = midpoint
		default:
			return midpoint - 1, nil
		}
	}

	return 0, fmt.Errorf("git: sha1 not found in index")
}

func OpenPackFile(path string) (*PackFile, error) {
	osfd, err := os.Open(path)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v", err)
		os.Exit(2)
	}

	var header PackHeader
	err = binary.Read(osfd, binary.BigEndian, &header)
	if err != nil {
		return nil, fmt.Errorf("git: could not read header: %v", err)
	}

	if string(header.Sig[:]) != "PACK" {
		return nil, fmt.Errorf("git: packfile signature error")
	}

	if header.Version != 2 {
		return nil, fmt.Errorf("git: unsupported packfile version")
	}

	fd := &PackFile{File: osfd,
		Version:  header.Version,
		ObjCount: header.Objects}

	return fd, nil
}

func (pf *PackFile) readRawObject(offset int64) (gitObject, error) {
	_, err := pf.Seek(offset, 0)
	if err != nil {
		return gitObject{}, fmt.Errorf("git: io error: %v", err)
	}

	b := make([]byte, 1)
	_, err = pf.Read(b)

	if err != nil {
		return gitObject{}, fmt.Errorf("git: io error: %v", err)
	}

	otype := ObjectType((b[0] & 0x70) >> 4)
	size := int64(b[0] & 0xF)

	for i := 0; b[0]&0x80 != 0; i++ {
		// TODO: overflow for i > 9
		_, err = pf.Read(b)
		if err != nil {
			return gitObject{}, fmt.Errorf("git io error: %v", err)
		}

		size += int64(b[0]&0x7F) << uint(4+i*7)
	}

	return gitObject{otype, size, nil}, nil
}

func (pf *PackFile) ReadPackObject(offset int64) (Object, error) {

	obj, err := pf.readRawObject(offset)

	if err != nil {
		return nil, err
	}

	switch obj.otype {
	case ObjCommit:
		r, err := zlib.NewReader(pf)
		if err != nil {
			return nil, fmt.Errorf("git: could not create zlib reader: %v", err)
		}
		obj.source = r
		commit, err := ParseCommit(obj)
		r.Close()
		return commit, err

	case ObjOFSDelta:
		doff, err := readVarint(pf)
		if err != nil {
			return nil, err
		}
		delta := DeltaOfs{gitObject: gitObject{ObjOFSDelta, obj.size, nil}, Offset: doff}
		return &delta, nil

	case OBjRefDelta:
		var ref SHA1
		_, err := pf.Read(ref[:])
		if err != nil {
			return nil, err
		}
		delta := DeltaRef{gitObject: gitObject{OBjRefDelta, obj.size, nil}, Base: ref}
		return &delta, nil

	default:
		return &obj, nil
	}
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
