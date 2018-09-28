// Package bnk implements access to the Wwise SoundBank file format.
package bnk

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
)

// A File represents an open Wwise SoundBank.
type File struct {
	closer   io.Closer
	Sections []*SectionHeader
}

// A SectionHeader represents a single Wwise SoundBank header.
type SectionHeader struct {
	Id     [4]byte
	Length uint32
}

// NewFile creates a new File for access Wwise SoundBank files. The file is
// expected to start at position 0 in the io.ReaderAt.
func NewFile(r io.ReaderAt) (*File, error) {
	var bnk File

	sr := io.NewSectionReader(r, 0, math.MaxInt64)
	for {
		var hdr SectionHeader
		err := binary.Read(sr, binary.LittleEndian, &hdr)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		bnk.Sections = append(bnk.Sections, &hdr)

		sr.Seek(int64(hdr.Length), io.SeekCurrent)
	}

	return &bnk, nil
}

// Open opens the File at the specified path using os.Open and prepares it
// for use as a Wwise SoundBank file.
func Open(path string) (*File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	bnk, err := NewFile(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	bnk.closer = f
	return bnk, nil
}

// Close closes the File
// If the File was created using NewFile directly instead of Open,
// Close has no effect.
func (bnk *File) Close() error {
	var err error
	if bnk.closer != nil {
		err = bnk.closer.Close()
		bnk.closer = nil
	}
	return err
}

func (bnk *File) String() string {
	var b strings.Builder
	for _, sec := range bnk.Sections {
		id := string(sec.Id[:len(sec.Id)])
		fmt.Fprintf(&b, "%s: len(%d)\n", id, sec.Length)
	}
	return b.String()
}
