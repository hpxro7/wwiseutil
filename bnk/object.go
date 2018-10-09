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
const EFFECT_BYTES = 7

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
	Type      byte
	Structure *SoundStructure
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

// A SoundStructure describes a variety of properties that define how an audio
// object should be played.
type SoundStructure struct {
	OverrideParentEffects bool
	EffectContainer       *EffectContainer
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

	ssOffset, _ := sr.Seek(0, io.SeekCurrent)
	remaining := dataLength - (ssOffset - startOffset)

	ss, err := NewSoundStructure(sr, remaining)
	if err != nil {
		return nil, err
	}

	return &SfxVoiceSoundObject{desc, sd, wd, soundType, ss}, nil
}

// WriteTo writes the full contents of this SfxVoiceSoundObject to the Writer
// specified by w.
func (sound *SfxVoiceSoundObject) WriteTo(w io.Writer) (written int64, err error) {
	err = binary.Write(w, binary.LittleEndian, sound.Descriptor)
	if err != nil {
		return
	}
	written = OBJECT_DESCRIPTOR_BYTES

	err = binary.Write(w, binary.LittleEndian, sound.SoundDescriptor)
	if err != nil {
		return
	}
	written += SOUND_DESCRIPTOR_BYTES

	if sound.SoundDescriptor.StreamSetting == streamSettingEmbedded {
		err = binary.Write(w, binary.LittleEndian, sound.WemDescriptor)
		if err != nil {
			return
		}
		written += OPTIONAL_WEM_DESCRIPTOR_BYTES
	}

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
	var override bool
	err := binary.Read(sr, binary.LittleEndian, &override)
	if err != nil {
		return nil, err
	}

	container, err := NewEffectContainer(sr)
	if err != nil {
		return nil, err
	}

	// Create a reader over the remaining elements in this object, then seek past
	// it.
	currOffset, _ := sr.Seek(0, io.SeekCurrent)
	remaining := length - (currOffset - startOffset)
	r := io.NewSectionReader(sr, currOffset, remaining)
	sr.Seek(remaining, io.SeekCurrent)
	return &SoundStructure{override, container, r}, nil
}

func (ss *SoundStructure) WriteTo(w io.Writer) (written int64, err error) {
	err = binary.Write(w, binary.LittleEndian, ss.OverrideParentEffects)
	if err != nil {
		return
	}
	written = int64(1)

	n, err := ss.EffectContainer.WriteTo(w)
	if err != nil {
		return
	}
	written += n

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
