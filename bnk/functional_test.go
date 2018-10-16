// Package bnk implements access to the Wwise SoundBank file format.
package bnk

// Large system tests for the bnk package.
import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

import (
	"github.com/hpxro7/wwiseutil/util"
	"github.com/hpxro7/wwiseutil/wwise"
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
	util.SkipIfShort(t)

	f, err := os.Open(filepath.Join(testDir, name))
	if err != nil {
		t.Error(err)
	}
	bnk, err := NewFile(f)
	if err != nil {
		t.Error(err)
	}
	wwise.AssertContainerEqualToFile(t, f, bnk)
}

func TestUnchangedWriteFileTwiceIsEqual(t *testing.T) {
	util.SkipIfShort(t)

	f, err := os.Open(filepath.Join(testDir, complexSoundBank))
	if err != nil {
		t.Error(err)
	}
	bnk, err := NewFile(f)
	if err != nil {
		t.Error(err)
	}
	wwise.AssertContainerEqualToFile(t, f, bnk)
	f, err = os.Open(filepath.Join(testDir, complexSoundBank))
	if err != nil {
		t.Error(err)
	}
	wwise.AssertContainerEqualToFile(t, f, bnk)
}

func TestReplaceFirstWemWithSmaller(t *testing.T) {
	util.SkipIfShort(t)

	wem, err := os.Open(filepath.Join(testDir, smallerWem))
	if err != nil {
		t.Error(err)
	}
	stat, _ := wem.Stat()
	rs := []*wwise.ReplacementWem{&wwise.ReplacementWem{wem, 0, stat.Size()}}
	assertReplacedFileCorrectness(t, complexSoundBank, rs...)
}

func TestReplaceFirstWemWithLarger(t *testing.T) {
	util.SkipIfShort(t)

	wem, err := os.Open(filepath.Join(testDir, largerWem))
	if err != nil {
		t.Error(err)
	}
	stat, _ := wem.Stat()
	rs := []*wwise.ReplacementWem{&wwise.ReplacementWem{wem, 0, stat.Size()}}
	assertReplacedFileCorrectness(t, complexSoundBank, rs...)
}

func TestReplaceLoopOfCases(t *testing.T) {
	util.SkipIfShort(t)

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

		wwise.AssertContainerEqualToFile(t, expect, bnk)
	}
}

func rereadFile(t *testing.T, org *File) *File {
	orgBytes := new(bytes.Buffer)
	_, err := org.WriteTo(orgBytes)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	bs := bytes.NewReader(orgBytes.Bytes())

	ctn, err := NewFile(bs)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	return ctn
}

func assertReplacedFileCorrectness(t *testing.T, bnkPath string,
	rs ...*wwise.ReplacementWem) {
	org, err := Open(filepath.Join(testDir, bnkPath))
	if err != nil {
		t.Error(err)
	}

	replaced, err := Open(filepath.Join(testDir, bnkPath))
	if err != nil {
		t.Error(err)
	}
	replaced.ReplaceWems(rs...)
	replaced = rereadFile(t, replaced)

	wwise.AssertReplacementOffsetsConsistent(t, org, replaced, rs...)
	actualLength := int64(0)
	for i, wem := range replaced.Wems() {
		actualLength += int64(wem.Descriptor.Length) + wem.Padding.Size()
		// Check that offsets are byte aligned.
		offset := wem.Descriptor.Offset
		if offset%wemAlignmentBytes != 0 {
			t.Errorf("The wem at index %d has an offset of 0x%X, which is not "+
				"byte aligned by %d", i, offset, wemAlignmentBytes)
		}
	}
	expectedLength := int64(replaced.DataSection.Header.Length)
	if expectedLength != actualLength {
		t.Errorf("The total size of wems is %d bytes, but the data section header "+
			"only reports %d bytes", actualLength, expectedLength)
	}

}
