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
