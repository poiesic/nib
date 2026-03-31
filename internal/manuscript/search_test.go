package manuscript

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearch_EmptyRange(t *testing.T) {
	err := Search(SearchOptions{Range: "", Query: "something"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "range is required")
}

func TestSearch_EmptyQuery(t *testing.T) {
	err := Search(SearchOptions{Range: "1-3", Query: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "search query is required")
}

func TestSearch_WhitespaceOnly(t *testing.T) {
	err := Search(SearchOptions{Range: "  ", Query: "something"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "range is required")

	err = Search(SearchOptions{Range: "1-3", Query: "   "})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "search query is required")
}
