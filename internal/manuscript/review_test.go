package manuscript

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCritique_EmptyRange(t *testing.T) {
	err := Critique(CritiqueOptions{Range: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "range is required")
}

func TestProof_EmptyRange(t *testing.T) {
	err := Proof(ProofOptions{Range: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "range is required")
}

func TestIsWholeChapters(t *testing.T) {
	tests := []struct {
		name     string
		spec     RangeSpec
		expected bool
	}{
		{"single chapter", RangeSpec{Kind: "list", Refs: []SceneRef{{Chapter: 1}}}, true},
		{"chapter range", RangeSpec{Kind: "range", Refs: []SceneRef{{Chapter: 1}, {Chapter: 3}}}, true},
		{"dotted ref", RangeSpec{Kind: "list", Refs: []SceneRef{{Chapter: 1, Position: 2}}}, false},
		{"mixed", RangeSpec{Kind: "list", Refs: []SceneRef{{Chapter: 1}, {Chapter: 2, Position: 1}}}, false},
		{"empty", RangeSpec{Kind: "list", Refs: []SceneRef{}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isWholeChapters(tt.spec))
		})
	}
}
