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
	testDir = "testdata"

	simpleSoundBank  = "simple.bnk"
	complexSoundBank = "complex.bnk"

	loopNoneSoundBank     = "loop_none.bnk"
	loop2SoundBank        = "loop_2.bnk"
	loop23SoundBank       = "loop_23.bnk"
	loopInfinitySoundBank = "loop_infinity.bnk"

	// This wem is smaller than the wem at index 0 of simpleSoundBank
	smallerWem = "small.wem"
	// This wem is larger than the wem at index 0 of simpleSoundBank
	largerWem = "large.wem"
)

func TestSimpleUnchangedFileIsEqual(t *testing.T) {
	unchangedFileIsEqual(simpleSoundBank, t)
}

func TestComplexUnchangedFileIsEqual(t *testing.T) {
	unchangedFileIsEqual(complexSoundBank, t)
}

func unchangedFileIsEqual(name string, t *testing.T) {
	skipIfShort(t)

	f, err := os.Open(filepath.Join(testDir, name))
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
	skipIfShort(t)

	bnk, err := Open(filepath.Join(testDir, complexSoundBank))
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
	skipIfShort(t)

	bnk, err := Open(filepath.Join(testDir, complexSoundBank))
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

func TestRegularLoopCases(t *testing.T) {
	skipIfShort(t)

	type loopCase struct {
		input      string
		loopChange LoopValue
		expected   string
	}

	inputs := []string{loop2SoundBank, loopNoneSoundBank, loopInfinitySoundBank}
	var cases []loopCase

	for _, input := range inputs {
		cases = append(cases,
			loopCase{input, LoopValue{false, 0}, loopNoneSoundBank})
		cases = append(cases,
			loopCase{input, LoopValue{true, 23}, loop23SoundBank})
		cases = append(cases,
			loopCase{input, LoopValue{true, 0}, loopInfinitySoundBank})
		cases = append(cases,
			loopCase{input, LoopValue{true, 2}, loop2SoundBank})
	}

	for _, c := range cases {
		bnk, err := Open(filepath.Join(testDir, c.input))
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		bnk.ReplaceLoopOf(0, c.loopChange)

		expect, err := os.Open(filepath.Join(testDir, c.expected))
		if err != nil {
			t.Error(err)
			t.FailNow()
		}

		AssertSoundBankEqualToFile(t, expect, bnk)
	}
}

func skipIfShort(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file comparison test.")
	}
}
