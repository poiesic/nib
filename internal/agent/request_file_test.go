package agent

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteReadRequestFile_roundTrip(t *testing.T) {
	req := Request{
		Operation: OpConverse,
		Prompt:    "hello world",
		Effort:    "high",
		Session:   &SessionOptions{ID: "test-session", Resume: true},
		Tools:     []string{"Read", "Write"},
	}

	path, err := WriteRequestFile(req)
	require.NoError(t, err)

	// File should exist after write
	_, err = os.Stat(path)
	require.NoError(t, err)

	got, err := ReadRequestFile(path)
	require.NoError(t, err)

	assert.Equal(t, req.Operation, got.Operation)
	assert.Equal(t, req.Prompt, got.Prompt)
	assert.Equal(t, req.Effort, got.Effort)
	assert.Equal(t, req.Session.ID, got.Session.ID)
	assert.True(t, got.Session.Resume)
	assert.Equal(t, req.Tools, got.Tools)

	// File should be removed after read
	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err))
}

func TestWriteRequestFile_validJSON(t *testing.T) {
	req := Request{
		Operation:   OpScaffold,
		Prompt:      "scaffold a project",
		ProjectName: "my-novel",
		Schema:      json.RawMessage(`{"type":"object"}`),
	}

	path, err := WriteRequestFile(req)
	require.NoError(t, err)
	defer os.Remove(path)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))
	assert.Equal(t, "scaffold", parsed["operation"])
	assert.Equal(t, "my-novel", parsed["project_name"])
}

func TestReadRequestFile_missingFile(t *testing.T) {
	_, err := ReadRequestFile("/nonexistent/path/file.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reading request file")
}

func TestReadRequestFile_invalidJSON(t *testing.T) {
	f, err := os.CreateTemp("", "nib-test-*.json")
	require.NoError(t, err)
	_, err = f.WriteString("not valid json")
	require.NoError(t, err)
	f.Close()

	_, err = ReadRequestFile(f.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing request file")

	// File should still be removed even on parse error
	_, err = os.Stat(f.Name())
	assert.True(t, os.IsNotExist(err))
}
