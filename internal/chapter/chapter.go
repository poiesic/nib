package chapter

import (
	"fmt"
	"strings"

	"github.com/poiesic/binder"
	"github.com/poiesic/nib/internal/bookio"
)

// AddOptions configures how a chapter is added.
type AddOptions struct {
	Name      string
	Interlude bool
	At        int // 1-based insertion position; 0 = append
}

// Add inserts a new chapter into book.yaml.
func Add(opts AddOptions) error {
	projectRoot, fm, book, err := bookio.Load()
	if err != nil {
		return err
	}

	ch := binder.Chapter{
		Name:      opts.Name,
		Interlude: opts.Interlude,
		Scenes:    []string{},
	}

	if opts.At == 0 {
		book.Chapters = append(book.Chapters, ch)
	} else {
		idx := opts.At - 1
		if idx < 0 || idx > len(book.Chapters) {
			return fmt.Errorf("position %d is out of range (1-%d)", opts.At, len(book.Chapters)+1)
		}
		book.Chapters = append(book.Chapters[:idx], append([]binder.Chapter{ch}, book.Chapters[idx:]...)...)
	}

	return bookio.Save(projectRoot, fm, book)
}

// ChapterInfo holds display information about a chapter.
type ChapterInfo struct {
	Index       int
	Heading     string
	SceneCount  int
	IsInterlude bool
}

// List returns info about all chapters in book.yaml.
func List() ([]ChapterInfo, error) {
	_, _, book, err := bookio.Load()
	if err != nil {
		return nil, err
	}

	var infos []ChapterInfo
	i := 0
	for ic := range book.GetChapters() {
		info := ChapterInfo{
			Index:       i + 1,
			Heading:     ic.Heading,
			SceneCount:  len(book.Chapters[i].Scenes),
			IsInterlude: book.Chapters[i].Interlude,
		}
		infos = append(infos, info)
		i++
	}

	return infos, nil
}

// Remove deletes a chapter from book.yaml by 1-based index.
// Scene files remain on disk.
func Remove(index int) error {
	projectRoot, fm, book, err := bookio.Load()
	if err != nil {
		return err
	}

	idx := index - 1
	if idx < 0 || idx >= len(book.Chapters) {
		return fmt.Errorf("chapter %d is out of range (1-%d)", index, len(book.Chapters))
	}

	book.Chapters = append(book.Chapters[:idx], book.Chapters[idx+1:]...)

	return bookio.Save(projectRoot, fm, book)
}

// Name sets the name of a chapter by 1-based index.
func Name(index int, name string) error {
	projectRoot, fm, book, err := bookio.Load()
	if err != nil {
		return err
	}

	idx := index - 1
	if idx < 0 || idx >= len(book.Chapters) {
		return fmt.Errorf("chapter %d is out of range (1-%d)", index, len(book.Chapters))
	}

	book.Chapters[idx].Name = name
	return bookio.Save(projectRoot, fm, book)
}

// ClearName removes the name from a chapter by 1-based index.
func ClearName(index int) error {
	return Name(index, "")
}

// Move relocates a chapter from one position to another in book.yaml.
// Both from and to are 1-based indices.
func Move(from, to int) error {
	projectRoot, fm, book, err := bookio.Load()
	if err != nil {
		return err
	}

	fromIdx := from - 1
	if fromIdx < 0 || fromIdx >= len(book.Chapters) {
		return fmt.Errorf("source chapter %d is out of range (1-%d)", from, len(book.Chapters))
	}

	toIdx := to - 1
	if toIdx < 0 || toIdx >= len(book.Chapters) {
		return fmt.Errorf("destination %d is out of range (1-%d)", to, len(book.Chapters))
	}

	if fromIdx == toIdx {
		return nil
	}

	ch := book.Chapters[fromIdx]
	book.Chapters = append(book.Chapters[:fromIdx], book.Chapters[fromIdx+1:]...)
	book.Chapters = append(book.Chapters[:toIdx], append([]binder.Chapter{ch}, book.Chapters[toIdx:]...)...)

	return bookio.Save(projectRoot, fm, book)
}

// FormatList formats chapter info for terminal output.
func FormatList(chapters []ChapterInfo) string {
	if len(chapters) == 0 {
		return "No chapters\n"
	}

	var b strings.Builder
	for _, ch := range chapters {
		heading := ch.Heading
		if heading == "" {
			heading = "(interlude)"
		}
		fmt.Fprintf(&b, "%d. %s (%d scenes)\n", ch.Index, heading, ch.SceneCount)
	}
	return b.String()
}
