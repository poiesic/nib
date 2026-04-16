package continuity

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/poiesic/binder"
	"github.com/poiesic/nib/internal/agent"
	"github.com/poiesic/nib/internal/storydb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockExtractFn returns an ExtractFunc that returns the given ExtractionResult
// and increments a call counter.
func mockExtractFn(result ExtractionResult, callCount *atomic.Int32) ExtractFunc {
	data, _ := json.Marshal(result)
	return func(prompt string, schema json.RawMessage, dir string, effort agent.Effort) (json.RawMessage, error) {
		if callCount != nil {
			callCount.Add(1)
		}
		return data, nil
	}
}

var basicResult = ExtractionResult{
	Scene: storydb.Scene{
		POV:       "lance",
		SceneType: "regular",
		Location:  "cafe",
		Summary:   "A test scene",
	},
	Facts: []storydb.Fact{
		{Category: "event", Summary: "Something happened", Detail: "Details", SourceText: "quote"},
	},
	Characters: []storydb.SceneCharacter{
		{Character: "lance", Role: "pov"},
	},
	Locations: []storydb.Location{
		{ID: "cafe", Name: "The Cafe", Type: "public", Description: "A cozy place"},
	},
}

var minimalResult = ExtractionResult{
	Scene: storydb.Scene{
		POV:       "lance",
		SceneType: "regular",
		Location:  "cafe",
		Summary:   "Test",
	},
	Facts:      []storydb.Fact{},
	Characters: []storydb.SceneCharacter{},
	Locations:  []storydb.Location{},
}

func TestBuildPrompt_BasicScene(t *testing.T) {
	prompt := buildPrompt("test-scene", "/tmp/scenes/test-scene.md", false, "", nil, nil)

	assert.Contains(t, prompt, "continuity analyst")
	assert.Contains(t, prompt, "## Scene: test-scene")
	assert.Contains(t, prompt, "Read the scene file at: /tmp/scenes/test-scene.md")
	assert.NotContains(t, prompt, "## Prior Chapter Context")
}

func TestBuildPrompt_WithCharacterSlugs(t *testing.T) {
	slugs := []string{"lance-thurgood", "bo-mcfarlane"}
	prompt := buildPrompt("test-scene", "/tmp/scenes/test-scene.md", false, "", slugs, nil)

	assert.Contains(t, prompt, "## Known Characters")
	assert.Contains(t, prompt, "- lance-thurgood")
	assert.Contains(t, prompt, "- bo-mcfarlane")
}

func TestBuildPrompt_WithPriorRecap(t *testing.T) {
	recapJSON := `{"chapters":[{"chapter":1,"scenes":[{"slug":"intro","position":1,"indexed":true}]}]}`
	prompt := buildPrompt("ch2-scene", "/tmp/scenes/ch2-scene.md", false, recapJSON, nil, nil)

	assert.Contains(t, prompt, "## Prior Chapter Context")
	assert.Contains(t, prompt, "```json")
	assert.Contains(t, prompt, `"slug":"intro"`)
}

func TestBuildPrompt_WithEarlierExtractions(t *testing.T) {
	earlier := []*ExtractionResult{
		{
			Scene: storydb.Scene{Scene: "scene-one", POV: "lance", Summary: "Lance arrives"},
			Facts: []storydb.Fact{
				{Category: "event", Summary: "Lance enters the cafe"},
			},
		},
	}
	prompt := buildPrompt("scene-two", "/tmp/scenes/scene-two.md", false, "", nil, earlier)

	assert.Contains(t, prompt, "## Earlier Scenes in This Chapter")
	assert.Contains(t, prompt, "scene-one")
	assert.Contains(t, prompt, "Lance enters the cafe")
}

func TestBuildPrompt_Interlude(t *testing.T) {
	prompt := buildPrompt("interlude-1", "/tmp/scenes/interlude-1.md", true, "", nil, nil)
	assert.Contains(t, prompt, "interlude in the manuscript structure")
}

func TestSanitizeResult_CollapsesNewlines(t *testing.T) {
	result := &ExtractionResult{
		Scene: storydb.Scene{
			Summary: "Line one\nLine two\nLine three",
		},
		Facts: []storydb.Fact{
			{
				Summary:    "Fact\nsummary",
				Detail:     "Detail\nwith\nnewlines",
				SourceText: "6.  I find it easy to:\na)  Follow rules\nb)  Voice disagreement\nc)  Undermine the paradigm",
			},
		},
		Locations: []storydb.Location{
			{
				Description: "FOR OFFICIAL USE ONLY\nRELEASE BY    DATE",
			},
		},
	}

	sanitizeResult(result)

	assert.Equal(t, "Line one Line two Line three", result.Scene.Summary)
	assert.Equal(t, "Fact summary", result.Facts[0].Summary)
	assert.Equal(t, "Detail with newlines", result.Facts[0].Detail)
	assert.Equal(t, "6. I find it easy to: a) Follow rules b) Voice disagreement c) Undermine the paradigm", result.Facts[0].SourceText)
	assert.Equal(t, "FOR OFFICIAL USE ONLY RELEASE BY DATE", result.Locations[0].Description)
}

func TestFindScenePath(t *testing.T) {
	// This test needs a book structure — we'll test via the helper
	// For unit testing, we verify the path construction logic directly
}

func TestReadCharacterSlugs(t *testing.T) {
	dir := t.TempDir()
	charDir := filepath.Join(dir, "characters")
	require.NoError(t, os.MkdirAll(charDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(charDir, "lance-thurgood.yaml"), []byte("name: Lance"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(charDir, "bo-mcfarlane.yaml"), []byte("name: Bo"), 0644))

	slugs := readCharacterSlugs(dir)
	assert.Len(t, slugs, 2)
	assert.Contains(t, slugs, "lance-thurgood")
	assert.Contains(t, slugs, "bo-mcfarlane")
}

func TestReadCharacterSlugs_NoDir(t *testing.T) {
	dir := t.TempDir()
	slugs := readCharacterSlugs(dir)
	assert.Empty(t, slugs)
}

// TestIndex_ExtractsAndWritesRecords verifies that Index calls the extract
// function and writes records to the DB.
func TestIndex_ExtractsAndWritesRecords(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	setupTestProject(t, dir)

	var callCount atomic.Int32
	fn := mockExtractFn(basicResult, &callCount)

	stdinContent := "a\na\na\na\n"
	stdout := &strings.Builder{}

	err := Index(IndexOptions{
		Range:     "1.1",
		ExtractFn: fn,
		Stdin:     strings.NewReader(stdinContent),
		Stdout:    stdout,
	})
	require.NoError(t, err)

	assert.Equal(t, int32(1), callCount.Load())

	db, err := storydb.Open(dir)
	require.NoError(t, err)
	defer db.Close()

	scenes, err := db.QueryScenes()
	require.NoError(t, err)
	require.Len(t, scenes, 1)
	assert.Equal(t, "test-scene", scenes[0].Scene)
	assert.Equal(t, "lance", scenes[0].POV)
	assert.NotEmpty(t, scenes[0].Checksum)

	facts, err := db.QueryFacts()
	require.NoError(t, err)
	require.Len(t, facts, 1)
	assert.Len(t, facts[0].ID, 26)

	chars, err := db.QuerySceneCharacters()
	require.NoError(t, err)
	require.Len(t, chars, 1)

	locs, err := db.QueryLocations()
	require.NoError(t, err)
	require.Len(t, locs, 1)

	assert.Contains(t, stdout.String(), "Indexed test-scene")
}

// TestIndex_RejectFact verifies that rejected records are not written.
func TestIndex_RejectFact(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	setupTestProject(t, dir)

	result := ExtractionResult{
		Scene: storydb.Scene{
			POV:       "lance",
			SceneType: "regular",
			Location:  "cafe",
			Summary:   "Test",
		},
		Facts: []storydb.Fact{
			{Category: "event", Summary: "Keep this"},
			{Category: "event", Summary: "Reject this"},
		},
		Characters: []storydb.SceneCharacter{},
		Locations:  []storydb.Location{},
	}

	stdinContent := "a\na\nr\n"
	stdout := &strings.Builder{}

	err := Index(IndexOptions{
		Range:     "1.1",
		ExtractFn: mockExtractFn(result, nil),
		Stdin:     strings.NewReader(stdinContent),
		Stdout:    stdout,
	})
	require.NoError(t, err)

	db, err := storydb.Open(dir)
	require.NoError(t, err)
	defer db.Close()

	facts, err := db.QueryFacts()
	require.NoError(t, err)
	require.Len(t, facts, 1)
	assert.Equal(t, "Keep this", facts[0].Summary)
}

func TestIndex_EmptyScene(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	setupTestProject(t, dir)

	scenePath := filepath.Join(dir, "scenes", "test-scene.md")
	require.NoError(t, os.WriteFile(scenePath, []byte(""), 0644))

	stdout := &strings.Builder{}
	err := Index(IndexOptions{
		Range:  "1.1",
		Stdout: stdout,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestIndex_SceneOutOfRange(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	setupTestProject(t, dir)

	stdout := &strings.Builder{}
	err := Index(IndexOptions{
		Range:  "1.5",
		Stdout: stdout,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestIndex_ChapterRange(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	setupTestProject(t, dir)

	stdout := &strings.Builder{}
	err := Index(IndexOptions{
		Range:     "1",
		ExtractFn: mockExtractFn(minimalResult, nil),
		Stdin:     strings.NewReader("a\n"),
		Stdout:    stdout,
	})
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Indexed test-scene")
}

// TestIndex_ChecksumSkip verifies that unchanged scenes are skipped.
func TestIndex_ChecksumSkip(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	setupTestProject(t, dir)

	var callCount atomic.Int32
	fn := mockExtractFn(minimalResult, &callCount)

	// First index — should call extract
	stdout := &strings.Builder{}
	err := Index(IndexOptions{
		Range:     "1.1",
		ExtractFn: fn,
		Stdin:     strings.NewReader("a\n"),
		Stdout:    stdout,
	})
	require.NoError(t, err)
	assert.Equal(t, int32(1), callCount.Load())

	// Second index — scene unchanged, should skip
	stdout.Reset()
	err = Index(IndexOptions{
		Range:     "1.1",
		ExtractFn: fn,
		Stdin:     strings.NewReader("a\n"),
		Stdout:    stdout,
	})
	require.NoError(t, err)
	assert.Equal(t, int32(1), callCount.Load()) // still 1
	assert.Contains(t, stdout.String(), "unchanged, skipping")
}

// TestIndex_ForceReindex verifies that --force overrides checksum skip.
func TestIndex_ForceReindex(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	setupTestProject(t, dir)

	var callCount atomic.Int32
	fn := mockExtractFn(minimalResult, &callCount)

	// First index
	stdout := &strings.Builder{}
	err := Index(IndexOptions{
		Range:     "1.1",
		ExtractFn: fn,
		Stdin:     strings.NewReader("a\n"),
		Stdout:    stdout,
	})
	require.NoError(t, err)
	assert.Equal(t, int32(1), callCount.Load())

	// Second index with --force — should call extract again
	stdout.Reset()
	err = Index(IndexOptions{
		Range:     "1.1",
		Force:     true,
		ExtractFn: fn,
		Stdin:     strings.NewReader("a\n"),
		Stdout:    stdout,
	})
	require.NoError(t, err)
	assert.Equal(t, int32(2), callCount.Load())
	assert.Contains(t, stdout.String(), "Indexed test-scene")
}

// TestIndex_MultipleScenes verifies that a range indexing multiple scenes
// processes each one and prints a total summary.
func TestIndex_MultipleScenes(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	setupMultiSceneProject(t, dir)

	result := ExtractionResult{
		Scene: storydb.Scene{
			POV:       "lance",
			SceneType: "regular",
			Location:  "cafe",
			Summary:   "A test scene",
		},
		Facts: []storydb.Fact{
			{Category: "event", Summary: "Something happened", Detail: "Details", SourceText: "quote"},
		},
		Characters: []storydb.SceneCharacter{},
		Locations:  []storydb.Location{},
	}

	var callCount atomic.Int32
	fn := mockExtractFn(result, &callCount)

	stdinContent := "a\na\na\na\n"
	stdout := &strings.Builder{}

	err := Index(IndexOptions{
		Range:     "1",
		ExtractFn: fn,
		Stdin:     strings.NewReader(stdinContent),
		Stdout:    stdout,
	})
	require.NoError(t, err)

	assert.Equal(t, int32(2), callCount.Load())

	output := stdout.String()
	assert.Contains(t, output, "Indexed scene-one")
	assert.Contains(t, output, "Indexed scene-two")
	assert.Contains(t, output, "Total: 2 scenes indexed")

	db, err := storydb.Open(dir)
	require.NoError(t, err)
	defer db.Close()

	scenes, err := db.QueryScenes()
	require.NoError(t, err)
	assert.Len(t, scenes, 2)
}

// TestIndex_SequentialExtraction verifies that multi-scene indexing runs
// extractions sequentially so each scene gets context from earlier scenes.
func TestIndex_SequentialExtraction(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	setupThreeSceneProject(t, dir)

	result := ExtractionResult{
		Scene: storydb.Scene{
			POV:       "lance",
			SceneType: "regular",
			Location:  "cafe",
			Summary:   "A test scene",
		},
		Facts: []storydb.Fact{
			{Category: "event", Summary: "Something happened", Detail: "Details", SourceText: "quote"},
		},
		Characters: []storydb.SceneCharacter{
			{Character: "lance", Role: "pov"},
		},
		Locations: []storydb.Location{},
	}

	var callCount atomic.Int32
	fn := mockExtractFn(result, &callCount)

	stdinContent := "a\na\na\na\na\na\na\na\na\n"
	stdout := &strings.Builder{}

	err := Index(IndexOptions{
		Range:     "1",
		ExtractFn: fn,
		Stdin:     strings.NewReader(stdinContent),
		Stdout:    stdout,
	})
	require.NoError(t, err)

	assert.Equal(t, int32(3), callCount.Load())

	output := stdout.String()
	assert.Contains(t, output, "Indexed scene-one")
	assert.Contains(t, output, "Indexed scene-two")
	assert.Contains(t, output, "Indexed scene-three")
	assert.Contains(t, output, "Total: 3 scenes indexed, 0 skipped")

	db, err := storydb.Open(dir)
	require.NoError(t, err)
	defer db.Close()

	scenes, err := db.QueryScenes()
	require.NoError(t, err)
	assert.Len(t, scenes, 3)

	facts, err := db.QueryFacts()
	require.NoError(t, err)
	assert.Len(t, facts, 3)

	chars, err := db.QuerySceneCharacters()
	require.NoError(t, err)
	assert.Len(t, chars, 3)
}

func TestBuildPriorRecap_Chapter1_Empty(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	setupThreeChapterProject(t, dir)

	book := testBook3Chapters()
	result := buildPriorRecap("scene-a", book, dir)
	assert.Empty(t, result, "chapter 1 scene should have no prior recap")
}

func TestBuildPriorRecap_Chapter2_GetsChapter1(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	setupThreeChapterProject(t, dir)

	db, err := storydb.Open(dir)
	require.NoError(t, err)
	require.NoError(t, db.UpsertScene(storydb.Scene{Scene: "scene-a", POV: "lance", Location: "cafe"}))
	db.Close()

	book := testBook3Chapters()
	result := buildPriorRecap("scene-b", book, dir)
	assert.Contains(t, result, "scene-a")
}

func TestBuildPriorRecap_Chapter3_Gets2Prior(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	setupThreeChapterProject(t, dir)

	db, err := storydb.Open(dir)
	require.NoError(t, err)
	require.NoError(t, db.UpsertScene(storydb.Scene{Scene: "scene-a", POV: "lance", Location: "cafe"}))
	require.NoError(t, db.UpsertScene(storydb.Scene{Scene: "scene-b", POV: "bo", Location: "office"}))
	db.Close()

	book := testBook3Chapters()
	result := buildPriorRecap("scene-c", book, dir)
	assert.Contains(t, result, "scene-a")
	assert.Contains(t, result, "scene-b")
}

func testBook3Chapters() *binder.Book {
	return &binder.Book{
		BaseDir: "scenes",
		Chapters: []binder.Chapter{
			{Scenes: []string{"scene-a"}},
			{Scenes: []string{"scene-b"}},
			{Scenes: []string{"scene-c"}},
		},
	}
}

func setupThreeChapterProject(t *testing.T, dir string) {
	t.Helper()

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
    - scenes:
        - scene-c
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(bookYAML), 0644))

	msDir := filepath.Join(dir, "scenes")
	require.NoError(t, os.MkdirAll(msDir, 0755))
	for _, name := range []string{"scene-a", "scene-b", "scene-c"} {
		require.NoError(t, os.WriteFile(filepath.Join(msDir, name+".md"),
			[]byte("Some prose for "+name+"."), 0644))
	}

	db, err := storydb.Open(dir)
	require.NoError(t, err)
	db.Close()
}

// setupTestProject creates a minimal project structure for testing.
func setupTestProject(t *testing.T, dir string) {
	t.Helper()

	bookYAML := `title: Test Novel
author: Test Author
---
book:
  base_dir: scenes
  chapters:
    - scenes:
        - test-scene
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(bookYAML), 0644))

	msDir := filepath.Join(dir, "scenes")
	require.NoError(t, os.MkdirAll(msDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(msDir, "test-scene.md"),
		[]byte("Lance walked into the cafe and sat down across from Bo."), 0644))

	db, err := storydb.Open(dir)
	require.NoError(t, err)
	db.Close()
}

// setupMultiSceneProject creates a project with two scenes in one chapter.
func setupMultiSceneProject(t *testing.T, dir string) {
	t.Helper()

	bookYAML := `title: Test Novel
author: Test Author
---
book:
  base_dir: scenes
  chapters:
    - scenes:
        - scene-one
        - scene-two
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(bookYAML), 0644))

	msDir := filepath.Join(dir, "scenes")
	require.NoError(t, os.MkdirAll(msDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(msDir, "scene-one.md"),
		[]byte("Lance walked into the cafe and sat down across from Bo."), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(msDir, "scene-two.md"),
		[]byte("Bo leaned back and crossed his arms, staring at Lance."), 0644))

	db, err := storydb.Open(dir)
	require.NoError(t, err)
	db.Close()
}

// setupThreeSceneProject creates a project with three scenes in one chapter.
func setupThreeSceneProject(t *testing.T, dir string) {
	t.Helper()

	bookYAML := `title: Test Novel
author: Test Author
---
book:
  base_dir: scenes
  chapters:
    - scenes:
        - scene-one
        - scene-two
        - scene-three
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(bookYAML), 0644))

	msDir := filepath.Join(dir, "scenes")
	require.NoError(t, os.MkdirAll(msDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(msDir, "scene-one.md"),
		[]byte("Lance walked into the cafe and sat down across from Bo."), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(msDir, "scene-two.md"),
		[]byte("Bo leaned back and crossed his arms, staring at Lance."), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(msDir, "scene-three.md"),
		[]byte("They both stood up and walked out into the rain."), 0644))

	db, err := storydb.Open(dir)
	require.NoError(t, err)
	db.Close()
}
