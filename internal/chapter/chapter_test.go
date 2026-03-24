package chapter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/poiesic/binder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const emptyBookYAML = `---
title: Test Book
author: Test Author
---
book:
  base_dir: manuscript
  chapters: []
`

const populatedBookYAML = `---
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
    - scenes:
        - "baz"
`

func setupProject(t *testing.T, yaml string) string {
	t.Helper()
	dir := t.TempDir()
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

func loadChapters(t *testing.T, dir string) []binder.Chapter {
	t.Helper()
	bookFile := filepath.Join(dir, "book.yaml")
	_, book, err := binder.LoadBook(bookFile)
	require.NoError(t, err)
	return book.Chapters
}

// --- Add tests ---

func TestAdd_Append(t *testing.T) {
	dir := setupProject(t, emptyBookYAML)
	chdirTo(t, dir)

	require.NoError(t, Add(AddOptions{}))
	require.NoError(t, Add(AddOptions{}))

	chapters := loadChapters(t, dir)
	assert.Len(t, chapters, 2)
}

func TestAdd_AppendWithName(t *testing.T) {
	dir := setupProject(t, emptyBookYAML)
	chdirTo(t, dir)

	require.NoError(t, Add(AddOptions{Name: "Prologue"}))

	chapters := loadChapters(t, dir)
	require.Len(t, chapters, 1)
	assert.Equal(t, "Prologue", chapters[0].Name)
	assert.Equal(t, []string{}, chapters[0].Scenes)
}

func TestAdd_AppendInterlude(t *testing.T) {
	dir := setupProject(t, emptyBookYAML)
	chdirTo(t, dir)

	require.NoError(t, Add(AddOptions{Interlude: true}))

	chapters := loadChapters(t, dir)
	require.Len(t, chapters, 1)
	assert.True(t, chapters[0].Interlude)
}

func TestAdd_InsertAtPosition(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	require.NoError(t, Add(AddOptions{Name: "Inserted", At: 2}))

	chapters := loadChapters(t, dir)
	require.Len(t, chapters, 4)
	assert.Equal(t, "Inserted", chapters[1].Name)
	// Original second chapter (interlude) should now be third
	assert.True(t, chapters[2].Interlude)
}

func TestAdd_InsertAtStart(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	require.NoError(t, Add(AddOptions{Name: "First", At: 1}))

	chapters := loadChapters(t, dir)
	require.Len(t, chapters, 4)
	assert.Equal(t, "First", chapters[0].Name)
}

func TestAdd_InsertAtEnd(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	// Position 4 = after all 3 existing chapters
	require.NoError(t, Add(AddOptions{Name: "Last", At: 4}))

	chapters := loadChapters(t, dir)
	require.Len(t, chapters, 4)
	assert.Equal(t, "Last", chapters[3].Name)
}

func TestAdd_InsertOutOfRange(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	err := Add(AddOptions{At: 10})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestAdd_InsertZeroPosition(t *testing.T) {
	dir := setupProject(t, emptyBookYAML)
	chdirTo(t, dir)

	// At=0 means append
	require.NoError(t, Add(AddOptions{Name: "Appended"}))

	chapters := loadChapters(t, dir)
	require.Len(t, chapters, 1)
	assert.Equal(t, "Appended", chapters[0].Name)
}

// --- List tests ---

func TestList_Empty(t *testing.T) {
	dir := setupProject(t, emptyBookYAML)
	chdirTo(t, dir)

	infos, err := List()
	require.NoError(t, err)
	assert.Empty(t, infos)
}

func TestList_Populated(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	infos, err := List()
	require.NoError(t, err)
	require.Len(t, infos, 3)

	// First chapter: auto-numbered "Chapter One"
	assert.Equal(t, 1, infos[0].Index)
	assert.Equal(t, "Chapter One", infos[0].Heading)
	assert.Equal(t, 2, infos[0].SceneCount)
	assert.False(t, infos[0].IsInterlude)

	// Second: interlude
	assert.Equal(t, 2, infos[1].Index)
	assert.Equal(t, "", infos[1].Heading)
	assert.Equal(t, 1, infos[1].SceneCount)
	assert.True(t, infos[1].IsInterlude)

	// Third: auto-numbered "Chapter Two"
	assert.Equal(t, 3, infos[2].Index)
	assert.Equal(t, "Chapter Two", infos[2].Heading)
	assert.Equal(t, 1, infos[2].SceneCount)
}

func TestList_MixedNamedAndAuto(t *testing.T) {
	yaml := `---
title: Test
author: Test
---
book:
  base_dir: manuscript
  chapters:
    - name: "Prologue"
      scenes: []
    - scenes:
        - "foo"
    - interlude: true
      scenes: []
    - name: "Epilogue"
      scenes:
        - "bar"
`
	dir := setupProject(t, yaml)
	chdirTo(t, dir)

	infos, err := List()
	require.NoError(t, err)
	require.Len(t, infos, 4)

	assert.Equal(t, "Prologue", infos[0].Heading)
	assert.Equal(t, "Chapter One", infos[1].Heading)
	assert.Equal(t, "", infos[2].Heading)
	assert.True(t, infos[2].IsInterlude)
	assert.Equal(t, "Epilogue", infos[3].Heading)
}

// --- Remove tests ---

func TestRemove_First(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	require.NoError(t, Remove(1))

	chapters := loadChapters(t, dir)
	assert.Len(t, chapters, 2)
	// First remaining should be the old interlude
	assert.True(t, chapters[0].Interlude)
}

func TestRemove_Middle(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	require.NoError(t, Remove(2))

	chapters := loadChapters(t, dir)
	assert.Len(t, chapters, 2)
	assert.Equal(t, []string{"foo", "bar"}, chapters[0].Scenes)
	assert.Equal(t, []string{"baz"}, chapters[1].Scenes)
}

func TestRemove_Last(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	require.NoError(t, Remove(3))

	chapters := loadChapters(t, dir)
	assert.Len(t, chapters, 2)
}

func TestRemove_Only(t *testing.T) {
	yaml := `---
title: Test
author: Test
---
book:
  base_dir: manuscript
  chapters:
    - scenes:
        - "foo"
`
	dir := setupProject(t, yaml)
	chdirTo(t, dir)

	require.NoError(t, Remove(1))

	chapters := loadChapters(t, dir)
	assert.Empty(t, chapters)
}

func TestRemove_InvalidIndex_Zero(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	err := Remove(0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestRemove_InvalidIndex_TooHigh(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	err := Remove(10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

// --- FormatList tests ---

func TestFormatList_Empty(t *testing.T) {
	output := FormatList(nil)
	assert.Equal(t, "No chapters\n", output)
}

func TestFormatList_Populated(t *testing.T) {
	infos := []ChapterInfo{
		{Index: 1, Heading: "Chapter One", SceneCount: 2},
		{Index: 2, Heading: "", SceneCount: 1, IsInterlude: true},
		{Index: 3, Heading: "Epilogue", SceneCount: 3},
	}

	output := FormatList(infos)
	assert.Contains(t, output, "1. Chapter One (2 scenes)")
	assert.Contains(t, output, "2. (interlude) (1 scenes)")
	assert.Contains(t, output, "3. Epilogue (3 scenes)")
}
