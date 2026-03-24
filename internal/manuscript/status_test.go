package manuscript

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCountWords(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/test.md"
	require.NoError(t, writeFile(path, "one two three four five"))

	count, err := countWords(path)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestCountWords_Empty(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/empty.md"
	require.NoError(t, writeFile(path, ""))

	count, err := countWords(path)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestCountWords_FileNotFound(t *testing.T) {
	_, err := countWords("/nonexistent/file.md")
	require.Error(t, err)
}

func TestFindUnassignedScenes(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, writeFile(dir+"/assigned.md", "content"))
	require.NoError(t, writeFile(dir+"/orphan.md", "content"))
	require.NoError(t, writeFile(dir+"/notes.txt", "not a scene"))

	assigned := map[string]bool{"assigned.md": true}
	unassigned, err := findUnassignedScenes(dir, assigned)
	require.NoError(t, err)

	assert.Equal(t, []string{"orphan.md"}, unassigned)
}

func TestFindUnassignedScenes_AllAssigned(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, writeFile(dir+"/scene1.md", "content"))
	require.NoError(t, writeFile(dir+"/scene2.md", "content"))

	assigned := map[string]bool{"scene1.md": true, "scene2.md": true}
	unassigned, err := findUnassignedScenes(dir, assigned)
	require.NoError(t, err)

	assert.Empty(t, unassigned)
}

func TestFindUnassignedScenes_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	assigned := map[string]bool{}
	unassigned, err := findUnassignedScenes(dir, assigned)
	require.NoError(t, err)

	assert.Empty(t, unassigned)
}

func TestFormatStatus(t *testing.T) {
	s := &Status{
		Scenes:     82,
		Chapters:   20,
		Interludes: 10,
		WordCount:  91420,
		EstPages:   365,
	}

	output := FormatStatus(s)
	assert.Contains(t, output, "Scenes: 82")
	assert.Contains(t, output, "Chapters: 20 + 10 interludes")
	assert.Contains(t, output, "91,420")
	assert.Contains(t, output, "Est. pages: 365")
}

func TestFormatStatus_NoInterludes(t *testing.T) {
	s := &Status{
		Scenes:    5,
		Chapters:  5,
		WordCount: 500,
		EstPages:  2,
	}

	output := FormatStatus(s)
	assert.Contains(t, output, "Chapters: 5\n")
	assert.NotContains(t, output, "interludes")
}

func TestFormatStatus_WithUnassigned(t *testing.T) {
	s := &Status{
		Scenes:           3,
		Chapters:         3,
		WordCount:        300,
		EstPages:         1,
		UnassignedScenes: []string{"orphan1.md", "orphan2.md"},
	}

	output := FormatStatus(s)
	assert.Contains(t, output, "Unassigned scenes: 2")
	assert.Contains(t, output, "orphan1.md")
	assert.Contains(t, output, "orphan2.md")
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{42, "42"},
		{999, "999"},
		{1000, "1,000"},
		{91420, "91,420"},
		{1234567, "1,234,567"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, formatNumber(tt.input), "formatNumber(%d)", tt.input)
	}
}
