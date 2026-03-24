package continuity

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/poiesic/nib/internal/storydb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedCharacters(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "book.yaml"), []byte("title: Test\n"), 0644))

	db, err := storydb.Open(dir)
	require.NoError(t, err)
	err = db.InsertSceneCharacters([]storydb.SceneCharacter{
		{Scene: "scene-a", Character: "zara", Role: "pov"},
		{Scene: "scene-a", Character: "mike", Role: "present"},
		{Scene: "scene-b", Character: "mike", Role: "pov"},
		{Scene: "scene-b", Character: "alice", Role: "mentioned"},
		{Scene: "scene-c", Character: "zara", Role: "present"},
	})
	require.NoError(t, err)
	db.Close()
	return dir
}

func seedCharactersWithBook(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	bookYAML := `title: Test Novel
author: Test Author
---
book:
  base_dir: scenes
  chapters:
    - scenes:
        - scene-a
        - scene-b
    - scenes:
        - scene-c
    - scenes:
        - scene-d
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(bookYAML), 0644))
	msDir := filepath.Join(dir, "scenes")
	require.NoError(t, os.MkdirAll(msDir, 0755))
	for _, name := range []string{"scene-a", "scene-b", "scene-c", "scene-d"} {
		require.NoError(t, os.WriteFile(filepath.Join(msDir, name+".md"), []byte("prose"), 0644))
	}

	db, err := storydb.Open(dir)
	require.NoError(t, err)
	require.NoError(t, db.InsertSceneCharacters([]storydb.SceneCharacter{
		{Scene: "scene-a", Character: "zara", Role: "pov"},
		{Scene: "scene-a", Character: "mike", Role: "present"},
		{Scene: "scene-b", Character: "mike", Role: "pov"},
		{Scene: "scene-b", Character: "alice", Role: "mentioned"},
		{Scene: "scene-c", Character: "zara", Role: "present"},
		{Scene: "scene-c", Character: "bo", Role: "pov"},
		{Scene: "scene-d", Character: "eddie", Role: "pov"},
	}))
	db.Close()
	return dir
}

func TestCharacters_DefaultExcludesMentioned(t *testing.T) {
	seedCharacters(t)

	var buf bytes.Buffer
	err := Characters(CharactersOptions{Stdout: &buf})
	require.NoError(t, err)

	var result []string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))

	assert.Equal(t, []string{"mike", "zara"}, result)
}

func TestCharacters_AllIncludesMentioned(t *testing.T) {
	seedCharacters(t)

	var buf bytes.Buffer
	err := Characters(CharactersOptions{All: true, Stdout: &buf})
	require.NoError(t, err)

	var result []string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))

	assert.Equal(t, []string{"alice", "mike", "zara"}, result)
}

func TestCharacters_EmptyStorydb(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "book.yaml"), []byte("title: Test\n"), 0644))

	db, err := storydb.Open(dir)
	require.NoError(t, err)
	db.Close()

	var buf bytes.Buffer
	err = Characters(CharactersOptions{Stdout: &buf})
	require.NoError(t, err)

	var result []string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))

	assert.Empty(t, result)
}

func TestCharacters_RangeChapter(t *testing.T) {
	seedCharactersWithBook(t)

	var buf bytes.Buffer
	// Chapter 1 has scene-a (zara pov, mike present) and scene-b (mike pov, alice mentioned)
	err := Characters(CharactersOptions{Range: "1", Stdout: &buf})
	require.NoError(t, err)

	var result []string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))

	assert.Equal(t, []string{"mike", "zara"}, result)
}

func TestCharacters_RangeChapterAll(t *testing.T) {
	seedCharactersWithBook(t)

	var buf bytes.Buffer
	// Chapter 1 with --all should include alice (mentioned)
	err := Characters(CharactersOptions{Range: "1", All: true, Stdout: &buf})
	require.NoError(t, err)

	var result []string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))

	assert.Equal(t, []string{"alice", "mike", "zara"}, result)
}

func TestCharacters_RangeMultipleChapters(t *testing.T) {
	seedCharactersWithBook(t)

	var buf bytes.Buffer
	// Chapters 2-3: scene-c (zara present, bo pov) and scene-d (eddie pov)
	err := Characters(CharactersOptions{Range: "2-3", Stdout: &buf})
	require.NoError(t, err)

	var result []string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))

	assert.Equal(t, []string{"bo", "eddie", "zara"}, result)
}

func TestCharacters_RangeDotted(t *testing.T) {
	seedCharactersWithBook(t)

	var buf bytes.Buffer
	// Single scene: 1.2 = scene-b (mike pov, alice mentioned)
	err := Characters(CharactersOptions{Range: "1.2", Stdout: &buf})
	require.NoError(t, err)

	var result []string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))

	// Default excludes mentioned, so only mike
	assert.Equal(t, []string{"mike"}, result)
}

func TestCharacters_RangeList(t *testing.T) {
	seedCharactersWithBook(t)

	var buf bytes.Buffer
	// Chapters 1 and 3: scene-a, scene-b, scene-d
	err := Characters(CharactersOptions{Range: "1,3", Stdout: &buf})
	require.NoError(t, err)

	var result []string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))

	assert.Equal(t, []string{"eddie", "mike", "zara"}, result)
}
