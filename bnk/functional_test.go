// Package bnk implements access to the Wwise SoundBank file format.
package bnk

// Large system tests for the bnk package.
import (
	"os"
	"path/filepath"
	"testing"
)

import (
	"github.com/hpxro7/bnkutil/util"
	"github.com/hpxro7/bnkutil/wwise"
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

	bnk, err := Open(filepath.Join(testDir, complexSoundBank))
	if err != nil {
		t.Error(err)
	}
	wem, err := os.Open(filepath.Join(testDir, smallerWem))
	if err != nil {
		t.Error(err)
	}
	stat, _ := wem.Stat()
	bnk.ReplaceWems(&wwise.ReplacementWem{wem, 0, stat.Size()})

	expect, err := os.Open(filepath.Join(testDir, "0_replaced_with_smaller.bnk"))
	if err != nil {
		t.Error(err)
	}

	wwise.AssertContainerEqualToFile(t, expect, bnk)
}

func TestReplaceFirstWemWithLargerTwice(t *testing.T) {
	util.SkipIfShort(t)

	bnk, err := Open(filepath.Join(testDir, complexSoundBank))
	if err != nil {
		t.Error(err)
	}
	wem, err := os.Open(filepath.Join(testDir, largerWem))
	if err != nil {
		t.Error(err)
	}
	stat, _ := wem.Stat()
	bnk.ReplaceWems(&wwise.ReplacementWem{wem, 0, stat.Size()})

	expect, err := os.Open(filepath.Join(testDir, "0_replaced_with_larger.bnk"))
	if err != nil {
		t.Error(err)
	}
	wwise.AssertContainerEqualToFile(t, expect, bnk)

	expect, err = os.Open(filepath.Join(testDir, "0_replaced_with_larger.bnk"))
	if err != nil {
		t.Error(err)
	}
	wwise.AssertContainerEqualToFile(t, expect, bnk)
}

func TestReplaceFirstWemWithLarger(t *testing.T) {
	util.SkipIfShort(t)

	bnk, err := Open(filepath.Join(testDir, complexSoundBank))
	if err != nil {
		t.Error(err)
	}
	wem, err := os.Open(filepath.Join(testDir, largerWem))
	if err != nil {
		t.Error(err)
	}
	stat, _ := wem.Stat()
	bnk.ReplaceWems(&wwise.ReplacementWem{wem, 0, stat.Size()})

	expect, err := os.Open(filepath.Join(testDir, "0_replaced_with_larger.bnk"))
	if err != nil {
		t.Error(err)
	}

	wwise.AssertContainerEqualToFile(t, expect, bnk)
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
