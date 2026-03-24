package scene

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/poiesic/binder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func focusTestBook() *binder.Book {
	return &binder.Book{
		BaseDir: "manuscript",
		Chapters: []binder.Chapter{
			{Scenes: []string{"opening", "conflict"}},
			{Interlude: true, Scenes: []string{"interlude1"}},
			{Scenes: []string{"climax"}},
		},
	}
}

// --- SetFocus tests ---

func TestSetFocus_WithPosition(t *testing.T) {
	dir := t.TempDir()
	book := focusTestBook()

	info, err := SetFocus(dir, book, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, 1, info.Chapter)
	assert.Equal(t, 2, info.Position)
	assert.Equal(t, "conflict", info.Slug)
}

func TestSetFocus_ChapterOnly(t *testing.T) {
	dir := t.TempDir()
	book := focusTestBook()

	info, err := SetFocus(dir, book, 2, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, info.Chapter)
	assert.Equal(t, 0, info.Position)
	assert.Equal(t, "", info.Slug)
}

func TestSetFocus_ChapterOutOfRange(t *testing.T) {
	dir := t.TempDir()
	book := focusTestBook()

	_, err := SetFocus(dir, book, 10, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestSetFocus_PositionOutOfRange(t *testing.T) {
	dir := t.TempDir()
	book := focusTestBook()

	_, err := SetFocus(dir, book, 1, 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

// --- ClearFocus tests ---

func TestClearFocus(t *testing.T) {
	dir := t.TempDir()
	book := focusTestBook()

	_, err := SetFocus(dir, book, 1, 1)
	require.NoError(t, err)

	require.NoError(t, ClearFocus(dir))

	info, err := GetFocus(dir, book)
	require.NoError(t, err)
	assert.Nil(t, info)
}

// --- GetFocus tests ---

func TestGetFocus_NoState(t *testing.T) {
	dir := t.TempDir()
	book := focusTestBook()

	info, err := GetFocus(dir, book)
	require.NoError(t, err)
	assert.Nil(t, info)
}

func TestGetFocus_WithSceneFocus(t *testing.T) {
	dir := t.TempDir()
	book := focusTestBook()

	_, err := SetFocus(dir, book, 1, 2)
	require.NoError(t, err)

	info, err := GetFocus(dir, book)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, 1, info.Chapter)
	assert.Equal(t, 2, info.Position)
	assert.Equal(t, "conflict", info.Slug)
}

func TestGetFocus_ChapterOnly(t *testing.T) {
	dir := t.TempDir()
	book := focusTestBook()

	_, err := SetFocus(dir, book, 2, 0)
	require.NoError(t, err)

	info, err := GetFocus(dir, book)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, 2, info.Chapter)
	assert.Equal(t, 0, info.Position)
	assert.Equal(t, "", info.Slug)
}

func TestGetFocus_SceneRemoved(t *testing.T) {
	dir := t.TempDir()
	book := focusTestBook()

	_, err := SetFocus(dir, book, 1, 2)
	require.NoError(t, err)

	// Remove the scene from the book
	book.Chapters[0].Scenes = []string{"opening"}

	_, err = GetFocus(dir, book)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no longer exists")
}

func TestGetFocus_ChapterRemoved(t *testing.T) {
	dir := t.TempDir()
	book := focusTestBook()

	_, err := SetFocus(dir, book, 3, 1)
	require.NoError(t, err)

	// Shrink book to only 1 chapter
	book.Chapters = book.Chapters[:1]

	_, err = GetFocus(dir, book)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no longer exists")
}

// --- resolveSlugOrFocus tests ---

func TestResolveSlugOrFocus_ExplicitSlug(t *testing.T) {
	dir := t.TempDir()
	book := focusTestBook()

	slug, err := resolveSlugOrFocus(dir, book, "opening")
	require.NoError(t, err)
	assert.Equal(t, "opening", slug)
}

func TestResolveSlugOrFocus_FromFocus(t *testing.T) {
	dir := t.TempDir()
	book := focusTestBook()

	_, err := SetFocus(dir, book, 1, 2)
	require.NoError(t, err)

	slug, err := resolveSlugOrFocus(dir, book, "")
	require.NoError(t, err)
	assert.Equal(t, "conflict", slug)
}

func TestResolveSlugOrFocus_NoFocusSet(t *testing.T) {
	dir := t.TempDir()
	book := focusTestBook()

	_, err := resolveSlugOrFocus(dir, book, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no scene specified and no focus set")
}

func TestResolveSlugOrFocus_ChapterOnlyFocus(t *testing.T) {
	dir := t.TempDir()
	book := focusTestBook()

	_, err := SetFocus(dir, book, 2, 0)
	require.NoError(t, err)

	_, err = resolveSlugOrFocus(dir, book, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no specific scene")
}

// --- Integration: focus survives reordering ---

func TestFocus_SurvivesReordering(t *testing.T) {
	dir := setupProject(t, populatedBookYAML)
	chdirTo(t, dir)

	book := focusTestBook()
	// Focus on "conflict" at position 2 in chapter 1
	_, err := SetFocus(dir, book, 1, 2)
	require.NoError(t, err)

	// Simulate reordering: "conflict" moves to position 1
	book.Chapters[0].Scenes = []string{"conflict", "opening"}

	info, err := GetFocus(dir, book)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "conflict", info.Slug)
	assert.Equal(t, 1, info.Position) // Position updated to new location
}

// --- Integration: focus resolves via project root ---

func TestFocus_RoundTrip_ViaProjectRoot(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(populatedBookYAML), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "manuscript"), 0755))

	book := &binder.Book{
		BaseDir: "manuscript",
		Chapters: []binder.Chapter{
			{Scenes: []string{"foo", "bar"}},
			{Interlude: true, Scenes: []string{"interlude1"}},
			{Scenes: []string{"baz"}},
		},
	}

	_, err := SetFocus(dir, book, 1, 1)
	require.NoError(t, err)

	info, err := GetFocus(dir, book)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "foo", info.Slug)
}
