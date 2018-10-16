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

func AssertReplacementOffsetsConsistent(t *testing.T, org Container,
	ctn Container, rs ...*ReplacementWem) {
	var expectedLengths []int64
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

	expectedOffset := int64(org.DataStart() + ctn.Wems()[0].Descriptor.Offset)

	for i, wem := range ctn.Wems() {
		expectedLength := expectedLengths[i]
		newLength := int64(wem.Descriptor.Length)
		if expectedLength != newLength {
			t.Errorf("Wem at index %d was expected to have length %d bytes "+
				"but instead was %d bytes", i, expectedLength, newLength)
		}

		actualOffset := int64(org.DataStart() + wem.Descriptor.Offset)
		if expectedOffset != actualOffset {
			t.Errorf("Wem at index %d was expected to have offset at 0x%X "+
				"but instead was 0x%X", i, expectedOffset, actualOffset)
		}
		expectedOffset += int64(newLength) + wem.Padding.Size()
	}
}
