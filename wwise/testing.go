// Package wwise implements access and modification iterfaces and functions to
// common WWise container formats.
package wwise

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"testing"
)

import "github.com/hpxro7/wwiseutil/util"

const (
	// The number of bytes to add or subtract from when testing replacing larger
	// or smaller wems
	wemDifference = 200
)

type replacementSpec struct {
	Index int
	// If true, Index will be ignored and the last wem will be used instead
	UseLast bool
	// If true, this test replacement will be larger than the original; if false,
	// it will be smaller
	Larger bool
}

type replacementTest []replacementSpec
type replacementTestCase struct {
	Name string
	Test replacementTest
}

var ReplacementTestCases = []replacementTestCase{
	{"ReplaceFirstWemWithSmaller", replacementTest{
		{Index: 0, Larger: false},
	}},
	{"ReplaceFirstWemWithLarger", replacementTest{
		{Index: 0, Larger: true},
	}},
	{"ReplaceLastWemWithSmaller", replacementTest{
		{UseLast: true, Larger: false},
	}},
	{"ReplaceLastWemWithLarger", replacementTest{
		{UseLast: true, Larger: true},
	}},
}

func AssertContainerEqualToFile(t *testing.T, f *os.File, pck Container) {
	equal, err := false, error(nil)
	f.Seek(0, os.SEEK_CUR)
	bs1 := bufio.NewReader(f)

	pckBytes := new(bytes.Buffer)
	total, err := pck.WriteTo(pckBytes)
	if err != nil {
		t.Error(err)
	}
	actualTotal := int64(len(pckBytes.Bytes()))
	if total != actualTotal {
		t.Errorf("%d bytes were actually written, but %d bytes were "+
			"reported to be written", actualTotal, total)
		t.FailNow()
	}
	stat, _ := f.Stat()
	fileSize := stat.Size()
	if total != fileSize {
		t.Errorf("The number of bytes written was %d bytes, but the file "+
			"was %d bytes", total, fileSize)
		t.FailNow()
	}
	bs2 := bufio.NewReader(bytes.NewReader(pckBytes.Bytes()))
	for {
		b1, err1 := bs1.ReadByte()
		b2, err2 := bs2.ReadByte()
		if err1 != nil && err1 != io.EOF {
			equal, err = false, err1
			break
		}
		if err2 != nil && err2 != io.EOF {
			equal, err = false, err2
			break
		}
		if err1 == io.EOF || err2 == io.EOF {
			equal, err = err1 == err2, nil
			break
		}
		if b1 != b2 {
			equal, err = false, nil
			break
		}
	}

	if err != nil {
		t.Error(err)
	}
	if !equal {
		t.Error("The two files have the same size but are not equal.")
	}
}

func AssertReplacementsConsistent(t *testing.T, org Container,
	replaced Container, reread Container, rs ...*ReplacementWem) (failed bool) {
	var expectedLengths []int64
	var expectedOffsets []int64
	replacementFrom := make(map[int]*ReplacementWem)

	for _, r := range rs {
		replacementFrom[r.WemIndex] = r
	}

	for i, wem := range org.Wems() {
		r, replacing := replacementFrom[i]
		if replacing {
			expectedLengths = append(expectedLengths, r.Length)
		} else {
			expectedLengths = append(expectedLengths, int64(wem.Descriptor.Length))
		}
	}

	currOffset := int64(replaced.DataStart() +
		replaced.Wems()[0].Descriptor.Offset)
	for _, wem := range replaced.Wems() {
		expectedOffsets = append(expectedOffsets, currOffset)
		currOffset += int64(wem.Descriptor.Length) + wem.Padding.Size()
	}

	for i, wem := range reread.Wems() {
		expectedLength := expectedLengths[i]
		newLength := int64(wem.Descriptor.Length)
		if expectedLength != newLength {
			t.Errorf("Wem at index %d was expected to have length %d bytes "+
				"but instead was %d bytes", i, expectedLength, newLength)
			failed = true
		}

		actualOffset := int64(reread.DataStart() + wem.Descriptor.Offset)
		if expectedOffsets[i] != actualOffset {
			t.Errorf("Wem at index %d was expected to have offset at 0x%X "+
				"but instead was 0x%X", i, expectedOffsets[i], actualOffset)
			failed = true
		}
	}
	return
}

func (rts replacementTest) Expand(org Container) []*ReplacementWem {
	var rs []*ReplacementWem
	for _, rt := range rts {
		index := rt.Index
		if rt.UseLast {
			index = len(org.Wems()) - 1
		}
		prevSize := int64(org.Wems()[index].Descriptor.Length)
		var newSize int64
		if rt.Larger {
			newSize = prevSize + wemDifference
		} else {
			newSize = prevSize - wemDifference
		}
		wem := util.NewConstantReader(newSize)

		rs = append(rs, &ReplacementWem{wem, index, newSize})
	}
	return rs
}
