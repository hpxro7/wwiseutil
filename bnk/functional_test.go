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
	// This wem is smaller than the wem at index 0 of simpleSoundBank
	smallerWem = "small.wem"
	// This wem is larger than the wem at index 0 of simpleSoundBank
	largerWem = "large.wem"
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

func TestReplaceFirstWemWithSmaller(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file comparison test.")
	}

	bnk, err := Open(filepath.Join(testDir, simpleSoundBank))
	if err != nil {
		t.Error(err)
	}
	wem, err := os.Open(filepath.Join(testDir, smallerWem))
	if err != nil {
		t.Error(err)
	}
	stat, _ := wem.Stat()
	bnk.ReplaceWems(&ReplacementWem{wem, 0, stat.Size()})

	expect, err := os.Open(filepath.Join(testDir, "0_replaced_with_smaller.bnk"))
	if err != nil {
		t.Error(err)
	}

	AssertSoundBankEqualToFile(t, expect, bnk)
}

func TestReplaceFirstWemWithLarger(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file comparison test.")
	}

	bnk, err := Open(filepath.Join(testDir, simpleSoundBank))
	if err != nil {
		t.Error(err)
	}
	wem, err := os.Open(filepath.Join(testDir, largerWem))
	if err != nil {
		t.Error(err)
	}
	stat, _ := wem.Stat()
	bnk.ReplaceWems(&ReplacementWem{wem, 0, stat.Size()})

	expect, err := os.Open(filepath.Join(testDir, "0_replaced_with_larger.bnk"))
	if err != nil {
		t.Error(err)
	}

	AssertSoundBankEqualToFile(t, expect, bnk)
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
	actualTotal := int64(len(bnkBytes.Bytes()))
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
