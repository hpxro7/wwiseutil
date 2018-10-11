// Package pck implements access to the Wwise File Package file format.
package pck

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
)

import (
	"github.com/hpxro7/bnkutil/util"
	"github.com/hpxro7/bnkutil/wwise"
)

// The number of bytes used to describe the File Package header.
const HEADER_BYTES = 4 + 4 + 44 + 4

// The number of bytes used to describe a single data index entry.
const DATA_INDEX_BYTES = 4 + 4 + 4 + 4

// A File represents an open Wwise File Package.
type File struct {
	closer  io.Closer
	Header  *Header
	Indexes []*DataIndex
	Padding uint32
	wems    []*wwise.Wem
}

// A Header represents a single Wwise File Package header.
type Header struct {
	Identifier [4]byte
	Length     uint32
	Unknown    [44]byte
	WemCount   uint32
}

// A DataIndex represents location and properties of a file within a File
// Package.
type DataIndex struct {
	// The type of data contained at this location.
	Type uint32
	// A descriptor of the wem contained at this location, if it is a wem.
	Descriptor *wwise.WemDescriptor
	Unknown    uint32
}

// NewFile creates a new File for access Wwise File Package files. The file is
// expected to start at position 0 in the io.ReaderAt.
func NewFile(r io.ReaderAt) (*File, error) {
	pck := new(File)
	sr := io.NewSectionReader(r, 0, math.MaxInt64)

	hdr, err := NewHeader(sr)
	if err != nil {
		return nil, err
	}
	pck.Header = hdr

	// Read in the data index.
	for i := uint32(0); i < pck.Header.WemCount; i++ {
		idx, err := NewDataIndex(sr)
		if err != nil {
			return nil, err
		}
		pck.Indexes = append(pck.Indexes, idx)
	}

	var padding uint32
	err = binary.Read(sr, binary.LittleEndian, &padding)
	if err != nil {
		return nil, err
	}
	pck.Padding = padding

	// Read in the data contained within this File Package
	for _, idx := range pck.Indexes {
		wem, err := newWem(sr, idx)
		if err != nil {
			return nil, err
		}
		pck.wems = append(pck.wems, wem)
	}

	return pck, nil
}

// WriteTo writes the full contents of this File to the Writer specified by w.
func (pck *File) WriteTo(w io.Writer) (written int64, err error) {
	written, err = pck.Header.WriteTo(w)
	if err != nil {
		return
	}

	for _, idx := range pck.Indexes {
		n, err := idx.WriteTo(w)
		if err != nil {
			return written, err
		}
		written += n
	}

	err = binary.Write(w, binary.LittleEndian, pck.Padding)
	if err != nil {
		return
	}
	written += int64(4)

	for _, wem := range pck.wems {
		n, err := io.Copy(w, wem)
		if err != nil {
			return written, err
		}
		written += int64(n)
	}

	return written, nil
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
	return pck.wems
}

func (pck *File) ReplaceWems(rs ...*wwise.ReplacementWem) {
	wwise.ReplaceWems(pck, rs...)
}

func (pck *File) DataStart() uint32 {
	return 0
}

func (pck *File) String() string {
	b := new(strings.Builder)

	tableParams := []string{"%-7", "%-15", "%-15", "%-8", "\n"}
	titleFmt := strings.Join(tableParams, "s|")
	wemFmt := strings.Join(tableParams, "d|")
	title := fmt.Sprintf(titleFmt,
		"Index", "Id", "Offset", "Length")
	fmt.Fprint(b, title)
	fmt.Fprintln(b, strings.Repeat("-", len(title)-1))

	for i, idx := range pck.Indexes {
		desc := idx.Descriptor

		fmt.Fprintf(b, wemFmt, i+1, desc.WemId, desc.Offset, desc.Length)
	}

	return b.String()
}

func NewHeader(sr util.ReadSeekerAt) (*Header, error) {
	hdr := new(Header)
	err := binary.Read(sr, binary.LittleEndian, hdr)
	if err != nil {
		return nil, err
	}
	return hdr, nil
}

func (hdr *Header) WriteTo(w io.Writer) (written int64, err error) {
	err = binary.Write(w, binary.LittleEndian, hdr)
	if err != nil {
		return
	}
	written = int64(HEADER_BYTES)
	return
}

func NewDataIndex(sr util.ReadSeekerAt) (*DataIndex, error) {
	var id uint32
	err := binary.Read(sr, binary.LittleEndian, &id)
	if err != nil {
		return nil, err
	}

	var dataType uint32
	err = binary.Read(sr, binary.LittleEndian, &dataType)
	if err != nil {
		return nil, err
	}

	var length uint32
	err = binary.Read(sr, binary.LittleEndian, &length)
	if err != nil {
		return nil, err
	}

	var offset uint32
	err = binary.Read(sr, binary.LittleEndian, &offset)
	if err != nil {
		return nil, err
	}

	var unknown uint32
	err = binary.Read(sr, binary.LittleEndian, &unknown)
	if err != nil {
		return nil, err
	}

	desc := wwise.WemDescriptor{id, offset, length}
	return &DataIndex{dataType, &desc, unknown}, nil
}

// WriteTo writes the full contents of this DataIndex to the Writer specified by
// w.
func (idx *DataIndex) WriteTo(w io.Writer) (written int64, err error) {
	err = binary.Write(w, binary.LittleEndian, idx.Descriptor.WemId)
	if err != nil {
		return
	}
	written = int64(4)

	err = binary.Write(w, binary.LittleEndian, idx.Type)
	if err != nil {
		return
	}
	written += int64(4)

	err = binary.Write(w, binary.LittleEndian, idx.Descriptor.Length)
	if err != nil {
		return
	}
	written += int64(4)

	err = binary.Write(w, binary.LittleEndian, idx.Descriptor.Offset)
	if err != nil {
		return
	}
	written += int64(4)

	err = binary.Write(w, binary.LittleEndian, idx.Unknown)
	if err != nil {
		return
	}
	written += int64(4)

	return written, nil
}

func newWem(sr util.ReadSeekerAt, idx *DataIndex) (*wwise.Wem, error) {
	offset, _ := sr.Seek(0, io.SeekCurrent)
	desc := idx.Descriptor
	if uint32(offset) != desc.Offset {
		msg := fmt.Sprintf("Wem %d was expected to start at offset %d "+
			"but instead started at offset %d", desc.WemId, desc.Offset, offset)
		return nil, errors.New(msg)
	}

	wemReader := util.NewResettingReader(sr, offset, int64(desc.Length))
	padding := util.NewResettingReader(&util.InfiniteReaderAt{0}, 0, 0)
	sr.Seek(int64(desc.Length), io.SeekCurrent)
	return &wwise.Wem{wemReader, desc, padding}, nil
}
