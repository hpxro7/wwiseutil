// Package pck implements access to the Wwise File Package file format.
package pck

import (
	"io"
	"os"
)

import (
	"github.com/hpxro7/bnkutil/wwise"
)

// A File represents an open Wwise File Package.
type File struct {
	closer io.Closer
}

// NewFile creates a new File for access Wwise File Package files. The file is
// expected to start at position 0 in the io.ReaderAt.
func NewFile(r io.ReaderAt) (*File, error) {
	pck := new(File)
	return pck, nil
}

// WriteTo writes the full contents of this File to the Writer specified by w.
func (pck *File) WriteTo(w io.Writer) (written int64, err error) {
	return
}

// Open opens the File at the specified path using os.Open and prepares it for
// use as a Wwise File Package file.
func Open(path string) (*File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	pck, err := NewFile(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	pck.closer = f
	return pck, nil
}

// Close closes the File
// If the File was created using NewFile directly instead of Open,
// Close has no effect.
func (pck *File) Close() error {
	var err error
	if pck.closer != nil {
		err = pck.closer.Close()
		pck.closer = nil
	}
	return err
}

func (pck *File) Wems() []*wwise.Wem {
	return nil
}

func (pck *File) ReplaceWems(rs ...*wwise.ReplacementWem) {
	wwise.ReplaceWems(pck, rs...)
}

func (pck *File) DataStart() uint32 {
	return 0
}
