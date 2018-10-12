// Package util implements common utility functions.
package util

import "testing"

func SkipIfShort(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file comparison test.")
	}
}
