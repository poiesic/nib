package continuity

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAskPrompt_basic(t *testing.T) {
	prompt := buildAskPrompt("Who is Lance?", "")

	assert.Contains(t, prompt, "Who is Lance?")
	assert.Contains(t, prompt, "nib ct recap")
	assert.Contains(t, prompt, "nib ct characters")
	assert.Contains(t, prompt, "nib ct chapters")
	assert.NotContains(t, prompt, "## Scope")
}

func TestBuildAskPrompt_withRange(t *testing.T) {
	prompt := buildAskPrompt("Who is Lance?", "1-5")

	assert.Contains(t, prompt, "Who is Lance?")
	assert.Contains(t, prompt, "## Scope")
	assert.Contains(t, prompt, "1-5")
}

func TestAsk_emptyQuestion(t *testing.T) {
	var buf bytes.Buffer
	err := Ask(AskOptions{
		Question: "",
		Stdout:   &buf,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "question is required")
}

func TestAsk_whitespaceQuestion(t *testing.T) {
	var buf bytes.Buffer
	err := Ask(AskOptions{
		Question: "   ",
		Stdout:   &buf,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "question is required")
}

func TestAsk_noProject(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	var buf bytes.Buffer
	err := Ask(AskOptions{
		Question: "Who is Lance?",
		Stdout:   &buf,
	})
	// Fails because there's no book.yaml
	require.Error(t, err)
}
