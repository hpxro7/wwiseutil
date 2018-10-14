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

import (
	"github.com/hpxro7/wwiseutil/util"
	"github.com/hpxro7/wwiseutil/wwise"
)

// The wem byte alignment requirement for SoundBank files.
const wemAlignmentBytes = 16

// A LoopValue identifier for looping infinite times.
const InfiniteLoops = 0

// A File represents an open Wwise SoundBank.
type File struct {
	closer io.Closer
	// The list of sections in this SoundBank, in the order that they are expected
	// to be found in the file.
	sections          []Section
	BankHeaderSection *BankHeaderSection
	IndexSection      *DataIndexSection
	DataSection       *DataSection
	ObjectSection     *ObjectHierarchySection
}

// LoopValue describes the loop parameters of a given audio object.
type LoopValue struct {
	// True if this audio object loops; and false if otherwise.
	Loops bool
	// The number of times this audio track will play. 0 means that this audio will
	// play infinite times. This value is not vaild if loops is false.
	Value uint32
}

// NewFile creates a new File for access Wwise SoundBank files. The file is
// expected to start at position 0 in the io.ReaderAt.
func NewFile(r io.ReaderAt) (*File, error) {
	bnk := new(File)

	sr := util.NewResettingReader(r, 0, math.MaxInt64)
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
			bnk.sections = append(bnk.sections, sec)
		case didxHeaderId:
			sec, err := hdr.NewDataIndexSection(sr)
			if err != nil {
				return nil, err
			}
			bnk.IndexSection = sec
			bnk.sections = append(bnk.sections, sec)
		case dataHeaderId:
			sec, err := hdr.NewDataSection(sr, bnk.IndexSection)
			if err != nil {
				return nil, err
			}
			bnk.DataSection = sec
			bnk.sections = append(bnk.sections, sec)
		case hircHeaderId:
			sec, err := hdr.NewObjectHierarchySection(sr)
			if err != nil {
				return nil, err
			}
			bnk.ObjectSection = sec
			bnk.sections = append(bnk.sections, sec)
		default:
			sec, err := hdr.NewUnknownSection(sr)
			if err != nil {
				return nil, err
			}
			bnk.sections = append(bnk.sections, sec)
		}
	}

	if bnk.DataSection == nil || len(bnk.Wems()) == 0 {
		return nil, errors.New("There are no wems stored within this file.")
	}

	return bnk, nil
}

// WriteTo writes the full contents of this File to the Writer specified by w.
func (bnk *File) WriteTo(w io.Writer) (written int64, err error) {
	for _, s := range bnk.sections {
		n, err := s.WriteTo(w)
		if err != nil {
			return written, err
		}
		written += n
	}
	return
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

func (bnk *File) Wems() []*wwise.Wem {
	if bnk.DataSection == nil {
		return nil
	}
	return bnk.DataSection.Wems
}

func (bnk *File) ReplaceWems(rs ...*wwise.ReplacementWem) {
	surplus := wwise.ReplaceWems(bnk, wemAlignmentBytes, rs...)

	if surplus > 0 {
		// Update the length of the DATA header to account for the change in size.
		bnk.DataSection.Header.Length += uint32(surplus)
	}
}

func (bnk *File) DataStart() uint32 {
	return bnk.DataSection.DataStart
}

// LoopOf returns the loop value of the wem stored in this SoundBank at index i.
// Returns a default LoopValue{false, 0} if the index is invalid.
func (bnk *File) LoopOf(i int) LoopValue {
	value := LoopValue{false, 0}
	if bnk.DataSection == nil {
		return value
	}

	wems := bnk.DataSection.Wems
	if i < 0 || i >= len(wems) {
		return value
	}

	desc := bnk.DataSection.Wems[i].Descriptor
	if bnk.ObjectSection == nil {
		return value
	}

	times, ok := bnk.ObjectSection.loopOf[desc.WemId]
	return LoopValue{ok, times}
}

// ReplaceLoopOf replaces the loop value of the wem stored in this SoundBank at
// index i with the new value. This method is idempotent.
func (bnk *File) ReplaceLoopOf(i int, loop LoopValue) {
	if bnk.DataSection == nil {
		return
	}

	wems := bnk.DataSection.Wems
	if i < 0 || i >= len(wems) {
		return
	}

	desc := bnk.DataSection.Wems[i].Descriptor
	if bnk.ObjectSection == nil {
		return
	}

	oldValue, oldLoops := bnk.ObjectSection.loopOf[desc.WemId]
	// Return if the loop values aren't changing.
	if oldLoops == false && loop.Loops == false || ((oldLoops == loop.Loops) &&
		oldValue == loop.Value) {
		return
	}

	// The HIRC object that maps to the target wem.
	object, ok := bnk.ObjectSection.wemToObject[desc.WemId]
	if !ok {
		return
	}
	// The sound structure that maps to the target wem.
	ss := object.Structure

	if loop.Loops == false {
		// We are removing looping from an audio object that already has a loop.
		for i, paramType := range ss.ParameterTypes {
			if paramType == parameterLoopType {
				ss.ParameterCount--
				ss.ParameterTypes =
					append(ss.ParameterTypes[:i], ss.ParameterTypes[i+1:]...)
				ss.ParameterValues =
					append(ss.ParameterValues[:i], ss.ParameterValues[i+1:]...)

				lengthDecrease := uint32(PARAMETER_TYPE_BYTES + PARAMETER_VALUE_BYTES)
				bnk.ObjectSection.Header.Length -= lengthDecrease
				object.Descriptor.Length -= lengthDecrease

				delete(bnk.ObjectSection.loopOf, desc.WemId)
				return
			}
		}
	} else {
		var lbs [4]byte
		binary.LittleEndian.PutUint32(lbs[:], loop.Value)
		if oldLoops {
			// We are modifying the existing loop value of an audio object.
			for i, paramType := range ss.ParameterTypes {
				if paramType == parameterLoopType {
					ss.ParameterValues[i] = lbs
					bnk.ObjectSection.loopOf[desc.WemId] = loop.Value
					return
				}
			}
		} else { // oldLoops == false
			// We are adding looping to an audio object that did not loop.
			ss := object.Structure
			ss.ParameterCount++
			ss.ParameterTypes = append(ss.ParameterTypes, parameterLoopType)
			ss.ParameterValues = append(ss.ParameterValues, lbs)
			bnk.ObjectSection.loopOf[desc.WemId] = loop.Value

			lengthIncrease := uint32(PARAMETER_TYPE_BYTES + PARAMETER_VALUE_BYTES)
			bnk.ObjectSection.Header.Length += lengthIncrease
			object.Descriptor.Length += lengthIncrease
		}
	}
}

func (bnk *File) String() string {
	b := new(strings.Builder)

	for _, sec := range bnk.sections {
		b.WriteString(sec.String())
	}

	tableParams := []string{"%-7", "%-15", "%-15", "%-15", "%-8", "%-12", "\n"}
	titleFmt := strings.Join(tableParams, "s|")
	wemFmt := strings.Join(tableParams, "d|")
	title := fmt.Sprintf(titleFmt,
		"Index", "Id", "Offset", "Length", "Padding", "Loop (0=Inf)")
	fmt.Fprint(b, title)
	fmt.Fprintln(b, strings.Repeat("-", len(title)-1))

	for i, wem := range bnk.DataSection.Wems {
		desc := wem.Descriptor
		l := bnk.LoopOf(i)
		loop := -1
		if l.Loops {
			loop = int(l.Value)
		}

		fmt.Fprintf(b, wemFmt, i+1, desc.WemId, desc.Offset, desc.Length,
			wem.Padding.Size(), loop)
	}

	return b.String()
}
