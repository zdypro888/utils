package utils

import (
	"bytes"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
)

type iReader interface {
	io.ReaderAt
	Size() int64
}

// type Cryptor interface {
// 	Crypt([]byte)
// 	Reset() error
// 	Clone() Cryptor
// }

type Reader struct {
	readers []iReader
	closers []io.Closer
	offset  int64
	size    int64
}

func NewReader(f ...any) (*Reader, error) {
	r := &Reader{offset: 0, size: 0}
	for _, v := range f {
		if err := r.Append(v); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *Reader) Append(f any) error {
	switch fo := f.(type) {
	case string:
		fi, err := os.OpenFile(fo, os.O_RDONLY, 0)
		if err != nil {
			return err
		}
		st, err := fi.Stat()
		if err != nil {
			fi.Close()
			return err
		}
		r.closers = append(r.closers, fi)
		r.size += st.Size()
		r.readers = append(r.readers, io.NewSectionReader(fi, 0, st.Size()))
		return nil
	case []byte:
		r.size += int64(len(fo))
		r.readers = append(r.readers, io.NewSectionReader(bytes.NewReader(fo), 0, int64(len(fo))))
		return nil
	case *bytes.Reader:
		r.size += int64(fo.Len())
		r.readers = append(r.readers, io.NewSectionReader(fo, 0, int64(fo.Len())))
		return nil
	case iReader:
		r.size += fo.Size()
		r.readers = append(r.readers, fo)
		return nil
	default:
		return fmt.Errorf("unsupported type: %T", f)
	}
}

func (r *Reader) Size() int64 {
	return r.size
}

func (r *Reader) Close() error {
	var err error
	for _, c := range r.closers {
		if nerr := c.Close(); nerr != nil {
			err = nerr
		}
	}
	return err
}

func (r *Reader) ReadAt(p []byte, off int64) (int, error) {
	if len(r.readers) == 0 {
		return 0, io.EOF
	}
	if off >= r.size {
		return 0, io.EOF
	}
	var roff int64
	totalRead := 0
	for _, rd := range r.readers {
		readerOff := roff
		roff += rd.Size()
		if off >= readerOff && off < readerOff+rd.Size() {
			n, err := rd.ReadAt(p[totalRead:], off-readerOff)
			off += int64(n)
			totalRead += n
			if err != nil && err != io.EOF {
				return totalRead, err
			}
			if totalRead == len(p) {
				return totalRead, nil
			}
			if err == io.EOF && off < r.size {
				continue
			}
			return totalRead, err
		}
	}
	return totalRead, io.EOF
}

func (r *Reader) Read(p []byte) (int, error) {
	n, err := r.ReadAt(p, r.offset)
	r.offset += int64(n)
	return n, err
}

// UnreadLength returns the number of bytes that can be read from the current offset.
func (r *Reader) UnLen() int64 {
	return r.size - r.offset
}

// UnreadLength returns the number of bytes that can be read from the current offset.
func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		r.offset = offset
	case io.SeekCurrent:
		r.offset += offset
	case io.SeekEnd:
		r.offset = r.size + offset
	default:
		return 0, errors.New("invalid whence")
	}
	if r.offset < 0 {
		r.offset = 0
		return 0, errors.New("negative position")
	}
	if r.offset > r.size {
		r.offset = r.size
		return r.size, errors.New("position out of bounds")
	}
	return r.offset, nil
}

// ReadSection returns a new Reader that reads from the specified offset and length of the original Reader.
func (r *Reader) ReadSection(offset, length int64) *Reader {
	if length < 0 || length > r.size-offset {
		length = r.size - offset
	}
	reader := &Reader{
		readers: []iReader{io.NewSectionReader(r, offset, length)},
		offset:  0,
		size:    length,
	}
	return reader
}

// ReadLimit returns a new Reader that reads from the current offset of the original Reader.
func (r *Reader) ReadLimit(length int64) *Reader {
	return r.ReadSection(r.offset, length)
}

// Temporary returns a new Reader that reads from the beginning of the original Reader.
func (r *Reader) Temporary() *Reader {
	return r.ReadSection(0, -1)
}

func (r *Reader) Sum(hasher hash.Hash) ([]byte, error) {
	curpos, _ := r.Seek(0, io.SeekCurrent)
	defer r.Seek(curpos, io.SeekStart)
	if _, err := io.Copy(hasher, r); err != nil {
		return nil, err
	}
	return hasher.Sum(nil), nil
}
