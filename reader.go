package utils

import (
	"bytes"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
)

type ReaderAtSize interface {
	io.ReaderAt
	Size() int64
}

type OffsetSize struct {
	Offset int64
	Size   int64
}

type Reader struct {
	readers []ReaderAtSize
	offset  int64
	size    int64
	// 进度回调
	progress chan *OffsetSize
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

func (r *Reader) Progress() <-chan *OffsetSize {
	return r.progress
}

type FileTag uint64

const (
	FileTagCloseHandle FileTag = 0x01
	FileTagDeleteFile  FileTag = 0x02
	FileTagAll         FileTag = FileTagCloseHandle | FileTagDeleteFile
)

type fileReader struct {
	Name   string
	Handle *os.File // 保存文件句柄，Close 时先关闭文件再删除
	Tag    FileTag
	ReaderAtSize
}

func (r *fileReader) Open() error {
	if r.Handle != nil {
		r.Handle.Close()
	}
	fi, err := os.OpenFile(r.Name, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	r.Handle = fi
	return nil
}

func (r *fileReader) Close() error {
	if r.Tag&FileTagCloseHandle == FileTagCloseHandle {
		if r.Handle != nil {
			r.Handle.Close()
		}
	}
	if r.Tag&FileTagDeleteFile == FileTagDeleteFile {
		os.Remove(r.Name)
	}
	return nil
}

func (r *fileReader) Remove() {
	if r.Handle != nil {
		r.Handle.Close()
	}
	os.Remove(r.Name)
}

func (r *Reader) AppendFile(filename string, tag FileTag) error {
	fi, err := os.OpenFile(filename, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	st, err := fi.Stat()
	if err != nil {
		fi.Close()
		return err
	}
	r.size += st.Size()
	r.readers = append(r.readers, &fileReader{Name: filename, Handle: fi, Tag: tag, ReaderAtSize: io.NewSectionReader(fi, 0, st.Size())})
	return nil
}

func (r *Reader) AppendBytes(b []byte) error {
	r.size += int64(len(b))
	r.readers = append(r.readers, io.NewSectionReader(bytes.NewReader(b), 0, int64(len(b))))
	return nil
}

func (r *Reader) Append(f any) error {
	switch fo := f.(type) {
	case string:
		return r.AppendFile(fo, FileTagAll)
	case []byte:
		return r.AppendBytes(fo)
	case *bytes.Reader:
		r.size += int64(fo.Len())
		r.readers = append(r.readers, io.NewSectionReader(fo, 0, int64(fo.Len())))
		return nil
	case ReaderAtSize:
		r.size += fo.Size()
		r.readers = append(r.readers, fo)
		return nil
	default:
		return fmt.Errorf("unsupported type: %T", f)
	}
}

type Openable interface {
	Open() error
}

func (r *Reader) Open() error {
	for _, rd := range r.readers {
		if reader, ok := rd.(Openable); ok {
			if err := reader.Open(); err != nil {
				return err
			}
		}
	}
	return nil
}

type Removeable interface {
	Remove()
}

func (r *Reader) Remove() {
	for _, rd := range r.readers {
		if remover, ok := rd.(Removeable); ok {
			remover.Remove()
		}
	}
}

func (r *Reader) Size() int64 {
	return r.size
}

func (r *Reader) Close() error {
	var err error
	for _, rd := range r.readers {
		if closer, ok := rd.(io.Closer); ok {
			if nerr := closer.Close(); nerr != nil {
				err = nerr
			}
		}
	}
	return err
}

func (r *Reader) ReadAt(data []byte, offset int64) (int, error) {
	if len(r.readers) == 0 {
		return 0, io.EOF
	}
	if offset >= r.size {
		return 0, io.EOF
	}
	var leftOffset int64
	readlen := 0
	for _, rd := range r.readers {
		rightOffset := leftOffset + rd.Size()
		if offset >= leftOffset && offset < rightOffset {
			n, err := rd.ReadAt(data[readlen:], offset-leftOffset)
			offset += int64(n)
			readlen += n
			if err != nil && err != io.EOF {
				// 遇到非EOF错误时，返回已读取的数据和错误
				return readlen, err
			}
			if readlen == len(data) {
				if err == nil {
					return readlen, nil
				} else if offset >= r.size {
					return readlen, io.EOF
				} else {
					return readlen, nil
				}
			}
			if offset < rightOffset {
				if err == io.EOF {
					return readlen, io.EOF
				}
				return readlen, nil
			}
		}
		leftOffset = rightOffset
	}
	return readlen, io.EOF
}

func (r *Reader) Read(p []byte) (int, error) {
	n, err := r.ReadAt(p, r.offset)
	r.offset += int64(n)
	if r.progress != nil {
		select {
		case r.progress <- &OffsetSize{Offset: r.offset, Size: r.size}:
		default:
		}
	}
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
		return -1, errors.New("invalid whence")
	}
	if r.offset < 0 {
		r.offset = 0
		return -1, errors.New("negative position")
	}
	if r.offset > r.size {
		r.offset = r.size
		return r.size, errors.New("position out of bounds")
	}
	return r.offset, nil
}

// PeekSection returns a new Reader that reads from the specified offset and length of the original Reader.
func (r *Reader) PeekSection(offset, length int64) *Reader {
	if length < 0 || length > r.size-offset {
		length = r.size - offset
	}
	reader := &Reader{
		readers: []ReaderAtSize{io.NewSectionReader(r, offset, length)},
		offset:  0,
		size:    length,
	}
	return reader
}

// ReadLimit returns a new Reader that reads from the current offset of the original Reader.
func (r *Reader) ReadLimit(length int64) *Reader {
	reader := r.PeekSection(r.offset, length)
	r.offset += reader.size
	return reader
}

// Temporary returns a new Reader that reads from the beginning of the original Reader.
func (r *Reader) Temporary() *Reader {
	return r.PeekSection(0, -1)
}

func (r *Reader) Reset() {
	r.offset = 0
}

// CopyTo copies the contents of the reader to the writer. and offset is not changed.
func (r *Reader) CopyTo(writer io.Writer) error {
	curpos, _ := r.Seek(0, io.SeekCurrent)
	defer r.Seek(curpos, io.SeekStart)
	if _, err := io.Copy(writer, r); err != nil {
		return err
	}
	return nil
}

// Sum returns the hash of the reader.
func (r *Reader) Sum(hasher hash.Hash) ([]byte, error) {
	if err := r.CopyTo(hasher); err != nil {
		return nil, err
	}
	return hasher.Sum(nil), nil
}
