package continuity

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/poiesic/nib/internal/storydb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedStorydb(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "book.yaml"), []byte("title: Test\n"), 0644))

	db, err := storydb.Open(dir)
	require.NoError(t, err)
	require.NoError(t, db.UpsertScene(storydb.Scene{Scene: "scene-a", POV: "lance", Summary: "test"}))
	require.NoError(t, db.InsertSceneCharacters([]storydb.SceneCharacter{
		{Scene: "scene-a", Character: "lance", Role: "pov"},
	}))
	require.NoError(t, db.InsertFacts([]storydb.Fact{
		{Scene: "scene-a", Category: "event", Summary: "something"},
	}))
	db.Close()
	return dir
}

func TestReset_WithYesFlag(t *testing.T) {
	dir := seedStorydb(t)

	var buf bytes.Buffer
	err := Reset(ResetOptions{Yes: true, Stdout: &buf})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Storydb reset.")

	// Verify tables are empty
	db, err := storydb.Open(dir)
	require.NoError(t, err)
	defer db.Close()

	scenes, err := db.QueryScenes()
	require.NoError(t, err)
	assert.Empty(t, scenes)

	chars, err := db.QuerySceneCharacters()
	require.NoError(t, err)
	assert.Empty(t, chars)

	facts, err := db.QueryFacts()
	require.NoError(t, err)
	assert.Empty(t, facts)
}

func TestReset_ConfirmYes(t *testing.T) {
	dir := seedStorydb(t)

	var buf bytes.Buffer
	err := Reset(ResetOptions{
		Stdin:  strings.NewReader("y\n"),
		Stdout: &buf,
	})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Continue?")
	assert.Contains(t, buf.String(), "Storydb reset.")

	db, err := storydb.Open(dir)
	require.NoError(t, err)
	defer db.Close()

	scenes, err := db.QueryScenes()
	require.NoError(t, err)
	assert.Empty(t, scenes)
}

func TestReset_ConfirmNo(t *testing.T) {
	dir := seedStorydb(t)

	var buf bytes.Buffer
	err := Reset(ResetOptions{
		Stdin:  strings.NewReader("n\n"),
		Stdout: &buf,
	})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Aborted.")

	// Data should still be there
	db, err := storydb.Open(dir)
	require.NoError(t, err)
	defer db.Close()

	scenes, err := db.QueryScenes()
	require.NoError(t, err)
	assert.Len(t, scenes, 1)
}

func TestReset_EmptyInputAborts(t *testing.T) {
	seedStorydb(t)

	var buf bytes.Buffer
	err := Reset(ResetOptions{
		Stdin:  strings.NewReader("\n"),
		Stdout: &buf,
	})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Aborted.")
}

func TestReset_EOFAborts(t *testing.T) {
	seedStorydb(t)

	var buf bytes.Buffer
	err := Reset(ResetOptions{
		Stdin:  strings.NewReader(""),
		Stdout: &buf,
	})
	require.NoError(t, err)
	assert.NotContains(t, buf.String(), "Storydb reset.")
}
