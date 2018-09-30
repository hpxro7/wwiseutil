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

// The number of bytes used to describe a single data index entry
// within the DIDX section.
const DIDX_ENTRY_BYTES = 12

// The identifier for the start of the DIDX (Data Index) section.
var didxHeaderId = [4]byte{'D', 'I', 'D', 'X'}

// The identifier for the start of the DATA section.
var dataHeaderId = [4]byte{'D', 'A', 'T', 'A'}

// A File represents an open Wwise SoundBank.
type File struct {
	closer       io.Closer
	IndexSection *DataIndexSection
	DataSection  *DataSection
	Others       []*SectionHeader
}

// A DataIndexSection represents the DIDX section of a SoundBank file.
type DataIndexSection struct {
	Header *SectionHeader
	// The count of wems in this SoundBank.
	WemCount uint32
	// A list of all wem IDs, in order of their offset into the file.
	WemIds []uint32
	// A mapping from wem ID to its descriptor.
	DescriptorMap map[uint32]WemDescriptor
}

// A DataIndexSection represents the DATA section of a SoundBank file.
type DataSection struct {
	Header *SectionHeader
	// The offset into the file where the data portion of the DATA section begins.
	// This is the location where wem entries are stored.
	DataStart uint32
	Wems      []*Wem
}

// A Wem represents a single sound entity contained within a SoundBank file.
type Wem struct {
	io.Reader
	sr         *io.SectionReader
	Descriptor WemDescriptor
}

// A WemDescriptor represents the location of a single wem entity within the
// SoundBank DATA section.
type WemDescriptor struct {
	// The number of bytes from the start of the DATA section's data (after the
	// header and length) that this wem begins.
	Offset uint32
	// The length in bytes of this wem.
	Length uint32
}

// A SectionHeader represents a single Wwise SoundBank header.
type SectionHeader struct {
	Identifier [4]byte
	Length     uint32
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
			bnk.Others = append(bnk.Others, hdr)
			sr.Seek(int64(hdr.Length), io.SeekCurrent)
		}
	}

	return bnk, nil
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

func (bnk *File) String() string {
	b := new(strings.Builder)
	for _, sec := range bnk.Others {
		fmt.Fprintf(b, "%s: len(%d)\n", sec.Identifier, sec.Length)
	}
	// TODO: Turn these into String() for each type.
	idx := bnk.IndexSection
	fmt.Fprintf(b, "%s: len(%d)\n", idx.Header.Identifier, idx.Header.Length)
	fmt.Fprintf(b, "WEM count: %d\n", idx.WemCount)
	total := uint32(0)
	for _, desc := range idx.DescriptorMap {
		total += desc.Length
	}
	fmt.Fprintf(b, "DIDX WEM total size: %d\n", total)
	data := bnk.DataSection
	fmt.Fprintf(b, "%s: len(%d)", data.Header.Identifier, data.Header.Length)
	return b.String()
}

// NewDataIndexSection creates a new DataIndexSection, reading from r, which must
// be seeked to the start of the DIDX section data.
// It is an error to call this method on a non-DIDX header.
func (hdr *SectionHeader) NewDataIndexSection(r io.Reader) (*DataIndexSection, error) {
	if hdr.Identifier != didxHeaderId {
		panic(fmt.Sprintf("Expected DIDX header but got: %s", hdr.Identifier))
	}
	wemCount := hdr.Length / DIDX_ENTRY_BYTES
	sec := DataIndexSection{hdr, wemCount, make([]uint32, 0),
		make(map[uint32]WemDescriptor)}
	for i := uint32(0); i < wemCount; i++ {
		var wemId uint32
		err := binary.Read(r, binary.LittleEndian, &wemId)
		if err != nil {
			return nil, err
		}
		if _, ok := sec.DescriptorMap[wemId]; ok {
			panic(fmt.Sprintf("%d is an illegal repeated wem ID in the DIDX", wemId))
		}
		sec.WemIds = append(sec.WemIds, wemId)

		var desc WemDescriptor
		err = binary.Read(r, binary.LittleEndian, &desc)
		if err != nil {
			return nil, err
		}
		sec.DescriptorMap[wemId] = desc
	}

	return &sec, nil
}

// NewDataSection creates a new DataSection, reading from sr, which must be
// seeked to the start of the DATA section data. idx specifies how each wem
// should be indexed from, given the current sr offset.
// It is an error to call this method on a non-DATA header.
func (hdr *SectionHeader) NewDataSection(sr *io.SectionReader,
	idx *DataIndexSection) (*DataSection, error) {
	if hdr.Identifier != dataHeaderId {
		panic(fmt.Sprintf("Expected DATA header but got: %s", hdr.Identifier))
	}
	dataOffset, err := sr.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}
	sec := DataSection{hdr, uint32(dataOffset), make([]*Wem, 0)}
	for _, id := range idx.WemIds {
		desc := idx.DescriptorMap[id]
		wemStartOffset := dataOffset + int64(desc.Offset)
		wemReader := io.NewSectionReader(sr, wemStartOffset, int64(desc.Length))
		wem := Wem{wemReader, wemReader, desc}
		sec.Wems = append(sec.Wems, &wem)
	}

	sr.Seek(int64(hdr.Length), io.SeekCurrent)
	return &sec, nil
}
