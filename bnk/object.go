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

const OVERRIDE_EFFECTS_BYTES = 1
const SFX_UNKNOWN_BYTES = 5
const OPTIONAL_WEM_DESCRIPTOR_BYTES = 8
const EFFECT_BYTES = 7
const PARAMETER_VALUE_BYTES = 4
const STRUCTURE_UNKNOWN_BYTES = 10

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

	Unknown       *[5]byte
	WemDescriptor *OptionalWemDescriptor
	// Determines whether this sound object is a SFX or Voice type.
	Type byte

	Structure *SoundStructure
}

// A OptionalWemDescriptor provides information about where a wem is stored from
// a SfxVoiceSourceObject. If the audio is streamed, this struct will still be
// read in, but it is unknown what its values correspond to.
type OptionalWemDescriptor struct {
	// This will be id of the wem referred to by this object.
	OptionalWemId uint32
	// If the sound is embedded, this will be length of the wem. If not, it is an
	// unknown number.
	OptionalWemLength uint32
}

// An UnknownObject represents an unknown object within the HIRC.
type UnknownObject struct {
	Descriptor *ObjectDescriptor
	// A reader to read the data of this section.
	Reader io.Reader
}

// A SoundStructure describes a variety of properties that define how an audio
// object should be played.
type SoundStructure struct {
	OverrideParentEffects byte
	EffectContainer       *EffectContainer
	Unknown               *[10]byte
	ParameterCount        byte
	ParameterTypes        []byte
	ParameterValues       [][4]byte
	// A reader to read the remaining data of this structure.
	RemainingReader io.Reader
}

// An EffectsContainer describes a set of effects applied to an audio object.
type EffectContainer struct {
	EffectCount byte
	// A bit mask specifying which effects are bypassed.
	Bypass  byte
	Effects []*Effect
}

// An Effect describes the type of effect applied to an audio object.
type Effect struct {
	Index   byte
	Id      uint32
	Padding [2]byte
}

// NewSfxVoiceSoundObject creates a new SfxVoiceSoundObject, reading from sr,
// which must be seeked to the start of the object's data.
func (desc *ObjectDescriptor) NewSfxVoiceSoundObject(sr *io.SectionReader) (*SfxVoiceSoundObject, error) {
	// Get the offset into the file where the data portion of this object begins.
	startOffset, _ := sr.Seek(0, io.SeekCurrent)
	// The descriptor length includes the Object ID, which has already been
	// written. Remove this from the remaining length.
	dataLength := int64(desc.Length) - OBJECT_DESCRIPTOR_ID_BYTES
	unknown := new([5]byte)
	err := binary.Read(sr, binary.LittleEndian, unknown)
	if err != nil {
		return nil, err
	}

	wd := new(OptionalWemDescriptor)
	err = binary.Read(sr, binary.LittleEndian, wd)
	if err != nil {
		return nil, err
	}

	var soundType byte
	err = binary.Read(sr, binary.LittleEndian, &soundType)
	if err != nil {
		return nil, err
	}

	ssOffset, _ := sr.Seek(0, io.SeekCurrent)
	remaining := dataLength - (ssOffset - startOffset)

	ss, err := NewSoundStructure(sr, remaining)
	if err != nil {
		return nil, err
	}

	return &SfxVoiceSoundObject{desc, unknown, wd, soundType, ss}, nil
}

// WriteTo writes the full contents of this SfxVoiceSoundObject to the Writer
// specified by w.
func (sound *SfxVoiceSoundObject) WriteTo(w io.Writer) (written int64, err error) {
	err = binary.Write(w, binary.LittleEndian, sound.Descriptor)
	if err != nil {
		return
	}
	written = OBJECT_DESCRIPTOR_BYTES

	err = binary.Write(w, binary.LittleEndian, sound.Unknown)
	if err != nil {
		return
	}
	written += SFX_UNKNOWN_BYTES

	err = binary.Write(w, binary.LittleEndian, sound.WemDescriptor)
	if err != nil {
		return
	}
	written += OPTIONAL_WEM_DESCRIPTOR_BYTES

	err = binary.Write(w, binary.LittleEndian, sound.Type)
	if err != nil {
		return
	}
	written += 1

	n, err := sound.Structure.WriteTo(w)
	if err != nil {
		return written, err
	}
	written += n

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

// NewSoundStructure creates a new SoundStructure, reading from sr, which must be
// seeked to the start of the structure's data.
func NewSoundStructure(sr *io.SectionReader, length int64) (*SoundStructure, error) {
	// Get the offset into the file where the structure begins.
	startOffset, _ := sr.Seek(0, io.SeekCurrent)
	var override byte
	err := binary.Read(sr, binary.LittleEndian, &override)
	if err != nil {
		return nil, err
	}

	ctr, err := NewEffectContainer(sr)
	if err != nil {
		return nil, err
	}

	unknown := new([10]byte)
	err = binary.Read(sr, binary.LittleEndian, unknown)
	if err != nil {
		return nil, err
	}

	var count byte
	err = binary.Read(sr, binary.LittleEndian, &count)
	if err != nil {
		return nil, err
	}

	var types []byte
	var values [][4]byte

	// Read in parameter types.
	for i := byte(0); i < count; i++ {
		var t byte
		err = binary.Read(sr, binary.LittleEndian, &t)
		if err != nil {
			return nil, err
		}
		types = append(types, t)
	}

	// Read in parameter values.
	for i := byte(0); i < count; i++ {
		var v [4]byte
		err = binary.Read(sr, binary.LittleEndian, &v)
		if err != nil {
			return nil, err
		}
		values = append(values, v)
	}

	// Create a reader over the remaining elements in this object, then seek past
	// it.
	currOffset, _ := sr.Seek(0, io.SeekCurrent)
	remaining := length - (currOffset - startOffset)
	r := io.NewSectionReader(sr, currOffset, remaining)
	sr.Seek(remaining, io.SeekCurrent)
	return &SoundStructure{override, ctr, unknown, count, types, values, r}, nil
}

func (ss *SoundStructure) WriteTo(w io.Writer) (written int64, err error) {
	err = binary.Write(w, binary.LittleEndian, ss.OverrideParentEffects)
	if err != nil {
		return
	}
	written = OVERRIDE_EFFECTS_BYTES

	n, err := ss.EffectContainer.WriteTo(w)
	if err != nil {
		return
	}
	written += n

	err = binary.Write(w, binary.LittleEndian, ss.Unknown)
	if err != nil {
		return
	}
	written += STRUCTURE_UNKNOWN_BYTES

	err = binary.Write(w, binary.LittleEndian, ss.ParameterCount)
	if err != nil {
		return
	}
	written += 1

	err = binary.Write(w, binary.LittleEndian, ss.ParameterTypes)
	if err != nil {
		return
	}
	written += int64(ss.ParameterCount)

	err = binary.Write(w, binary.LittleEndian, ss.ParameterValues)
	if err != nil {
		return
	}
	written += int64(ss.ParameterCount) * PARAMETER_VALUE_BYTES

	n, err = io.Copy(w, ss.RemainingReader)
	if err != nil {
		return written, err
	}
	written += n

	return written, nil
}

// NewEffectContainer creates a new EffectContainer, reading from sr, which must
// be seeked to the start of the container.
func NewEffectContainer(sr *io.SectionReader) (*EffectContainer, error) {
	var count byte
	err := binary.Read(sr, binary.LittleEndian, &count)
	if err != nil {
		return nil, err
	}

	var bypass byte
	var effects []*Effect
	if count > 0 {
		err := binary.Read(sr, binary.LittleEndian, &bypass)
		if err != nil {
			return nil, err
		}

		for i := byte(0); i < count; i++ {
			effect := new(Effect)
			err := binary.Read(sr, binary.LittleEndian, effect)
			if err != nil {
				return nil, err
			}
			effects = append(effects, effect)
		}
	}
	return &EffectContainer{count, bypass, effects}, nil
}

// WriteTo writes the full contents of this EffectContainer to the Writer
// specified by w.
func (e *EffectContainer) WriteTo(w io.Writer) (written int64, err error) {
	err = binary.Write(w, binary.LittleEndian, e.EffectCount)
	if err != nil {
		return
	}
	written = 1

	if e.EffectCount > 0 {
		err = binary.Write(w, binary.LittleEndian, e.Bypass)
		if err != nil {
			return
		}
		written += 1
		for _, effect := range e.Effects {
			err = binary.Write(w, binary.LittleEndian, effect)
			if err != nil {
				return
			}
			written += EFFECT_BYTES
		}
	}
	return
}
