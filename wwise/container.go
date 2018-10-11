// Package wwise implements access and modification iterfaces and functions to
// common WWise container formats.
package wwise

import (
	"fmt"
	"io"
	"sort"
)

import (
	"github.com/hpxro7/bnkutil/util"
)

type Container interface {
	io.WriterTo
	fmt.Stringer

	// Wems returns a list of pointers to wems stored in this container. The
	// pointers should point directly to the wem objects used by the container;
	// modifying the contents of these wems should modify the original container.
	Wems() []*Wem

	// ReplaceWems replaces the wems of this Container with all the replacements in
	// rs. The container is updated to match the new expected lengths and offsets.
	ReplaceWems(rs ...*ReplacementWem)

	// DataStart returns the offset into the file where the logical data portion
	// begins. DataStart() + WemDescriptor.Length gives you the true offset of a
	// wem in a file.
	DataStart() uint32
}

// A Wem represents a single sound entity contained within a SoundBank file.
type Wem struct {
	io.Reader
	Descriptor *WemDescriptor
	// A reader over the bytes that remain until the next wem if there is one, or
	// the end of the data section. These bytes are NUL(0x00) padding up until the
	// next 16-aligned byte (i.e. nextWem.Offset % 16 = 0).
	Padding util.ReadSeekerAt
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

// A ReplacementWem defines a wem to be replaced into an original SoundBank File.
type ReplacementWem struct {
	// The reader pointing to the contents of the new wem.
	Wem io.ReaderAt
	// The index, where zero is the first wem, into the original SoundBank's wems
	// to replace.
	WemIndex int
	// The number of bytes to read in for this wem.
	Length int64
}

type ReplacementWems []*ReplacementWem

// ByWemIndex implements the sort.Interface for sorting a slice of
// ReplacementWems in ascending order of their WemIndex.
type ByWemIndex struct {
	ReplacementWems
}

// ReplaceWems replaces the wems of ctn with all the replacements in rs. The
// ctv is updated to match the new expected lengths and offsets. The amount
// of additional space taken up by the new wems is returned. This should be
// used to update the headers of any container as appropriate.
func ReplaceWems(ctn Container, rs ...*ReplacementWem) int64 {
	// Ammending offsets in case of a surplus in a single pass, in O(n) time, as
	// opposed to O(n^2), requires that the replacements happen in the order
	// that their wem will appear in the file; sorting them by index achives this.
	sort.Sort(ByWemIndex{rs})
	// Surplus is the number of bytes a wem offset needs to be increased by,
	// because of a increase in a previous wem's size.
	surplus := int64(0)
	for i, r := range rs {
		wem := ctn.Wems()[r.WemIndex]

		newLength, oldLength := r.Length, int64(wem.Descriptor.Length)
		wem.Reader = util.NewResettingReader(r.Wem, 0, newLength)

		padding := wem.Padding.Size()
		if newLength > oldLength {
			surplus += newLength - oldLength
			// Compute the new amount of padding needed to align the next offset (true
			// end of this wem section) with 16 bytes.
			padding = (16 - (int64(wem.Descriptor.Offset)+newLength)%16)
			// Subsequent wem's will need to have their offsets aligned with the end
			// of our new wem's padding. The offset difference will need to include
			// the difference in padding between the old wem and the replacement wem.
			surplus += padding - wem.Padding.Size()
		} else { // newLength <= oldLength
			// Pad from the end of the new wem to the offset of the next wem.
			padding += int64(oldLength - newLength)
		}

		// Update the length of the descriptor. This, by pointer dereference,
		// updates the descriptor stored in the IndexSection's DescriptorMap, as
		// well.
		wem.Descriptor.Length = uint32(newLength)
		wem.Padding = util.NewResettingReader(&util.InfiniteReaderAt{0}, 0, padding)

		if surplus > 0 {
			// Shift the offsets for the next wems, since the current wem is going to
			// take up more space than it originally was. Do this up to and including
			// the next replacement wem, if any. After that point, we'll need to
			// re-evaluate our surplus.
			for wi := r.WemIndex + 1; wi <= len(ctn.Wems())-1; wi++ {
				wem := ctn.Wems()[wi]
				wem.Descriptor.Offset += uint32(surplus)
				if i+1 < len(rs) && wi == rs[i+1].WemIndex {
					// We have just replaced the offset for the next replacement wem. Stop
					// ammending offsets as we might have a different surplus after
					// replacing that wem.
					break
				}
			}
		}
	}

	return surplus
}

func (rs ReplacementWems) Len() int {
	return len(rs)
}

func (rs ReplacementWems) Swap(i, j int) {
	rs[i], rs[j] = rs[j], rs[i]
}

func (b ByWemIndex) Less(i, j int) bool {
	return b.ReplacementWems[i].WemIndex < b.ReplacementWems[j].WemIndex
}
