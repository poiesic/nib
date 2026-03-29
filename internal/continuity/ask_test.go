package continuity

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
