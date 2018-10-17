// Package pck implements access to the Wwise File Package file format.
package pck

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

	simpleFilePackage  = "simple.pck"
	complexFilePackage = "complex.pck"
)

func TestSimpleUnchangedFileIsEqual(t *testing.T) {
	unchangedFileIsEqual(simpleFilePackage, t)
}

func unchangedFileIsEqual(name string, t *testing.T) {
	util.SkipIfShort(t)

	f, err := os.Open(filepath.Join(testDir, name))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	pck, err := NewFile(f)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	wwise.AssertContainerEqualToFile(t, f, pck)
}

func TestUnchangedWriteFileTwiceIsEqual(t *testing.T) {
	util.SkipIfShort(t)

	f, err := os.Open(filepath.Join(testDir, simpleFilePackage))
	if err != nil {
		t.Error(err)
	}
	pck, err := NewFile(f)
	if err != nil {
		t.Error(err)
	}
	wwise.AssertContainerEqualToFile(t, f, pck)
	f, err = os.Open(filepath.Join(testDir, simpleFilePackage))
	if err != nil {
		t.Error(err)
	}
	wwise.AssertContainerEqualToFile(t, f, pck)
}

func TestReplaceWemCases(t *testing.T) {
	util.SkipIfShort(t)

	for _, c := range wwise.ReplacementTestCases {
		org, err := Open(filepath.Join(testDir, complexFilePackage))
		if err != nil {
			t.Error(c.Name+"failed: ", err)
			t.FailNow()
		}

		rs := c.Test.Expand(org)
		failed := assertReplacedFileCorrectness(t, complexFilePackage, rs...)
		if failed {
			t.Error("The", c.Name, "test case has failed.\n")
		}
	}
}

func assertReplacedFileCorrectness(t *testing.T, pckPath string,
	rs ...*wwise.ReplacementWem) (failed bool) {
	org, err := Open(filepath.Join(testDir, pckPath))
	if err != nil {
		t.Error(err)
		return true
	}

	replaced, err := Open(filepath.Join(testDir, pckPath))
	if err != nil {
		t.Error(err)
		return true
	}
	replaced.ReplaceWems(rs...)
	reread := rereadFile(t, replaced)

	failed =
		wwise.AssertReplacementsConsistent(t, org, replaced, reread, rs...)
	return
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
