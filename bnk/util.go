// Package bnk implements access to the Wwise SoundBank file format.
package bnk

// A utility ReaderAt that emits an infinite stream of a specific value.
type InfiniteReaderAt struct {
	// The value that this padding writer will write.
	Value byte
}

// ReadAt fills all of len(p) bytes with the Value of this InfiniteReaderAt.
func (r *InfiniteReaderAt) ReadAt(p []byte, off int64) (int, error) {
	for i, _ := range p {
		p[i] = r.Value
	}
	return 1, nil
}
