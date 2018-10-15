// Package util implements common utility functions.
package util

import (
	"io"
)

type ReadSeekerAt interface {
	io.ReadSeeker
	io.ReaderAt
	Size() int64
}

type ResettingReader struct {
	*io.SectionReader
}

func NewResettingReader(r io.ReaderAt, off int64, n int64) ReadSeekerAt {
	return &ResettingReader{io.NewSectionReader(r, off, n)}
}

func (r *ResettingReader) Read(p []byte) (n int, err error) {
	n, err = r.SectionReader.Read(p)
	if err == io.EOF {
		r.SectionReader.Seek(0, io.SeekStart)
	}
	return
}

// A utility ReaderAt that emits an infinite stream of a specific value.
type InfiniteReaderAt struct {
	// The value that this padding writer will write.
	Value byte
}

// ReadAt fills all of len(p) bytes with the Value of this InfiniteReaderAt.
func (r *InfiniteReaderAt) ReadAt(p []byte, off int64) (int, error) {
	for i, _ := range p {
		p[i] = r.Value
	}
	return len(p), nil
}
