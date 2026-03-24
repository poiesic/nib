package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_MissingFile(t *testing.T) {
	dir := t.TempDir()

	s, err := Load(dir)
	require.NoError(t, err)
	assert.Nil(t, s.Focus)
}

func TestSave_Load_RoundTrip(t *testing.T) {
	dir := t.TempDir()

	original := &State{
		Focus: &Focus{Chapter: 3, Scene: "opening"},
	}
	require.NoError(t, Save(dir, original))

	loaded, err := Load(dir)
	require.NoError(t, err)
	require.NotNil(t, loaded.Focus)
	assert.Equal(t, 3, loaded.Focus.Chapter)
	assert.Equal(t, "opening", loaded.Focus.Scene)
}

func TestSave_NilFocus(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, Save(dir, &State{}))

	loaded, err := Load(dir)
	require.NoError(t, err)
	assert.Nil(t, loaded.Focus)
}

func TestSave_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, Save(dir, &State{Focus: &Focus{Chapter: 1, Scene: "test"}}))

	_, err := os.Stat(filepath.Join(dir, ".nib"))
	assert.NoError(t, err, ".nib directory should exist")
}

func TestSave_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, Save(dir, &State{Focus: &Focus{Chapter: 1, Scene: "first"}}))
	require.NoError(t, Save(dir, &State{Focus: &Focus{Chapter: 2, Scene: "second"}}))

	loaded, err := Load(dir)
	require.NoError(t, err)
	require.NotNil(t, loaded.Focus)
	assert.Equal(t, 2, loaded.Focus.Chapter)
	assert.Equal(t, "second", loaded.Focus.Scene)
}
