package scene

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/poiesic/nib/internal/bookio"
	"github.com/poiesic/nib/internal/state"
	"github.com/poiesic/nib/internal/storydb"
)

// AddOptions configures how a scene is added to a chapter.
type AddOptions struct {
	ChapterIndex int    // 1-based chapter index
	Slug         string // scene slug (filename without .md)
	At           int    // 1-based insertion position; 0 = append
}

// AddResult holds information about a successfully added scene.
type AddResult struct {
	ChapterIndex int    // 1-based chapter index
	Slug         string // scene slug
	Position     int    // 1-based position within chapter
}

// Add inserts a scene slug into a chapter's scene list and creates the scene file.
func Add(opts AddOptions) (*AddResult, error) {
	if opts.Slug == "" {
		return nil, fmt.Errorf("scene slug must not be empty")
	}
	opts.Slug = strings.TrimSuffix(opts.Slug, ".md")

	projectRoot, fm, book, err := bookio.Load()
	if err != nil {
		return nil, err
	}

	idx := opts.ChapterIndex - 1
	if idx < 0 || idx >= len(book.Chapters) {
		return nil, fmt.Errorf("chapter %d is out of range (1-%d)", opts.ChapterIndex, len(book.Chapters))
	}

	ch := &book.Chapters[idx]
	for _, s := range ch.Scenes {
		if s == opts.Slug {
			return nil, fmt.Errorf("scene %q already exists in chapter %d", opts.Slug, opts.ChapterIndex)
		}
	}

	var position int
	if opts.At == 0 {
		ch.Scenes = append(ch.Scenes, opts.Slug)
		position = len(ch.Scenes)
	} else {
		pos := opts.At - 1
		if pos < 0 || pos > len(ch.Scenes) {
			return nil, fmt.Errorf("position %d is out of range (1-%d)", opts.At, len(ch.Scenes)+1)
		}
		ch.Scenes = append(ch.Scenes[:pos], append([]string{opts.Slug}, ch.Scenes[pos:]...)...)
		position = opts.At
	}

	// Create scene file
	sceneDir := book.BaseDir
	if ch.Subdir != "" {
		sceneDir = filepath.Join(book.BaseDir, ch.Subdir)
	}
	if !filepath.IsAbs(sceneDir) {
		sceneDir = filepath.Join(projectRoot, sceneDir)
	}
	if err := os.MkdirAll(sceneDir, 0755); err != nil {
		return nil, fmt.Errorf("creating scene directory: %w", err)
	}
	scenePath := filepath.Join(sceneDir, opts.Slug+".md")
	if _, err := os.Stat(scenePath); os.IsNotExist(err) {
		if err := os.WriteFile(scenePath, []byte{}, 0644); err != nil {
			return nil, fmt.Errorf("creating scene file: %w", err)
		}
	}

	if err := bookio.Save(projectRoot, fm, book); err != nil {
		return nil, err
	}

	return &AddResult{
		ChapterIndex: opts.ChapterIndex,
		Slug:         opts.Slug,
		Position:     position,
	}, nil
}

// RemoveOptions configures how a scene is removed from a chapter.
type RemoveOptions struct {
	ChapterIndex int    // 1-based chapter index
	Slug         string // scene slug to remove
}

// Remove deletes a scene slug from a chapter's scene list.
// The scene file remains on disk.
func Remove(opts RemoveOptions) error {
	projectRoot, fm, book, err := bookio.Load()
	if err != nil {
		return err
	}

	idx := opts.ChapterIndex - 1
	if idx < 0 || idx >= len(book.Chapters) {
		return fmt.Errorf("chapter %d is out of range (1-%d)", opts.ChapterIndex, len(book.Chapters))
	}

	ch := &book.Chapters[idx]
	found := -1
	for i, s := range ch.Scenes {
		if s == opts.Slug {
			found = i
			break
		}
	}
	if found == -1 {
		return fmt.Errorf("scene %q not found in chapter %d", opts.Slug, opts.ChapterIndex)
	}

	ch.Scenes = append(ch.Scenes[:found], ch.Scenes[found+1:]...)

	return bookio.Save(projectRoot, fm, book)
}

// SceneInfo holds display information about a single scene.
type SceneInfo struct {
	Index int    // 1-based position within chapter
	Slug  string // scene slug
}

// ChapterScenes groups scene info by chapter.
type ChapterScenes struct {
	ChapterIndex int
	Heading      string
	IsInterlude  bool
	Scenes       []SceneInfo
}

// ListOptions configures which chapters to list scenes for.
type ListOptions struct {
	ChapterIndex int // 1-based; 0 = all chapters
}

// List returns scene information grouped by chapter.
func List(opts ListOptions) ([]ChapterScenes, error) {
	_, _, book, err := bookio.Load()
	if err != nil {
		return nil, err
	}

	if opts.ChapterIndex != 0 {
		idx := opts.ChapterIndex - 1
		if idx < 0 || idx >= len(book.Chapters) {
			return nil, fmt.Errorf("chapter %d is out of range (1-%d)", opts.ChapterIndex, len(book.Chapters))
		}
	}

	var groups []ChapterScenes
	i := 0
	for ic := range book.GetChapters() {
		chIdx := i + 1
		i++
		if opts.ChapterIndex != 0 && chIdx != opts.ChapterIndex {
			continue
		}

		ch := book.Chapters[chIdx-1]
		var scenes []SceneInfo
		for j, slug := range ch.Scenes {
			scenes = append(scenes, SceneInfo{Index: j + 1, Slug: slug})
		}

		groups = append(groups, ChapterScenes{
			ChapterIndex: chIdx,
			Heading:      ic.Heading,
			IsInterlude:  ch.Interlude,
			Scenes:       scenes,
		})
	}

	return groups, nil
}

// MoveOptions configures how a scene is moved.
type MoveOptions struct {
	ChapterIndex int // 1-based source chapter (--chapter flag)
	FromPosition int // 1-based source scene position (first positional arg)
	To           int // 1-based destination chapter (--to flag; 0 = same chapter)
	ToPosition   int // 1-based destination position (second positional arg)
}

// Move removes a scene from one position and inserts it at another,
// within the same chapter or across chapters.
func Move(opts MoveOptions) error {
	projectRoot, fm, book, err := bookio.Load()
	if err != nil {
		return err
	}

	srcIdx := opts.ChapterIndex - 1
	if srcIdx < 0 || srcIdx >= len(book.Chapters) {
		return fmt.Errorf("source chapter %d is out of range (1-%d)", opts.ChapterIndex, len(book.Chapters))
	}

	dstChapter := opts.ChapterIndex
	if opts.To != 0 {
		dstChapter = opts.To
	}
	dstIdx := dstChapter - 1
	if dstIdx < 0 || dstIdx >= len(book.Chapters) {
		return fmt.Errorf("destination chapter %d is out of range (1-%d)", dstChapter, len(book.Chapters))
	}

	src := &book.Chapters[srcIdx]
	dst := &book.Chapters[dstIdx]

	if src.Subdir != dst.Subdir {
		return fmt.Errorf("cannot move scene between chapters with different subdirs (%q vs %q)", src.Subdir, dst.Subdir)
	}

	// Validate and remove from source by position
	fromPos := opts.FromPosition - 1
	if fromPos < 0 || fromPos >= len(src.Scenes) {
		return fmt.Errorf("source position %d is out of range (1-%d)", opts.FromPosition, len(src.Scenes))
	}
	slug := src.Scenes[fromPos]
	src.Scenes = append(src.Scenes[:fromPos], src.Scenes[fromPos+1:]...)

	// Insert into destination at position
	if opts.ToPosition == 0 {
		dst.Scenes = append(dst.Scenes, slug)
	} else {
		toPos := opts.ToPosition - 1
		if toPos < 0 || toPos > len(dst.Scenes) {
			return fmt.Errorf("destination position %d is out of range (1-%d)", opts.ToPosition, len(dst.Scenes)+1)
		}
		dst.Scenes = append(dst.Scenes[:toPos], append([]string{slug}, dst.Scenes[toPos:]...)...)
	}

	return bookio.Save(projectRoot, fm, book)
}

// RenameOptions configures how a scene is renamed.
type RenameOptions struct {
	OldSlug string
	NewSlug string
}

// Rename changes a scene's slug in book.yaml, renames the file on disk,
// updates storydb records if present, and updates focus if the renamed
// scene is currently focused.
func Rename(opts RenameOptions) error {
	opts.OldSlug = strings.TrimSuffix(opts.OldSlug, ".md")
	opts.NewSlug = strings.TrimSuffix(opts.NewSlug, ".md")

	if opts.OldSlug == "" {
		return fmt.Errorf("old slug must not be empty")
	}
	if opts.NewSlug == "" {
		return fmt.Errorf("new slug must not be empty")
	}
	if opts.OldSlug == opts.NewSlug {
		return fmt.Errorf("old and new slugs are the same")
	}

	projectRoot, fm, book, err := bookio.Load()
	if err != nil {
		return err
	}

	// Find the old slug and check the new slug doesn't already exist
	foundChapter := -1
	foundPos := -1
	for i, ch := range book.Chapters {
		for j, s := range ch.Scenes {
			if s == opts.NewSlug {
				return fmt.Errorf("scene %q already exists in chapter %d", opts.NewSlug, i+1)
			}
			if s == opts.OldSlug {
				foundChapter = i
				foundPos = j
			}
		}
	}
	if foundChapter == -1 {
		return fmt.Errorf("scene %q not found in book.yaml", opts.OldSlug)
	}

	// Replace slug in book
	book.Chapters[foundChapter].Scenes[foundPos] = opts.NewSlug

	// Rename file on disk
	ch := book.Chapters[foundChapter]
	sceneDir := book.BaseDir
	if ch.Subdir != "" {
		sceneDir = filepath.Join(book.BaseDir, ch.Subdir)
	}
	if !filepath.IsAbs(sceneDir) {
		sceneDir = filepath.Join(projectRoot, sceneDir)
	}
	oldPath := filepath.Join(sceneDir, opts.OldSlug+".md")
	newPath := filepath.Join(sceneDir, opts.NewSlug+".md")
	if _, err := os.Stat(oldPath); err == nil {
		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("renaming file: %w", err)
		}
	}

	// Save book.yaml
	if err := bookio.Save(projectRoot, fm, book); err != nil {
		return err
	}

	// Update focus if the renamed scene is currently focused
	st, err := state.Load(projectRoot)
	if err == nil && st.Focus != nil && st.Focus.Scene == opts.OldSlug {
		st.Focus.Scene = opts.NewSlug
		_ = state.Save(projectRoot, st)
	}

	// Update storydb if it exists
	dbDir := filepath.Join(projectRoot, "storydb")
	if _, err := os.Stat(dbDir); err == nil {
		db, err := storydb.Open(projectRoot)
		if err == nil {
			_ = db.RenameScene(opts.OldSlug, opts.NewSlug)
			db.Close()
		}
	}

	return nil
}

// FormatList formats scene groups for terminal output.
func FormatList(groups []ChapterScenes) string {
	if len(groups) == 0 {
		return "No chapters\n"
	}

	var b strings.Builder
	for _, g := range groups {
		heading := g.Heading
		if heading == "" {
			heading = "(interlude)"
		}
		fmt.Fprintf(&b, "[%d] %s\n", g.ChapterIndex, heading)
		if len(g.Scenes) == 0 {
			fmt.Fprintf(&b, "    (no scenes)\n")
		} else {
			for _, s := range g.Scenes {
				fmt.Fprintf(&b, "    [%d] %s\n", s.Index, s.Slug)
			}
		}
	}
	return b.String()
}

// CommandRunner creates an exec.Cmd. Injected for testing.
type CommandRunner func(name string, args ...string) *exec.Cmd

var ErrEditorNotSet = errors.New("no editor set; set NIB_EDITOR, VISUAL, or EDITOR")

// EditOptions configures how a scene is opened in an editor.
type EditOptions struct {
	Slug   string
	Runner CommandRunner // nil = exec.Command
}

// Edit opens a scene file in the user's preferred editor.
// If slug is empty, falls back to the focused scene.
func Edit(opts EditOptions) error {
	projectRoot, _, book, err := bookio.Load()
	if err != nil {
		return err
	}

	slug, err := resolveSlugOrFocus(projectRoot, book, opts.Slug)
	if err != nil {
		return err
	}
	opts.Slug = slug

	// Find the scene and its chapter to resolve the file path
	var subdir string
	found := false
	for _, ch := range book.Chapters {
		for _, s := range ch.Scenes {
			if s == opts.Slug {
				subdir = ch.Subdir
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		return fmt.Errorf("scene %q not found in book.yaml; use nib scene add to add the scene", opts.Slug)
	}

	sceneDir := book.BaseDir
	if subdir != "" {
		sceneDir = filepath.Join(book.BaseDir, subdir)
	}
	if !filepath.IsAbs(sceneDir) {
		sceneDir = filepath.Join(projectRoot, sceneDir)
	}
	scenePath := filepath.Join(sceneDir, opts.Slug+".md")

	editor := editorFromEnv()
	if editor == "" {
		return ErrEditorNotSet
	}

	runner := opts.Runner
	if runner == nil {
		runner = exec.Command
	}

	parts := strings.Fields(editor)
	cmd := runner(parts[0], append(parts[1:], scenePath)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func editorFromEnv() string {
	for _, key := range []string{"NIB_EDITOR", "VISUAL", "EDITOR"} {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return ""
}
