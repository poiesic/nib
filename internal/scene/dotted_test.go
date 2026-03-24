package scene

import (
	"testing"

	"github.com/poiesic/binder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ParseDotted tests ---

func TestParseDotted_ChapterOnly(t *testing.T) {
	ch, pos, err := ParseDotted("3")
	require.NoError(t, err)
	assert.Equal(t, 3, ch)
	assert.Equal(t, 0, pos)
}

func TestParseDotted_ChapterAndPosition(t *testing.T) {
	ch, pos, err := ParseDotted("3.2")
	require.NoError(t, err)
	assert.Equal(t, 3, ch)
	assert.Equal(t, 2, pos)
}

func TestParseDotted_SingleDigits(t *testing.T) {
	ch, pos, err := ParseDotted("1.1")
	require.NoError(t, err)
	assert.Equal(t, 1, ch)
	assert.Equal(t, 1, pos)
}

func TestParseDotted_InvalidChapter(t *testing.T) {
	_, _, err := ParseDotted("abc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid chapter")
}

func TestParseDotted_InvalidPosition(t *testing.T) {
	_, _, err := ParseDotted("3.abc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid position")
}

func TestParseDotted_ZeroChapter(t *testing.T) {
	_, _, err := ParseDotted("0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chapter must be >= 1")
}

func TestParseDotted_ZeroPosition(t *testing.T) {
	_, _, err := ParseDotted("3.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "position must be >= 1")
}

// --- ResolveSlug tests ---

func testBook() *binder.Book {
	return &binder.Book{
		BaseDir: "manuscript",
		Chapters: []binder.Chapter{
			{Scenes: []string{"foo", "bar"}},
			{Interlude: true, Scenes: []string{"interlude1"}},
			{Scenes: []string{"baz"}},
		},
	}
}

func TestResolveSlug_Valid(t *testing.T) {
	slug, err := ResolveSlug(testBook(), 1, 2)
	require.NoError(t, err)
	assert.Equal(t, "bar", slug)
}

func TestResolveSlug_FirstScene(t *testing.T) {
	slug, err := ResolveSlug(testBook(), 1, 1)
	require.NoError(t, err)
	assert.Equal(t, "foo", slug)
}

func TestResolveSlug_ChapterOutOfRange(t *testing.T) {
	_, err := ResolveSlug(testBook(), 10, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestResolveSlug_PositionOutOfRange(t *testing.T) {
	_, err := ResolveSlug(testBook(), 1, 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

// --- ParseMoveArgs tests ---

func TestParseMoveArgs_SameChapter(t *testing.T) {
	opts, err := ParseMoveArgs("3.1", "3.3")
	require.NoError(t, err)
	assert.Equal(t, 3, opts.ChapterIndex)
	assert.Equal(t, 1, opts.FromPosition)
	assert.Equal(t, 0, opts.To) // same chapter, no To set
	assert.Equal(t, 3, opts.ToPosition)
}

func TestParseMoveArgs_CrossChapter(t *testing.T) {
	opts, err := ParseMoveArgs("3.1", "4.1")
	require.NoError(t, err)
	assert.Equal(t, 3, opts.ChapterIndex)
	assert.Equal(t, 1, opts.FromPosition)
	assert.Equal(t, 4, opts.To)
	assert.Equal(t, 1, opts.ToPosition)
}

func TestParseMoveArgs_AppendToOtherChapter(t *testing.T) {
	opts, err := ParseMoveArgs("3.1", "4")
	require.NoError(t, err)
	assert.Equal(t, 3, opts.ChapterIndex)
	assert.Equal(t, 1, opts.FromPosition)
	assert.Equal(t, 4, opts.To)
	assert.Equal(t, 0, opts.ToPosition)
}

func TestParseMoveArgs_AppendToSameChapter(t *testing.T) {
	opts, err := ParseMoveArgs("3.1", "")
	require.NoError(t, err)
	assert.Equal(t, 3, opts.ChapterIndex)
	assert.Equal(t, 1, opts.FromPosition)
	assert.Equal(t, 0, opts.To)
	assert.Equal(t, 0, opts.ToPosition)
}

func TestParseMoveArgs_SourceMissingPosition(t *testing.T) {
	_, err := ParseMoveArgs("3", "4.1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source must include a scene position")
}

func TestParseMoveArgs_InvalidSource(t *testing.T) {
	_, err := ParseMoveArgs("abc", "4.1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid source")
}

func TestParseMoveArgs_InvalidDest(t *testing.T) {
	_, err := ParseMoveArgs("3.1", "abc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid destination")
}
