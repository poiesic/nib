package storydb

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openTestDB(t *testing.T) (*DB, string) {
	t.Helper()
	dir := t.TempDir()
	db, err := Open(dir)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db, dir
}

func TestOpen_CreatesCSVFiles(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	require.NoError(t, err)
	defer db.Close()

	for name := range tableSchemas {
		path := filepath.Join(dir, "storydb", name+".csv")
		_, err := os.Stat(path)
		require.NoError(t, err, "CSV file for %s should exist", name)
	}
}

func TestOpen_DoesNotOverwriteExisting(t *testing.T) {
	dir := t.TempDir()
	db1, err := Open(dir)
	require.NoError(t, err)

	// Insert a scene
	require.NoError(t, db1.UpsertScene(Scene{Scene: "test", POV: "lance"}))
	db1.Close()

	// Reopen — should not lose data
	db2, err := Open(dir)
	require.NoError(t, err)
	defer db2.Close()

	scenes, err := db2.QueryScenes()
	require.NoError(t, err)
	require.Len(t, scenes, 1)
	assert.Equal(t, "test", scenes[0].Scene)
}

func TestUpsertScene_Insert(t *testing.T) {
	db, _ := openTestDB(t)

	err := db.UpsertScene(Scene{
		Scene:    "intro",
		POV:      "lance",
		Location: "cafe",
		Summary:  "Opening scene",
		Checksum: "abc123",
	})
	require.NoError(t, err)

	scenes, err := db.QueryScenes()
	require.NoError(t, err)
	require.Len(t, scenes, 1)
	assert.Equal(t, "intro", scenes[0].Scene)
	assert.Equal(t, "lance", scenes[0].POV)
	assert.Equal(t, "abc123", scenes[0].Checksum)
}

func TestUpsertScene_Replace(t *testing.T) {
	db, _ := openTestDB(t)

	require.NoError(t, db.UpsertScene(Scene{Scene: "intro", POV: "lance", Summary: "v1"}))
	require.NoError(t, db.UpsertScene(Scene{Scene: "intro", POV: "bo", Summary: "v2"}))

	scenes, err := db.QueryScenes()
	require.NoError(t, err)
	require.Len(t, scenes, 1)
	assert.Equal(t, "bo", scenes[0].POV)
	assert.Equal(t, "v2", scenes[0].Summary)
}

func TestUpsertScene_MultipleScenes(t *testing.T) {
	db, _ := openTestDB(t)

	require.NoError(t, db.UpsertScene(Scene{Scene: "scene-a", POV: "lance"}))
	require.NoError(t, db.UpsertScene(Scene{Scene: "scene-b", POV: "bo"}))

	// Replace scene-a, scene-b should be untouched
	require.NoError(t, db.UpsertScene(Scene{Scene: "scene-a", POV: "eddie"}))

	scenes, err := db.QueryScenes()
	require.NoError(t, err)
	require.Len(t, scenes, 2)
}

func TestSceneChecksum(t *testing.T) {
	db, _ := openTestDB(t)

	// No scene yet
	cs, err := db.SceneChecksum("intro")
	require.NoError(t, err)
	assert.Empty(t, cs)

	// Insert scene with checksum
	require.NoError(t, db.UpsertScene(Scene{Scene: "intro", Checksum: "sha256abc"}))

	cs, err = db.SceneChecksum("intro")
	require.NoError(t, err)
	assert.Equal(t, "sha256abc", cs)
}

func TestInsertFacts_GeneratesULIDs(t *testing.T) {
	db, _ := openTestDB(t)

	facts := []Fact{
		{Scene: "intro", Category: "event", Summary: "Something happened"},
		{Scene: "intro", Category: "description", Summary: "A place"},
	}
	require.NoError(t, db.InsertFacts(facts))

	got, err := db.QueryFacts()
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.NotEmpty(t, got[0].ID)
	assert.NotEmpty(t, got[1].ID)
	assert.NotEqual(t, got[0].ID, got[1].ID)
	assert.Len(t, got[0].ID, 26) // ULID length
}

func TestInsertFacts_PreservesExplicitID(t *testing.T) {
	db, _ := openTestDB(t)

	facts := []Fact{
		{ID: "custom-id", Scene: "intro", Summary: "test"},
	}
	require.NoError(t, db.InsertFacts(facts))

	got, err := db.QueryFacts()
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "custom-id", got[0].ID)
}

func TestInsertFacts_Empty(t *testing.T) {
	db, _ := openTestDB(t)
	require.NoError(t, db.InsertFacts(nil))
	require.NoError(t, db.InsertFacts([]Fact{}))
}

func TestDeleteByScene(t *testing.T) {
	db, _ := openTestDB(t)

	// Insert facts from two scenes
	require.NoError(t, db.InsertFacts([]Fact{
		{Scene: "scene-a", Summary: "fact a1"},
		{Scene: "scene-a", Summary: "fact a2"},
		{Scene: "scene-b", Summary: "fact b1"},
	}))

	// Delete scene-a facts
	require.NoError(t, db.DeleteByScene("facts", "scene-a"))

	got, err := db.QueryFacts()
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "scene-b", got[0].Scene)
}

func TestDeleteByScene_NoMatch(t *testing.T) {
	db, _ := openTestDB(t)

	require.NoError(t, db.InsertFacts([]Fact{
		{Scene: "scene-a", Summary: "test"},
	}))

	require.NoError(t, db.DeleteByScene("facts", "nonexistent"))

	got, err := db.QueryFacts()
	require.NoError(t, err)
	require.Len(t, got, 1)
}

func TestInsertSceneCharacters(t *testing.T) {
	db, _ := openTestDB(t)

	chars := []SceneCharacter{
		{Scene: "intro", Character: "lance", Role: "pov"},
		{Scene: "intro", Character: "bo", Role: "present"},
	}
	require.NoError(t, db.InsertSceneCharacters(chars))

	got, err := db.QuerySceneCharacters()
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "lance", got[0].Character)
	assert.Equal(t, "bo", got[1].Character)
}

func TestInsertLocations_GeneratesULIDs(t *testing.T) {
	db, _ := openTestDB(t)

	locs := []Location{
		{Name: "The Cafe", Type: "public", FirstScene: "intro"},
	}
	require.NoError(t, db.InsertLocations(locs))

	got, err := db.QueryLocations()
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.NotEmpty(t, got[0].ID)
	assert.Len(t, got[0].ID, 26)
	assert.Equal(t, "The Cafe", got[0].Name)
}

func TestInsertLocations_PreservesExplicitID(t *testing.T) {
	db, _ := openTestDB(t)

	locs := []Location{
		{ID: "cafe", Name: "The Cafe", Type: "public"},
	}
	require.NoError(t, db.InsertLocations(locs))

	got, err := db.QueryLocations()
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "cafe", got[0].ID)
}

func TestNewID_IsULID(t *testing.T) {
	id := NewID()
	assert.Len(t, id, 26)

	// Generate a second — should be different
	id2 := NewID()
	assert.NotEqual(t, id, id2)
}

func TestQueryScenesBySlugs(t *testing.T) {
	db, _ := openTestDB(t)

	require.NoError(t, db.UpsertScene(Scene{Scene: "scene-a", POV: "lance", Summary: "A"}))
	require.NoError(t, db.UpsertScene(Scene{Scene: "scene-b", POV: "bo", Summary: "B"}))
	require.NoError(t, db.UpsertScene(Scene{Scene: "scene-c", POV: "eddie", Summary: "C"}))

	got, err := db.QueryScenesBySlugs([]string{"scene-a", "scene-c"})
	require.NoError(t, err)
	require.Len(t, got, 2)

	slugs := []string{got[0].Scene, got[1].Scene}
	assert.Contains(t, slugs, "scene-a")
	assert.Contains(t, slugs, "scene-c")
}

func TestQueryScenesBySlugs_Empty(t *testing.T) {
	db, _ := openTestDB(t)

	got, err := db.QueryScenesBySlugs(nil)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestQueryScenesBySlugs_NoMatch(t *testing.T) {
	db, _ := openTestDB(t)

	require.NoError(t, db.UpsertScene(Scene{Scene: "scene-a", POV: "lance"}))

	got, err := db.QueryScenesBySlugs([]string{"nonexistent"})
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestQueryFactsBySlugs(t *testing.T) {
	db, _ := openTestDB(t)

	require.NoError(t, db.InsertFacts([]Fact{
		{ID: "f1", Scene: "scene-a", Category: "event", Summary: "A happened"},
		{ID: "f2", Scene: "scene-b", Category: "state", Summary: "B state"},
		{ID: "f3", Scene: "scene-a", Category: "description", Summary: "A desc"},
		{ID: "f4", Scene: "scene-c", Category: "event", Summary: "C happened"},
	}))

	got, err := db.QueryFactsBySlugs([]string{"scene-a"})
	require.NoError(t, err)
	require.Len(t, got, 2)
	for _, f := range got {
		assert.Equal(t, "scene-a", f.Scene)
	}
}

func TestQueryFactsBySlugs_Empty(t *testing.T) {
	db, _ := openTestDB(t)

	got, err := db.QueryFactsBySlugs(nil)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestQueryCharactersBySlugs(t *testing.T) {
	db, _ := openTestDB(t)

	require.NoError(t, db.InsertSceneCharacters([]SceneCharacter{
		{Scene: "scene-a", Character: "lance", Role: "pov"},
		{Scene: "scene-a", Character: "bo", Role: "present"},
		{Scene: "scene-b", Character: "eddie", Role: "pov"},
	}))

	got, err := db.QueryCharactersBySlugs([]string{"scene-a"})
	require.NoError(t, err)
	require.Len(t, got, 2)
	for _, c := range got {
		assert.Equal(t, "scene-a", c.Scene)
	}
}

func TestQueryCharactersBySlugs_Empty(t *testing.T) {
	db, _ := openTestDB(t)

	got, err := db.QueryCharactersBySlugs(nil)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestQuerySceneSlugsForCharacters(t *testing.T) {
	db, _ := openTestDB(t)

	require.NoError(t, db.InsertSceneCharacters([]SceneCharacter{
		{Scene: "scene-a", Character: "lance", Role: "pov"},
		{Scene: "scene-a", Character: "bo", Role: "present"},
		{Scene: "scene-b", Character: "eddie", Role: "pov"},
		{Scene: "scene-b", Character: "lance", Role: "mentioned"},
		{Scene: "scene-c", Character: "bo", Role: "pov"},
	}))

	// Single character
	slugs, err := db.QuerySceneSlugsForCharacters([]string{"lance"})
	require.NoError(t, err)
	assert.Len(t, slugs, 2)
	assert.Contains(t, slugs, "scene-a")
	assert.Contains(t, slugs, "scene-b")

	// Multiple characters (union)
	slugs, err = db.QuerySceneSlugsForCharacters([]string{"eddie", "bo"})
	require.NoError(t, err)
	assert.Len(t, slugs, 3)
	assert.Contains(t, slugs, "scene-a")
	assert.Contains(t, slugs, "scene-b")
	assert.Contains(t, slugs, "scene-c")

	// No match
	slugs, err = db.QuerySceneSlugsForCharacters([]string{"nobody"})
	require.NoError(t, err)
	assert.Empty(t, slugs)

	// Empty input
	slugs, err = db.QuerySceneSlugsForCharacters(nil)
	require.NoError(t, err)
	assert.Nil(t, slugs)
}

func TestQuerySceneSlugsForCharactersWithRoles(t *testing.T) {
	db, _ := openTestDB(t)

	require.NoError(t, db.InsertSceneCharacters([]SceneCharacter{
		{Scene: "scene-a", Character: "lance", Role: "pov"},
		{Scene: "scene-a", Character: "bo", Role: "present"},
		{Scene: "scene-b", Character: "lance", Role: "mentioned"},
		{Scene: "scene-c", Character: "bo", Role: "pov"},
	}))

	// pov+present only
	slugs, err := db.QuerySceneSlugsForCharactersWithRoles([]string{"lance"}, []string{"pov", "present"})
	require.NoError(t, err)
	assert.Equal(t, []string{"scene-a"}, slugs)

	// mentioned excluded
	assert.NotContains(t, slugs, "scene-b")

	// all roles
	slugs, err = db.QuerySceneSlugsForCharactersWithRoles([]string{"lance"}, []string{"pov", "present", "mentioned"})
	require.NoError(t, err)
	assert.Len(t, slugs, 2)
	assert.Contains(t, slugs, "scene-a")
	assert.Contains(t, slugs, "scene-b")

	// empty characters
	slugs, err = db.QuerySceneSlugsForCharactersWithRoles(nil, []string{"pov"})
	require.NoError(t, err)
	assert.Nil(t, slugs)

	// empty roles
	slugs, err = db.QuerySceneSlugsForCharactersWithRoles([]string{"lance"}, nil)
	require.NoError(t, err)
	assert.Nil(t, slugs)
}

func TestQueryDistinctCharacters_AllRoles(t *testing.T) {
	db, _ := openTestDB(t)

	require.NoError(t, db.InsertSceneCharacters([]SceneCharacter{
		{Scene: "scene-a", Character: "zara", Role: "pov"},
		{Scene: "scene-a", Character: "mike", Role: "present"},
		{Scene: "scene-b", Character: "mike", Role: "pov"},
		{Scene: "scene-b", Character: "alice", Role: "mentioned"},
		{Scene: "scene-c", Character: "zara", Role: "present"},
	}))

	characters, err := db.QueryDistinctCharacters(nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"alice", "mike", "zara"}, characters)
}

func TestQueryDistinctCharacters_FilterByRole(t *testing.T) {
	db, _ := openTestDB(t)

	require.NoError(t, db.InsertSceneCharacters([]SceneCharacter{
		{Scene: "scene-a", Character: "zara", Role: "pov"},
		{Scene: "scene-a", Character: "mike", Role: "present"},
		{Scene: "scene-b", Character: "mike", Role: "pov"},
		{Scene: "scene-b", Character: "alice", Role: "mentioned"},
		{Scene: "scene-c", Character: "zara", Role: "present"},
	}))

	// pov+present only, excludes alice (mentioned only)
	characters, err := db.QueryDistinctCharacters([]string{"pov", "present"})
	require.NoError(t, err)
	assert.Equal(t, []string{"mike", "zara"}, characters)

	// pov only
	characters, err = db.QueryDistinctCharacters([]string{"pov"})
	require.NoError(t, err)
	assert.Equal(t, []string{"mike", "zara"}, characters)

	// mentioned only
	characters, err = db.QueryDistinctCharacters([]string{"mentioned"})
	require.NoError(t, err)
	assert.Equal(t, []string{"alice"}, characters)
}

func TestQueryDistinctCharacters_Empty(t *testing.T) {
	db, _ := openTestDB(t)

	characters, err := db.QueryDistinctCharacters(nil)
	require.NoError(t, err)
	assert.Nil(t, characters)
}

func TestRenameScene_UpdatesAllTables(t *testing.T) {
	db, _ := openTestDB(t)

	// Seed data across all three tables
	require.NoError(t, db.UpsertScene(Scene{Scene: "old-name", POV: "lance", Summary: "test"}))
	require.NoError(t, db.InsertFacts([]Fact{
		{ID: "f1", Scene: "old-name", Category: "event", Summary: "something"},
		{ID: "f2", Scene: "other-scene", Category: "event", Summary: "unrelated"},
	}))
	require.NoError(t, db.InsertSceneCharacters([]SceneCharacter{
		{Scene: "old-name", Character: "lance", Role: "pov"},
		{Scene: "other-scene", Character: "bo", Role: "pov"},
	}))

	require.NoError(t, db.RenameScene("old-name", "new-name"))

	// scenes table: old-name -> new-name
	scenes, err := db.QueryScenesBySlugs([]string{"new-name"})
	require.NoError(t, err)
	require.Len(t, scenes, 1)
	assert.Equal(t, "new-name", scenes[0].Scene)
	assert.Equal(t, "lance", scenes[0].POV)

	oldScenes, err := db.QueryScenesBySlugs([]string{"old-name"})
	require.NoError(t, err)
	assert.Empty(t, oldScenes)

	// facts table: old-name -> new-name, other-scene untouched
	facts, err := db.QueryFactsBySlugs([]string{"new-name"})
	require.NoError(t, err)
	require.Len(t, facts, 1)
	assert.Equal(t, "new-name", facts[0].Scene)

	otherFacts, err := db.QueryFactsBySlugs([]string{"other-scene"})
	require.NoError(t, err)
	require.Len(t, otherFacts, 1)

	// scene_characters table: old-name -> new-name, other-scene untouched
	chars, err := db.QueryCharactersBySlugs([]string{"new-name"})
	require.NoError(t, err)
	require.Len(t, chars, 1)
	assert.Equal(t, "new-name", chars[0].Scene)

	otherChars, err := db.QueryCharactersBySlugs([]string{"other-scene"})
	require.NoError(t, err)
	require.Len(t, otherChars, 1)
}

func TestReset_ClearsAllTables(t *testing.T) {
	db, _ := openTestDB(t)

	// Seed data across multiple tables
	require.NoError(t, db.UpsertScene(Scene{Scene: "scene-a", POV: "lance", Summary: "test"}))
	require.NoError(t, db.InsertFacts([]Fact{
		{Scene: "scene-a", Category: "event", Summary: "something happened"},
	}))
	require.NoError(t, db.InsertSceneCharacters([]SceneCharacter{
		{Scene: "scene-a", Character: "lance", Role: "pov"},
	}))
	require.NoError(t, db.InsertLocations([]Location{
		{Name: "office", Type: "workplace"},
	}))

	require.NoError(t, db.Reset())

	// All tables should be empty
	scenes, err := db.QueryScenes()
	require.NoError(t, err)
	assert.Empty(t, scenes)

	facts, err := db.QueryFacts()
	require.NoError(t, err)
	assert.Empty(t, facts)

	chars, err := db.QuerySceneCharacters()
	require.NoError(t, err)
	assert.Empty(t, chars)

	locs, err := db.QueryLocations()
	require.NoError(t, err)
	assert.Empty(t, locs)

	// Tables should still be usable after reset
	require.NoError(t, db.UpsertScene(Scene{Scene: "scene-b", POV: "bo", Summary: "new"}))
	scenes, err = db.QueryScenes()
	require.NoError(t, err)
	require.Len(t, scenes, 1)
	assert.Equal(t, "scene-b", scenes[0].Scene)
}

func TestRenameScene_NoMatchIsNoOp(t *testing.T) {
	db, _ := openTestDB(t)

	require.NoError(t, db.UpsertScene(Scene{Scene: "existing", POV: "lance"}))

	require.NoError(t, db.RenameScene("nonexistent", "new-name"))

	// existing scene untouched
	scenes, err := db.QueryScenes()
	require.NoError(t, err)
	require.Len(t, scenes, 1)
	assert.Equal(t, "existing", scenes[0].Scene)
}
