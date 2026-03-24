package continuity

import (
	"bufio"
	"io"
	"strings"
	"testing"

	"github.com/poiesic/nib/internal/storydb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBufReader(r io.Reader) *bufio.Reader {
	return bufio.NewReader(r)
}

func TestReviewProposals_AcceptAll(t *testing.T) {
	result := &ExtractionResult{
		Scene: storydb.Scene{
			Scene:     "test-scene",
			POV:       "lance",
			SceneType: "regular",
			Location:  "cafe",
			Summary:   "A scene",
		},
		Facts: []storydb.Fact{
			{ID: "f00001", Category: "event", Summary: "Something", Detail: "Details", SourceText: "quote"},
		},
		Characters: []storydb.SceneCharacter{
			{Character: "lance", Role: "pov"},
		},
		Locations: []storydb.Location{
			{ID: "cafe", Name: "Cafe", Type: "public", Description: "A cafe"},
		},
	}

	input := "a\na\na\na\n"
	stdout := &strings.Builder{}

	accepted, err := reviewProposals(result, strings.NewReader(input), stdout)
	require.NoError(t, err)

	assert.Equal(t, "test-scene", accepted.Scene.Scene)
	assert.Len(t, accepted.Facts, 1)
	assert.Len(t, accepted.Characters, 1)
	assert.Len(t, accepted.Locations, 1)
}

func TestReviewProposals_RejectFact(t *testing.T) {
	result := &ExtractionResult{
		Scene: storydb.Scene{Scene: "test", POV: "lance", SceneType: "regular"},
		Facts: []storydb.Fact{
			{ID: "f00001", Summary: "Keep"},
			{ID: "f00002", Summary: "Reject"},
			{ID: "f00003", Summary: "Also keep"},
		},
	}

	input := "a\na\nr\na\n"
	stdout := &strings.Builder{}

	accepted, err := reviewProposals(result, strings.NewReader(input), stdout)
	require.NoError(t, err)

	require.Len(t, accepted.Facts, 2)
	assert.Equal(t, "f00001", accepted.Facts[0].ID)
	assert.Equal(t, "f00003", accepted.Facts[1].ID)
}

func TestReviewProposals_RejectScene_ReturnsError(t *testing.T) {
	result := &ExtractionResult{
		Scene: storydb.Scene{Scene: "test", POV: "lance"},
	}

	input := "r\n"
	stdout := &strings.Builder{}

	_, err := reviewProposals(result, strings.NewReader(input), stdout)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scene metadata rejected")
}

func TestReviewProposals_EOFTreatedAsAccept(t *testing.T) {
	result := &ExtractionResult{
		Scene: storydb.Scene{Scene: "test", POV: "lance", SceneType: "regular"},
		Facts: []storydb.Fact{
			{ID: "f00001", Summary: "auto-accept"},
		},
	}

	// Empty input — EOF on first read
	stdout := &strings.Builder{}

	accepted, err := reviewProposals(result, strings.NewReader(""), stdout)
	require.NoError(t, err)

	assert.Equal(t, "test", accepted.Scene.Scene)
	assert.Len(t, accepted.Facts, 1)
}

func TestReviewProposals_DisplaysContent(t *testing.T) {
	result := &ExtractionResult{
		Scene: storydb.Scene{
			Scene:     "lunch-scene",
			POV:       "lance",
			SceneType: "regular",
			Location:  "wellness-center",
			Date:      "2025-03-14",
			Time:      "12:00",
			Summary:   "Lance and Bo have lunch",
		},
		Facts: []storydb.Fact{
			{ID: "f00001", Category: "event", Summary: "Lunch meeting", Detail: "They meet for lunch", SourceText: "sat down across from Bo"},
		},
		Characters: []storydb.SceneCharacter{
			{Character: "lance", Role: "pov"},
			{Character: "bo", Role: "present"},
		},
		Locations: []storydb.Location{
			{ID: "wellness-center", Name: "Wellness Center", Type: "workplace", Description: "A medical facility"},
		},
	}

	input := "a\na\na\na\na\n"
	stdout := &strings.Builder{}

	_, err := reviewProposals(result, strings.NewReader(input), stdout)
	require.NoError(t, err)

	output := stdout.String()

	// Scene metadata displayed
	assert.Contains(t, output, "Scene: lunch-scene")
	assert.Contains(t, output, "POV: lance")
	assert.Contains(t, output, "Location: wellness-center")
	assert.Contains(t, output, "Date: 2025-03-14")
	assert.Contains(t, output, "Summary: Lance and Bo have lunch")

	// Facts displayed
	assert.Contains(t, output, "Facts (1)")
	assert.Contains(t, output, "event: Lunch meeting")

	// Characters displayed
	assert.Contains(t, output, "Characters (2)")
	assert.Contains(t, output, "lance (pov)")
	assert.Contains(t, output, "bo (present)")

	// Locations displayed
	assert.Contains(t, output, "Locations (1)")
	assert.Contains(t, output, "Wellness Center")
}

func TestPromptAction_Variations(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"a\n", "a"},
		{"accept\n", "a"},
		{"r\n", "r"},
		{"reject\n", "r"},
		{"e\n", "e"},
		{"edit\n", "e"},
		{"A\n", "A"},
		{"R\n", "r"},
		{"E\n", "e"},
		{"unknown\n", "a"}, // default to accept
		{"  a  \n", "a"},   // trimmed
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			stdout := &strings.Builder{}
			action, err := promptAction(newBufReader(reader), stdout)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, action)
		})
	}
}

func TestReviewProposals_RejectCharacters(t *testing.T) {
	result := &ExtractionResult{
		Scene: storydb.Scene{Scene: "test", POV: "lance", SceneType: "regular"},
		Characters: []storydb.SceneCharacter{
			{Character: "lance", Role: "pov"},
			{Character: "bo", Role: "present"},
		},
	}

	input := "a\na\nr\n" // accept scene, accept lance, reject bo
	stdout := &strings.Builder{}

	accepted, err := reviewProposals(result, strings.NewReader(input), stdout)
	require.NoError(t, err)

	require.Len(t, accepted.Characters, 1)
	assert.Equal(t, "lance", accepted.Characters[0].Character)
}

func TestReviewProposals_AcceptAllFromScene(t *testing.T) {
	result := &ExtractionResult{
		Scene: storydb.Scene{Scene: "test", POV: "lance", SceneType: "regular"},
		Facts: []storydb.Fact{
			{Category: "event", Summary: "fact1"},
			{Category: "event", Summary: "fact2"},
		},
		Characters: []storydb.SceneCharacter{
			{Character: "lance", Role: "pov"},
			{Character: "bo", Role: "present"},
		},
		Locations: []storydb.Location{
			{ID: "cafe", Name: "Cafe"},
		},
	}

	// Only one input: 'A' at the scene prompt, rest auto-accepted
	input := "A\n"
	stdout := &strings.Builder{}

	accepted, err := reviewProposals(result, strings.NewReader(input), stdout)
	require.NoError(t, err)

	assert.Equal(t, "lance", accepted.Scene.POV)
	assert.Len(t, accepted.Facts, 2)
	assert.Len(t, accepted.Characters, 2)
	assert.Len(t, accepted.Locations, 1)
	assert.Contains(t, stdout.String(), "[auto-accepted]")
}

func TestReviewProposals_AcceptAllMidway(t *testing.T) {
	result := &ExtractionResult{
		Scene: storydb.Scene{Scene: "test", POV: "lance", SceneType: "regular"},
		Facts: []storydb.Fact{
			{Category: "event", Summary: "fact1"},
			{Category: "event", Summary: "fact2"},
			{Category: "event", Summary: "fact3"},
		},
	}

	// Accept scene, reject first fact, accept-all on second fact, third auto-accepted
	input := "a\nr\nA\n"
	stdout := &strings.Builder{}

	accepted, err := reviewProposals(result, strings.NewReader(input), stdout)
	require.NoError(t, err)

	require.Len(t, accepted.Facts, 2)
	assert.Equal(t, "fact2", accepted.Facts[0].Summary)
	assert.Equal(t, "fact3", accepted.Facts[1].Summary)
}

func TestReviewProposals_RejectLocations(t *testing.T) {
	result := &ExtractionResult{
		Scene: storydb.Scene{Scene: "test", POV: "lance", SceneType: "regular"},
		Locations: []storydb.Location{
			{ID: "cafe", Name: "Cafe"},
			{ID: "park", Name: "Park"},
		},
	}

	input := "a\nr\na\n" // accept scene, reject cafe, accept park
	stdout := &strings.Builder{}

	accepted, err := reviewProposals(result, strings.NewReader(input), stdout)
	require.NoError(t, err)

	require.Len(t, accepted.Locations, 1)
	assert.Equal(t, "park", accepted.Locations[0].ID)
}
