// Package bnk implements access to the Wwise SoundBank file format.
package bnk

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
)

// A File represents an open Wwise SoundBank.
type File struct {
	closer            io.Closer
	BankHeaderSection *BankHeaderSection
	IndexSection      *DataIndexSection
	DataSection       *DataSection
	Others            []*UnknownSection
}

// NewFile creates a new File for access Wwise SoundBank files. The file is
// expected to start at position 0 in the io.ReaderAt.
func NewFile(r io.ReaderAt) (*File, error) {
	bnk := new(File)

	sr := io.NewSectionReader(r, 0, math.MaxInt64)
	for {
		hdr := new(SectionHeader)
		err := binary.Read(sr, binary.LittleEndian, hdr)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		switch id := hdr.Identifier; id {
		case bkhdHeaderId:
			sec, err := hdr.NewBankHeaderSection(sr)
			if err != nil {
				return nil, err
			}
			bnk.BankHeaderSection = sec
		case didxHeaderId:
			sec, err := hdr.NewDataIndexSection(sr)
			if err != nil {
				return nil, err
			}
			bnk.IndexSection = sec
		case dataHeaderId:
			sec, err := hdr.NewDataSection(sr, bnk.IndexSection)
			if err != nil {
				return nil, err
			}
			bnk.DataSection = sec
		default:
			sec, err := hdr.NewUnknownSection(sr)
			if err != nil {
				return nil, err
			}
			bnk.Others = append(bnk.Others, sec)
		}
	}

	if bnk.DataSection == nil {
		return nil, errors.New("There are no wems stored within this SoundBank.")
	}

	return bnk, nil
}

// WriteTo writes the full contents of this File to the Writer specified by w.
func (bnk *File) WriteTo(w io.Writer) (written int64, err error) {
	written, err = bnk.BankHeaderSection.WriteTo(w)
	if err != nil {
		return
	}
	n, err := bnk.IndexSection.WriteTo(w)
	if err != nil {
		return
	}
	written += n
	n, err = bnk.DataSection.WriteTo(w)
	if err != nil {
		return
	}
	written += n
	for _, other := range bnk.Others {
		n, err = other.WriteTo(w)
		if err != nil {
			return
		}
		written += n
	}
	return written, err
}

// Open opens the File at the specified path using os.Open and prepares it for
// use as a Wwise SoundBank file.
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

// ReplaceWem replaces the wem of File at index i, reading the wem, with
// specified length in from r.
func (bnk *File) ReplaceWem(r io.ReaderAt, i int, length int64) {
	wem := bnk.DataSection.Wems[i]
	oldLength := int64(wem.Descriptor.Length)
	if length > oldLength {
		panic(fmt.Sprintf("Target wem at index %d (%d bytes) is larger than the "+
			"original wem (%d bytes).\nUsing target wems that are larger than "+
			"the original wem is not yet supported", i, length, oldLength))
	}
	diff := oldLength - length
	wem.Reader = io.NewSectionReader(r, 0, length)
	remaining := int64(diff) + wem.RemainingLength
	wem.RemainingReader = io.NewSectionReader(&InfiniteReaderAt{0}, 0, remaining)

	oldDesc := wem.Descriptor
	desc := WemDescriptor{oldDesc.WemId, oldDesc.Offset, uint32(length)}
	wem.Descriptor = desc
	bnk.IndexSection.DescriptorMap[desc.WemId] = desc
}

func (bnk *File) String() string {
	b := new(strings.Builder)

	// TODO: Turn these into String() for each type.
	hdr := bnk.BankHeaderSection
	fmt.Fprintf(b, "%s: len(%d) version(%d) id(%d)\n", hdr.Header.Identifier,
		hdr.Header.Length, hdr.Descriptor.Version, hdr.Descriptor.BankId)

	idx := bnk.IndexSection
	total := uint32(0)
	for _, desc := range idx.DescriptorMap {
		total += desc.Length
	}
	fmt.Fprintf(b, "%s: len(%d) wem_count(%d)\n", idx.Header.Identifier,
		idx.Header.Length, idx.WemCount)
	fmt.Fprintf(b, "DIDX WEM total size: %d\n", total)

	data := bnk.DataSection
	fmt.Fprintf(b, "%s: len(%d)\n", data.Header.Identifier, data.Header.Length)

	for _, sec := range bnk.Others {
		fmt.Fprintf(b, "%s: len(%d)\n", sec.Header.Identifier, sec.Header.Length)
	}

	return b.String()
}
