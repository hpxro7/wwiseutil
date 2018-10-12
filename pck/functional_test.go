// Package pck implements access to the Wwise File Package file format.
package pck

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

	simpleFilePackage = "simple.pck"
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
