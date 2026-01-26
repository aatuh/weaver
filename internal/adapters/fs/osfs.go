package fs

import (
	"io/fs"
	"os"
	"path/filepath"
)

// OSFS implements FileSystem using the local OS.
type OSFS struct{}

func (OSFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}

func (OSFS) ReadFile(path string) ([]byte, error) {
	// #nosec G304 -- paths are derived from the configured root and filter.
	return os.ReadFile(path)
}
