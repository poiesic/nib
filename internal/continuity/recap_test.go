package continuity

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/poiesic/binder"
	"github.com/poiesic/nib/internal/manuscript"
	"github.com/poiesic/nib/internal/storydb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveChapters_Single(t *testing.T) {
	book := &binder.Book{
		Chapters: []binder.Chapter{
			{Name: "Opening", Scenes: []string{"scene-a", "scene-b"}},
			{Name: "Rising", Scenes: []string{"scene-c"}},
		},
	}

	spec, err := manuscript.ParseRange("1")
	require.NoError(t, err)

	chapters, err := resolveChapters(book, spec)
	require.NoError(t, err)
	require.Len(t, chapters, 1)
	assert.Equal(t, 1, chapters[0].number)
	assert.Equal(t, "Opening", chapters[0].name)
	assert.Equal(t, []string{"scene-a", "scene-b"}, chapters[0].slugs)
}

func TestResolveChapters_Range(t *testing.T) {
	book := &binder.Book{
		Chapters: []binder.Chapter{
			{Scenes: []string{"s1"}},
			{Scenes: []string{"s2"}},
			{Scenes: []string{"s3"}},
		},
	}

	spec, err := manuscript.ParseRange("1-3")
	require.NoError(t, err)

	chapters, err := resolveChapters(book, spec)
	require.NoError(t, err)
	require.Len(t, chapters, 3)
	assert.Equal(t, 1, chapters[0].number)
	assert.Equal(t, 3, chapters[2].number)
}

func TestResolveChapters_List(t *testing.T) {
	book := &binder.Book{
		Chapters: []binder.Chapter{
			{Scenes: []string{"s1"}},
			{Scenes: []string{"s2"}},
			{Scenes: []string{"s3"}},
		},
	}

	spec, err := manuscript.ParseRange("1,3")
	require.NoError(t, err)

	chapters, err := resolveChapters(book, spec)
	require.NoError(t, err)
	require.Len(t, chapters, 2)
	assert.Equal(t, 1, chapters[0].number)
	assert.Equal(t, 3, chapters[1].number)
}

func TestResolveChapters_RejectsDottedRefs(t *testing.T) {
	book := &binder.Book{
		Chapters: []binder.Chapter{
			{Scenes: []string{"s1", "s2"}},
		},
	}

	spec, err := manuscript.ParseRange("1.1")
	require.NoError(t, err)

	_, err = resolveChapters(book, spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "whole chapters")
}

func TestResolveChapters_RejectsDottedRange(t *testing.T) {
	book := &binder.Book{
		Chapters: []binder.Chapter{
			{Scenes: []string{"s1", "s2"}},
			{Scenes: []string{"s3"}},
		},
	}

	spec, err := manuscript.ParseRange("1.1-2.1")
	require.NoError(t, err)

	_, err = resolveChapters(book, spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "whole chapters")
}

func TestResolveChapters_OutOfRange(t *testing.T) {
	book := &binder.Book{
		Chapters: []binder.Chapter{
			{Scenes: []string{"s1"}},
		},
	}

	spec, err := manuscript.ParseRange("5")
	require.NoError(t, err)

	_, err = resolveChapters(book, spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestResolveChapters_Interlude(t *testing.T) {
	book := &binder.Book{
		Chapters: []binder.Chapter{
			{Scenes: []string{"s1"}},
			{Interlude: true, Scenes: []string{"interlude-1"}},
			{Scenes: []string{"s2"}},
		},
	}

	spec, err := manuscript.ParseRange("2")
	require.NoError(t, err)

	chapters, err := resolveChapters(book, spec)
	require.NoError(t, err)
	require.Len(t, chapters, 1)
	assert.True(t, chapters[0].interlude)
	assert.Equal(t, []string{"interlude-1"}, chapters[0].slugs)
}

func TestRecapOutput_JSONShape(t *testing.T) {
	output := RecapOutput{
		Chapters: []RecapChapter{
			{
				Chapter: 1,
				Name:    "Opening",
				Scenes: []RecapScene{
					{
						Slug:     "lance-arrives",
						Position: 1,
						POV:      "lance",
						Location: "cafe",
						Summary:  "Lance walks into the cafe",
						Indexed:  true,
						Facts: []RecapFact{
							{Category: "event", Summary: "Lance enters"},
						},
						Characters: []RecapCharacter{
							{Character: "lance", Role: "pov"},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	require.NoError(t, enc.Encode(output))

	// Round-trip to verify structure
	var decoded RecapOutput
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))

	require.Len(t, decoded.Chapters, 1)
	ch := decoded.Chapters[0]
	assert.Equal(t, 1, ch.Chapter)
	assert.Equal(t, "Opening", ch.Name)

	require.Len(t, ch.Scenes, 1)
	sc := ch.Scenes[0]
	assert.Equal(t, "lance-arrives", sc.Slug)
	assert.Equal(t, 1, sc.Position)
	assert.Equal(t, "lance", sc.POV)
	assert.True(t, sc.Indexed)

	require.Len(t, sc.Facts, 1)
	assert.Equal(t, "event", sc.Facts[0].Category)

	require.Len(t, sc.Characters, 1)
	assert.Equal(t, "lance", sc.Characters[0].Character)
}

func TestRecap_CharacterFilter(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	// Set up a 2-chapter project
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
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(bookYAML), 0644))
	msDir := filepath.Join(dir, "scenes")
	require.NoError(t, os.MkdirAll(msDir, 0755))
	for _, name := range []string{"scene-a", "scene-b", "scene-c"} {
		require.NoError(t, os.WriteFile(filepath.Join(msDir, name+".md"), []byte("prose"), 0644))
	}

	// Populate storydb
	db, err := storydb.Open(dir)
	require.NoError(t, err)
	require.NoError(t, db.UpsertScene(storydb.Scene{Scene: "scene-a", POV: "lance", Summary: "A"}))
	require.NoError(t, db.UpsertScene(storydb.Scene{Scene: "scene-b", POV: "bo", Summary: "B"}))
	require.NoError(t, db.UpsertScene(storydb.Scene{Scene: "scene-c", POV: "lance", Summary: "C"}))
	require.NoError(t, db.InsertSceneCharacters([]storydb.SceneCharacter{
		{Scene: "scene-a", Character: "lance", Role: "pov"},
		{Scene: "scene-a", Character: "bo", Role: "present"},
		{Scene: "scene-b", Character: "bo", Role: "pov"},
		{Scene: "scene-c", Character: "lance", Role: "pov"},
		{Scene: "scene-c", Character: "eddie", Role: "present"},
	}))
	db.Close()

	// Recap filtered to lance only
	var buf bytes.Buffer
	err = Recap(RecapOptions{
		Range:      "1-2",
		Characters: []string{"lance"},
		Stdout:     &buf,
		Stderr:     io.Discard,
	})
	require.NoError(t, err)

	var output RecapOutput
	require.NoError(t, json.Unmarshal(buf.Bytes(), &output))

	// Chapter 1 should have scene-a (lance is pov) but not scene-b (bo only)
	// Chapter 2 should have scene-c (lance is pov)
	require.Len(t, output.Chapters, 2)

	ch1 := output.Chapters[0]
	require.Len(t, ch1.Scenes, 1)
	assert.Equal(t, "scene-a", ch1.Scenes[0].Slug)

	ch2 := output.Chapters[1]
	require.Len(t, ch2.Scenes, 1)
	assert.Equal(t, "scene-c", ch2.Scenes[0].Slug)
}

func TestRecap_CharacterFilter_DropsEmptyChapters(t *testing.T) {
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
    - scenes:
        - scene-b
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(bookYAML), 0644))
	msDir := filepath.Join(dir, "scenes")
	require.NoError(t, os.MkdirAll(msDir, 0755))
	for _, name := range []string{"scene-a", "scene-b"} {
		require.NoError(t, os.WriteFile(filepath.Join(msDir, name+".md"), []byte("prose"), 0644))
	}

	db, err := storydb.Open(dir)
	require.NoError(t, err)
	require.NoError(t, db.UpsertScene(storydb.Scene{Scene: "scene-a", POV: "lance"}))
	require.NoError(t, db.UpsertScene(storydb.Scene{Scene: "scene-b", POV: "bo"}))
	require.NoError(t, db.InsertSceneCharacters([]storydb.SceneCharacter{
		{Scene: "scene-a", Character: "lance", Role: "pov"},
		{Scene: "scene-b", Character: "bo", Role: "pov"},
	}))
	db.Close()

	// Filter to bo — only chapter 2 has bo
	var buf bytes.Buffer
	err = Recap(RecapOptions{
		Range:      "1-2",
		Characters: []string{"bo"},
		Stdout:     &buf,
		Stderr:     io.Discard,
	})
	require.NoError(t, err)

	var output RecapOutput
	require.NoError(t, json.Unmarshal(buf.Bytes(), &output))

	// Chapter 1 should be dropped entirely
	require.Len(t, output.Chapters, 1)
	assert.Equal(t, 2, output.Chapters[0].Chapter)
}

func TestRecap_DefaultOmitsFactsAndMentioned(t *testing.T) {
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
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(bookYAML), 0644))
	msDir := filepath.Join(dir, "scenes")
	require.NoError(t, os.MkdirAll(msDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(msDir, "scene-a.md"), []byte("prose"), 0644))

	db, err := storydb.Open(dir)
	require.NoError(t, err)
	require.NoError(t, db.UpsertScene(storydb.Scene{
		Scene: "scene-a", POV: "lance", Location: "cafe", Date: "day-1", Time: "morning", Summary: "A",
	}))
	require.NoError(t, db.InsertFacts([]storydb.Fact{
		{ID: "f1", Scene: "scene-a", Category: "event", Summary: "something"},
	}))
	require.NoError(t, db.InsertSceneCharacters([]storydb.SceneCharacter{
		{Scene: "scene-a", Character: "lance", Role: "pov"},
		{Scene: "scene-a", Character: "bo", Role: "present"},
		{Scene: "scene-a", Character: "eddie", Role: "mentioned"},
	}))
	db.Close()

	// Default (compact) mode
	var buf bytes.Buffer
	err = Recap(RecapOptions{
		Range:  "1",
		Stdout: &buf,
		Stderr: io.Discard,
	})
	require.NoError(t, err)

	var output RecapOutput
	require.NoError(t, json.Unmarshal(buf.Bytes(), &output))

	sc := output.Chapters[0].Scenes[0]
	assert.Equal(t, "lance", sc.POV)
	assert.Equal(t, "A", sc.Summary)
	// Compact: no location, date, time, facts
	assert.Empty(t, sc.Location)
	assert.Empty(t, sc.Date)
	assert.Empty(t, sc.Time)
	assert.Nil(t, sc.Facts)
	// Compact: mentioned characters excluded
	require.Len(t, sc.Characters, 2)
	for _, c := range sc.Characters {
		assert.NotEqual(t, "mentioned", c.Role)
	}
}

func TestRecap_DetailedIncludesEverything(t *testing.T) {
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
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(bookYAML), 0644))
	msDir := filepath.Join(dir, "scenes")
	require.NoError(t, os.MkdirAll(msDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(msDir, "scene-a.md"), []byte("prose"), 0644))

	db, err := storydb.Open(dir)
	require.NoError(t, err)
	require.NoError(t, db.UpsertScene(storydb.Scene{
		Scene: "scene-a", POV: "lance", Location: "cafe", Date: "day-1", Time: "morning", Summary: "A",
	}))
	require.NoError(t, db.InsertFacts([]storydb.Fact{
		{ID: "f1", Scene: "scene-a", Category: "event", Summary: "something"},
	}))
	require.NoError(t, db.InsertSceneCharacters([]storydb.SceneCharacter{
		{Scene: "scene-a", Character: "lance", Role: "pov"},
		{Scene: "scene-a", Character: "bo", Role: "present"},
		{Scene: "scene-a", Character: "eddie", Role: "mentioned"},
	}))
	db.Close()

	// Detailed mode
	var buf bytes.Buffer
	err = Recap(RecapOptions{
		Range:    "1",
		Detailed: true,
		Stdout:   &buf,
		Stderr:   io.Discard,
	})
	require.NoError(t, err)

	var output RecapOutput
	require.NoError(t, json.Unmarshal(buf.Bytes(), &output))

	sc := output.Chapters[0].Scenes[0]
	assert.Equal(t, "cafe", sc.Location)
	assert.Equal(t, "day-1", sc.Date)
	assert.Equal(t, "morning", sc.Time)
	require.Len(t, sc.Facts, 1)
	assert.Equal(t, "event", sc.Facts[0].Category)
	// Detailed: all characters including mentioned
	require.Len(t, sc.Characters, 3)
	roles := map[string]bool{}
	for _, c := range sc.Characters {
		roles[c.Role] = true
	}
	assert.True(t, roles["mentioned"])
}

func TestRecap_UnknownCharacter(t *testing.T) {
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
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(bookYAML), 0644))
	msDir := filepath.Join(dir, "scenes")
	require.NoError(t, os.MkdirAll(msDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(msDir, "scene-a.md"), []byte("prose"), 0644))

	db, err := storydb.Open(dir)
	require.NoError(t, err)
	require.NoError(t, db.InsertSceneCharacters([]storydb.SceneCharacter{
		{Scene: "scene-a", Character: "lance", Role: "pov"},
	}))
	db.Close()

	var buf bytes.Buffer
	err = Recap(RecapOptions{
		Range:      "1",
		Characters: []string{"nobody"},
		Stdout:     &buf,
		Stderr:     io.Discard,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `character slug "nobody" not found`)
	assert.Contains(t, err.Error(), "slug format")
}

func TestRecapOutput_OmitsEmptyFields(t *testing.T) {
	output := RecapOutput{
		Chapters: []RecapChapter{
			{
				Chapter: 1,
				Scenes: []RecapScene{
					{
						Slug:     "unindexed",
						Position: 1,
						Indexed:  false,
					},
				},
			},
		},
	}

	data, err := json.Marshal(output)
	require.NoError(t, err)

	// Unindexed scene should omit pov, location, summary, facts, characters
	var raw map[string]any
	require.NoError(t, json.Unmarshal(data, &raw))

	chapters := raw["chapters"].([]any)
	ch := chapters[0].(map[string]any)
	// name should be omitted (empty)
	_, hasName := ch["name"]
	assert.False(t, hasName)
	// interlude should be omitted (false)
	_, hasInterlude := ch["interlude"]
	assert.False(t, hasInterlude)

	scenes := ch["scenes"].([]any)
	sc := scenes[0].(map[string]any)
	_, hasPOV := sc["pov"]
	assert.False(t, hasPOV)
	_, hasFacts := sc["facts"]
	assert.False(t, hasFacts)
	_, hasChars := sc["characters"]
	assert.False(t, hasChars)
}

func TestRecap_DefaultOutputIsCompact(t *testing.T) {
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
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(bookYAML), 0644))
	msDir := filepath.Join(dir, "scenes")
	require.NoError(t, os.MkdirAll(msDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(msDir, "scene-a.md"), []byte("prose"), 0644))

	db, err := storydb.Open(dir)
	require.NoError(t, err)
	require.NoError(t, db.UpsertScene(storydb.Scene{Scene: "scene-a", POV: "lance", Summary: "A"}))
	db.Close()

	// Default: compact JSON (single line)
	var buf bytes.Buffer
	err = Recap(RecapOptions{
		Range:  "1",
		Stdout: &buf,
		Stderr: io.Discard,
	})
	require.NoError(t, err)
	output := buf.String()
	assert.NotContains(t, output, "\n  ")

	// Pretty: indented JSON
	buf.Reset()
	err = Recap(RecapOptions{
		Range:  "1",
		Pretty: true,
		Stdout: &buf,
		Stderr: io.Discard,
	})
	require.NoError(t, err)
	output = buf.String()
	assert.Contains(t, output, "\n  ")
}
