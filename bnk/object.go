// Package bnk implements access to the Wwise SoundBank file format.
package bnk

import (
	"encoding/binary"
	"fmt"
	"io"
)

// The number of bytes used to describe the a HIRC object.
const OBJECT_DESCRIPTOR_BYTES = 9

// The number of bytes used to describe the ID of a HIRC object.
const OBJECT_DESCRIPTOR_ID_BYTES = 4

// Object represents a single object within the HIRC section.
type Object interface {
	io.WriterTo
	fmt.Stringer
}

// A ObjectDescriptor describes a single object within a HIRC section.
type ObjectDescriptor struct {
	Type byte
	// The length in bytes of the id and data portion of this object.
	Length   uint32
	ObjectId uint32
}

// An UnknownObject represents an unknown object within the HIRC.
type UnknownObject struct {
	Descriptor *ObjectDescriptor
	// A reader to read the data of this section.
	Reader io.Reader
}

// NewUnknownObject creates a new UnknownObject, reading from sr, which must
// be seeked to the start of the unknown object's data.
func (desc *ObjectDescriptor) NewUnknownObject(sr *io.SectionReader) (*UnknownObject, error) {
	// Get the offset into the file where the data portion of this object begins.
	dataOffset, _ := sr.Seek(0, io.SeekCurrent)
	// The descriptor length includes the Object ID, which has already been
	// written. Remove this from the remaining length
	dataLength := int64(desc.Length) - OBJECT_DESCRIPTOR_ID_BYTES
	r := io.NewSectionReader(sr, dataOffset, dataLength)
	sr.Seek(dataLength, io.SeekCurrent)
	return &UnknownObject{desc, r}, nil
}

// WriteTo writes the full contents of this UnknownObject to the Writer
// specified by w.
func (unknown *UnknownObject) WriteTo(w io.Writer) (written int64, err error) {
	err = binary.Write(w, binary.LittleEndian, unknown.Descriptor)
	if err != nil {
		return
	}
	written = int64(OBJECT_DESCRIPTOR_BYTES)

	n, err := io.Copy(w, unknown.Reader)
	if err != nil {
		return written, err
	}
	written += int64(n)

	return written, nil
}

func (unknown *UnknownObject) String() string {
	return ""
}
