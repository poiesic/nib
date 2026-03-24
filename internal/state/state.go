package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const stateDir = ".nib"
const stateFile = "state.json"

// Focus records which scene the user is currently working on.
type Focus struct {
	Chapter int    `json:"chapter"` // 1-based chapter index
	Scene   string `json:"scene"`   // scene slug
}

// State holds persistent project state stored in .nib/state.json.
type State struct {
	Focus *Focus `json:"focus,omitempty"`
}

// Load reads state from .nib/state.json under projectRoot.
// Returns an empty State if the file does not exist.
func Load(projectRoot string) (*State, error) {
	path := filepath.Join(projectRoot, stateDir, stateFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &State{}, nil
		}
		return nil, fmt.Errorf("reading state: %w", err)
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing state: %w", err)
	}
	return &s, nil
}

// Save writes state to .nib/state.json under projectRoot,
// creating the .nib/ directory if needed.
func Save(projectRoot string, s *State) error {
	dir := filepath.Join(projectRoot, stateDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding state: %w", err)
	}
	path := filepath.Join(dir, stateFile)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing state: %w", err)
	}
	return nil
}
