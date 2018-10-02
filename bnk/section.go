// Package bnk implements access to the Wwise SoundBank file format.
package bnk

import (
	"encoding/binary"
	"fmt"
	"io"
)

// The number of bytes used to describe the header of a section.
const SECTION_HEADER_BYTES = 8

// The number of bytes used to describe the known portion of a BKHD section,
// excluding its own header.
const BKHD_SECTION_BYTES = 8

// The number of bytes used to describe a single data index
// entry (a WemDescriptor) within the DIDX section.
const DIDX_ENTRY_BYTES = 12

// The identifier for the start of the BKHD (Bank Header) section.
var bkhdHeaderId = [4]byte{'B', 'K', 'H', 'D'}

// The identifier for the start of the DIDX (Data Index) section.
var didxHeaderId = [4]byte{'D', 'I', 'D', 'X'}

// The identifier for the start of the DATA section.
var dataHeaderId = [4]byte{'D', 'A', 'T', 'A'}

// A SectionHeader represents a single Wwise SoundBank header.
type SectionHeader struct {
	Identifier [4]byte
	Length     uint32
}

// A BankHeaderSection represents the BKHD section of a SoundBank file.
type BankHeaderSection struct {
	Header          *SectionHeader
	Descriptor      BankDescriptor
	RemainingReader io.Reader
}

// A BankDescriptor provides metadata about the overall SoundBank file.
type BankDescriptor struct {
	Version uint32
	BankId  uint32
}

// A DataIndexSection represents the DIDX section of a SoundBank file.
type DataIndexSection struct {
	Header *SectionHeader
	// The count of wems in this SoundBank.
	WemCount int
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
	Descriptor WemDescriptor
	// A reader over the bytes that remain until the next wem if there is one, or
	// the end of the data section. These bytes are generally NUL(0x00) padding.
	RemainingReader io.Reader
	// The number of bytes remaining until the next wem if there is one, or the
	// end of the data section.
	RemainingLength int64
}

// A WemDescriptor represents the location of a single wem entity within the
// SoundBank DATA section.
type WemDescriptor struct {
	WemId uint32
	// The number of bytes from the start of the DATA section's data (after the
	// header and length) that this wem begins.
	Offset uint32
	// The length in bytes of this wem.
	Length uint32
}

// An UnknownSection represents an unknown section in a SoundBank file.
type UnknownSection struct {
	Header *SectionHeader
	// A reader to read the data of this section.
	Reader io.Reader
}

// NewBankHeaderSection creates a new BankHeaderSection, reading from sr, which
// must be seeked to the start of the BKHD section data.
// It is an error to call this method on a non-BKHD header.
func (hdr *SectionHeader) NewBankHeaderSection(sr *io.SectionReader) (*BankHeaderSection, error) {
	if hdr.Identifier != bkhdHeaderId {
		panic(fmt.Sprintf("Expected BKHD header but got: %s", hdr.Identifier))
	}
	sec := new(BankHeaderSection)
	sec.Header = hdr
	desc := BankDescriptor{}
	err := binary.Read(sr, binary.LittleEndian, &desc)
	if err != nil {
		return nil, err
	}
	sec.Descriptor = desc
	// Get the offset into the file where the known portion of the BKHD ends.
	knownOffset, _ := sr.Seek(0, io.SeekCurrent)
	remaining := int64(hdr.Length - BKHD_SECTION_BYTES)
	sec.RemainingReader = io.NewSectionReader(sr, knownOffset, remaining)
	sr.Seek(remaining, io.SeekCurrent)

	return sec, nil
}

// WriteTo writes the full contents of this BankHeaderSection to the Writer
// specified by w.
func (hdr *BankHeaderSection) WriteTo(w io.Writer) (written int64, err error) {
	err = binary.Write(w, binary.LittleEndian, hdr.Header)
	if err != nil {
		return
	}
	written = int64(SECTION_HEADER_BYTES)
	err = binary.Write(w, binary.LittleEndian, hdr.Descriptor)
	if err != nil {
		return
	}
	written += int64(BKHD_SECTION_BYTES)
	n, err := io.Copy(w, hdr.RemainingReader)
	if err != nil {
		return
	}
	written += int64(n)
	return written, nil
}

// NewDataIndexSection creates a new DataIndexSection, reading from r, which must
// be seeked to the start of the DIDX section data.
// It is an error to call this method on a non-DIDX header.
func (hdr *SectionHeader) NewDataIndexSection(r io.Reader) (*DataIndexSection, error) {
	if hdr.Identifier != didxHeaderId {
		panic(fmt.Sprintf("Expected DIDX header but got: %s", hdr.Identifier))
	}
	wemCount := int(hdr.Length / DIDX_ENTRY_BYTES)
	sec := DataIndexSection{hdr, wemCount, make([]uint32, 0),
		make(map[uint32]WemDescriptor)}
	for i := 0; i < wemCount; i++ {
		var desc WemDescriptor
		err := binary.Read(r, binary.LittleEndian, &desc)
		if err != nil {
			return nil, err
		}

		if _, ok := sec.DescriptorMap[desc.WemId]; ok {
			panic(fmt.Sprintf(
				"%d is an illegal repeated wem ID in the DIDX", desc.WemId))
		}
		sec.WemIds = append(sec.WemIds, desc.WemId)
		sec.DescriptorMap[desc.WemId] = desc
	}

	return &sec, nil
}

// WriteTo writes the full contents of this DataIndexSection to the Writer
// specified by w.
func (idx *DataIndexSection) WriteTo(w io.Writer) (written int64, err error) {
	err = binary.Write(w, binary.LittleEndian, idx.Header)
	if err != nil {
		return
	}
	written = int64(SECTION_HEADER_BYTES)

	for _, id := range idx.WemIds {
		desc := idx.DescriptorMap[id]
		err = binary.Write(w, binary.LittleEndian, desc)
		if err != nil {
			return
		}
		written += int64(DIDX_ENTRY_BYTES)
	}
	return written, nil
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
	dataOffset, _ := sr.Seek(0, io.SeekCurrent)

	sec := DataSection{hdr, uint32(dataOffset), make([]*Wem, 0)}
	for i, id := range idx.WemIds {
		desc := idx.DescriptorMap[id]
		wemStartOffset := dataOffset + int64(desc.Offset)
		wemReader := io.NewSectionReader(sr, wemStartOffset, int64(desc.Length))

		var remReader io.Reader
		remaining := int64(0)

		if i <= len(idx.WemIds)-1 {
			wemEndOffset := wemStartOffset + int64(desc.Length)
			var nextOffset int64
			if i == len(idx.WemIds)-1 {
				// This is the last wem, check how many bytes remain until the end of
				// the data section.
				nextOffset = dataOffset + int64(hdr.Length)
			} else {
				// This is not the last wem, check how many bytes remain until the next
				// wem.
				nextDesc := idx.DescriptorMap[idx.WemIds[i+1]]
				nextOffset = dataOffset + int64(nextDesc.Offset)
			}
			remaining = nextOffset - wemEndOffset
			// Pass a Reader over the remaining section if we have remaining bytes to
			// read, or an empty Reader if remaining is 0 (no bytes will be read).
			remReader = io.NewSectionReader(sr, wemEndOffset, remaining)
		}

		wem := Wem{wemReader, desc, remReader, remaining}
		sec.Wems = append(sec.Wems, &wem)
	}

	sr.Seek(int64(hdr.Length), io.SeekCurrent)
	return &sec, nil
}

// WriteTo writes the full contents of this DataSection to the Writer specified
// by w.
func (data *DataSection) WriteTo(w io.Writer) (written int64, err error) {
	err = binary.Write(w, binary.LittleEndian, data.Header)
	if err != nil {
		return
	}
	written = int64(SECTION_HEADER_BYTES)
	for _, wem := range data.Wems {
		n, err := io.Copy(w, wem)
		if err != nil {
			return written, err
		}
		written += int64(n)
		n, err = io.Copy(w, wem.RemainingReader)
		if err != nil {
			return written, err
		}
		written += int64(n)
	}

	return written, nil
}

// NewUnknownSection creates a new UnknownSection, reading from sr, which
// must be seeked to the start of the unknown section data.
func (hdr *SectionHeader) NewUnknownSection(sr *io.SectionReader) (*UnknownSection, error) {
	// Get the offset into the file where the data portion of this section begins.
	dataOffset, _ := sr.Seek(0, io.SeekCurrent)
	r := io.NewSectionReader(sr, dataOffset, int64(hdr.Length))
	sr.Seek(int64(hdr.Length), io.SeekCurrent)
	return &UnknownSection{hdr, r}, nil
}

// WriteTo writes the full contents of this UnknownSection to the Writer
// specified by w.
func (unknown *UnknownSection) WriteTo(w io.Writer) (written int64, err error) {
	err = binary.Write(w, binary.LittleEndian, unknown.Header)
	if err != nil {
		return
	}
	written = int64(SECTION_HEADER_BYTES)

	n, err := io.Copy(w, unknown.Reader)
	if err != nil {
		return written, err
	}
	written += int64(n)

	return written, nil
}
