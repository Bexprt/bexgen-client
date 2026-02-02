package types

import "io"

type ObjectStorage interface {
	Store(path string, r io.Reader) error
	Get(path string) (io.ReadCloser, error)
}
