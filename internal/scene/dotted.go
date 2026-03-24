package scene

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/poiesic/binder"
)

// ParseDotted parses a dotted notation string like "3" or "3.2" into
// a chapter index (1-based) and scene position (1-based, 0 if omitted).
func ParseDotted(s string) (chapter, position int, err error) {
	parts := strings.SplitN(s, ".", 2)
	chapter, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid chapter in %q: %w", s, err)
	}
	if chapter < 1 {
		return 0, 0, fmt.Errorf("chapter must be >= 1, got %d", chapter)
	}
	if len(parts) == 2 {
		position, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, fmt.Errorf("invalid position in %q: %w", s, err)
		}
		if position < 1 {
			return 0, 0, fmt.Errorf("position must be >= 1, got %d", position)
		}
	}
	return chapter, position, nil
}

// ResolveSlug looks up the scene slug at a given chapter index and position
// in the book. Both chapter and position are 1-based.
func ResolveSlug(book *binder.Book, chapter, position int) (string, error) {
	idx := chapter - 1
	if idx < 0 || idx >= len(book.Chapters) {
		return "", fmt.Errorf("chapter %d is out of range (1-%d)", chapter, len(book.Chapters))
	}
	ch := book.Chapters[idx]
	pos := position - 1
	if pos < 0 || pos >= len(ch.Scenes) {
		return "", fmt.Errorf("position %d is out of range (1-%d) in chapter %d", position, len(ch.Scenes), chapter)
	}
	return ch.Scenes[pos], nil
}

// ParseMoveArgs converts dotted notation from/to arguments into MoveOptions.
// from must include a position (e.g. "3.2"). to may be chapter-only ("4" = append)
// or include position ("4.1"). Empty to means append to the same chapter.
func ParseMoveArgs(from, to string) (*MoveOptions, error) {
	fromChapter, fromPos, err := ParseDotted(from)
	if err != nil {
		return nil, fmt.Errorf("invalid source: %w", err)
	}
	if fromPos == 0 {
		return nil, fmt.Errorf("source must include a scene position (e.g. %d.1)", fromChapter)
	}

	opts := &MoveOptions{
		ChapterIndex: fromChapter,
		FromPosition: fromPos,
	}

	if to == "" {
		// Append to same chapter
		return opts, nil
	}

	toChapter, toPos, err := ParseDotted(to)
	if err != nil {
		return nil, fmt.Errorf("invalid destination: %w", err)
	}

	if toChapter != fromChapter {
		opts.To = toChapter
	}
	opts.ToPosition = toPos

	return opts, nil
}
