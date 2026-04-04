package home

import "errors"

// ErrNotFound is returned by Get when the requested file does not exist.
var ErrNotFound = errors.New("file not found")

// Home abstracts file operations on a directory.
type Home interface {
	Get(name string) ([]byte, error)        // returns ErrNotFound if missing
	Search(pattern string) ([]string, error) // glob, returns bare filenames
	Upsert(name string, data []byte) error   // create or overwrite; creates intermediate dirs
	Delete(name string) error                // returns ErrNotFound if missing
	Sub(name string) (Home, error)           // returns a Home rooted at a subdirectory
	ListDirs() ([]string, error)             // lists first-level subdirectory names
	DeleteDir(name string) error             // removes a subdirectory and all its contents
}
