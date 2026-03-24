package character

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupProject(t *testing.T) string {
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
        - test-scene
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(bookYAML), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "characters"), 0755))
	return dir
}

func writeCharacter(t *testing.T, dir, slug, name string) {
	t.Helper()
	content := "---\nname: \"" + name + "\"\n---\n"
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "characters", slug+".yaml"),
		[]byte(content), 0644,
	))
}

// --- Add tests ---

func TestAdd_CreatesFile(t *testing.T) {
	dir := setupProject(t)

	path, err := Add("lance-thurgood")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "characters", "lance-thurgood.yaml"), path)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "name:")
	assert.Contains(t, string(content), "role:")
	assert.Contains(t, string(content), "relationships:")
}

func TestAdd_RejectsInvalidSlug(t *testing.T) {
	setupProject(t)

	tests := []string{
		"Bad-Name",
		"has spaces",
		"has_underscores",
		"",
		"-leading-dash",
	}
	for _, slug := range tests {
		_, err := Add(slug)
		assert.Error(t, err, "slug %q should be invalid", slug)
	}
}

func TestAdd_RejectsDuplicate(t *testing.T) {
	dir := setupProject(t)
	writeCharacter(t, dir, "lance", "Lance")

	_, err := Add("lance")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

// --- List tests ---

func TestList_Empty(t *testing.T) {
	setupProject(t)

	infos, err := List()
	require.NoError(t, err)
	assert.Empty(t, infos)
}

func TestList_ReturnsAlphabetical(t *testing.T) {
	dir := setupProject(t)
	writeCharacter(t, dir, "bo-dupuis", "Bo Dupuis")
	writeCharacter(t, dir, "lance-thurgood", "Lance Thurgood")
	writeCharacter(t, dir, "ashley-santos", "Ashley Santos")

	infos, err := List()
	require.NoError(t, err)
	require.Len(t, infos, 3)
	assert.Equal(t, "ashley-santos", infos[0].Slug)
	assert.Equal(t, "Ashley Santos", infos[0].Name)
	assert.Equal(t, "bo-dupuis", infos[1].Slug)
	assert.Equal(t, "lance-thurgood", infos[2].Slug)
}

// --- Remove tests ---

func TestRemove_DeletesFile(t *testing.T) {
	dir := setupProject(t)
	writeCharacter(t, dir, "lance", "Lance")

	err := Remove("lance")
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "characters", "lance.yaml"))
	assert.True(t, os.IsNotExist(err))
}

func TestRemove_NotFound(t *testing.T) {
	setupProject(t)

	err := Remove("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

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

func TestEdit_NotFound(t *testing.T) {
	setupProject(t)

	err := Edit(EditOptions{Slug: "nonexistent"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEdit_NoEditor(t *testing.T) {
	dir := setupProject(t)
	writeCharacter(t, dir, "lance", "Lance")
	clearEditorEnv(t)

	err := Edit(EditOptions{Slug: "lance"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEditorNotSet)
}

func TestEdit_LaunchesEditor(t *testing.T) {
	dir := setupProject(t)
	writeCharacter(t, dir, "lance", "Lance")
	clearEditorEnv(t)
	t.Setenv("EDITOR", "myeditor")

	var capturedName string
	var capturedArgs []string
	runner := func(name string, args ...string) *exec.Cmd {
		capturedName = name
		capturedArgs = args
		return exec.Command("true")
	}

	err := Edit(EditOptions{Slug: "lance", Runner: runner})
	require.NoError(t, err)
	assert.Equal(t, "myeditor", capturedName)
	require.Len(t, capturedArgs, 1)
	assert.Contains(t, capturedArgs[0], "lance.yaml")
}

// --- Talk tests ---

func TestTalk_CharacterNotFound(t *testing.T) {
	setupProject(t)

	err := Talk(TalkOptions{Slug: "nonexistent", Scene: "1.1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestTalk_InvalidSceneRef(t *testing.T) {
	dir := setupProject(t)
	writeCharacter(t, dir, "lance", "Lance Thurgood")

	err := Talk(TalkOptions{Slug: "lance", Scene: "abc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid scene reference")
}

func TestTalkSessionID_Deterministic(t *testing.T) {
	id1 := talkSessionID("lance", "3.2")
	id2 := talkSessionID("lance", "3.2")
	id3 := talkSessionID("lance", "4.1")

	assert.Equal(t, id1, id2, "same inputs should produce same ID")
	assert.NotEqual(t, id1, id3, "different inputs should produce different IDs")
	assert.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-5[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`, id1)
}

func TestBuildTalkPrompt_WithRecap(t *testing.T) {
	profile := "---\nname: Lance\ngoal: disappear\n---\n"
	recap := `{"chapters":[{"chapter":1}]}`

	prompt := buildTalkPrompt("Lance Thurgood", "3.2", profile, recap, nil)

	assert.Contains(t, prompt, "You are Lance Thurgood")
	assert.Contains(t, prompt, "scene 3.2")
	assert.Contains(t, prompt, "## Character Profile")
	assert.Contains(t, prompt, "goal: disappear")
	assert.Contains(t, prompt, "## Story So Far")
	assert.Contains(t, prompt, recap)
	assert.Contains(t, prompt, "Stay in character")
}

func TestBuildTalkPrompt_WithoutRecap(t *testing.T) {
	profile := "---\nname: Lance\n---\n"

	prompt := buildTalkPrompt("Lance Thurgood", "3.2", profile, "", fmt.Errorf("no data"))

	assert.Contains(t, prompt, "## Character Profile")
	assert.Contains(t, prompt, "No indexed continuity data available")
	assert.NotContains(t, prompt, "## Story So Far")
}

// --- FormatList tests ---

func TestFormatList_Empty(t *testing.T) {
	assert.Equal(t, "No characters\n", FormatList(nil))
}

func TestFormatList_TabSeparated(t *testing.T) {
	chars := []CharacterInfo{
		{Slug: "lance-thurgood", Name: "Lance Thurgood"},
		{Slug: "bo-dupuis", Name: "Bo Dupuis"},
	}
	output := FormatList(chars)
	assert.Equal(t, "lance-thurgood\tLance Thurgood\nbo-dupuis\tBo Dupuis\n", output)
}
