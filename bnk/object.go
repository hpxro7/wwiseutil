// Package bnk implements access to the Wwise SoundBank file format.
package bnk

import (
	"encoding/binary"
	"io"
)

// The number of bytes used to describe the a HIRC object.
const OBJECT_DESCRIPTOR_BYTES = 9

// The number of bytes used to describe the ID of a HIRC object.
const OBJECT_DESCRIPTOR_ID_BYTES = 4

const SOUND_DESCRIPTOR_BYTES = 16
const OPTIONAL_WEM_DESCRIPTOR_BYTES = 8

// The identifier for SFX or Voice sound objects.
const soundObjectId = 0x02

// The wem is embedded in this sound file.
const streamSettingEmbedded = 0x00

// Object represents a single object within the HIRC section.
type Object interface {
	io.WriterTo
}

// A ObjectDescriptor describes a single object within a HIRC section.
type ObjectDescriptor struct {
	Type byte
	// The length in bytes of the id and data portion of this object.
	Length   uint32
	ObjectId uint32
}

// An SfxVoiceSoundObject represents a Voice/SFX Sound object within the HIRC
// section.
type SfxVoiceSoundObject struct {
	Descriptor *ObjectDescriptor

	SoundDescriptor *SoundObjectDescriptor
	WemDescriptor   *OptionalWemDescriptor
	// Determines whether this sound object is a SFX or Voice type.
	Type byte
	// A reader to read the remaining data of this section.
	RemainingReader io.Reader
}

// A SoundObjectDescriptor describes the location and properties of a sound
// object.
type SoundObjectDescriptor struct {
	Unknown [4]byte
	// Determines whether the sound is embedded is the SoundBank or streamed.
	StreamSetting uint32
	AudioId       uint32
	// If the file is embedded, this will be the source SoundBank id from the STID
	// section. If the file is being streamed, this will be the same as AudioId.
	SourceId uint32
}

// A OptionalWemDescriptor provides information about where a wem is stored from
// a SfxVoiceSourceObject. This will only be in the SoundObject if the sound
// is not streamed.
type OptionalWemDescriptor struct {
	// If the sound is embedded, this will be offset of the wem from the start of
	// the file. If not, it will not exist.
	OptionalWemOffset uint32
	// If the sound is embedded, this will be length of the wem. If not, it will
	// not exist.
	OptionalWemLength uint32
}

// An UnknownObject represents an unknown object within the HIRC.
type UnknownObject struct {
	Descriptor *ObjectDescriptor
	// A reader to read the data of this section.
	Reader io.Reader
}

// NewSfxVoiceSoundObject creates a new SfxVoiceSoundObject, reading from sr,
// which must be seeked to the start of the object's data.
func (desc *ObjectDescriptor) NewSfxVoiceSoundObject(sr *io.SectionReader) (*SfxVoiceSoundObject, error) {
	// Get the offset into the file where the data portion of this object begins.
	startOffset, _ := sr.Seek(0, io.SeekCurrent)
	// The descriptor length includes the Object ID, which has already been
	// written. Remove this from the remaining length.
	dataLength := int64(desc.Length) - OBJECT_DESCRIPTOR_ID_BYTES
	//fmt.Print("TO READ:", dataLength)
	sd := new(SoundObjectDescriptor)
	err := binary.Read(sr, binary.LittleEndian, sd)
	if err != nil {
		return nil, err
	}
	var wd *OptionalWemDescriptor
	if sd.StreamSetting == streamSettingEmbedded {
		wd = new(OptionalWemDescriptor)
		err := binary.Read(sr, binary.LittleEndian, wd)
		if err != nil {
			return nil, err
		}
	}

	var soundType byte
	wd = new(OptionalWemDescriptor)
	err = binary.Read(sr, binary.LittleEndian, &soundType)
	if err != nil {
		return nil, err
	}

	// The start offset of the sound structure.
	ssOffset, _ := sr.Seek(0, io.SeekCurrent)
	//fmt.Print(" . Read:", ssOffset-startOffset)
	remaining := dataLength - (ssOffset - startOffset)
	//fmt.Println(" . Remaining:", remaining)
	r := io.NewSectionReader(sr, ssOffset, remaining)
	sr.Seek(remaining, io.SeekCurrent)
	return &SfxVoiceSoundObject{desc, sd, wd, soundType, r}, nil
}

// WriteTo writes the full contents of this SfxVoiceSoundObject to the Writer
// specified by w.
func (sound *SfxVoiceSoundObject) WriteTo(w io.Writer) (written int64, err error) {
	err = binary.Write(w, binary.LittleEndian, sound.Descriptor)
	if err != nil {
		return
	}
	written = int64(OBJECT_DESCRIPTOR_BYTES)

	err = binary.Write(w, binary.LittleEndian, sound.SoundDescriptor)
	if err != nil {
		return
	}
	written += int64(SOUND_DESCRIPTOR_BYTES)

	if sound.SoundDescriptor.StreamSetting == streamSettingEmbedded {
		err = binary.Write(w, binary.LittleEndian, sound.WemDescriptor)
		if err != nil {
			return
		}
		written += int64(OPTIONAL_WEM_DESCRIPTOR_BYTES)
	}

	err = binary.Write(w, binary.LittleEndian, sound.Type)
	if err != nil {
		return
	}
	written += int64(1)

	n, err := io.Copy(w, sound.RemainingReader)
	if err != nil {
		return written, err
	}
	written += int64(n)

	return written, nil
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
