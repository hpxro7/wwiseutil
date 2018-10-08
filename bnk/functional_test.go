// Package bnk implements access to the Wwise SoundBank file format.
package bnk

// Large system tests for the bnk package.
import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

const (
	testDir         = "testdata"
	simpleSoundBank = "simple.bnk"
)

func TestUnchangedFileIsEqual(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file comparison test.")
	}

	f, err := os.Open(filepath.Join(testDir, simpleSoundBank))
	if err != nil {
		t.Error(err)
	}
	bnk, err := NewFile(f)
	if err != nil {
		t.Error(err)
	}
	AssertSoundBankEqualToFile(t, f, bnk)
}

func AssertSoundBankEqualToFile(t *testing.T, f *os.File, bnk *File) {
	equal, err := false, error(nil)
	f.Seek(0, os.SEEK_CUR)
	bs1 := bufio.NewReader(f)

	bnkBytes := new(bytes.Buffer)
	total, err := bnk.WriteTo(bnkBytes)
	if err != nil {
		t.Error(err)
	}
	stat, _ := f.Stat()
	fileSize := stat.Size()
	if total != fileSize {
		t.Errorf("The number of bytes written was %d bytes, but the file "+
			"was %d bytes", total, fileSize)
	}
	bs2 := bufio.NewReader(bytes.NewReader(bnkBytes.Bytes()))
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
