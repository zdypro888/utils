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

type IProgress interface {
	OnProgress(offset, size int64)
}

type Reader struct {
	readers []iReader
	offset  int64
	size    int64
	// 进度回调
	Progress IProgress
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

type autoDeleteReader struct {
	filename string
	file     *os.File // 保存文件句柄，Close 时先关闭文件再删除
	iReader
}

func (r *autoDeleteReader) Close() error {
	// 先关闭文件句柄
	if r.file != nil {
		r.file.Close()
	}
	// 再删除文件（忽略删除错误，避免影响上传流程）
	os.Remove(r.filename)
	return nil
}

func (r *Reader) AppendFile(filename string, autoDelete bool) error {
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
	filereader := io.NewSectionReader(fi, 0, st.Size())
	if autoDelete {
		r.readers = append(r.readers, &autoDeleteReader{filename: filename, file: fi, iReader: filereader})
	} else {
		r.readers = append(r.readers, filereader)
	}
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
		return r.AppendFile(fo, false)
	case []byte:
		return r.AppendBytes(fo)
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
	if r.Progress != nil {
		r.Progress.OnProgress(r.offset, r.size)
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
		readers: []iReader{io.NewSectionReader(r, offset, length)},
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
