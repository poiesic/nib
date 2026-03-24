package manuscript

import (
	"path/filepath"
	"testing"

	"github.com/poiesic/binder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ParseRange tests ---

func TestParseRange_SingleChapter(t *testing.T) {
	spec, err := ParseRange("3")
	require.NoError(t, err)
	assert.Equal(t, "list", spec.Kind)
	require.Len(t, spec.Refs, 1)
	assert.Equal(t, SceneRef{Chapter: 3, Position: 0}, spec.Refs[0])
}

func TestParseRange_SingleDotted(t *testing.T) {
	spec, err := ParseRange("3.2")
	require.NoError(t, err)
	assert.Equal(t, "list", spec.Kind)
	require.Len(t, spec.Refs, 1)
	assert.Equal(t, SceneRef{Chapter: 3, Position: 2}, spec.Refs[0])
}

func TestParseRange_ChapterRange(t *testing.T) {
	spec, err := ParseRange("1-3")
	require.NoError(t, err)
	assert.Equal(t, "range", spec.Kind)
	require.Len(t, spec.Refs, 2)
	assert.Equal(t, SceneRef{Chapter: 1, Position: 0}, spec.Refs[0])
	assert.Equal(t, SceneRef{Chapter: 3, Position: 0}, spec.Refs[1])
}

func TestParseRange_DottedRange(t *testing.T) {
	spec, err := ParseRange("1.1-3.2")
	require.NoError(t, err)
	assert.Equal(t, "range", spec.Kind)
	require.Len(t, spec.Refs, 2)
	assert.Equal(t, SceneRef{Chapter: 1, Position: 1}, spec.Refs[0])
	assert.Equal(t, SceneRef{Chapter: 3, Position: 2}, spec.Refs[1])
}

func TestParseRange_ChapterList(t *testing.T) {
	spec, err := ParseRange("1,3,5")
	require.NoError(t, err)
	assert.Equal(t, "list", spec.Kind)
	require.Len(t, spec.Refs, 3)
	assert.Equal(t, SceneRef{Chapter: 1, Position: 0}, spec.Refs[0])
	assert.Equal(t, SceneRef{Chapter: 3, Position: 0}, spec.Refs[1])
	assert.Equal(t, SceneRef{Chapter: 5, Position: 0}, spec.Refs[2])
}

func TestParseRange_DottedList(t *testing.T) {
	spec, err := ParseRange("1.1,2.3")
	require.NoError(t, err)
	assert.Equal(t, "list", spec.Kind)
	require.Len(t, spec.Refs, 2)
	assert.Equal(t, SceneRef{Chapter: 1, Position: 1}, spec.Refs[0])
	assert.Equal(t, SceneRef{Chapter: 2, Position: 3}, spec.Refs[1])
}

func TestParseRange_MixedDottedChapterList(t *testing.T) {
	_, err := ParseRange("1,2.3")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mixed")
}

func TestParseRange_MixedDottedChapterRange(t *testing.T) {
	_, err := ParseRange("1-2.3")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mixed")
}

func TestParseRange_Empty(t *testing.T) {
	_, err := ParseRange("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestParseRange_InvalidSyntax(t *testing.T) {
	_, err := ParseRange("abc")
	require.Error(t, err)
}

func TestParseRange_EndBeforeStart_Chapter(t *testing.T) {
	_, err := ParseRange("3-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "before start")
}

func TestParseRange_EndBeforeStart_Dotted(t *testing.T) {
	_, err := ParseRange("2.3-2.1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "before start")
}

// --- ResolveScenePaths tests ---

func testBook() *binder.Book {
	return &binder.Book{
		BaseDir: "manuscript",
		Chapters: []binder.Chapter{
			{Scenes: []string{"ch1-s1", "ch1-s2", "ch1-s3"}},
			{Scenes: []string{"ch2-s1", "ch2-s2"}},
			{Subdir: "part2", Scenes: []string{"ch3-s1", "ch3-s2"}},
			{Scenes: []string{"ch4-s1"}},
		},
	}
}

func TestResolveScenePaths_ChapterRange(t *testing.T) {
	book := testBook()
	spec := RangeSpec{Kind: "range", Refs: []SceneRef{{1, 0}, {2, 0}}}

	paths, err := ResolveScenePaths("/project", book, spec)
	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join("/project", "manuscript", "ch1-s1.md"),
		filepath.Join("/project", "manuscript", "ch1-s2.md"),
		filepath.Join("/project", "manuscript", "ch1-s3.md"),
		filepath.Join("/project", "manuscript", "ch2-s1.md"),
		filepath.Join("/project", "manuscript", "ch2-s2.md"),
	}, paths)
}

func TestResolveScenePaths_DottedRange(t *testing.T) {
	book := testBook()
	spec := RangeSpec{Kind: "range", Refs: []SceneRef{{1, 2}, {2, 1}}}

	paths, err := ResolveScenePaths("/project", book, spec)
	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join("/project", "manuscript", "ch1-s2.md"),
		filepath.Join("/project", "manuscript", "ch1-s3.md"),
		filepath.Join("/project", "manuscript", "ch2-s1.md"),
	}, paths)
}

func TestResolveScenePaths_DottedRange_SameChapter(t *testing.T) {
	book := testBook()
	spec := RangeSpec{Kind: "range", Refs: []SceneRef{{1, 1}, {1, 3}}}

	paths, err := ResolveScenePaths("/project", book, spec)
	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join("/project", "manuscript", "ch1-s1.md"),
		filepath.Join("/project", "manuscript", "ch1-s2.md"),
		filepath.Join("/project", "manuscript", "ch1-s3.md"),
	}, paths)
}

func TestResolveScenePaths_SingleChapter(t *testing.T) {
	book := testBook()
	spec := RangeSpec{Kind: "list", Refs: []SceneRef{{2, 0}}}

	paths, err := ResolveScenePaths("/project", book, spec)
	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join("/project", "manuscript", "ch2-s1.md"),
		filepath.Join("/project", "manuscript", "ch2-s2.md"),
	}, paths)
}

func TestResolveScenePaths_SingleDotted(t *testing.T) {
	book := testBook()
	spec := RangeSpec{Kind: "list", Refs: []SceneRef{{1, 2}}}

	paths, err := ResolveScenePaths("/project", book, spec)
	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join("/project", "manuscript", "ch1-s2.md"),
	}, paths)
}

func TestResolveScenePaths_Subdir(t *testing.T) {
	book := testBook()
	spec := RangeSpec{Kind: "list", Refs: []SceneRef{{3, 0}}}

	paths, err := ResolveScenePaths("/project", book, spec)
	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join("/project", "manuscript", "part2", "ch3-s1.md"),
		filepath.Join("/project", "manuscript", "part2", "ch3-s2.md"),
	}, paths)
}

func TestResolveScenePaths_ChapterOutOfRange(t *testing.T) {
	book := testBook()
	spec := RangeSpec{Kind: "list", Refs: []SceneRef{{10, 0}}}

	_, err := ResolveScenePaths("/project", book, spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestResolveScenePaths_PositionOutOfRange(t *testing.T) {
	book := testBook()
	spec := RangeSpec{Kind: "list", Refs: []SceneRef{{1, 10}}}

	_, err := ResolveScenePaths("/project", book, spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestResolveScenePaths_EmptyChapter(t *testing.T) {
	book := &binder.Book{
		BaseDir:  "manuscript",
		Chapters: []binder.Chapter{{Scenes: nil}},
	}
	spec := RangeSpec{Kind: "list", Refs: []SceneRef{{1, 0}}}

	_, err := ResolveScenePaths("/project", book, spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no scenes")
}

func TestResolveScenePaths_ChapterList(t *testing.T) {
	book := testBook()
	spec := RangeSpec{Kind: "list", Refs: []SceneRef{{1, 0}, {4, 0}}}

	paths, err := ResolveScenePaths("/project", book, spec)
	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join("/project", "manuscript", "ch1-s1.md"),
		filepath.Join("/project", "manuscript", "ch1-s2.md"),
		filepath.Join("/project", "manuscript", "ch1-s3.md"),
		filepath.Join("/project", "manuscript", "ch4-s1.md"),
	}, paths)
}

func TestResolveScenePaths_DottedList(t *testing.T) {
	book := testBook()
	spec := RangeSpec{Kind: "list", Refs: []SceneRef{{1, 1}, {2, 2}}}

	paths, err := ResolveScenePaths("/project", book, spec)
	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join("/project", "manuscript", "ch1-s1.md"),
		filepath.Join("/project", "manuscript", "ch2-s2.md"),
	}, paths)
}

func TestResolveScenePaths_DottedRange_PositionOutOfRange(t *testing.T) {
	book := testBook()
	spec := RangeSpec{Kind: "range", Refs: []SceneRef{{1, 1}, {2, 10}}}

	_, err := ResolveScenePaths("/project", book, spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

// --- ResolveSlugs tests ---

func testBookWithInterludes() *binder.Book {
	return &binder.Book{
		BaseDir: "manuscript",
		Chapters: []binder.Chapter{
			{Scenes: []string{"ch1-s1", "ch1-s2", "ch1-s3"}},
			{Interlude: true, Scenes: []string{"int-s1"}},
			{Scenes: []string{"ch3-s1", "ch3-s2"}},
		},
	}
}

func TestResolveSlugs_SingleDotted(t *testing.T) {
	book := testBookWithInterludes()
	spec := RangeSpec{Kind: "list", Refs: []SceneRef{{1, 2}}}

	scenes, err := ResolveSlugs(book, spec)
	require.NoError(t, err)
	require.Len(t, scenes, 1)
	assert.Equal(t, ResolvedScene{Slug: "ch1-s2", Interlude: false}, scenes[0])
}

func TestResolveSlugs_SingleChapter(t *testing.T) {
	book := testBookWithInterludes()
	spec := RangeSpec{Kind: "list", Refs: []SceneRef{{1, 0}}}

	scenes, err := ResolveSlugs(book, spec)
	require.NoError(t, err)
	require.Len(t, scenes, 3)
	assert.Equal(t, "ch1-s1", scenes[0].Slug)
	assert.Equal(t, "ch1-s2", scenes[1].Slug)
	assert.Equal(t, "ch1-s3", scenes[2].Slug)
	for _, s := range scenes {
		assert.False(t, s.Interlude)
	}
}

func TestResolveSlugs_InterludeChapter(t *testing.T) {
	book := testBookWithInterludes()
	spec := RangeSpec{Kind: "list", Refs: []SceneRef{{2, 0}}}

	scenes, err := ResolveSlugs(book, spec)
	require.NoError(t, err)
	require.Len(t, scenes, 1)
	assert.Equal(t, ResolvedScene{Slug: "int-s1", Interlude: true}, scenes[0])
}

func TestResolveSlugs_ChapterRange(t *testing.T) {
	book := testBookWithInterludes()
	spec := RangeSpec{Kind: "range", Refs: []SceneRef{{1, 0}, {2, 0}}}

	scenes, err := ResolveSlugs(book, spec)
	require.NoError(t, err)
	require.Len(t, scenes, 4)
	assert.Equal(t, "ch1-s1", scenes[0].Slug)
	assert.False(t, scenes[0].Interlude)
	assert.Equal(t, "int-s1", scenes[3].Slug)
	assert.True(t, scenes[3].Interlude)
}

func TestResolveSlugs_DottedRange(t *testing.T) {
	book := testBookWithInterludes()
	spec := RangeSpec{Kind: "range", Refs: []SceneRef{{1, 2}, {3, 1}}}

	scenes, err := ResolveSlugs(book, spec)
	require.NoError(t, err)
	require.Len(t, scenes, 4)
	assert.Equal(t, "ch1-s2", scenes[0].Slug)
	assert.Equal(t, "ch1-s3", scenes[1].Slug)
	assert.Equal(t, "int-s1", scenes[2].Slug)
	assert.True(t, scenes[2].Interlude)
	assert.Equal(t, "ch3-s1", scenes[3].Slug)
}

func TestResolveSlugs_ChapterOutOfRange(t *testing.T) {
	book := testBookWithInterludes()
	spec := RangeSpec{Kind: "list", Refs: []SceneRef{{10, 0}}}

	_, err := ResolveSlugs(book, spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestResolveSlugs_PositionOutOfRange(t *testing.T) {
	book := testBookWithInterludes()
	spec := RangeSpec{Kind: "list", Refs: []SceneRef{{1, 10}}}

	_, err := ResolveSlugs(book, spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestResolveSlugs_ChapterList(t *testing.T) {
	book := testBookWithInterludes()
	spec := RangeSpec{Kind: "list", Refs: []SceneRef{{1, 0}, {3, 0}}}

	scenes, err := ResolveSlugs(book, spec)
	require.NoError(t, err)
	require.Len(t, scenes, 5)
	assert.Equal(t, "ch1-s1", scenes[0].Slug)
	assert.Equal(t, "ch3-s2", scenes[4].Slug)
}

func TestResolveSlugs_DottedList(t *testing.T) {
	book := testBookWithInterludes()
	spec := RangeSpec{Kind: "list", Refs: []SceneRef{{1, 1}, {2, 1}}}

	scenes, err := ResolveSlugs(book, spec)
	require.NoError(t, err)
	require.Len(t, scenes, 2)
	assert.Equal(t, ResolvedScene{Slug: "ch1-s1", Interlude: false}, scenes[0])
	assert.Equal(t, ResolvedScene{Slug: "int-s1", Interlude: true}, scenes[1])
}
