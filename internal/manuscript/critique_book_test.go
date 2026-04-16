package manuscript

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcatChapters_OrderAndSeparators(t *testing.T) {
	dir := t.TempDir()

	files := []string{
		filepath.Join(dir, "001-alpha.md"),
		filepath.Join(dir, "002-bravo.md"),
		filepath.Join(dir, "003-charlie.md"),
	}
	contents := []string{
		"# Alpha\n\nbody of alpha.\n",
		"# Bravo\n\nbody of bravo.\n",
		"# Charlie\n\nbody of charlie.\n",
	}
	for i, path := range files {
		require.NoError(t, os.WriteFile(path, []byte(contents[i]), 0644))
	}

	dest := filepath.Join(dir, FullManuscriptFile)
	require.NoError(t, concatChapters(files, dest))

	got, err := os.ReadFile(dest)
	require.NoError(t, err)

	expected := contents[0] + "\n\n" + contents[1] + "\n\n" + contents[2]
	assert.Equal(t, expected, string(got))
}

func TestConcatChapters_OverwritesExistingFile(t *testing.T) {
	dir := t.TempDir()

	chapter := filepath.Join(dir, "001-one.md")
	require.NoError(t, os.WriteFile(chapter, []byte("fresh\n"), 0644))

	dest := filepath.Join(dir, FullManuscriptFile)
	require.NoError(t, os.WriteFile(dest, []byte("stale content that should be wiped\n"), 0644))

	require.NoError(t, concatChapters([]string{chapter}, dest))

	got, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, "fresh\n", string(got))
}

func TestConcatChapters_SingleFile(t *testing.T) {
	dir := t.TempDir()

	chapter := filepath.Join(dir, "001-only.md")
	require.NoError(t, os.WriteFile(chapter, []byte("solo chapter\n"), 0644))

	dest := filepath.Join(dir, FullManuscriptFile)
	require.NoError(t, concatChapters([]string{chapter}, dest))

	got, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, "solo chapter\n", string(got))
}
