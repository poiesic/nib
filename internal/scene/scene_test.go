package scene

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/poiesic/binder"
	"github.com/poiesic/nib/internal/bookio"
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

const oneChapterBookYAML = `---
title: Test Book
author: Test Author
---
book:
  base_dir: manuscript
  chapters:
    - scenes: []
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

const subdirBookYAML = `---
title: Test Book
author: Test Author
---
book:
  base_dir: manuscript
  chapters:
    - subdir: part1
      scenes:
        - "foo"
    - subdir: part2
      scenes:
        - "bar"
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
	dir := setupProject(t, oneChapterBookYAML)
	chdirTo(t, dir)

	result, err := Add(AddOptions{ChapterIndex: 1, Slug: "opening"})
	require.NoError(t, err)
	assert.Equal(t, 1, result.ChapterIndex)
	assert.Equal(t, "opening", result.Slug)
	assert.Equal(t, 1, result.Position)

	chapters := loadChapters(t, dir)
	require.Len(t, chapters, 1)
	assert.Equal(t, []string{"opening"}, chapters[0].Scenes)
}

func TestAdd_AppendMultiple(t *testing.T) {
	dir := setupProject(t, oneChapterBookYAML)
	chdirTo(t, dir)

	r1, err := Add(AddOptions{ChapterIndex: 1, Slug: "opening"})
	require.NoError(t, err)
	assert.Equal(t, 1, r1.Position)

	r2, err := Add(AddOptions{ChapterIndex: 1, Slug: "conflict"})
	require.NoError(t, err)
	assert.Equal(t, 2, r2.Position)

	chapters := loadChapters(t, dir)
	assert.Equal(t, []string{"opening", "conflict"}, chapters[0].Scenes)
}

func TestAdd_InsertAtPosition(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	result, err := Add(AddOptions{ChapterIndex: 1, Slug: "inserted", At: 2})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Position)

	chapters := loadChapters(t, dir)
	assert.Equal(t, []string{"foo", "inserted", "bar"}, chapters[0].Scenes)
}

func TestAdd_InsertAtStart(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	result, err := Add(AddOptions{ChapterIndex: 1, Slug: "first", At: 1})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Position)

	chapters := loadChapters(t, dir)
	assert.Equal(t, []string{"first", "foo", "bar"}, chapters[0].Scenes)
}

func TestAdd_CreatesFile(t *testing.T) {
	dir := setupProject(t, oneChapterBookYAML)
	chdirTo(t, dir)

	_, err := Add(AddOptions{ChapterIndex: 1, Slug: "opening"})
	require.NoError(t, err)

	scenePath := filepath.Join(dir, "manuscript", "opening.md")
	_, err = os.Stat(scenePath)
	assert.NoError(t, err, "scene file should exist")
}

func TestAdd_StripsMdExtension(t *testing.T) {
	dir := setupProject(t, oneChapterBookYAML)
	chdirTo(t, dir)

	result, err := Add(AddOptions{ChapterIndex: 1, Slug: "opening.md"})
	require.NoError(t, err)
	assert.Equal(t, "opening", result.Slug)

	// File should be opening.md, not opening.md.md
	scenePath := filepath.Join(dir, "manuscript", "opening.md")
	_, err = os.Stat(scenePath)
	assert.NoError(t, err, "scene file should be opening.md")

	badPath := filepath.Join(dir, "manuscript", "opening.md.md")
	_, err = os.Stat(badPath)
	assert.True(t, os.IsNotExist(err), "opening.md.md should not exist")

	chapters := loadChapters(t, dir)
	assert.Contains(t, chapters[0].Scenes, "opening")
}

func TestAdd_CreatesFileInSubdir(t *testing.T) {
	dir := setupProject(t, subdirBookYAML)
	chdirTo(t, dir)

	_, err := Add(AddOptions{ChapterIndex: 1, Slug: "new-scene"})
	require.NoError(t, err)

	scenePath := filepath.Join(dir, "manuscript", "part1", "new-scene.md")
	_, err = os.Stat(scenePath)
	assert.NoError(t, err, "scene file should exist in subdir")
}

func TestAdd_FileAlreadyExists(t *testing.T) {
	dir := setupProject(t, oneChapterBookYAML)
	chdirTo(t, dir)

	// Pre-create the file with content
	scenePath := filepath.Join(dir, "manuscript", "existing.md")
	require.NoError(t, os.WriteFile(scenePath, []byte("existing content"), 0644))

	_, err := Add(AddOptions{ChapterIndex: 1, Slug: "existing"})
	require.NoError(t, err)

	// File should still have original content
	content, err := os.ReadFile(scenePath)
	require.NoError(t, err)
	assert.Equal(t, "existing content", string(content))
}

func TestAdd_DuplicateSlugError(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	_, err := Add(AddOptions{ChapterIndex: 1, Slug: "foo"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestAdd_ChapterOutOfRange(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	_, err := Add(AddOptions{ChapterIndex: 10, Slug: "new"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestAdd_ChapterZero(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	_, err := Add(AddOptions{ChapterIndex: 0, Slug: "new"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestAdd_PositionOutOfRange(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	_, err := Add(AddOptions{ChapterIndex: 1, Slug: "new", At: 10})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestAdd_EmptySlug(t *testing.T) {
	dir := setupProject(t, oneChapterBookYAML)
	chdirTo(t, dir)

	_, err := Add(AddOptions{ChapterIndex: 1, Slug: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must not be empty")
}

// --- Remove tests ---

func TestRemove_Existing(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	require.NoError(t, Remove(RemoveOptions{ChapterIndex: 1, Slug: "foo"}))

	chapters := loadChapters(t, dir)
	assert.Equal(t, []string{"bar"}, chapters[0].Scenes)
}

func TestRemove_NotFound(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	err := Remove(RemoveOptions{ChapterIndex: 1, Slug: "nonexistent"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRemove_OnlyScene(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	require.NoError(t, Remove(RemoveOptions{ChapterIndex: 2, Slug: "interlude1"}))

	chapters := loadChapters(t, dir)
	assert.Empty(t, chapters[1].Scenes)
}

func TestRemove_FileStaysOnDisk(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	// Create the scene file first
	scenePath := filepath.Join(dir, "manuscript", "foo.md")
	require.NoError(t, os.WriteFile(scenePath, []byte("content"), 0644))

	require.NoError(t, Remove(RemoveOptions{ChapterIndex: 1, Slug: "foo"}))

	// File should still exist
	_, err := os.Stat(scenePath)
	assert.NoError(t, err, "scene file should remain on disk")
}

func TestRemove_ChapterOutOfRange(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	err := Remove(RemoveOptions{ChapterIndex: 10, Slug: "foo"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

// --- List tests ---

func TestList_AllChapters(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	groups, err := List(ListOptions{})
	require.NoError(t, err)
	require.Len(t, groups, 3)

	// Chapter 1
	assert.Equal(t, 1, groups[0].ChapterIndex)
	assert.Equal(t, "Chapter One", groups[0].Heading)
	assert.False(t, groups[0].IsInterlude)
	require.Len(t, groups[0].Scenes, 2)
	assert.Equal(t, SceneInfo{Index: 1, Slug: "foo"}, groups[0].Scenes[0])
	assert.Equal(t, SceneInfo{Index: 2, Slug: "bar"}, groups[0].Scenes[1])

	// Interlude
	assert.Equal(t, 2, groups[1].ChapterIndex)
	assert.Equal(t, "", groups[1].Heading)
	assert.True(t, groups[1].IsInterlude)
	require.Len(t, groups[1].Scenes, 1)
	assert.Equal(t, SceneInfo{Index: 1, Slug: "interlude1"}, groups[1].Scenes[0])

	// Chapter 2
	assert.Equal(t, 3, groups[2].ChapterIndex)
	assert.Equal(t, "Chapter Two", groups[2].Heading)
	require.Len(t, groups[2].Scenes, 1)
}

func TestList_SingleChapter(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	groups, err := List(ListOptions{ChapterIndex: 2})
	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, 2, groups[0].ChapterIndex)
	assert.True(t, groups[0].IsInterlude)
}

func TestList_EmptyBook(t *testing.T) {
	dir := setupProject(t, emptyBookYAML)
	chdirTo(t, dir)

	groups, err := List(ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, groups)
}

func TestList_ChapterOutOfRange(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	_, err := List(ListOptions{ChapterIndex: 10})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestList_ChapterWithNoScenes(t *testing.T) {
	dir := setupProject(t, oneChapterBookYAML)
	chdirTo(t, dir)

	groups, err := List(ListOptions{})
	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Empty(t, groups[0].Scenes)
}

// --- Move tests ---

func TestMove_WithinChapter(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	// Move scene at position 2 ("bar") to position 1 within chapter 1
	require.NoError(t, Move(MoveOptions{ChapterIndex: 1, FromPosition: 2, ToPosition: 1}))

	chapters := loadChapters(t, dir)
	assert.Equal(t, []string{"bar", "foo"}, chapters[0].Scenes)
}

func TestMove_WithinChapter_ToEnd(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	// Move scene at position 1 ("foo") to position 2 (end) within chapter 1
	// After removing "foo": ["bar"], insert at pos 2 is out of range (only 1 element)
	// Actually after remove we have ["bar"], so valid positions are 1..1
	// Move to pos 1 keeps it at start. We need to append, which is pos len+1
	// With ["bar"], inserting at position 1 gives ["foo", "bar"]
	// So to put foo at the end, toPosition should be 2 (after remove, ["bar"] has 1 scene, pos 2 = append after)
	// Wait: after removing pos 1, we have ["bar"]. toPos 2-1=1, which is > len(["bar"])=1? No, 1 is not > 1.
	// dst.Scenes[:1] = ["bar"], append slug + dst.Scenes[1:] = ["bar", "foo"]. That's right.
	require.NoError(t, Move(MoveOptions{ChapterIndex: 1, FromPosition: 1, ToPosition: 2}))

	chapters := loadChapters(t, dir)
	assert.Equal(t, []string{"bar", "foo"}, chapters[0].Scenes)
}

func TestMove_WithinChapter_NoOp(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	// Move scene at position 1 to position 1 — no change
	require.NoError(t, Move(MoveOptions{ChapterIndex: 1, FromPosition: 1, ToPosition: 1}))

	chapters := loadChapters(t, dir)
	assert.Equal(t, []string{"foo", "bar"}, chapters[0].Scenes)
}

func TestMove_WithinChapter_Append(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	// Move scene at position 1 ("foo") to end of chapter 1 (ToPosition=0 means append)
	require.NoError(t, Move(MoveOptions{ChapterIndex: 1, FromPosition: 1}))

	chapters := loadChapters(t, dir)
	assert.Equal(t, []string{"bar", "foo"}, chapters[0].Scenes)
}

func TestMove_BetweenChapters_Append(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	// Move scene at position 1 ("foo") from chapter 1 to end of chapter 3
	require.NoError(t, Move(MoveOptions{ChapterIndex: 1, FromPosition: 1, To: 3}))

	chapters := loadChapters(t, dir)
	assert.Equal(t, []string{"bar"}, chapters[0].Scenes)
	assert.Equal(t, []string{"baz", "foo"}, chapters[2].Scenes)
}

func TestMove_BetweenChapters(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	// Move scene at position 1 ("foo") from chapter 1 to end of chapter 3
	// Chapter 3 has ["baz"], so toPosition 2 = append after "baz"
	require.NoError(t, Move(MoveOptions{ChapterIndex: 1, FromPosition: 1, To: 3, ToPosition: 2}))

	chapters := loadChapters(t, dir)
	assert.Equal(t, []string{"bar"}, chapters[0].Scenes)
	assert.Equal(t, []string{"baz", "foo"}, chapters[2].Scenes)
}

func TestMove_BetweenChapters_AtStart(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	// Move scene at position 1 ("foo") from chapter 1 to position 1 in chapter 3
	require.NoError(t, Move(MoveOptions{ChapterIndex: 1, FromPosition: 1, To: 3, ToPosition: 1}))

	chapters := loadChapters(t, dir)
	assert.Equal(t, []string{"bar"}, chapters[0].Scenes)
	assert.Equal(t, []string{"foo", "baz"}, chapters[2].Scenes)
}

func TestMove_CrossSubdirError(t *testing.T) {
	dir := setupProject(t, subdirBookYAML)
	chdirTo(t, dir)

	err := Move(MoveOptions{ChapterIndex: 1, FromPosition: 1, To: 2, ToPosition: 1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "different subdirs")
}

func TestMove_CrossSubdirError_SubdirToNoSubdir(t *testing.T) {
	dir := setupProject(t, subdirBookYAML)
	chdirTo(t, dir)

	err := Move(MoveOptions{ChapterIndex: 1, FromPosition: 1, To: 3, ToPosition: 1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "different subdirs")
}

func TestMove_SourcePositionOutOfRange(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	err := Move(MoveOptions{ChapterIndex: 1, FromPosition: 10, ToPosition: 1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source position")
	assert.Contains(t, err.Error(), "out of range")
}

func TestMove_DestPositionOutOfRange(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	err := Move(MoveOptions{ChapterIndex: 1, FromPosition: 1, ToPosition: 10})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "destination position")
	assert.Contains(t, err.Error(), "out of range")
}

func TestMove_SourceChapterOutOfRange(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	err := Move(MoveOptions{ChapterIndex: 10, FromPosition: 1, ToPosition: 1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestMove_DestChapterOutOfRange(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	err := Move(MoveOptions{ChapterIndex: 1, FromPosition: 1, To: 10, ToPosition: 1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

// --- Rename tests ---

func TestRename_HappyPath(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	// Create the scene file
	scenePath := filepath.Join(dir, "manuscript", "foo.md")
	require.NoError(t, os.WriteFile(scenePath, []byte("scene content"), 0644))

	require.NoError(t, Rename(RenameOptions{OldSlug: "foo", NewSlug: "renamed-foo"}))

	// book.yaml updated
	chapters := loadChapters(t, dir)
	assert.Equal(t, []string{"renamed-foo", "bar"}, chapters[0].Scenes)

	// File renamed
	_, err := os.Stat(filepath.Join(dir, "manuscript", "renamed-foo.md"))
	assert.NoError(t, err)
	_, err = os.Stat(scenePath)
	assert.True(t, os.IsNotExist(err))

	// Content preserved
	content, err := os.ReadFile(filepath.Join(dir, "manuscript", "renamed-foo.md"))
	require.NoError(t, err)
	assert.Equal(t, "scene content", string(content))
}

func TestRename_StripsMdExtension(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "manuscript", "foo.md"), []byte{}, 0644))

	require.NoError(t, Rename(RenameOptions{OldSlug: "foo.md", NewSlug: "renamed.md"}))

	chapters := loadChapters(t, dir)
	assert.Equal(t, []string{"renamed", "bar"}, chapters[0].Scenes)
}

func TestRename_OldSlugNotFound(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	err := Rename(RenameOptions{OldSlug: "nonexistent", NewSlug: "new"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in book.yaml")
}

func TestRename_NewSlugAlreadyExists(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	err := Rename(RenameOptions{OldSlug: "foo", NewSlug: "bar"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestRename_EmptyOldSlug(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	err := Rename(RenameOptions{OldSlug: "", NewSlug: "new"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "old slug must not be empty")
}

func TestRename_EmptyNewSlug(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	err := Rename(RenameOptions{OldSlug: "foo", NewSlug: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "new slug must not be empty")
}

func TestRename_SameSlugs(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	err := Rename(RenameOptions{OldSlug: "foo", NewSlug: "foo"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "same")
}

func TestRename_WithSubdir(t *testing.T) {
	dir := setupProject(t, subdirBookYAML)
	chdirTo(t, dir)

	subdirPath := filepath.Join(dir, "manuscript", "part1")
	require.NoError(t, os.MkdirAll(subdirPath, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subdirPath, "foo.md"), []byte("content"), 0644))

	require.NoError(t, Rename(RenameOptions{OldSlug: "foo", NewSlug: "renamed-foo"}))

	chapters := loadChapters(t, dir)
	assert.Equal(t, []string{"renamed-foo"}, chapters[0].Scenes)

	_, err := os.Stat(filepath.Join(subdirPath, "renamed-foo.md"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(subdirPath, "foo.md"))
	assert.True(t, os.IsNotExist(err))
}

func TestRename_FileDoesNotExist(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	// No file on disk — should still update book.yaml without error
	require.NoError(t, Rename(RenameOptions{OldSlug: "foo", NewSlug: "renamed-foo"}))

	chapters := loadChapters(t, dir)
	assert.Equal(t, []string{"renamed-foo", "bar"}, chapters[0].Scenes)
}

func TestRename_UpdatesFocus(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	// Load book and set focus to foo (chapter 1, position 1)
	_, _, book, err := bookio.Load()
	require.NoError(t, err)
	_, err = SetFocus(dir, book, 1, 1)
	require.NoError(t, err)

	require.NoError(t, Rename(RenameOptions{OldSlug: "foo", NewSlug: "renamed-foo"}))

	// Reload book after rename and check focus
	_, _, book, err = bookio.Load()
	require.NoError(t, err)
	info, err := GetFocus(dir, book)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "renamed-foo", info.Slug)
}

func TestRename_FocusUnchangedForOtherScene(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	// Focus on bar (chapter 1, position 2)
	_, _, book, err := bookio.Load()
	require.NoError(t, err)
	_, err = SetFocus(dir, book, 1, 2)
	require.NoError(t, err)

	require.NoError(t, Rename(RenameOptions{OldSlug: "foo", NewSlug: "renamed-foo"}))

	_, _, book, err = bookio.Load()
	require.NoError(t, err)
	info, err := GetFocus(dir, book)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "bar", info.Slug)
}

// --- FormatList tests ---

// --- Edit tests ---

func clearEditorEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"NIB_EDITOR", "VISUAL", "EDITOR"} {
		orig, set := os.LookupEnv(key)
		os.Unsetenv(key)
		if set {
			t.Cleanup(func() { os.Setenv(key, orig) })
		} else {
			t.Cleanup(func() { os.Unsetenv(key) })
		}
	}
}

func TestEdit_NoSlugNoFocus(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	err := Edit(EditOptions{Slug: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no scene specified and no focus set")
}

func TestEdit_SlugNotInBook(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	err := Edit(EditOptions{Slug: "nonexistent"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in book.yaml")
}

func TestEdit_NoEditorSet(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)
	clearEditorEnv(t)

	err := Edit(EditOptions{Slug: "foo"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEditorNotSet)
}

func TestEdit_ScribEditorTakesPrecedence(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)
	clearEditorEnv(t)
	t.Setenv("NIB_EDITOR", "myeditor")
	t.Setenv("VISUAL", "other")
	t.Setenv("EDITOR", "other")

	var capturedName string
	var capturedArgs []string
	runner := func(name string, args ...string) *exec.Cmd {
		capturedName = name
		capturedArgs = args
		return exec.Command("true")
	}

	err := Edit(EditOptions{Slug: "foo", Runner: runner})
	require.NoError(t, err)
	assert.Equal(t, "myeditor", capturedName)
	require.Len(t, capturedArgs, 1)
	assert.Contains(t, capturedArgs[0], "foo.md")
}

func TestEdit_FallsBackToVisual(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)
	clearEditorEnv(t)
	t.Setenv("VISUAL", "vis")

	var capturedName string
	runner := func(name string, args ...string) *exec.Cmd {
		capturedName = name
		return exec.Command("true")
	}

	err := Edit(EditOptions{Slug: "foo", Runner: runner})
	require.NoError(t, err)
	assert.Equal(t, "vis", capturedName)
}

func TestEdit_FallsBackToEditor(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)
	clearEditorEnv(t)
	t.Setenv("EDITOR", "ed")

	var capturedName string
	runner := func(name string, args ...string) *exec.Cmd {
		capturedName = name
		return exec.Command("true")
	}

	err := Edit(EditOptions{Slug: "foo", Runner: runner})
	require.NoError(t, err)
	assert.Equal(t, "ed", capturedName)
}

func TestEdit_ResolvesSubdirPath(t *testing.T) {
	dir := setupProject(t, subdirBookYAML)
	chdirTo(t, dir)
	clearEditorEnv(t)
	t.Setenv("EDITOR", "ed")

	var capturedArgs []string
	runner := func(name string, args ...string) *exec.Cmd {
		capturedArgs = args
		return exec.Command("true")
	}

	err := Edit(EditOptions{Slug: "foo", Runner: runner})
	require.NoError(t, err)
	require.Len(t, capturedArgs, 1)
	assert.Contains(t, capturedArgs[0], filepath.Join("manuscript", "part1", "foo.md"))
}

func TestFormatList_Empty(t *testing.T) {
	output := FormatList(nil)
	assert.Equal(t, "No chapters\n", output)
}

func TestFormatList_Populated(t *testing.T) {
	groups := []ChapterScenes{
		{
			ChapterIndex: 1,
			Heading:      "Chapter One",
			Scenes: []SceneInfo{
				{Index: 1, Slug: "foo"},
				{Index: 2, Slug: "bar"},
			},
		},
		{
			ChapterIndex: 2,
			Heading:      "",
			IsInterlude:  true,
			Scenes: []SceneInfo{
				{Index: 1, Slug: "interlude1"},
			},
		},
		{
			ChapterIndex: 3,
			Heading:      "Chapter Two",
			Scenes:       nil,
		},
	}

	output := FormatList(groups)
	expected := "[1] Chapter One\n    [1] foo\n    [2] bar\n[2] (interlude)\n    [1] interlude1\n[3] Chapter Two\n    (no scenes)\n"
	assert.Equal(t, expected, output)
}

func TestFormatList_AllEmpty(t *testing.T) {
	groups := []ChapterScenes{
		{
			ChapterIndex: 1,
			Heading:      "Chapter One",
			Scenes:       nil,
		},
	}

	output := FormatList(groups)
	assert.Contains(t, output, "[1] Chapter One")
	assert.Contains(t, output, "    (no scenes)")
}
