package manuscript

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/poiesic/binder"
	"github.com/poiesic/nib/internal/scene"
)

// ResolvedScene identifies a scene by slug with its interlude flag.
type ResolvedScene struct {
	Slug      string
	Interlude bool
}

// SceneRef identifies a scene by chapter and position.
// Position=0 means all scenes in the chapter.
type SceneRef struct {
	Chapter  int
	Position int
}

// RangeSpec describes a parsed range expression.
// Kind is "list" (explicit refs) or "range" (start/end pair).
type RangeSpec struct {
	Kind string     // "list" or "range"
	Refs []SceneRef // list: all refs; range: exactly 2 (start, end)
}

// ParseRange parses a range string into a RangeSpec.
// Supported formats:
//
//	"3"         -> list with {3, 0}
//	"3.2"       -> list with {3, 2}
//	"1-3"       -> range with {1, 0} and {3, 0}
//	"1.1-3.2"   -> range with {1, 1} and {3, 2}
//	"1,3,5"     -> list with three chapter refs
//	"1.1,2.3"   -> list with two dotted refs
func ParseRange(input string) (RangeSpec, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return RangeSpec{}, fmt.Errorf("empty range")
	}

	if strings.Contains(input, ",") {
		return parseList(input)
	}
	if strings.Contains(input, "-") {
		return parseRange(input)
	}
	// Single value
	ref, err := parseRef(input)
	if err != nil {
		return RangeSpec{}, err
	}
	return RangeSpec{Kind: "list", Refs: []SceneRef{ref}}, nil
}

func parseRef(s string) (SceneRef, error) {
	ch, pos, err := scene.ParseDotted(s)
	if err != nil {
		return SceneRef{}, err
	}
	return SceneRef{Chapter: ch, Position: pos}, nil
}

func isDotted(ref SceneRef) bool {
	return ref.Position > 0
}

func parseList(input string) (RangeSpec, error) {
	parts := strings.Split(input, ",")
	if len(parts) < 2 {
		return RangeSpec{}, fmt.Errorf("invalid list: %q", input)
	}

	refs := make([]SceneRef, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return RangeSpec{}, fmt.Errorf("empty element in list: %q", input)
		}
		ref, err := parseRef(p)
		if err != nil {
			return RangeSpec{}, err
		}
		refs = append(refs, ref)
	}

	// All must be consistently dotted or all chapter-only
	allDotted := isDotted(refs[0])
	for _, ref := range refs[1:] {
		if isDotted(ref) != allDotted {
			return RangeSpec{}, fmt.Errorf("mixed dotted and chapter-only refs in list: %q", input)
		}
	}

	return RangeSpec{Kind: "list", Refs: refs}, nil
}

func parseRange(input string) (RangeSpec, error) {
	parts := strings.SplitN(input, "-", 2)
	if len(parts) != 2 {
		return RangeSpec{}, fmt.Errorf("invalid range: %q", input)
	}

	start, err := parseRef(strings.TrimSpace(parts[0]))
	if err != nil {
		return RangeSpec{}, fmt.Errorf("invalid range start: %w", err)
	}
	end, err := parseRef(strings.TrimSpace(parts[1]))
	if err != nil {
		return RangeSpec{}, fmt.Errorf("invalid range end: %w", err)
	}

	if isDotted(start) != isDotted(end) {
		return RangeSpec{}, fmt.Errorf("mixed dotted and chapter-only refs in range: %q", input)
	}

	// Validate end >= start
	if end.Chapter < start.Chapter {
		return RangeSpec{}, fmt.Errorf("range end chapter %d is before start chapter %d", end.Chapter, start.Chapter)
	}
	if end.Chapter == start.Chapter && end.Position > 0 && start.Position > 0 && end.Position < start.Position {
		return RangeSpec{}, fmt.Errorf("range end %d.%d is before start %d.%d", end.Chapter, end.Position, start.Chapter, start.Position)
	}

	return RangeSpec{Kind: "range", Refs: []SceneRef{start, end}}, nil
}

// ResolveScenePaths expands a RangeSpec into absolute file paths in narrative order.
func ResolveScenePaths(projectRoot string, book *binder.Book, spec RangeSpec) ([]string, error) {
	switch spec.Kind {
	case "list":
		return resolveList(projectRoot, book, spec.Refs)
	case "range":
		if len(spec.Refs) != 2 {
			return nil, fmt.Errorf("range spec must have exactly 2 refs, got %d", len(spec.Refs))
		}
		return resolveRange(projectRoot, book, spec.Refs[0], spec.Refs[1])
	default:
		return nil, fmt.Errorf("unknown range kind: %q", spec.Kind)
	}
}

// ResolveSlugs expands a RangeSpec into scene slugs with interlude flags in narrative order.
func ResolveSlugs(book *binder.Book, spec RangeSpec) ([]ResolvedScene, error) {
	switch spec.Kind {
	case "list":
		return resolveSlugList(book, spec.Refs)
	case "range":
		if len(spec.Refs) != 2 {
			return nil, fmt.Errorf("range spec must have exactly 2 refs, got %d", len(spec.Refs))
		}
		return resolveSlugRange(book, spec.Refs[0], spec.Refs[1])
	default:
		return nil, fmt.Errorf("unknown range kind: %q", spec.Kind)
	}
}

func resolveSlugList(book *binder.Book, refs []SceneRef) ([]ResolvedScene, error) {
	var scenes []ResolvedScene
	for _, ref := range refs {
		chIdx := ref.Chapter - 1
		if chIdx < 0 || chIdx >= len(book.Chapters) {
			return nil, fmt.Errorf("chapter %d is out of range (1-%d)", ref.Chapter, len(book.Chapters))
		}
		ch := book.Chapters[chIdx]

		if ref.Position == 0 {
			if len(ch.Scenes) == 0 {
				return nil, fmt.Errorf("chapter %d has no scenes", ref.Chapter)
			}
			for _, slug := range ch.Scenes {
				scenes = append(scenes, ResolvedScene{Slug: slug, Interlude: ch.Interlude})
			}
		} else {
			if ref.Position < 1 || ref.Position > len(ch.Scenes) {
				return nil, fmt.Errorf("position %d is out of range (1-%d) in chapter %d", ref.Position, len(ch.Scenes), ref.Chapter)
			}
			slug := ch.Scenes[ref.Position-1]
			scenes = append(scenes, ResolvedScene{Slug: slug, Interlude: ch.Interlude})
		}
	}
	if len(scenes) == 0 {
		return nil, fmt.Errorf("no scenes found for the given range")
	}
	return scenes, nil
}

func resolveSlugRange(book *binder.Book, start, end SceneRef) ([]ResolvedScene, error) {
	if start.Chapter < 1 || start.Chapter > len(book.Chapters) {
		return nil, fmt.Errorf("chapter %d is out of range (1-%d)", start.Chapter, len(book.Chapters))
	}
	if end.Chapter < 1 || end.Chapter > len(book.Chapters) {
		return nil, fmt.Errorf("chapter %d is out of range (1-%d)", end.Chapter, len(book.Chapters))
	}

	if isDotted(start) {
		return resolveSlugDottedRange(book, start, end)
	}
	return resolveSlugChapterRange(book, start.Chapter, end.Chapter)
}

func resolveSlugChapterRange(book *binder.Book, startCh, endCh int) ([]ResolvedScene, error) {
	var scenes []ResolvedScene
	for ch := startCh; ch <= endCh; ch++ {
		chapter := book.Chapters[ch-1]
		for _, slug := range chapter.Scenes {
			scenes = append(scenes, ResolvedScene{Slug: slug, Interlude: chapter.Interlude})
		}
	}
	if len(scenes) == 0 {
		return nil, fmt.Errorf("no scenes found in chapters %d-%d", startCh, endCh)
	}
	return scenes, nil
}

func resolveSlugDottedRange(book *binder.Book, start, end SceneRef) ([]ResolvedScene, error) {
	startCh := book.Chapters[start.Chapter-1]
	if start.Position < 1 || start.Position > len(startCh.Scenes) {
		return nil, fmt.Errorf("position %d is out of range (1-%d) in chapter %d", start.Position, len(startCh.Scenes), start.Chapter)
	}
	endCh := book.Chapters[end.Chapter-1]
	if end.Position < 1 || end.Position > len(endCh.Scenes) {
		return nil, fmt.Errorf("position %d is out of range (1-%d) in chapter %d", end.Position, len(endCh.Scenes), end.Chapter)
	}

	var scenes []ResolvedScene
	for ch := start.Chapter; ch <= end.Chapter; ch++ {
		chapter := book.Chapters[ch-1]

		startPos := 0
		endPos := len(chapter.Scenes) - 1

		if ch == start.Chapter {
			startPos = start.Position - 1
		}
		if ch == end.Chapter {
			endPos = end.Position - 1
		}

		for i := startPos; i <= endPos; i++ {
			scenes = append(scenes, ResolvedScene{Slug: chapter.Scenes[i], Interlude: chapter.Interlude})
		}
	}
	if len(scenes) == 0 {
		return nil, fmt.Errorf("no scenes found in range %d.%d-%d.%d", start.Chapter, start.Position, end.Chapter, end.Position)
	}
	return scenes, nil
}

func resolveList(projectRoot string, book *binder.Book, refs []SceneRef) ([]string, error) {
	var paths []string
	for _, ref := range refs {
		chIdx := ref.Chapter - 1
		if chIdx < 0 || chIdx >= len(book.Chapters) {
			return nil, fmt.Errorf("chapter %d is out of range (1-%d)", ref.Chapter, len(book.Chapters))
		}
		ch := book.Chapters[chIdx]

		if ref.Position == 0 {
			// All scenes in chapter
			if len(ch.Scenes) == 0 {
				return nil, fmt.Errorf("chapter %d has no scenes", ref.Chapter)
			}
			for _, slug := range ch.Scenes {
				paths = append(paths, scenePath(projectRoot, book.BaseDir, ch.Subdir, slug))
			}
		} else {
			if ref.Position < 1 || ref.Position > len(ch.Scenes) {
				return nil, fmt.Errorf("position %d is out of range (1-%d) in chapter %d", ref.Position, len(ch.Scenes), ref.Chapter)
			}
			slug := ch.Scenes[ref.Position-1]
			paths = append(paths, scenePath(projectRoot, book.BaseDir, ch.Subdir, slug))
		}
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no scenes found for the given range")
	}
	return paths, nil
}

func resolveRange(projectRoot string, book *binder.Book, start, end SceneRef) ([]string, error) {
	// Validate chapter bounds
	if start.Chapter < 1 || start.Chapter > len(book.Chapters) {
		return nil, fmt.Errorf("chapter %d is out of range (1-%d)", start.Chapter, len(book.Chapters))
	}
	if end.Chapter < 1 || end.Chapter > len(book.Chapters) {
		return nil, fmt.Errorf("chapter %d is out of range (1-%d)", end.Chapter, len(book.Chapters))
	}

	if isDotted(start) {
		return resolveDottedRange(projectRoot, book, start, end)
	}
	return resolveChapterRange(projectRoot, book, start.Chapter, end.Chapter)
}

func resolveChapterRange(projectRoot string, book *binder.Book, startCh, endCh int) ([]string, error) {
	var paths []string
	for ch := startCh; ch <= endCh; ch++ {
		chIdx := ch - 1
		chapter := book.Chapters[chIdx]
		for _, slug := range chapter.Scenes {
			paths = append(paths, scenePath(projectRoot, book.BaseDir, chapter.Subdir, slug))
		}
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no scenes found in chapters %d-%d", startCh, endCh)
	}
	return paths, nil
}

func resolveDottedRange(projectRoot string, book *binder.Book, start, end SceneRef) ([]string, error) {
	// Validate positions
	startCh := book.Chapters[start.Chapter-1]
	if start.Position < 1 || start.Position > len(startCh.Scenes) {
		return nil, fmt.Errorf("position %d is out of range (1-%d) in chapter %d", start.Position, len(startCh.Scenes), start.Chapter)
	}
	endCh := book.Chapters[end.Chapter-1]
	if end.Position < 1 || end.Position > len(endCh.Scenes) {
		return nil, fmt.Errorf("position %d is out of range (1-%d) in chapter %d", end.Position, len(endCh.Scenes), end.Chapter)
	}

	var paths []string
	for ch := start.Chapter; ch <= end.Chapter; ch++ {
		chIdx := ch - 1
		chapter := book.Chapters[chIdx]

		startPos := 0
		endPos := len(chapter.Scenes) - 1

		if ch == start.Chapter {
			startPos = start.Position - 1
		}
		if ch == end.Chapter {
			endPos = end.Position - 1
		}

		for i := startPos; i <= endPos; i++ {
			paths = append(paths, scenePath(projectRoot, book.BaseDir, chapter.Subdir, chapter.Scenes[i]))
		}
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no scenes found in range %d.%d-%d.%d", start.Chapter, start.Position, end.Chapter, end.Position)
	}
	return paths, nil
}

func scenePath(projectRoot, baseDir, subdir, slug string) string {
	dir := baseDir
	if subdir != "" {
		dir = filepath.Join(baseDir, subdir)
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(projectRoot, dir)
	}
	return filepath.Join(dir, slug+".md")
}
