package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindProjectRoot_Found(t *testing.T) {
	// Create a temp directory with book.yaml
	root := t.TempDir()
	err := os.WriteFile(filepath.Join(root, BookFile), []byte("title: test\n"), 0644)
	require.NoError(t, err)

	result, err := FindProjectRoot(root)
	require.NoError(t, err)
	assert.Equal(t, root, result)
}

func TestFindProjectRoot_FoundFromSubdir(t *testing.T) {
	root := t.TempDir()
	err := os.WriteFile(filepath.Join(root, BookFile), []byte("title: test\n"), 0644)
	require.NoError(t, err)

	subdir := filepath.Join(root, "manuscript", "scenes")
	require.NoError(t, os.MkdirAll(subdir, 0755))

	result, err := FindProjectRoot(subdir)
	require.NoError(t, err)
	assert.Equal(t, root, result)
}

func TestFindProjectRoot_NotFound(t *testing.T) {
	// Use a temp dir with no book.yaml anywhere
	dir := t.TempDir()

	_, err := FindProjectRoot(dir)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotInProject)
}
