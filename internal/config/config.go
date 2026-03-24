package config

import (
	"errors"
	"os"
	"path/filepath"
)

const BookFile = "book.yaml"

var ErrNotInProject = errors.New("not in a nib project (no book.yaml found)")

// FindProjectRoot walks up from startDir looking for a directory
// containing book.yaml. Returns the absolute path to that directory
// or ErrNotInProject if none is found.
func FindProjectRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, BookFile)
		if _, err := os.Stat(candidate); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotInProject
		}
		dir = parent
	}
}
