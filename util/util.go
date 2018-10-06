// Package util implements common utility functions.
package util

import (
	"os"
	"runtime"
)

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
