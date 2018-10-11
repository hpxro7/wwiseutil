// Package util implements common utility functions.
package util

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
)

type ReadSeekerAt interface {
	io.ReadSeeker
	io.ReaderAt
	Size() int64
}

type ResettingReader struct {
	*io.SectionReader
}

func NewResettingReader(r io.ReaderAt, off int64, n int64) ReadSeekerAt {
	return &ResettingReader{io.NewSectionReader(r, off, n)}
}

func (r *ResettingReader) Read(p []byte) (n int, err error) {
	n, err = r.SectionReader.Read(p)
	if err == io.EOF {
		r.SectionReader.Seek(0, io.SeekStart)
	}
	return
}

// UserHome returns the platform-specific path to the user's home directory.
func UserHome() string {
	if runtime.GOOS == "windows" {
		path := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if path == "" {
			return os.Getenv("USERPROFILE")
		} else {
			return path
		}
	} else { // We are on a POSIX system.
		return os.Getenv("HOME")
	}
}

// CanonicalWemName returns the canonical string name for a wem based on its
// index in a file.
func CanonicalWemName(index, wemCount int) string {
	// Grow or shrink the number of leading '0's in a filename, based on the
	// maximum number of wems being unpacked.
	maxDigits := strconv.Itoa(len(strconv.Itoa(wemCount)))
	nameFmt := strings.Join([]string{"%0", maxDigits, "d.wem"}, "")
	// Wems are indexed internally starting from 0, but the names start at 1.
	return fmt.Sprintf(nameFmt, index+1)
}
