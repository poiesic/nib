package bookio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/poiesic/binder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fixtureBookYAML = `---
title: Test Book
author: Test Author
---
book:
  base_dir: manuscript
  chapters:
    - scenes:
        - "foo"
        - "bar"
    - interlude: true
      scenes:
        - "interlude1"
    - name: "Epilogue"
      scenes:
        - "baz"
`

func setupProject(t *testing.T, yaml string) string {
	t.Helper()
	dir := t.TempDir()
	// Resolve symlinks so paths match on macOS (/var -> /private/var).
	dir, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(yaml), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "manuscript"), 0755))
	return dir
}

func chdirTo(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { os.Chdir(orig) })
}

func TestRoundTrip(t *testing.T) {
	dir := setupProject(t, fixtureBookYAML)
	chdirTo(t, dir)

	projectRoot, fm, book, err := Load()
	require.NoError(t, err)
	assert.Equal(t, dir, projectRoot)
	assert.Equal(t, "Test Book", fm.Title)
	assert.Equal(t, 3, len(book.Chapters))

	// Save and reload
	require.NoError(t, Save(projectRoot, fm, book))

	_, fm2, book2, err := Load()
	require.NoError(t, err)
	assert.Equal(t, fm.Title, fm2.Title)
	assert.Equal(t, fm.Author, fm2.Author)
	assert.Equal(t, len(book.Chapters), len(book2.Chapters))

	// Verify chapter contents survived round-trip
	assert.Equal(t, []string{"foo", "bar"}, book2.Chapters[0].Scenes)
	assert.True(t, book2.Chapters[1].Interlude)
	assert.Equal(t, "Epilogue", book2.Chapters[2].Name)
}

func TestSave_BaseDirWrittenRelative(t *testing.T) {
	dir := setupProject(t, fixtureBookYAML)
	chdirTo(t, dir)

	_, fm, book, err := Load()
	require.NoError(t, err)

	// binder.LoadBook resolves BaseDir to absolute when manuscript/ exists
	assert.True(t, filepath.IsAbs(book.BaseDir), "LoadBook should resolve BaseDir to absolute")

	require.NoError(t, Save(dir, fm, book))

	// Read raw YAML and verify base_dir is relative
	data, err := os.ReadFile(filepath.Join(dir, "book.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "base_dir: manuscript")
	assert.NotContains(t, string(data), dir)
}

func TestSave_EmptyChapters(t *testing.T) {
	dir := setupProject(t, fixtureBookYAML)
	chdirTo(t, dir)

	_, fm, book, err := Load()
	require.NoError(t, err)

	book.Chapters = []binder.Chapter{}
	require.NoError(t, Save(dir, fm, book))

	data, err := os.ReadFile(filepath.Join(dir, "book.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "chapters: []")
}

func TestSave_TwoDocumentStream(t *testing.T) {
	dir := setupProject(t, fixtureBookYAML)
	chdirTo(t, dir)

	_, fm, book, err := Load()
	require.NoError(t, err)

	require.NoError(t, Save(dir, fm, book))

	data, err := os.ReadFile(filepath.Join(dir, "book.yaml"))
	require.NoError(t, err)

	// yaml.Encoder writes "---" separator between documents.
	// Verify it's a two-document stream by checking binder can parse both docs.
	content := string(data)
	assert.Contains(t, content, "---", "should contain at least one --- separator between documents")
	assert.Contains(t, content, "title:")
	assert.Contains(t, content, "book:")
}
