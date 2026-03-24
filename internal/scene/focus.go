package scene

import (
	"fmt"

	"github.com/poiesic/binder"
	"github.com/poiesic/nib/internal/state"
)

// FocusInfo holds resolved focus details for display.
type FocusInfo struct {
	Chapter  int    // 1-based chapter index
	Position int    // 1-based scene position within chapter
	Slug     string // scene slug
}

// SetFocus validates the chapter/position, resolves the slug, and persists focus.
// If position is 0, focuses on the chapter only (no specific scene).
func SetFocus(projectRoot string, book *binder.Book, chapter, position int) (*FocusInfo, error) {
	idx := chapter - 1
	if idx < 0 || idx >= len(book.Chapters) {
		return nil, fmt.Errorf("chapter %d is out of range (1-%d)", chapter, len(book.Chapters))
	}

	var slug string
	if position > 0 {
		var err error
		slug, err = ResolveSlug(book, chapter, position)
		if err != nil {
			return nil, err
		}
	}

	s := &state.State{
		Focus: &state.Focus{
			Chapter: chapter,
			Scene:   slug,
		},
	}
	if err := state.Save(projectRoot, s); err != nil {
		return nil, err
	}

	return &FocusInfo{
		Chapter:  chapter,
		Position: position,
		Slug:     slug,
	}, nil
}

// ClearFocus removes the current focus.
func ClearFocus(projectRoot string) error {
	return state.Save(projectRoot, &state.State{})
}

// GetFocus reads the current focus. Returns nil if no focus is set.
// Returns an error if the focused scene no longer exists in the book.
func GetFocus(projectRoot string, book *binder.Book) (*FocusInfo, error) {
	s, err := state.Load(projectRoot)
	if err != nil {
		return nil, err
	}
	if s.Focus == nil {
		return nil, nil
	}

	chapter := s.Focus.Chapter
	idx := chapter - 1
	if idx < 0 || idx >= len(book.Chapters) {
		return nil, fmt.Errorf("focused chapter %d no longer exists (book has %d chapters)", chapter, len(book.Chapters))
	}

	if s.Focus.Scene == "" {
		// Chapter-only focus
		return &FocusInfo{Chapter: chapter}, nil
	}

	// Find the scene's current position
	ch := book.Chapters[idx]
	for i, slug := range ch.Scenes {
		if slug == s.Focus.Scene {
			return &FocusInfo{
				Chapter:  chapter,
				Position: i + 1,
				Slug:     slug,
			}, nil
		}
	}

	return nil, fmt.Errorf("focused scene %q no longer exists in chapter %d", s.Focus.Scene, chapter)
}

// resolveSlugOrFocus returns the provided slug if non-empty, otherwise
// loads focus and returns the focused scene slug. Errors if no focus is set
// or the focus has no scene.
func resolveSlugOrFocus(projectRoot string, book *binder.Book, slug string) (string, error) {
	if slug != "" {
		return slug, nil
	}
	focus, err := GetFocus(projectRoot, book)
	if err != nil {
		return "", err
	}
	if focus == nil {
		return "", fmt.Errorf("no scene specified and no focus set; use nib scene focus to set one")
	}
	if focus.Slug == "" {
		return "", fmt.Errorf("focus is set to chapter %d but no specific scene; specify a slug or use nib scene focus <chapter.scene>", focus.Chapter)
	}
	return focus.Slug, nil
}
