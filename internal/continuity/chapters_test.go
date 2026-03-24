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

func setupChaptersProject(t *testing.T) string {
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
        - scene-e
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(bookYAML), 0644))
	msDir := filepath.Join(dir, "scenes")
	require.NoError(t, os.MkdirAll(msDir, 0755))
	for _, name := range []string{"scene-a", "scene-b", "scene-c", "scene-d", "scene-e"} {
		require.NoError(t, os.WriteFile(filepath.Join(msDir, name+".md"), []byte("prose"), 0644))
	}

	db, err := storydb.Open(dir)
	require.NoError(t, err)
	require.NoError(t, db.InsertSceneCharacters([]storydb.SceneCharacter{
		{Scene: "scene-a", Character: "lance", Role: "pov"},
		{Scene: "scene-a", Character: "bo", Role: "present"},
		{Scene: "scene-b", Character: "lance", Role: "present"},
		{Scene: "scene-c", Character: "bo", Role: "pov"},
		{Scene: "scene-c", Character: "lance", Role: "mentioned"},
		{Scene: "scene-d", Character: "eddie", Role: "pov"},
		{Scene: "scene-e", Character: "bo", Role: "present"},
	}))
	db.Close()
	return dir
}

func TestChapters_SingleCharacter(t *testing.T) {
	setupChaptersProject(t)

	var buf bytes.Buffer
	err := Chapters(ChaptersOptions{Characters: []string{"lance"}, Stdout: &buf})
	require.NoError(t, err)

	var result []string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))

	// lance is pov in scene-a (1.1), present in scene-b (1.2)
	// lance is mentioned in scene-c -- excluded
	assert.Equal(t, []string{"1.1", "1.2"}, result)
}

func TestChapters_AndDefault(t *testing.T) {
	setupChaptersProject(t)

	var buf bytes.Buffer
	// Default is AND: scenes where both lance AND bo are pov/present
	err := Chapters(ChaptersOptions{Characters: []string{"lance", "bo"}, Stdout: &buf})
	require.NoError(t, err)

	var result []string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))

	// Only scene-a (1.1) has both lance (pov) and bo (present)
	assert.Equal(t, []string{"1.1"}, result)
}

func TestChapters_AndNoOverlap(t *testing.T) {
	setupChaptersProject(t)

	var buf bytes.Buffer
	// lance and eddie never share a scene
	err := Chapters(ChaptersOptions{Characters: []string{"lance", "eddie"}, Stdout: &buf})
	require.NoError(t, err)

	var result []string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))

	assert.Empty(t, result)
}

func TestChapters_OrMode(t *testing.T) {
	setupChaptersProject(t)

	var buf bytes.Buffer
	err := Chapters(ChaptersOptions{Characters: []string{"lance", "bo"}, Or: true, Stdout: &buf})
	require.NoError(t, err)

	var result map[string][]string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))

	// lance: pov in scene-a (1.1), present in scene-b (1.2)
	assert.Equal(t, []string{"1.1", "1.2"}, result["lance"])

	// bo: present in scene-a (1.1), pov in scene-c (2.1), present in scene-e (3.2)
	assert.Equal(t, []string{"1.1", "2.1", "3.2"}, result["bo"])
}

func TestChapters_OrSingleCharacter(t *testing.T) {
	setupChaptersProject(t)

	var buf bytes.Buffer
	err := Chapters(ChaptersOptions{Characters: []string{"eddie"}, Or: true, Stdout: &buf})
	require.NoError(t, err)

	var result map[string][]string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))

	assert.Equal(t, []string{"3.1"}, result["eddie"])
}

func TestChapters_UnknownCharacter(t *testing.T) {
	setupChaptersProject(t)

	var buf bytes.Buffer
	err := Chapters(ChaptersOptions{Characters: []string{"nobody"}, Stdout: &buf})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `character slug "nobody" not found`)
	assert.Contains(t, err.Error(), "slug format")
}

func TestChapters_ExcludesMentioned(t *testing.T) {
	setupChaptersProject(t)

	var buf bytes.Buffer
	err := Chapters(ChaptersOptions{Characters: []string{"lance"}, Stdout: &buf})
	require.NoError(t, err)

	var result []string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))

	// lance is mentioned in scene-c (2.1) but not pov/present
	assert.NotContains(t, result, "2.1")
}
