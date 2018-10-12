// Package util implements common utility functions.
package util

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type ContainerType int

const (
	UnknownFileType     ContainerType = iota
	SoundBankFileType   ContainerType = iota
	FilePackageFileType ContainerType = iota
)

var soundBankExtensions = []string{".nbnk", ".bnk"}
var filePackageExtensions = []string{".npck", ".pck"}

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

// GetFileType determies what the file type is path is based off of its
// extension.
func GetFileType(path string) (t ContainerType, ext string) {
	ext = filepath.Ext(path)
	if contains(soundBankExtensions, ext) {
		return SoundBankFileType, ext
	}
	if contains(filePackageExtensions, ext) {
		return FilePackageFileType, ext
	}

	return UnknownFileType, ext
}

func contains(sources []string, target string) bool {
	for _, s := range sources {
		if s == target {
			return true
		}
	}
	return false
}
